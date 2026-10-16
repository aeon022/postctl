package tui

import (
	"charm.land/lipgloss/v2"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/service"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/generator"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/store"
	"gopkg.in/yaml.v3"
)

// PostStats hält aggregierte Post-Statistiken
type PostStats struct {
	posted    int
	scheduled int
	drafts    int
	failed    int
}

// WantsProfileSwitch reports whether the program ended because the user asked to change profile.
func (m Model) WantsProfileSwitch() bool { return m.switchProfile }

// Model repräsentiert den Zustand der Bubbletea-Anwendung
type Model struct {
	store     *store.SQLiteStore
	activeTab int         // one of the tab* constants (tabs.go)
	calDay    time.Time   // selected day of the calendar tab (zero = today)
	calCursor int         // selected post within that day
	ask       *publishAsk // pending "publish now?" confirmation
	health    healthInfo  // token expiry, scheduler state (dashboard)
	cursor    int

	// Geladene Daten
	posts     []models.Post
	history   []models.HistoryEntry
	stats     PostStats
	campaigns []models.Campaign
	platforms map[string]bool
	nextUp    []models.Post

	// UI Zustand
	selectedPost       *models.Post
	selectedHistory    *models.HistoryEntry
	filterCampaign     string
	isEditing          bool
	showHelp           bool
	switchProfile      bool // set by Keys.Profile; cmd/tui.go re-runs with another profile
	err                error
	loading            bool
	repurposing        bool
	statusMessage      string
	detailScrollOffset int

	// Editor Zustand
	editorPostID      string
	editorPlatform    string
	editorCampaign    textinput.Model
	editorScheduledAt textinput.Model
	editorImages      textinput.Model
	editorBody        textarea.Model
	editorFocus       int
	editorAlts        textinput.Model // alt text per image, separated by " | "
	editorType        string          // kept when editing (e.g. "story")
	editorTargets     map[string]bool // platforms selected for a new post (space toggles)
	inPalette         bool            // ":" command palette open
	spin              int             // spinner frame
	searching         bool            // "/" search input open (Posts tab)
	logSource         int             // logSourceTUI or logSourceDaemon
	searchInput       textinput.Model
	searchQuery       string
	spinning          bool // a spinner tick is in flight
	paletteInput      textinput.Model
	paletteCursor     int
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

	// Queue Slots Editor
	editingQueueSlots bool
	queueSlotsInput   textinput.Model

	// Massen-Aktionen (Bulk Actions)
	selectedPosts map[string]bool
}

// Msg-Typen
type dataLoadedMsg struct {
	posts     []models.Post
	history   []models.HistoryEntry
	stats     PostStats
	campaigns []models.Campaign
	platforms map[string]bool
	nextUp    []models.Post
	health    healthInfo
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
	noData           int // posts the platform returned no metrics for
	dailyEngagement  []int
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
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Platform    string    `json:"platform"`
	PostedAt    string    `json:"posted_at"`
	Likes       int       `json:"likes"`
	Shares      int       `json:"shares"`
	Comments    int       `json:"comments"`
	Impressions int       `json:"impressions"`
	At          time.Time `json:"-"`
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

type publishFinishedMsg struct {
	count int
	err   error
}

type tickMsg struct{}

// spinMsg advances the loading spinner.
type spinMsg struct{}

func spinTick() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m Model) spinner() string { return spinFrames[m.spin%len(spinFrames)] }

// NewModel initialisiert ein TUI-Model
func NewModel(s *store.SQLiteStore) Model {
	qInput := textinput.New()
	qInput.Placeholder = "Mon 09:00, Wed 14:00, Fri 17:30"
	qInput.CharLimit = 150
	qInput.SetWidth(60)

	return Model{
		store:             s,
		activeTab:         0,
		cursor:            0,
		platforms:         make(map[string]bool),
		loading:           true,
		queueSlotsInput:   qInput,
		editingQueueSlots: false,
		selectedPosts:     make(map[string]bool),
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
		models.PlatformTwitter:   false,
		models.PlatformLinkedIn:  false,
		models.PlatformThreads:   false,
		models.PlatformMastodon:  false,
		models.PlatformBluesky:   false,
		models.PlatformFacebook:  false,
		models.PlatformInstagram: config.ActiveConfig.Instagram.AccessToken != "" && config.ActiveConfig.Instagram.AccountID != "",
		models.PlatformTelegram:  config.ActiveConfig.Telegram.BotToken != "" && config.ActiveConfig.Telegram.ChatID != "",
		models.PlatformDiscord:   config.ActiveConfig.Discord.WebhookURL != "",
		models.PlatformDevTo:     config.ActiveConfig.DevTo.APIToken != "",
		models.PlatformReddit:    config.ActiveConfig.Reddit.ClientID != "" && config.ActiveConfig.Reddit.ClientSecret != "" && config.ActiveConfig.Reddit.Username != "" && config.ActiveConfig.Reddit.Password != "",
		models.PlatformHashnode:  config.ActiveConfig.Hashnode.APIToken != "" && config.ActiveConfig.Hashnode.PublicationID != "",
		models.PlatformMedium:    config.ActiveConfig.Medium.IntegrationToken != "",
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
		health:    collectHealth(ctx, m.store, posts),
	}
}

// deletePostCmd löscht einen Post asynchron (und entfernt ihn von der Plattform, falls veröffentlicht)
func (m Model) deletePostCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		post, err := m.store.GetPost(ctx, id)
		if err == nil && post != nil && post.Status == models.StatusPosted && post.PlatformID != "" {
			// Real platform delete must succeed before the local record
			// goes away — used to discard this error and delete the local
			// row regardless, which on a failure (or on one of the several
			// platforms whose Delete is still an unimplemented stub) left
			// the post live and untracked while postctl believed it gone.
			// Same bug class fixed in calctl's DeleteEvent on 2026-08-01.
			plat, err := platforms.GetPlatform(post.Platform, m.store, false)
			if err != nil {
				return errorMsg{fmt.Errorf("resolve platform %q: %w", post.Platform, err)}
			}
			if err := plat.Delete(ctx, post.PlatformID); err != nil {
				return errorMsg{err}
			}
		}
		if err := m.store.DeletePost(ctx, id); err != nil {
			return errorMsg{err}
		}
		return postDeletedMsg{id}
	}
}

// deletePostsCmd löscht mehrere Posts asynchron
func (m Model) deletePostsCmd(ids []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		for _, id := range ids {
			post, err := m.store.GetPost(ctx, id)
			if err == nil && post != nil && post.Status == models.StatusPosted && post.PlatformID != "" {
				// Same real-delete-before-local-delete ordering as
				// deletePostCmd above.
				plat, err := platforms.GetPlatform(post.Platform, m.store, false)
				if err != nil {
					return errorMsg{fmt.Errorf("resolve platform %q: %w", post.Platform, err)}
				}
				if err := plat.Delete(ctx, post.PlatformID); err != nil {
					return errorMsg{err}
				}
			}
			if err := m.store.DeletePost(ctx, id); err != nil {
				return errorMsg{err}
			}
		}
		dummyId := ""
		if len(ids) > 0 {
			dummyId = ids[0]
		}
		return postDeletedMsg{dummyId}
	}
}

// bulkSchedulePostsCmd plant mehrere Posts asynchron ein
func (m Model) bulkSchedulePostsCmd(ids []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		for _, id := range ids {
			post, err := m.store.GetPost(ctx, id)
			if err != nil {
				return errorMsg{err}
			}

			if post.Status != models.StatusDraft && post.Status != models.StatusFailed {
				continue
			}

			slot, err := scheduler.GetNextQueueSlot(ctx, m.store, post.Platform)
			if err != nil {
				return errorMsg{err}
			}

			post.Status = models.StatusScheduled
			post.ScheduledAt = &slot
			post.UpdatedAt = time.Now()

			if err := m.store.SavePost(ctx, post); err != nil {
				return errorMsg{err}
			}
			scheduler.LogPost("scheduled", post)
		}
		return m.loadDataCmd()
	}
}

// bulkPublishPostsCmd veröffentlicht mehrere Posts asynchron sofort
func (m Model) bulkPublishPostsCmd(ids []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		publishedCount := 0
		var lastErr error
		for _, id := range ids {
			post, err := m.store.GetPost(ctx, id)
			if err != nil {
				lastErr = err
				continue
			}

			_, err = scheduler.PublishPost(ctx, m.store, post, false)
			if err != nil {
				lastErr = err
			} else {
				publishedCount++
			}
		}
		return publishFinishedMsg{
			count: publishedCount,
			err:   lastErr,
		}
	}
}

// runAuthCmd startet den Authentifizierungs-Flow für eine Plattform asynchron im Hintergrund
func (m Model) runAuthCmd(platformName string) tea.Cmd {
	return func() tea.Msg {
		// Auth flows print instructions with fmt.Println; on the TUI's alt screen that
		// scrolls the terminal and desyncs the renderer. The TUI writes through its own
		// file handle, so pointing the globals at /dev/null only mutes the platform code.
		if null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
			out, errw := os.Stdout, os.Stderr
			os.Stdout, os.Stderr = null, null
			defer func() { os.Stdout, os.Stderr = out, errw; null.Close() }()
		}
		ctx := context.Background()
		plat, err := platforms.GetPlatform(platformName, m.store, false)
		if err != nil {
			return authResultMsg{platform: platformName, err: err}
		}

		err = plat.Auth(ctx)
		return authResultMsg{platform: platformName, err: err}
	}
}

// selfCmd runs postctl itself as a subprocess on the active profile; without
// POSTCTL_PROFILE the child would silently fall back to the default profile.
func selfCmd(args ...string) *exec.Cmd {
	c := exec.Command(getExecutablePath(), args...)
	c.Env = append(os.Environ(), "POSTCTL_PROFILE="+config.ActiveProfile)
	return c
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
		return (config.ActiveConfig.Facebook.AppID == "" || config.ActiveConfig.Facebook.AppSecret == "") && config.ActiveConfig.Facebook.PageToken == ""
	case models.PlatformInstagram:
		return config.ActiveConfig.Instagram.AccessToken == "" || config.ActiveConfig.Instagram.AccountID == ""
	case models.PlatformTelegram:
		return config.ActiveConfig.Telegram.BotToken == "" || config.ActiveConfig.Telegram.ChatID == ""
	case models.PlatformDiscord:
		return config.ActiveConfig.Discord.WebhookURL == ""
	case models.PlatformDevTo:
		return config.ActiveConfig.DevTo.APIToken == ""
	case models.PlatformReddit:
		return config.ActiveConfig.Reddit.ClientID == "" || config.ActiveConfig.Reddit.ClientSecret == "" || config.ActiveConfig.Reddit.Username == "" || config.ActiveConfig.Reddit.Password == ""
	case models.PlatformHashnode:
		return config.ActiveConfig.Hashnode.APIToken == "" || config.ActiveConfig.Hashnode.PublicationID == ""
	case models.PlatformMedium:
		return config.ActiveConfig.Medium.IntegrationToken == ""
	}
	return false
}

// runSetupWizardCmd startet das interaktive CLI Setup für eine Plattform
func (m Model) runSetupWizardCmd(platformName string) tea.Cmd {
	c := selfCmd("config", "setup", platformName)
	return m.execWithStoreReleased(c, func(err error) tea.Msg {
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
		case models.PlatformInstagram:
			config.ActiveConfig.Instagram.AccessToken = ""
			config.ActiveConfig.Instagram.AccountID = ""
		case models.PlatformTelegram:
			config.ActiveConfig.Telegram.BotToken = ""
			config.ActiveConfig.Telegram.ChatID = ""
		case models.PlatformDiscord:
			config.ActiveConfig.Discord.WebhookURL = ""
		case models.PlatformDevTo:
			config.ActiveConfig.DevTo.APIToken = ""
		case models.PlatformReddit:
			config.ActiveConfig.Reddit.ClientID = ""
			config.ActiveConfig.Reddit.ClientSecret = ""
			config.ActiveConfig.Reddit.Username = ""
			config.ActiveConfig.Reddit.Password = ""
		case models.PlatformHashnode:
			config.ActiveConfig.Hashnode.APIToken = ""
			config.ActiveConfig.Hashnode.PublicationID = ""
		case models.PlatformMedium:
			config.ActiveConfig.Medium.IntegrationToken = ""
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
	c := selfCmd("config", "export")
	return m.execWithStoreReleased(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: true, err: err}
	})
}

// runBackupImportCmd pausiert die TUI und startet interaktiv den CLI-Import
func (m Model) runBackupImportCmd() tea.Cmd {
	c := selfCmd("config", "import")
	return m.execWithStoreReleased(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: false, err: err}
	})
}

// runImportPostsCmd pausiert die TUI und startet interaktiv den CLI-Post-Import
func (m Model) runImportPostsCmd() tea.Cmd {
	c := selfCmd("import")
	return m.execWithStoreReleased(c, func(err error) tea.Msg {
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
		rulerNum = " 000      030      060      090      120      150      180      210      240      270 280!\n"
		rulerLine = " |--------|--------|--------|--------|--------|--------|--------|--------|--------|-|\n"
	case "bluesky":
		limit = 300
		rulerNum = " 000      030      060      090      120      150      180      210      240      270      300!\n"
		rulerLine = " |--------|--------|--------|--------|--------|--------|--------|--------|--------|--------|\n"
	case "mastodon":
		limit = 500
		rulerNum = " 000      050      100      150      200      250      300      350      400      450      500!\n"
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
		"twitter":   {Name: "Twitter/X"},
		"linkedin":  {Name: "LinkedIn"},
		"threads":   {Name: "Threads"},
		"mastodon":  {Name: "Mastodon"},
		"bluesky":   {Name: "Bluesky"},
		"facebook":  {Name: "Facebook"},
		"instagram": {Name: "Instagram"},
	}

	noData := 0
	dailyEngagement := make([]int, days)

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
			noData++ // never invent numbers: a post without data counts as a post, with zeros
			metrics = models.AnalyticsData{}
		}

		postedAtStr := ""
		if p.PostedAt != nil {
			postedAtStr = p.PostedAt.Format("02.01. 15:04")

			// Tägliche Interaktionen berechnen
			postDate := time.Date(p.PostedAt.Year(), p.PostedAt.Month(), p.PostedAt.Day(), 0, 0, 0, 0, time.Local)
			nowDate := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
			daysAgo := int(nowDate.Sub(postDate).Hours() / 24)
			if daysAgo >= 0 && daysAgo < days {
				dailyEngagement[days-1-daysAgo] += metrics.Likes + metrics.Shares + metrics.Comments
			}
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
		if p.PostedAt != nil {
			metricItem.At = *p.PostedAt
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
		noData:           noData,
		platStats:        platStats,
		analyzedPosts:    analyzedPosts,
		dailyEngagement:  dailyEngagement,
	}
}

// dbSettleQuiet is how long the local DB file must have been unmodified
// before the TUI trusts it enough to auto-publish due posts — see
// config.DBSettled.
const dbSettleQuiet = 45 * time.Second

// publishDuePostsCmd prüft und veröffentlicht fällige Posts im TUI-Hintergrund
func (m Model) publishDuePostsCmd() tea.Msg {
	ctx := context.Background()

	// Sicherheits-Rescheduling für überfällige Beiträge durchführen
	if err := scheduler.RescheduleOverdue(ctx, m.store); err != nil {
		platforms.Log("[SCHEDULER FEHLER] Sicherheits-Rescheduling fehlgeschlagen: %v", err)
	}

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
	if _, ok := msg.(spinMsg); ok {
		m.spin++
		if !m.loading {
			m.spinning = false
			return m, nil
		}
		return m, spinTick()
	}
	nm, cmd := m.update(msg)
	if mm, ok := nm.(Model); ok && mm.loading && !mm.spinning {
		mm.spinning = true // the spinner only ticks while something loads
		return mm, tea.Batch(cmd, spinTick())
	}
	return nm, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.isEditing {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
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
				m.editorFocus = (m.editorFocus + 1) % focusCount
				m.updateEditorFocus()
				return m, nil
			case "shift+tab":
				m.editorFocus = (m.editorFocus - 1 + focusCount) % focusCount
				m.updateEditorFocus()
				return m, nil
			case "enter":
				if m.editorFocus == focusSave {
					if err := m.saveEditedPost(); err != nil {
						m.err = err
						return m, nil
					}
					m.isEditing = false
					return m, m.loadDataCmd
				} else if m.editorFocus == focusCancel {
					m.isEditing = false
					return m, nil
				}
			case "ctrl+v":
				return m, m.runExternalEditorCmd()
			case "ctrl+d":
				if m.editorFocus == focusSchedule {
					m.showDatePicker = true
					m.datePickerDate = time.Now()
					return m, nil
				}
			}

			if m.editorFocus == focusPlatform {
				platformsList := editorPlatforms
				currIdx := -1
				for idx, p := range platformsList {
					if p == m.editorPlatform {
						currIdx = idx
						break
					}
				}
				if keyMsg.String() == "space" && m.editorPostID == "" { // pick several platforms for a new post
					if m.editorTargets == nil {
						m.editorTargets = map[string]bool{}
					}
					m.editorTargets[m.editorPlatform] = !m.editorTargets[m.editorPlatform]
					return m, nil
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
			case focusCampaign:
				m.editorCampaign, cmd = m.editorCampaign.Update(msg)
			case focusSchedule:
				m.editorScheduledAt, cmd = m.editorScheduledAt.Update(msg)
			case focusImages:
				m.editorImages, cmd = m.editorImages.Update(msg)
			case focusAlts:
				m.editorAlts, cmd = m.editorAlts.Update(msg)
			case focusBody:
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
		cmds := []tea.Cmd{
			tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
				return tickMsg{}
			}),
		}
		// Only this machine's designated publisher auto-fires the
		// scheduler, and only once its local (possibly Dropbox-synced) DB
		// copy has gone quiet for a bit — see config.AutoPublishEnabled
		// and config.DBSettled for why: otherwise two machines can each
		// see the same post as still due and both publish it.
		if config.AutoPublishEnabled() && config.DBSettled(dbSettleQuiet) {
			cmds = append(cmds, m.publishDuePostsCmd)
		}
		return m, tea.Batch(cmds...)

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
		m.statusMessage = fmt.Sprintf(Tr("st_converted"), strings.Join(msg.files, ", "))
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
		m.statusMessage = fmt.Sprintf(Tr("st_open_browser"), msg.platform)
		return m, m.runAuthCmd(msg.platform)

	case platformClearedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("Fehler beim Zurücksetzen von %s: %w", msg.platform, msg.err)
			m.statusMessage = ""
			return m, nil
		} else {
			m.statusMessage = fmt.Sprintf(Tr("st_reset_done"), strings.ToUpper(msg.platform))
			return m, m.loadDataCmd
		}

	case authResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("Authentifizierung fehlgeschlagen: %w", msg.err)
			m.statusMessage = ""
			return m, nil
		} else {
			m.statusMessage = fmt.Sprintf(Tr("st_connected"), msg.platform)
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
			m.statusMessage = fmt.Sprintf(Tr("st_config_done"), action)
			return m, m.loadDataCmd
		}

	case importFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf(Tr("st_import_failed"), msg.err)
		} else {
			m.statusMessage = Tr("st_imported")
		}
		return m, m.loadDataCmd

	case exportFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Fehler beim Exportieren: %w", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf(Tr("st_exported"), msg.filename)
		}
		return m, nil

	case publishFinishedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf(Tr("st_publish_failed"), msg.err)
		} else {
			if msg.count == 1 {
				m.statusMessage = Tr("st_published")
			} else {
				m.statusMessage = fmt.Sprintf(Tr("st_published_n"), msg.count)
			}
		}
		m.selectedPost = nil
		m.detailScrollOffset = 0
		return m, m.loadDataCmd

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
		m.health = msg.health
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

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && m.activeTab == tabCalendar && msg.Y > tabsRow && !m.loading && m.err == nil && !m.isEditing &&
			m.selectedPost == nil && m.selectedHistory == nil && !m.showReadme && !m.showHelp && !m.inPalette && m.ask == nil {
			if nm, ok := m.calendarClick(msg.X, msg.Y); ok {
				return nm, nil
			}
		}
		if msg.Button == tea.MouseLeft && msg.Y == tabsRow && !m.loading && m.err == nil && !m.isEditing &&
			m.selectedPost == nil && m.selectedHistory == nil && !m.showReadme && !m.showHelp {
			if i := m.tabAt(msg.X); i >= 0 && i != m.activeTab {
				return m.switchTab(i)
			}
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.activeTab == tabCalendar && m.selectedPost == nil && !m.showReadme && !m.showHelp && !m.inPalette && m.ask == nil {
			switch msg.Button {
			case tea.MouseWheelUp:
				m.calDay, m.calCursor = dayStart(m.calSel().AddDate(0, 0, -7)), 0
			case tea.MouseWheelDown:
				m.calDay, m.calCursor = dayStart(m.calSel().AddDate(0, 0, 7)), 0
			}
			return m, nil
		}
		if m.showReadme {
			if m.readmeFocus == 0 {
				switch msg.Button {
				case tea.MouseWheelUp:
					m.tocCursor = max(0, m.tocCursor-1)
					return m, nil
				case tea.MouseWheelDown:
					m.tocCursor = min(len(m.readmeTOC)-1, m.tocCursor+1)
					return m, nil
				}
			} else {
				switch msg.Button {
				case tea.MouseWheelUp:
					m.readmeScroll = max(0, m.readmeScroll-1)
					return m, nil
				case tea.MouseWheelDown:
					maxScroll := len(m.readmeLines) - m.getReadmeViewportHeight()
					if maxScroll < 0 {
						maxScroll = 0
					}
					m.readmeScroll = min(maxScroll, m.readmeScroll+1)
					return m, nil
				}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.ask != nil {
			nm, cmd := m.askKey(msg)
			return nm, cmd
		}
		if m.inPalette {
			nm, cmd := m.paletteKey(msg)
			return nm, cmd
		}
		if m.searching {
			nm, cmd := m.searchKey(msg)
			return nm, cmd
		}
		if msg.String() == "/" && m.activeTab == tabPosts && m.selectedPost == nil && !m.isEditing && !m.editingQueueSlots && !m.showReadme && m.err == nil && !m.loading {
			return m.openSearch(), nil
		}
		if msg.String() == "esc" && m.searchQuery != "" && m.activeTab == tabPosts && m.selectedPost == nil && !m.isEditing {
			m.searchQuery, m.cursor = "", 0
			return m, nil
		}
		if msg.String() == ":" && !m.isEditing && !m.editingQueueSlots && !m.showReadme && m.err == nil && !m.loading {
			return m.openPalette(), nil
		}
		// Wenn wir die Queue Slots bearbeiten, verarbeite nur diese Tasten
		if m.editingQueueSlots {
			var cmd tea.Cmd
			switch {
			case msg.String() == "esc":
				m.editingQueueSlots = false
				return m, nil
			case msg.String() == "enter":
				inputVal := m.queueSlotsInput.Value()
				parts := strings.Split(inputVal, ",")
				var newSlots []string
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						newSlots = append(newSlots, trimmed)
					}
				}
				config.ActiveConfig.Scheduler.Slots = newSlots
				if err := config.SaveConfig(); err != nil {
					m.err = err
				}
				m.editingQueueSlots = false
				return m, nil
			default:
				m.queueSlotsInput, cmd = m.queueSlotsInput.Update(msg)
				return m, cmd
			}
		}

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
					m.tocCursor = max(0, m.tocCursor-1)
				} else {
					m.readmeScroll = max(0, m.readmeScroll-1)
				}
				return m, nil

			case key.Matches(msg, Keys.Down):
				if m.readmeFocus == 0 {
					m.tocCursor = min(len(m.readmeTOC)-1, m.tocCursor+1)
				} else {
					maxScroll := len(m.readmeLines) - m.getReadmeViewportHeight()
					if maxScroll < 0 {
						maxScroll = 0
					}
					m.readmeScroll = min(maxScroll, m.readmeScroll+1)
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

		// Filter zurücksetzen oder Auswahl aufheben bei Esc im Posts-Tab, wenn keine Detailansicht offen ist
		if m.activeTab == tabPosts && m.selectedPost == nil && !m.showReadme && key.Matches(msg, Keys.Esc) {
			hasSelected := false
			for _, sel := range m.selectedPosts {
				if sel {
					hasSelected = true
					break
				}
			}
			if hasSelected {
				m.selectedPosts = make(map[string]bool)
			} else if m.filterCampaign != "" {
				m.filterCampaign = ""
				m.cursor = 0
			}
			return m, nil
		}

		// Leertaste wählt Post aus / wählt Post ab im Posts-Tab (Massenaktionen)
		if m.activeTab == tabPosts && m.selectedPost == nil && !m.showReadme && msg.String() == "space" {
			filtered := m.getFilteredPosts()
			if len(filtered) > 0 && m.cursor < len(filtered) {
				p := filtered[m.cursor]
				m.selectedPosts[p.ID] = !m.selectedPosts[p.ID]
			}
			return m, nil
		}

		// History-Detailansicht-Steuerung
		if m.selectedHistory != nil {
			m.statusMessage = ""
			switch {
			case key.Matches(msg, Keys.Esc):
				m.selectedHistory = nil
				m.detailScrollOffset = 0
				return m, nil
			case key.Matches(msg, Keys.Export):
				return m, m.exportHistoryEntryCmd(m.selectedHistory)
			case key.Matches(msg, Keys.Up) || msg.String() == "k":
				if m.detailScrollOffset > 0 {
					m.detailScrollOffset--
				}
				return m, nil
			case key.Matches(msg, Keys.Down) || msg.String() == "j":
				m.detailScrollOffset++
				return m, nil
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
				m.detailScrollOffset = 0
				return m, nil
			case key.Matches(msg, Keys.Delete):
				idToDelete := m.selectedPost.ID
				m.detailScrollOffset = 0
				return m, m.deletePostCmd(idToDelete)
			case key.Matches(msg, Keys.Edit):
				postToEdit := *m.selectedPost
				m.selectedPost = nil
				m.detailScrollOffset = 0
				m.initEditor(&postToEdit)
				return m, nil
			case key.Matches(msg, Keys.Up) || msg.String() == "k":
				if m.detailScrollOffset > 0 {
					m.detailScrollOffset--
				}
				return m, nil
			case key.Matches(msg, Keys.Down) || msg.String() == "j":
				m.detailScrollOffset++
				return m, nil
			case key.Matches(msg, Keys.Post):
				return m.askPublish([]string{m.selectedPost.ID})
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
				m.statusMessage = Tr("st_ai_converting")
				return m, m.repurposePostCmd(p, targets)
			}
			return m, nil
		}

		if m.activeTab == tabLogs && msg.String() == "s" && !m.showReadme && !m.showHelp {
			m.logSource = (m.logSource + 1) % 2
			return m, nil
		}
		if m.activeTab == tabCalendar && m.selectedPost == nil && !m.showReadme && !m.showHelp {
			if nm, cmd, ok := m.calendarKey(msg); ok {
				return nm, cmd
			}
		}

		// Hauptmenü-Steuerung
		switch {
		case key.Matches(msg, Keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, Keys.Profile):
			m.switchProfile = true
			return m, tea.Quit

		case key.Matches(msg, Keys.Tab):
			return m.switchTab(m.activeTab + 1)

		case key.Matches(msg, Keys.ShiftTab):
			return m.switchTab(m.activeTab - 1)

		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
				if m.activeTab == tabSettings && m.settingsOptions()[m.cursor].kind == settingDisplay {
					m.cursor--
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Down):
			maxItems := m.maxCursorItems()
			if m.cursor < maxItems-1 {
				m.cursor++
				if m.activeTab == tabSettings && m.settingsOptions()[m.cursor].kind == settingDisplay {
					m.cursor++
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Left), key.Matches(msg, Keys.Right):
			if m.activeTab == tabSettings && m.settingsOptions()[m.cursor].kind == settingCyclable {
				dir := 1
				if key.Matches(msg, Keys.Left) {
					dir = -1
				}
				m.cycleSettingDir(dir)
				return m, nil
			}
			return m, nil

		case key.Matches(msg, Keys.Enter):
			if m.activeTab == tabSettings {
				opt := m.settingsOptions()[m.cursor]
				switch opt.kind {
				case settingPlatformAuth:
					platName := opt.platform
					if platformNeedsSetup(platName) || (platName == models.PlatformTwitter && config.ActiveConfig.Twitter.AuthMode == "cookie") {
						return m, m.runSetupWizardCmd(platName)
					}
					m.loading = true
					m.statusMessage = fmt.Sprintf(Tr("st_open_browser"), platName)
					return m, m.runAuthCmd(platName)
				case settingAction:
					switch opt.action {
					case actionExport:
						return m, m.runBackupExportCmd()
					case actionImport:
						return m, m.runBackupImportCmd()
					case actionScheduler:
						var err error
						if service.Running(config.ActiveProfile) {
							err = service.Uninstall(config.ActiveProfile)
						} else {
							err = service.Install(config.ActiveProfile)
						}
						if err != nil {
							m.err = err
						}
						return m, nil
					case actionEditSlots:
						m.editingQueueSlots = true
						m.queueSlotsInput.SetValue(strings.Join(config.ActiveConfig.Scheduler.Slots, ", "))
						m.queueSlotsInput.Focus()
						return m, nil
					}
				}
				m.cycleSetting()
				return m, nil
			}
			if m.activeTab == tabDashboard && len(m.campaigns) > 0 {
				m.filterCampaign = m.campaigns[m.cursor].Slug
				m.activeTab = tabPosts
				m.cursor = 0
				return m, nil
			}
			if m.activeTab == tabPosts {
				filtered := m.getFilteredPosts()
				if len(filtered) > 0 {
					m.selectedPost = &filtered[m.cursor]
				}
			}
			if m.activeTab == tabSchedule {
				if len(m.nextUp) > 0 && m.cursor < len(m.nextUp) {
					m.selectedPost = &m.nextUp[m.cursor]
				}
			}
			if m.activeTab == tabHistory {
				if len(m.history) > 0 && m.cursor < len(m.history) {
					m.selectedHistory = &m.history[m.cursor]
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Delete):
			if m.activeTab == tabPosts {
				var ids []string
				for id, selected := range m.selectedPosts {
					if selected {
						ids = append(ids, id)
					}
				}
				if len(ids) > 0 {
					m.selectedPosts = make(map[string]bool)
					return m, m.deletePostsCmd(ids)
				}

				filtered := m.getFilteredPosts()
				if len(filtered) > 0 {
					idToDelete := filtered[m.cursor].ID
					return m, m.deletePostCmd(idToDelete)
				}
			} else if m.activeTab == tabSettings { // Settings
				opt := m.settingsOptions()[m.cursor]
				if opt.kind == settingPlatformAuth {
					m.loading = true
					m.statusMessage = fmt.Sprintf(Tr("st_resetting"), opt.platform)
					return m, m.clearPlatformCmd(opt.platform)
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Export):
			if m.selectedPost == nil && !m.showReadme {
				if m.activeTab == tabHistory {
					return m, m.exportHistoryCmd(m.history)
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Filter):
			if m.selectedPost == nil && !m.showReadme && m.activeTab == tabPosts {
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
				if m.activeTab == tabPosts { // Posts
					filtered := m.getFilteredPosts()
					if len(filtered) > 0 && m.cursor < len(filtered) {
						postToEdit := filtered[m.cursor]
						m.initEditor(&postToEdit)
						return m, nil
					}
				} else if m.activeTab == tabSchedule { // Schedule
					if len(m.nextUp) > 0 && m.cursor < len(m.nextUp) {
						postToEdit := m.nextUp[m.cursor]
						m.initEditor(&postToEdit)
						return m, nil
					}
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Schedule):
			if m.selectedPost == nil && !m.showReadme {
				if m.activeTab == tabPosts { // Posts
					var ids []string
					for id, selected := range m.selectedPosts {
						if selected {
							ids = append(ids, id)
						}
					}
					if len(ids) > 0 {
						m.selectedPosts = make(map[string]bool)
						return m, m.bulkSchedulePostsCmd(ids)
					}

					filtered := m.getFilteredPosts()
					if len(filtered) > 0 && m.cursor < len(filtered) {
						idToSchedule := filtered[m.cursor].ID
						return m, m.bulkSchedulePostsCmd([]string{idToSchedule})
					}
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Post):
			if m.selectedPost == nil && !m.showReadme {
				if m.activeTab == tabPosts { // Posts
					var ids []string
					for id, selected := range m.selectedPosts {
						if selected {
							ids = append(ids, id)
						}
					}
					if len(ids) > 0 {
						m.selectedPosts = make(map[string]bool)
						return m.askPublish(ids)
					}

					filtered := m.getFilteredPosts()
					if len(filtered) > 0 && m.cursor < len(filtered) {
						return m.askPublish([]string{filtered[m.cursor].ID})
					}
				} else if m.activeTab == tabSchedule { // Schedule
					if len(m.nextUp) > 0 && m.cursor < len(m.nextUp) {
						return m.askPublish([]string{m.nextUp[m.cursor].ID})
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
	case tabDashboard:
		return len(m.campaigns)
	case tabPosts:
		return len(m.getFilteredPosts())
	case tabSchedule:
		return len(m.nextUp)
	case tabHistory:
		return len(m.history)
	case tabSettings:
		return len(m.settingsOptions())
	default:
		return 0
	}
}

// getFilteredPosts gibt die Liste der Posts unter Berücksichtigung des Kampagnen-Filters zurück
func (m Model) getFilteredPosts() []models.Post {
	if m.filterCampaign == "" && strings.TrimSpace(m.searchQuery) == "" {
		return m.posts
	}
	var filtered []models.Post
	for _, p := range m.posts {
		if (m.filterCampaign == "" || p.Campaign == m.filterCampaign) && m.matchesSearch(p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// View rendert den Bildschirm als Zeichenkette
func (m Model) View() tea.View {
	v := tea.NewView(m.viewContent())
	// v1's WithAltScreen()/WithMouseCellMotion() Program options are gone in
	// v2 — they are per-View fields now.
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) viewContent() string {
	w, h := m.bodyDims()
	var body string
	switch {
	case m.loading:
		msg := m.statusMessage
		if msg == "" {
			msg = Tr("loading_data")
		}
		body = m.indentLines(lipgloss.NewStyle().Foreground(theme.BlueV2).Render(m.spinner()) + " " + msg)
	case m.err != nil:
		body = m.indentLines(ui.Pill(Tr("error_title"), ui.Err) + " " + fmt.Sprint(m.err) + "\n\n" + Tr("error_close"))
	default:
		body = m.indentLines(m.tabBody(w, h))
	}
	header := m.indentLines(strings.Join(m.headerLines(), "\n"))
	footer := m.indentLines(m.footerLine())
	return ui.Frame(m.frameHeight(), header, body, footer)
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

		result, err := generator.RepurposeContent(ctx, aiCfg, p.Platform, p.Type, p.Title, srcContent, targets, "")
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
// cycleSetting changes the currently selected cyclable setting's value and
// persists it. Keyed by settingOption.settingKey (that row's stable i18n
// label key — see settingsOptions' doc comment) rather than m.cursor
// directly, so this stays correct regardless of where a row sits in the
// list.
func (m *Model) cycleSetting() { m.cycleSettingDir(1) }

// cycleSettingDir cycles the selected setting; dir is -1 for ← and +1 for → / Enter
// (only pickers with more than two values, like the theme, care about the direction).
func (m *Model) cycleSettingDir(dir int) {
	switch m.settingsOptions()[m.cursor].settingKey {
	case "settings_theme":
		names := theme.Presets()
		idx := 0
		for i, n := range names {
			if n == theme.Current() {
				idx = i
			}
		}
		next := names[(idx+dir+len(names))%len(names)]
		if err := theme.Apply(next); err != nil {
			m.err = err
			return
		}
		_ = theme.Save(next) // every tool of the suite starts with it next time
		m.statusMessage = fmt.Sprintf(Tr("st_theme"), next)
		return
	case "settings_ai_provider":
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
	case "settings_ai_model":
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
	case "settings_dry_run":
		config.ActiveConfig.Defaults.DryRun = !config.ActiveConfig.Defaults.DryRun
	case "settings_auto_publish":
		// Gates the TUI's own 10s background-tick publisher (see
		// publishDuePostsCmd/tickMsg), off by default so a second
		// Dropbox-synced machine running postctl doesn't also see the same
		// post as due and double-publish it. Toggling this here is meant
		// for deciding, per machine, which one auto-fires the scheduler —
		// see DBSettled's doc comment for the other half of that safety
		// net.
		config.ActiveConfig.Scheduler.AutoPublish = !config.ActiveConfig.Scheduler.AutoPublish
	case "settings_language":
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

// getReadmeViewportHeight is the number of README lines the panel shows.
func (m Model) getReadmeViewportHeight() int {
	_, h := m.bodyDims()
	return max(h-2, 1)
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

// storeReleasingCmd runs a child process while this TUI's database lock is released: postctl's
// store is exclusive, so a wizard or import started from the TUI could not open the database
// at all (it failed, or — for the X cookie wizard — saved nothing). The lock is taken back when
// the child exits.
type storeReleasingCmd struct {
	cmd   *exec.Cmd
	store *store.SQLiteStore
}

func (c *storeReleasingCmd) SetStdin(r io.Reader)  { c.cmd.Stdin = r }
func (c *storeReleasingCmd) SetStdout(w io.Writer) { c.cmd.Stdout = w }
func (c *storeReleasingCmd) SetStderr(w io.Writer) { c.cmd.Stderr = w }

func (c *storeReleasingCmd) Run() error {
	if c.store != nil {
		_ = c.store.Suspend()
		defer func() { _ = c.store.Resume() }()
	}
	return c.cmd.Run()
}

func (m Model) execWithStoreReleased(c *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
	return tea.Exec(&storeReleasingCmd{cmd: c, store: m.store}, cb)
}
