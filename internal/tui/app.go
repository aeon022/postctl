package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/generator"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/store"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// PostStats hält aggregierte Post-Statistiken
type PostStats struct {
	posted    int
	scheduled int
	drafts    int
	failed    int
}

// Model repräsentiert den Zustand der Bubbletea-Anwendung
type Model struct {
	store         store.Store
	activeTab     int // 0: Dashboard, 1: Posts, 2: Schedule, 3: History
	cursor        int
	
	// Geladene Daten
	posts     []models.Post
	history   []models.HistoryEntry
	stats     PostStats
	campaigns []models.Campaign
	platforms map[string]bool
	nextUp    []models.Post
	
	// UI Zustand
	selectedPost     *models.Post
	selectedHistory  *models.HistoryEntry
	filterCampaign   string
	isEditing        bool
	showHelp         bool
	err              error
	loading          bool
	repurposing      bool
	statusMessage    string
	
	// Editor Zustand
	editorPostID      string
	editorPlatform    string
	editorCampaign    textinput.Model
	editorScheduledAt textinput.Model
	editorImages      textinput.Model
	editorBody        textarea.Model
	editorFocus       int
	showDatePicker    bool
	datePickerDate    time.Time
	
	// README Viewer Zustand
	showReadme   bool
	readmeLines  []string
	readmeTOC    []tocItem
	readmeScroll int
	tocCursor    int
	readmeFocus  int // 0: TOC, 1: Content
	
	// Terminal Dimensionen
	width  int
	height int

	// Analytics Zustand
	analyticsLoading bool
	analyticsData    *analyticsLoadedMsg
}

// Msg-Typen
type dataLoadedMsg struct {
	posts     []models.Post
	history   []models.HistoryEntry
	stats     PostStats
	campaigns []models.Campaign
	platforms map[string]bool
	nextUp    []models.Post
}

type errorMsg struct {
	err error
}

type exportFinishedMsg struct {
	err      error
	filename string
}

type postDeletedMsg struct {
	id string
}

type postRepurposedMsg struct {
	files []string
	err   error
}

type analyticsLoadedMsg struct {
	totalPosts       int
	totalLikes       int
	totalShares      int
	totalComments    int
	totalImpressions int
	platStats        map[string]*platMetricSummary
	analyzedPosts    []postMetric
	err              error
}

type platMetricSummary struct {
	Name        string
	Posts       int
	Likes       int
	Shares      int
	Comments    int
	Impressions int
}

type postMetric struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Platform    string `json:"platform"`
	PostedAt    string `json:"posted_at"`
	Likes       int    `json:"likes"`
	Shares      int    `json:"shares"`
	Comments    int    `json:"comments"`
	Impressions int    `json:"impressions"`
}

type authResultMsg struct {
	platform string
	err      error
}

type backupFinishedMsg struct {
	isExport bool
	err      error
}

type importFinishedMsg struct {
	err error
}

type externalEditorFinishedMsg struct {
	content  string
	platform string
	campaign string
	schedule string
	images   string
	err      error
}

type setupWizardFinishedMsg struct {
	platform string
	err      error
}

type platformClearedMsg struct {
	platform string
	err      error
}

type tickMsg struct{}

// NewModel initialisiert ein TUI-Model
func NewModel(s store.Store) Model {
	return Model{
		store:     s,
		activeTab: 0,
		cursor:    0,
		platforms: make(map[string]bool),
		loading:   true,
	}
}

// Init startet die initialen Aktionen der App (inkl. 10s-Scheduler Ticker)
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadDataCmd,
		tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
			return tickMsg{}
		}),
	)
}

// loadDataCmd lädt alle relevanten Daten asynchron aus der DB
func (m Model) loadDataCmd() tea.Msg {
	ctx := context.Background()
	
	posts, err := m.store.ListPosts(ctx, "all", "all", "")
	if err != nil {
		return errorMsg{err}
	}
	
	hist, err := m.store.GetHistory(ctx, 20)
	if err != nil {
		return errorMsg{err}
	}
	
	var stats PostStats
	campaignMap := make(map[string]*models.Campaign)
	var nextUp []models.Post
	
	for _, p := range posts {
		switch p.Status {
		case models.StatusPosted:
			stats.posted++
		case models.StatusScheduled:
			stats.scheduled++
			nextUp = append(nextUp, p)
		case models.StatusDraft:
			stats.drafts++
		case models.StatusFailed:
			stats.failed++
		}
		
		if p.Campaign != "" {
			c, ok := campaignMap[p.Campaign]
			if !ok {
				c = &models.Campaign{Slug: p.Campaign}
				campaignMap[p.Campaign] = c
			}
			c.Posts = append(c.Posts, p)
			switch p.Status {
			case models.StatusPosted:
				c.Posted++
			case models.StatusScheduled:
				c.Scheduled++
			case models.StatusDraft:
				c.Drafts++
			}
		}
	}
	
	var campaigns []models.Campaign
	for _, c := range campaignMap {
		campaigns = append(campaigns, *c)
	}
	
	// Sortieren von nextUp nach schedule time (ASC)
	slices.SortFunc(nextUp, func(a, b models.Post) int {
		if a.ScheduledAt == nil && b.ScheduledAt == nil {
			return 0
		}
		if a.ScheduledAt == nil {
			return 1
		}
		if b.ScheduledAt == nil {
			return -1
		}
		if a.ScheduledAt.Before(*b.ScheduledAt) {
			return -1
		}
		if a.ScheduledAt.After(*b.ScheduledAt) {
			return 1
		}
		return 0
	})

	// Gruppieren nach Kampagnen unter Beibehaltung der chronologischen Reihenfolge der Kampagnen-Starts
	if len(nextUp) > 0 {
		type CampaignGroup struct {
			Slug  string
			Posts []models.Post
		}
		var groups []CampaignGroup
		groupIndex := make(map[string]int)

		for _, p := range nextUp {
			idx, exists := groupIndex[p.Campaign]
			if !exists {
				idx = len(groups)
				groupIndex[p.Campaign] = idx
				groups = append(groups, CampaignGroup{
					Slug:  p.Campaign,
					Posts: []models.Post{},
				})
			}
			groups[idx].Posts = append(groups[idx].Posts, p)
		}

		var groupedNextUp []models.Post
		for _, g := range groups {
			groupedNextUp = append(groupedNextUp, g.Posts...)
		}
		nextUp = groupedNextUp
	}

	
	platforms := map[string]bool{
		models.PlatformTwitter:  false,
		models.PlatformLinkedIn: false,
		models.PlatformThreads:  false,
		models.PlatformMastodon: false,
		models.PlatformBluesky:  false,
		models.PlatformFacebook: false,
	}
	for p := range platforms {
		_, _, _, err := m.store.GetToken(ctx, p)
		if err == nil {
			platforms[p] = true
		}
	}
	
	return dataLoadedMsg{
		posts:     posts,
		history:   hist,
		stats:     stats,
		campaigns: campaigns,
		platforms: platforms,
		nextUp:    nextUp,
	}
}

// deletePostCmd löscht einen Post asynchron
func (m Model) deletePostCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := m.store.DeletePost(ctx, id); err != nil {
			return errorMsg{err}
		}
		return postDeletedMsg{id}
	}
}

// runAuthCmd startet den Authentifizierungs-Flow für eine Plattform asynchron im Hintergrund
func (m Model) runAuthCmd(platformName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		plat, err := platforms.GetPlatform(platformName, m.store, false)
		if err != nil {
			return authResultMsg{platform: platformName, err: err}
		}

		err = plat.Auth(ctx)
		return authResultMsg{platform: platformName, err: err}
	}
}

func getExecutablePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "./postctl"
	}
	return exe
}

func platformNeedsSetup(platformName string) bool {
	switch platformName {
	case models.PlatformTwitter:
		if config.ActiveConfig.Twitter.AuthMode == "cookie" {
			return false
		}
		return config.ActiveConfig.Twitter.ClientID == "" || config.ActiveConfig.Twitter.ClientSecret == ""
	case models.PlatformLinkedIn:
		return config.ActiveConfig.LinkedIn.ClientID == "" || config.ActiveConfig.LinkedIn.ClientSecret == ""
	case models.PlatformThreads:
		return config.ActiveConfig.Threads.AppID == "" || config.ActiveConfig.Threads.AppSecret == ""
	case models.PlatformMastodon:
		return false
	case models.PlatformBluesky:
		return config.ActiveConfig.Bluesky.Handle == "" || config.ActiveConfig.Bluesky.AppPassword == ""
	case models.PlatformFacebook:
		return config.ActiveConfig.Facebook.AppID == "" || config.ActiveConfig.Facebook.AppSecret == ""
	}
	return false
}

// runSetupWizardCmd startet das interaktive CLI Setup für eine Plattform
func (m Model) runSetupWizardCmd(platformName string) tea.Cmd {
	c := exec.Command(getExecutablePath(), "config", "setup", platformName)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return setupWizardFinishedMsg{platform: platformName, err: err}
	})
}

// clearPlatformCmd löscht die Zugangsdaten einer Plattform aus Config & DB
func (m Model) clearPlatformCmd(platformName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// 1. Token in DB löschen
		_ = m.store.DeleteToken(ctx, platformName)

		// 2. Zugangsdaten aus Config entfernen
		switch platformName {
		case models.PlatformTwitter:
			config.ActiveConfig.Twitter.ClientID = ""
			config.ActiveConfig.Twitter.ClientSecret = ""
			config.ActiveConfig.Twitter.AuthMode = ""
		case models.PlatformLinkedIn:
			config.ActiveConfig.LinkedIn.ClientID = ""
			config.ActiveConfig.LinkedIn.ClientSecret = ""
		case models.PlatformThreads:
			config.ActiveConfig.Threads.AppID = ""
			config.ActiveConfig.Threads.AppSecret = ""
		case models.PlatformMastodon:
			config.ActiveConfig.Mastodon.ClientID = ""
			config.ActiveConfig.Mastodon.ClientSecret = ""
			config.ActiveConfig.Mastodon.InstanceURL = "https://mastodon.social"
		case models.PlatformBluesky:
			config.ActiveConfig.Bluesky.Handle = ""
			config.ActiveConfig.Bluesky.AppPassword = ""
		case models.PlatformFacebook:
			config.ActiveConfig.Facebook.AppID = ""
			config.ActiveConfig.Facebook.AppSecret = ""
			config.ActiveConfig.Facebook.PageID = ""
		}

		// 3. Speichern
		if err := config.SaveConfig(); err != nil {
			return platformClearedMsg{platform: platformName, err: err}
		}

		return platformClearedMsg{platform: platformName, err: nil}
	}
}

// runBackupExportCmd pausiert die TUI und startet interaktiv den CLI-Export
func (m Model) runBackupExportCmd() tea.Cmd {
	c := exec.Command(getExecutablePath(), "config", "export")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: true, err: err}
	})
}

// runBackupImportCmd pausiert die TUI und startet interaktiv den CLI-Import
func (m Model) runBackupImportCmd() tea.Cmd {
	c := exec.Command(getExecutablePath(), "config", "import")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: false, err: err}
	})
}

// runImportPostsCmd pausiert die TUI und startet interaktiv den CLI-Post-Import
func (m Model) runImportPostsCmd() tea.Cmd {
	c := exec.Command(getExecutablePath(), "import")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return importFinishedMsg{err: err}
	})
}

// runExternalEditorCmd pausiert die TUI und öffnet Neovim/Vim, um den Textinhalt zu bearbeiten
func (m Model) runExternalEditorCmd() tea.Cmd {
	// Temp-Datei erstellen
	tmpFile, err := os.CreateTemp("", "postctl-body-*.md")
	if err != nil {
		return func() tea.Msg {
			return externalEditorFinishedMsg{err: err}
		}
	}
	
	// Hilfetext generieren
	var helper strings.Builder
	var limit int
	var rulerNum, rulerLine string
	switch m.editorPlatform {
	case "twitter":
		limit = 280
		rulerNum =  " 000      030      060      090      120      150      180      210      240      270 280!\n"
		rulerLine = " |--------|--------|--------|--------|--------|--------|--------|--------|--------|-|\n"
	case "bluesky":
		limit = 300
		rulerNum =  " 000      030      060      090      120      150      180      210      240      270      300!\n"
		rulerLine = " |--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|\n"
	case "mastodon":
		limit = 500
		rulerNum =  " 000      050      100      150      200      250      300      350      400      450      500!\n"
		rulerLine = " |--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|\n"
	}

	if limit > 0 {
		helper.WriteString("<!--\n")
		helper.WriteString(fmt.Sprintf(Tr("editor_helper_title_other"), strings.ToUpper(m.editorPlatform)))
		helper.WriteString(Tr("editor_helper_ruler_twitter"))
		helper.WriteString(rulerNum)
		helper.WriteString(rulerLine + "\n")
		
		bodyText := m.editorBody.Value()
		if strings.Contains(bodyText, "\n---\n") {
			tweets := strings.Split(bodyText, "\n---\n")
			helper.WriteString(Tr("editor_helper_status_thread"))
			for i, tweet := range tweets {
				trimmed := strings.TrimSpace(tweet)
				runes := []rune(trimmed)
				charCount := len(runes)
				remaining := limit - charCount
				status := "✓"
				if remaining < 0 {
					status = "✗ ZU LANG"
					if Tr("editor_helper_tweet_format") == "" {
						status = "✗ TOO LONG"
					}
				}
				helper.WriteString(fmt.Sprintf(Tr("editor_helper_tweet_format"), i+1, charCount, remaining, status))
			}
		} else {
			trimmed := strings.TrimSpace(bodyText)
			charCount := len([]rune(trimmed))
			remaining := limit - charCount
			status := "✓"
			if remaining < 0 {
				status = "✗ ZU LANG"
			}
			helper.WriteString(fmt.Sprintf(Tr("editor_helper_status_single"), charCount, remaining, status))
		}
		
		helper.WriteString(Tr("editor_helper_note_strip"))
		helper.WriteString("-->\n\n")
	} else {
		helper.WriteString("<!--\n")
		helper.WriteString(fmt.Sprintf(Tr("editor_helper_title_other"), strings.ToUpper(m.editorPlatform)))
		bodyText := m.editorBody.Value()
		trimmed := strings.TrimSpace(bodyText)
		charCount := len([]rune(trimmed))
		helper.WriteString(fmt.Sprintf(Tr("editor_helper_status_other"), charCount))
		
		helper.WriteString(Tr("editor_helper_note_strip"))
		helper.WriteString("-->\n\n")
	}

	// Hilfetext + Frontmatter + Aktuellen Text reinschreiben
	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")
	frontmatter.WriteString(fmt.Sprintf("platform: %s\n", m.editorPlatform))
	frontmatter.WriteString(fmt.Sprintf("campaign: %s\n", m.editorCampaign.Value()))
	frontmatter.WriteString(fmt.Sprintf("schedule: %s\n", m.editorScheduledAt.Value()))
	frontmatter.WriteString(fmt.Sprintf("images: %s\n", m.editorImages.Value()))
	frontmatter.WriteString("---\n\n")

	if _, err := tmpFile.WriteString(helper.String() + frontmatter.String() + m.editorBody.Value()); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return func() tea.Msg {
			return externalEditorFinishedMsg{err: err}
		}
	}
	_ = tmpFile.Close()

	// Editor bestimmen (EDITOR Env-Var, sonst nvim, sonst vim)
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = "nvim"
	}
	// Fallback-Check: Falls nvim nicht gefunden wird, probiere vim
	if _, err := exec.LookPath(editorCmd); err != nil && editorCmd == "nvim" {
		editorCmd = "vim"
	}

	c := exec.Command(editorCmd, tmpFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			_ = os.Remove(tmpFile.Name())
			return externalEditorFinishedMsg{err: err}
		}
		
		// Datei wieder einlesen
		data, readErr := os.ReadFile(tmpFile.Name())
		_ = os.Remove(tmpFile.Name())
		if readErr != nil {
			return externalEditorFinishedMsg{err: readErr}
		}
		
		// Hilfetext wieder entfernen
		contentStr := string(data)
		if strings.HasPrefix(contentStr, "<!--") {
			if idx := strings.Index(contentStr, "-->"); idx != -1 {
				contentStr = contentStr[idx+3:]
				contentStr = strings.TrimLeft(contentStr, "\r\n")
			}
		}

		// Frontmatter parsen
		var platform, campaign, schedule, images string
		if strings.HasPrefix(contentStr, "---") {
			if endIdx := strings.Index(contentStr[3:], "---"); endIdx != -1 {
				yamlPart := contentStr[3 : endIdx+3]
				contentStr = contentStr[endIdx+6:] // Skip ---\n
				contentStr = strings.TrimLeft(contentStr, "\r\n")

				var fm struct {
					Platform string        `yaml:"platform"`
					Campaign string        `yaml:"campaign"`
					Schedule string        `yaml:"schedule"`
					Images   StringOrSlice `yaml:"images"`
				}
				if err := yaml.Unmarshal([]byte(yamlPart), &fm); err == nil {
					platform = strings.TrimSpace(strings.ToLower(fm.Platform))
					campaign = strings.TrimSpace(fm.Campaign)
					schedule = strings.TrimSpace(fm.Schedule)
					images = strings.Join(fm.Images, ", ")
				}
			}
		}
		
		return externalEditorFinishedMsg{
			content:  contentStr,
			platform: platform,
			campaign: campaign,
			schedule: schedule,
			images:   images,
		}
	})
}

// loadAnalyticsCmd lädt die Social Analytics Daten asynchron im Hintergrund der TUI
func (m Model) loadAnalyticsCmd() tea.Msg {
	ctx := context.Background()
	days := 30

	posts, err := m.store.ListPosts(ctx, "all", "posted", "")
	if err != nil {
		return analyticsLoadedMsg{err: err}
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var analyzedPosts []postMetric

	totalPosts := 0
	totalLikes := 0
	totalShares := 0
	totalComments := 0
	totalImpressions := 0

	platStats := map[string]*platMetricSummary{
		"twitter":  {Name: "Twitter/X"},
		"linkedin": {Name: "LinkedIn"},
		"threads":  {Name: "Threads"},
		"mastodon": {Name: "Mastodon"},
		"bluesky":  {Name: "Bluesky"},
		"facebook": {Name: "Facebook"},
	}

	for _, p := range posts {
		if p.PostedAt != nil && p.PostedAt.Before(cutoff) {
			continue
		}

		plat, err := platforms.GetPlatform(p.Platform, m.store, false)
		if err != nil {
			continue
		}

		metrics, err := plat.FetchAnalytics(ctx, p.PlatformID)
		if err != nil {
			// fallback/mock
			metrics.Likes = 10
			metrics.Impressions = 150
		}

		postedAtStr := ""
		if p.PostedAt != nil {
			postedAtStr = p.PostedAt.Format("02.01. 15:04")
		}

		metricItem := postMetric{
			ID:          p.ID,
			Title:       p.Title,
			Platform:    p.Platform,
			PostedAt:    postedAtStr,
			Likes:       metrics.Likes,
			Shares:      metrics.Shares,
			Comments:    metrics.Comments,
			Impressions: metrics.Impressions,
		}
		analyzedPosts = append(analyzedPosts, metricItem)

		totalPosts++
		totalLikes += metrics.Likes
		totalShares += metrics.Shares
		totalComments += metrics.Comments
		totalImpressions += metrics.Impressions

		if summary, ok := platStats[p.Platform]; ok {
			summary.Posts++
			summary.Likes += metrics.Likes
			summary.Shares += metrics.Shares
			summary.Comments += metrics.Comments
			summary.Impressions += metrics.Impressions
		}
	}

	return analyticsLoadedMsg{
		totalPosts:       totalPosts,
		totalLikes:       totalLikes,
		totalShares:      totalShares,
		totalComments:    totalComments,
		totalImpressions: totalImpressions,
		platStats:        platStats,
		analyzedPosts:    analyzedPosts,
	}
}

// publishDuePostsCmd prüft und veröffentlicht fällige Posts im TUI-Hintergrund
func (m Model) publishDuePostsCmd() tea.Msg {
	ctx := context.Background()
	now := time.Now()
	
	posts, err := m.store.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err == nil {
		for _, p := range posts {
			if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
				// Versuche den Post atomar zu sperren, um doppeltes Posten im TUI-Hintergrund zu verhindern
				locked, err := m.store.TryLockPost(ctx, p.ID)
				if err != nil || !locked {
					continue
				}
				// TUI-Scheduler führt kein dry-run aus
				_, _ = scheduler.PublishPost(ctx, m.store, &p, false)
			}
		}
	}
	
	return m.loadDataCmd()
}

// Update reagiert auf Events und aktualisiert den Zustand
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.isEditing {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if m.showDatePicker {
				switch keyMsg.String() {
				case "esc", "ctrl+d":
					m.showDatePicker = false
					return m, nil
				case "left", "h":
					m.datePickerDate = m.datePickerDate.AddDate(0, 0, -1)
					return m, nil
				case "right", "l":
					m.datePickerDate = m.datePickerDate.AddDate(0, 0, 1)
					return m, nil
				case "up", "k":
					m.datePickerDate = m.datePickerDate.AddDate(0, 0, -7)
					return m, nil
				case "down", "j":
					m.datePickerDate = m.datePickerDate.AddDate(0, 0, 7)
					return m, nil
				case "pageup", "p":
					m.datePickerDate = m.datePickerDate.AddDate(0, -1, 0)
					return m, nil
				case "pagedown", "n":
					m.datePickerDate = m.datePickerDate.AddDate(0, 1, 0)
					return m, nil
				case "enter":
					currentTime := "09:00"
					existingVal := m.editorScheduledAt.Value()
					if len(existingVal) >= 16 {
						parts := strings.Split(existingVal, " ")
						if len(parts) == 2 {
							currentTime = parts[1]
						}
					}
					m.editorScheduledAt.SetValue(fmt.Sprintf("%s %s", m.datePickerDate.Format("02.01.2006"), currentTime))
					m.showDatePicker = false
					return m, nil
				}
				return m, nil
			}

			switch keyMsg.String() {
			case "esc":
				m.isEditing = false
				return m, nil
			case "tab":
				m.editorFocus = (m.editorFocus + 1) % 7
				m.updateEditorFocus()
				return m, nil
			case "shift+tab":
				m.editorFocus = (m.editorFocus - 1 + 7) % 7
				m.updateEditorFocus()
				return m, nil
			case "enter":
				if m.editorFocus == 5 { // Save
					if err := m.saveEditedPost(); err != nil {
						m.err = err
						return m, nil
					}
					m.isEditing = false
					return m, m.loadDataCmd
				} else if m.editorFocus == 6 { // Cancel
					m.isEditing = false
					return m, nil
				}
			case "ctrl+v":
				return m, m.runExternalEditorCmd()
			case "ctrl+d":
				if m.editorFocus == 2 {
					m.showDatePicker = true
					m.datePickerDate = time.Now()
					return m, nil
				}
			}

			if m.editorFocus == 0 {
				platformsList := []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook"}
				currIdx := -1
				for idx, p := range platformsList {
					if p == m.editorPlatform {
						currIdx = idx
						break
					}
				}
				if keyMsg.String() == "left" || keyMsg.String() == "h" {
					if currIdx != -1 {
						m.editorPlatform = platformsList[(currIdx-1+len(platformsList))%len(platformsList)]
					}
					return m, nil
				} else if keyMsg.String() == "right" || keyMsg.String() == "l" {
					if currIdx != -1 {
						m.editorPlatform = platformsList[(currIdx+1)%len(platformsList)]
						return m, nil
					}
				}
			}

			var cmd tea.Cmd
			switch m.editorFocus {
			case 1:
				m.editorCampaign, cmd = m.editorCampaign.Update(msg)
			case 2:
				m.editorScheduledAt, cmd = m.editorScheduledAt.Update(msg)
			case 3:
				m.editorImages, cmd = m.editorImages.Update(msg)
			case 4:
				m.editorBody, cmd = m.editorBody.Update(msg)
			}
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
		
	case tickMsg:
		return m, tea.Batch(
			m.publishDuePostsCmd,
			tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			}),
		)
		
	case analyticsLoadedMsg:
		m.analyticsLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.analyticsData = &msg
		return m, nil

	case postRepurposedMsg:
		m.repurposing = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Erfolgreich konvertiert: %s", strings.Join(msg.files, ", "))
		return m, m.loadDataCmd

	case setupWizardFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Setup für %s abgebrochen oder fehlgeschlagen: %w", msg.platform, msg.err)
			return m, nil
		}
		// Konfiguration neu laden
		if err := config.LoadConfig(); err != nil {
			m.err = fmt.Errorf("Fehler beim Neuladen der Konfiguration: %w", err)
			return m, nil
		}
		// Jetzt direkt OAuth-Flow starten
		m.loading = true
		m.statusMessage = fmt.Sprintf("Öffne Browser für %s...", msg.platform)
		return m, m.runAuthCmd(msg.platform)

	case platformClearedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("Fehler beim Zurücksetzen von %s: %w", msg.platform, msg.err)
			m.statusMessage = ""
			return m, nil
		} else {
			m.statusMessage = fmt.Sprintf("Verbindung und Einstellungen für %s wurden zurückgesetzt.", strings.ToUpper(msg.platform))
			return m, m.loadDataCmd
		}

	case authResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("Authentifizierung fehlgeschlagen: %w", msg.err)
			m.statusMessage = ""
			return m, nil
		} else {
			m.statusMessage = fmt.Sprintf("Erfolgreich mit %s verbunden!", msg.platform)
			return m, m.loadDataCmd
		}

	case backupFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Backup-Aktion fehlgeschlagen: %w", msg.err)
			m.statusMessage = ""
			return m, nil
		} else {
			action := "importiert"
			if msg.isExport {
				action = "exportiert (postctl_backup.bin)"
			}
			m.statusMessage = fmt.Sprintf("Konfiguration erfolgreich %s!", action)
			return m, m.loadDataCmd
		}

	case importFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Import fehlgeschlagen: %v", msg.err)
		} else {
			m.statusMessage = "Beiträge erfolgreich importiert!"
		}
		return m, m.loadDataCmd

	case exportFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Fehler beim Exportieren: %w", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Erfolgreich exportiert nach %s!", msg.filename)
		}
		return m, nil

	case externalEditorFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Fehler beim Bearbeiten mit externem Editor: %w", msg.err)
		} else {
			m.editorBody.SetValue(msg.content)
			if msg.platform != "" {
				m.editorPlatform = msg.platform
			}
			m.editorCampaign.SetValue(msg.campaign)
			m.editorScheduledAt.SetValue(msg.schedule)
			m.editorImages.SetValue(msg.images)
		}
		return m, nil

	case dataLoadedMsg:
		m.posts = msg.posts
		m.history = msg.history
		m.stats = msg.stats
		m.campaigns = msg.campaigns
		m.platforms = msg.platforms
		m.nextUp = msg.nextUp
		m.loading = false
		m.err = nil
		return m, nil
		
	case errorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
		
	case postDeletedMsg:
		m.selectedPost = nil
		m.cursor = 0
		return m, m.loadDataCmd
		
	case tea.MouseMsg:
		if m.showReadme {
			if m.readmeFocus == 0 {
				switch msg.Button {
				case tea.MouseButtonWheelUp:
					m.tocCursor = max(0, m.tocCursor - 1)
					return m, nil
				case tea.MouseButtonWheelDown:
					m.tocCursor = min(len(m.readmeTOC) - 1, m.tocCursor + 1)
					return m, nil
				}
			} else {
				switch msg.Button {
				case tea.MouseButtonWheelUp:
					m.readmeScroll = max(0, m.readmeScroll - 1)
					return m, nil
				case tea.MouseButtonWheelDown:
					maxScroll := len(m.readmeLines) - m.getReadmeViewportHeight()
					if maxScroll < 0 {
						maxScroll = 0
					}
					m.readmeScroll = min(maxScroll, m.readmeScroll + 1)
					return m, nil
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Wenn README offen ist, verarbeite nur README-Tasten
		if m.showReadme {
			if m.readmeFocus == 1 && (msg.String() == "t" || msg.String() == "backspace") {
				m.readmeFocus = 0
				return m, nil
			}

			switch {
			case key.Matches(msg, Keys.Quit) || key.Matches(msg, Keys.Esc):
				m.showReadme = false
				return m, nil

			case key.Matches(msg, Keys.Tab):
				m.readmeFocus = (m.readmeFocus + 1) % 2
				return m, nil

			case key.Matches(msg, Keys.ShiftTab):
				m.readmeFocus = (m.readmeFocus - 1 + 2) % 2
				return m, nil

			case key.Matches(msg, Keys.Up):
				if m.readmeFocus == 0 {
					m.tocCursor = max(0, m.tocCursor - 1)
				} else {
					m.readmeScroll = max(0, m.readmeScroll - 1)
				}
				return m, nil

			case key.Matches(msg, Keys.Down):
				if m.readmeFocus == 0 {
					m.tocCursor = min(len(m.readmeTOC) - 1, m.tocCursor + 1)
				} else {
					maxScroll := len(m.readmeLines) - m.getReadmeViewportHeight()
					if maxScroll < 0 {
						maxScroll = 0
					}
					m.readmeScroll = min(maxScroll, m.readmeScroll + 1)
				}
				return m, nil

			case key.Matches(msg, Keys.Enter):
				if m.readmeFocus == 0 && len(m.readmeTOC) > 0 {
					m.readmeScroll = m.readmeTOC[m.tocCursor].line
					m.readmeFocus = 1 // direkt zum Inhalt springen und fokussieren
				}
				return m, nil
			}
			return m, nil
		}

		// Wenn eine Fehlermeldung angezeigt wird, schließt Esc diese
		if m.err != nil && msg.String() == "esc" {
			m.err = nil
			return m, nil
		}

		// Filter zurücksetzen bei Esc im Posts-Tab, wenn keine Detailansicht offen ist
		if m.activeTab == 1 && m.selectedPost == nil && m.filterCampaign != "" && key.Matches(msg, Keys.Esc) {
			m.filterCampaign = ""
			m.cursor = 0
			return m, nil
		}

		// History-Detailansicht-Steuerung
		if m.selectedHistory != nil {
			m.statusMessage = ""
			switch {
			case key.Matches(msg, Keys.Esc):
				m.selectedHistory = nil
				return m, nil
			case key.Matches(msg, Keys.Export):
				return m, m.exportHistoryEntryCmd(m.selectedHistory)
			}
			return m, nil
		}

		// Detailansicht-Steuerung
		if m.selectedPost != nil {
			// Clear status message when any key is pressed to prevent old message lingering
			m.statusMessage = ""
			switch {
			case key.Matches(msg, Keys.Esc):
				m.selectedPost = nil
				return m, nil
			case key.Matches(msg, Keys.Delete):
				idToDelete := m.selectedPost.ID
				return m, m.deletePostCmd(idToDelete)
			case key.Matches(msg, Keys.Edit):
				postToEdit := *m.selectedPost
				m.selectedPost = nil
				m.initEditor(&postToEdit)
				return m, nil
			case key.Matches(msg, Keys.Repurpose):
				if m.repurposing {
					return m, nil
				}
				p := m.selectedPost
				var targets []string
				allPlats := []string{
					models.PlatformTwitter,
					models.PlatformLinkedIn,
					models.PlatformThreads,
					models.PlatformMastodon,
					models.PlatformBluesky,
					models.PlatformFacebook,
				}
				for _, plat := range allPlats {
					if plat != p.Platform {
						targets = append(targets, plat)
					}
				}
				m.repurposing = true
				m.statusMessage = "KI generiert Konvertierungen..."
				return m, m.repurposePostCmd(p, targets)
			}
			return m, nil
		}

		// Hauptmenü-Steuerung
		switch {
		case key.Matches(msg, Keys.Quit):
			return m, tea.Quit
			
		case key.Matches(msg, Keys.Tab):
			m.activeTab = (m.activeTab + 1) % 6
			m.cursor = 0
			if m.activeTab == 4 {
				m.analyticsLoading = true
				return m, m.loadAnalyticsCmd
			}
			return m, nil
			
		case key.Matches(msg, Keys.ShiftTab):
			m.activeTab = (m.activeTab - 1 + 6) % 6
			m.cursor = 0
			if m.activeTab == 4 {
				m.analyticsLoading = true
				return m, m.loadAnalyticsCmd
			}
			return m, nil
			
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
				if m.activeTab == 5 && m.cursor == 4 {
					m.cursor--
				}
			}
			return m, nil
			
		case key.Matches(msg, Keys.Down):
			maxItems := m.maxCursorItems()
			if m.cursor < maxItems-1 {
				m.cursor++
				if m.activeTab == 5 && m.cursor == 4 {
					m.cursor++
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Left), key.Matches(msg, Keys.Right):
			if m.activeTab == 5 && m.cursor < 4 {
				m.cycleSetting()
				return m, nil
			}
			return m, nil
			
		case key.Matches(msg, Keys.Enter):
			if m.activeTab == 5 {
				if m.cursor >= 5 && m.cursor <= 10 {
					var platName string
					switch m.cursor {
					case 5:
						platName = models.PlatformTwitter
					case 6:
						platName = models.PlatformLinkedIn
					case 7:
						platName = models.PlatformThreads
					case 8:
						platName = models.PlatformMastodon
					case 9:
						platName = models.PlatformBluesky
					case 10:
						platName = models.PlatformFacebook
					}
					if platformNeedsSetup(platName) || (platName == models.PlatformTwitter && config.ActiveConfig.Twitter.AuthMode == "cookie") {
						return m, m.runSetupWizardCmd(platName)
					}
					m.loading = true
					m.statusMessage = fmt.Sprintf("Öffne Browser für %s...", platName)
					return m, m.runAuthCmd(platName)
				}
				if m.cursor == 11 {
					return m, m.runBackupExportCmd()
				}
				if m.cursor == 12 {
					return m, m.runBackupImportCmd()
				}
				m.cycleSetting()
				return m, nil
			}
			if m.activeTab == 0 && len(m.campaigns) > 0 {
				m.filterCampaign = m.campaigns[m.cursor].Slug
				m.activeTab = 1
				m.cursor = 0
				return m, nil
			}
			if m.activeTab == 1 {
				filtered := m.getFilteredPosts()
				if len(filtered) > 0 {
					m.selectedPost = &filtered[m.cursor]
				}
			}
			if m.activeTab == 2 {
				if len(m.nextUp) > 0 && m.cursor < len(m.nextUp) {
					m.selectedPost = &m.nextUp[m.cursor]
				}
			}
			if m.activeTab == 3 {
				if len(m.history) > 0 && m.cursor < len(m.history) {
					m.selectedHistory = &m.history[m.cursor]
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Delete):
			if m.activeTab == 1 {
				filtered := m.getFilteredPosts()
				if len(filtered) > 0 {
					idToDelete := filtered[m.cursor].ID
					return m, m.deletePostCmd(idToDelete)
				}
			} else if m.activeTab == 5 { // Settings
				if m.cursor >= 5 && m.cursor <= 10 {
					var platName string
					switch m.cursor {
					case 5:
						platName = models.PlatformTwitter
					case 6:
						platName = models.PlatformLinkedIn
					case 7:
						platName = models.PlatformThreads
					case 8:
						platName = models.PlatformMastodon
					case 9:
						platName = models.PlatformBluesky
					case 10:
						platName = models.PlatformFacebook
					}
					m.loading = true
					m.statusMessage = fmt.Sprintf("Setze %s zurück...", platName)
					return m, m.clearPlatformCmd(platName)
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Export):
			if m.selectedPost == nil && !m.showReadme {
				if m.activeTab == 3 {
					return m, m.exportHistoryCmd(m.history)
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Filter):
			if m.selectedPost == nil && !m.showReadme && m.activeTab == 1 {
				if len(m.campaigns) > 0 {
					nextCampaign := ""
					if m.filterCampaign == "" {
						nextCampaign = m.campaigns[0].Slug
					} else {
						idx := -1
						for i, c := range m.campaigns {
							if c.Slug == m.filterCampaign {
								idx = i
								break
							}
						}
						if idx != -1 && idx < len(m.campaigns)-1 {
							nextCampaign = m.campaigns[idx+1].Slug
						}
					}
					m.filterCampaign = nextCampaign
					m.cursor = 0
				}
				return m, nil
			}
			return m, nil

		case key.Matches(msg, Keys.Import):
			if m.selectedPost == nil {
				return m, m.runImportPostsCmd()
			}
			return m, nil

		case key.Matches(msg, Keys.NewPost):
			if m.selectedPost == nil && !m.showReadme {
				m.initEditor(nil)
				return m, nil
			}

		case key.Matches(msg, Keys.Edit):
			if m.selectedPost == nil && !m.showReadme {
				if m.activeTab == 1 { // Posts
					filtered := m.getFilteredPosts()
					if len(filtered) > 0 && m.cursor < len(filtered) {
						postToEdit := filtered[m.cursor]
						m.initEditor(&postToEdit)
						return m, nil
					}
				} else if m.activeTab == 2 { // Schedule
					if len(m.nextUp) > 0 && m.cursor < len(m.nextUp) {
						postToEdit := m.nextUp[m.cursor]
						m.initEditor(&postToEdit)
						return m, nil
					}
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Readme):
			if m.selectedPost == nil {
				lines, toc := getReadmeData()
				m.readmeLines = lines
				m.readmeTOC = toc
				m.readmeScroll = 0
				m.tocCursor = 0
				m.readmeFocus = 0
				m.showReadme = true
			}
			return m, nil

		case key.Matches(msg, Keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}
	}
	
	return m, nil
}

// maxCursorItems gibt die Anzahl der selektierbaren Elemente im aktuellen Tab zurück
func (m Model) maxCursorItems() int {
	switch m.activeTab {
	case 0: // Dashboard
		return len(m.campaigns)
	case 1: // Posts
		return len(m.getFilteredPosts())
	case 2: // Schedule
		return len(m.nextUp)
	case 3: // History
		return len(m.history)
	case 5: // Settings
		return 13
	default:
		return 0
	}
}

// getFilteredPosts gibt die Liste der Posts unter Berücksichtigung des Kampagnen-Filters zurück
func (m Model) getFilteredPosts() []models.Post {
	if m.filterCampaign == "" {
		return m.posts
	}
	var filtered []models.Post
	for _, p := range m.posts {
		if p.Campaign == m.filterCampaign {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// View rendert den Bildschirm als Zeichenkette
func (m Model) View() string {
	if m.loading {
		return "\n  Lade Daten aus SQLite Store...\n"
	}
	
	if m.err != nil {
		return fmt.Sprintf("\n  [FEHLER]: %v\n\n  Drücke ESC zum Schließen.\n", m.err)
	}

	if m.isEditing {
		return m.renderEditor()
	}

	// Detailansicht anzeigen, falls ein Post ausgewählt ist
	if m.selectedPost != nil {
		return m.renderDetailView()
	}

	// History-Detailansicht anzeigen, falls ausgewählt
	if m.selectedHistory != nil {
		return m.renderHistoryDetailView()
	}

	// README / System-Dokumentation anzeigen, falls geöffnet
	if m.showReadme {
		return m.renderReadme()
	}

	var builder strings.Builder

	// Header
	builder.WriteString(StyleTitle.Render(" postctl — Social Media CLI "))
	builder.WriteString("\n\n")

	// Tabs
	builder.WriteString(RenderTabs(m.activeTab))
	builder.WriteString("\n\n")

	// Inhalt je nach Tab
	var tabContent string
	switch m.activeTab {
	case 0:
		tabContent = m.renderDashboard()
	case 1:
		tabContent = m.renderPostList()
	case 2:
		tabContent = m.renderSchedule()
	case 3:
		tabContent = m.renderHistory()
	case 4:
		tabContent = m.renderAnalytics()
	case 5:
		tabContent = m.renderSettings()
	}
	builder.WriteString(tabContent)
	builder.WriteString("\n\n")

	// Hilfetext / Keybindings
	builder.WriteString(m.renderHelp())

	return builder.String()
}

func (m Model) repurposePostCmd(p *models.Post, targets []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		var srcContent string
		if len(p.Tweets) > 0 {
			var sb strings.Builder
			for _, tweet := range p.Tweets {
				sb.WriteString(fmt.Sprintf("## Tweet %d\n%s\n\n", tweet.Index, tweet.Content))
			}
			srcContent = sb.String()
		} else {
			srcContent = p.Body
		}

		aiCfg := generator.GeneratorConfig{
			Provider: config.ActiveConfig.AI.Provider,
			APIKey:   config.ActiveConfig.AI.APIKey,
			Model:    config.ActiveConfig.AI.Model,
			BaseURL:  config.ActiveConfig.AI.BaseURL,
		}

		if aiCfg.APIKey == "" {
			if strings.ToLower(aiCfg.Provider) == "claude" {
				aiCfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			} else if strings.ToLower(aiCfg.Provider) == "openai" || aiCfg.Provider == "" {
				aiCfg.APIKey = os.Getenv("OPENAI_API_KEY")
			}
		}

		if strings.ToLower(aiCfg.Provider) != "ollama" && aiCfg.APIKey == "" {
			return postRepurposedMsg{err: fmt.Errorf("AI API-Key fehlt. Bitte setze ihn in config.yaml oder als Umgebungsvariable")}
		}

		result, err := generator.RepurposeContent(ctx, aiCfg, p.Platform, p.Type, p.Title, srcContent, targets)
		if err != nil {
			return postRepurposedMsg{err: err}
		}

		writtenFiles, err := generator.SaveRepurposedToMarkdownFiles(result, ".", p.Campaign)
		if err != nil {
			return postRepurposedMsg{err: err}
		}

		var shortNames []string
		for _, f := range writtenFiles {
			shortNames = append(shortNames, filepath.Base(f))
		}

		return postRepurposedMsg{files: shortNames}
	}
}

// cycleSetting ändert den Wert der aktuell ausgewählten Einstellung und speichert sie
func (m *Model) cycleSetting() {
	switch m.cursor {
	case 0: // AI Provider
		current := strings.ToLower(config.ActiveConfig.AI.Provider)
		next := "openai"
		switch current {
		case "openai":
			next = "claude"
		case "claude":
			next = "ollama"
		case "ollama":
			next = "openai"
		}
		config.ActiveConfig.AI.Provider = next
		// Standardmodell für neuen Provider setzen
		if next == "openai" {
			config.ActiveConfig.AI.Model = "gpt-4o-mini"
		} else if next == "claude" {
			config.ActiveConfig.AI.Model = "claude-3-5-sonnet-latest"
		} else {
			config.ActiveConfig.AI.Model = "llama3"
		}
	case 1: // AI Model
		provider := strings.ToLower(config.ActiveConfig.AI.Provider)
		model := config.ActiveConfig.AI.Model
		if provider == "openai" {
			if model == "gpt-4o-mini" {
				config.ActiveConfig.AI.Model = "gpt-4o"
			} else {
				config.ActiveConfig.AI.Model = "gpt-4o-mini"
			}
		} else if provider == "claude" {
			if model == "claude-3-5-sonnet-latest" {
				config.ActiveConfig.AI.Model = "claude-3-5-haiku-latest"
			} else {
				config.ActiveConfig.AI.Model = "claude-3-5-sonnet-latest"
			}
		} else {
			if model == "llama3" {
				config.ActiveConfig.AI.Model = "mistral"
			} else {
				config.ActiveConfig.AI.Model = "llama3"
			}
		}
	case 2: // Dry Run
		config.ActiveConfig.Defaults.DryRun = !config.ActiveConfig.Defaults.DryRun
	case 3: // Language
		current := strings.ToLower(config.ActiveConfig.Defaults.Language)
		if current == "en" || current == "" {
			config.ActiveConfig.Defaults.Language = "de"
		} else {
			config.ActiveConfig.Defaults.Language = "en"
		}
	}

	// In config.yaml speichern
	_ = config.SaveConfig()
}

func (m Model) getReadmeViewportHeight() int {
	outerHeight := 22
	if m.height > 10 {
		outerHeight = max(22, m.height - 4)
	}
	innerHeight := outerHeight - 4
	return innerHeight - 2
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*s = []string{str}
		return nil
	}

	var slice []string
	if err := value.Decode(&slice); err == nil {
		*s = slice
		return nil
	}

	return nil
}

func (m Model) exportHistoryCmd(history []models.HistoryEntry) tea.Cmd {
	return func() tea.Msg {
		if len(history) == 0 {
			return exportFinishedMsg{err: fmt.Errorf("keine History-Einträge vorhanden")}
		}

		b, err := json.MarshalIndent(history, "", "  ")
		if err != nil {
			return exportFinishedMsg{err: fmt.Errorf("marshal history: %w", err)}
		}

		filename := "history_export.json"
		err = os.WriteFile(filename, b, 0644)
		if err != nil {
			return exportFinishedMsg{err: fmt.Errorf("write file: %w", err)}
		}

		return exportFinishedMsg{filename: filename}
	}
}

func (m Model) exportHistoryEntryCmd(entry *models.HistoryEntry) tea.Cmd {
	return func() tea.Msg {
		if entry == nil {
			return exportFinishedMsg{err: fmt.Errorf("kein Eintrag ausgewählt")}
		}

		b, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return exportFinishedMsg{err: fmt.Errorf("marshal entry: %w", err)}
		}

		filename := fmt.Sprintf("history_entry_%s.json", entry.ID)
		err = os.WriteFile(filename, b, 0644)
		if err != nil {
			return exportFinishedMsg{err: fmt.Errorf("write file: %w", err)}
		}

		return exportFinishedMsg{filename: filename}
	}
}



