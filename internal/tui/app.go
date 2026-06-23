package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	selectedPost   *models.Post
	filterCampaign string
	isEditing      bool
	showHelp       bool
	err            error
	loading        bool
	repurposing    bool
	statusMessage  string
	
	// Editor Zustand
	editorPostID      string
	editorPlatform    string
	editorCampaign    textinput.Model
	editorScheduledAt textinput.Model
	editorImages      textinput.Model
	editorBody        textarea.Model
	editorFocus       int
	
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

type postDeletedMsg struct {
	id string
}

type postRepurposedMsg struct {
	files []string
	err   error
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
	// Ein einfacher Bubble-Sort, um externe Bibliotheken zu vermeiden
	for i := 0; i < len(nextUp)-1; i++ {
		for j := 0; j < len(nextUp)-i-1; j++ {
			if nextUp[j].ScheduledAt != nil && nextUp[j+1].ScheduledAt != nil {
				if nextUp[j].ScheduledAt.After(*nextUp[j+1].ScheduledAt) {
					nextUp[j], nextUp[j+1] = nextUp[j+1], nextUp[j]
				}
			}
		}
	}
	
	platforms := map[string]bool{
		models.PlatformTwitter:  false,
		models.PlatformLinkedIn: false,
		models.PlatformThreads:  false,
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

// runBackupExportCmd pausiert die TUI und startet interaktiv den CLI-Export
func (m Model) runBackupExportCmd() tea.Cmd {
	c := exec.Command("./postctl", "config", "export")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: true, err: err}
	})
}

// runBackupImportCmd pausiert die TUI und startet interaktiv den CLI-Import
func (m Model) runBackupImportCmd() tea.Cmd {
	c := exec.Command("./postctl", "config", "import")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return backupFinishedMsg{isExport: false, err: err}
	})
}

// runImportPostsCmd pausiert die TUI und startet interaktiv den CLI-Post-Import
func (m Model) runImportPostsCmd() tea.Cmd {
	c := exec.Command("./postctl", "import")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return importFinishedMsg{err: err}
	})
}

// publishDuePostsCmd prüft und veröffentlicht fällige Posts im TUI-Hintergrund
func (m Model) publishDuePostsCmd() tea.Msg {
	ctx := context.Background()
	now := time.Now()
	
	posts, err := m.store.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err == nil {
		for _, p := range posts {
			if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
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
			}

			if m.editorFocus == 0 {
				platformsList := []string{"twitter", "linkedin", "threads"}
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
		
	case postRepurposedMsg:
		m.repurposing = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Erfolgreich konvertiert: %s", strings.Join(msg.files, ", "))
		return m, m.loadDataCmd

	case authResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("Authentifizierung fehlgeschlagen: %w", msg.err)
			m.statusMessage = ""
		} else {
			m.statusMessage = fmt.Sprintf("Erfolgreich mit %s verbunden!", msg.platform)
		}
		return m, m.loadDataCmd

	case backupFinishedMsg:
		if msg.err != nil {
			m.err = fmt.Errorf("Backup-Aktion fehlgeschlagen: %w", msg.err)
			m.statusMessage = ""
		} else {
			action := "importiert"
			if msg.isExport {
				action = "exportiert (postctl_backup.bin)"
			}
			m.statusMessage = fmt.Sprintf("Konfiguration erfolgreich %s!", action)
		}
		return m, m.loadDataCmd

	case importFinishedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Import fehlgeschlagen: %v", msg.err)
		} else {
			m.statusMessage = "Beiträge erfolgreich importiert!"
		}
		return m, m.loadDataCmd

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
				switch p.Platform {
				case models.PlatformTwitter:
					targets = []string{models.PlatformLinkedIn, models.PlatformThreads}
				case models.PlatformLinkedIn:
					targets = []string{models.PlatformTwitter, models.PlatformThreads}
				case models.PlatformThreads:
					targets = []string{models.PlatformTwitter, models.PlatformLinkedIn}
				default:
					targets = []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads}
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
			m.activeTab = (m.activeTab + 1) % 5
			m.cursor = 0
			return m, nil
			
		case key.Matches(msg, Keys.ShiftTab):
			m.activeTab = (m.activeTab - 1 + 5) % 5
			m.cursor = 0
			return m, nil
			
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
				if m.activeTab == 4 && m.cursor == 3 {
					m.cursor--
				}
			}
			return m, nil
			
		case key.Matches(msg, Keys.Down):
			maxItems := m.maxCursorItems()
			if m.cursor < maxItems-1 {
				m.cursor++
				if m.activeTab == 4 && m.cursor == 3 {
					m.cursor++
				}
			}
			return m, nil

		case key.Matches(msg, Keys.Left), key.Matches(msg, Keys.Right):
			if m.activeTab == 4 && m.cursor < 3 {
				m.cycleSetting()
				return m, nil
			}
			return m, nil
			
		case key.Matches(msg, Keys.Enter):
			if m.activeTab == 4 {
				if m.cursor >= 4 && m.cursor <= 6 {
					var platName string
					switch m.cursor {
					case 4:
						platName = models.PlatformTwitter
					case 5:
						platName = models.PlatformLinkedIn
					case 6:
						platName = models.PlatformThreads
					}
					m.loading = true
					m.statusMessage = fmt.Sprintf("Öffne Browser für %s...", platName)
					return m, m.runAuthCmd(platName)
				}
				if m.cursor == 7 {
					return m, m.runBackupExportCmd()
				}
				if m.cursor == 8 {
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
			return m, nil

		case key.Matches(msg, Keys.Delete):
			if m.activeTab == 1 {
				filtered := m.getFilteredPosts()
				if len(filtered) > 0 {
					idToDelete := filtered[m.cursor].ID
					return m, m.deletePostCmd(idToDelete)
				}
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
	case 4: // Settings
		return 9
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

