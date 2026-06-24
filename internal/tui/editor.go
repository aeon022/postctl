package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// initEditor initialisiert die Eingabefelder des Editors mit Werten eines bestehenden Beitrags oder leer
func (m *Model) initEditor(p *models.Post) {
	campaignInput := textinput.New()
	campaignInput.Placeholder = "z.B. launch-2026"
	
	schedInput := textinput.New()
	schedInput.Placeholder = "leer lassen für Entwurf (Draft) oder 'DD.MM.YYYY HH:MM'"
	
	imagesInput := textinput.New()
	imagesInput.Placeholder = "z.B. bild1.png, bild2.png (Komma-separiert)"
	
	bodyArea := textarea.New()
	bodyArea.Placeholder = "Schreibe deinen Beitrag hier..."
	bodyArea.SetWidth(70)
	bodyArea.SetHeight(8)
	
	if p == nil {
		m.editorPostID = ""
		m.editorPlatform = "twitter"
		campaignInput.SetValue("")
		schedInput.SetValue("")
		imagesInput.SetValue("")
		bodyArea.SetValue("")
	} else {
		m.editorPostID = p.ID
		m.editorPlatform = p.Platform
		campaignInput.SetValue(p.Campaign)
		
		if p.ScheduledAt != nil {
			schedInput.SetValue(p.ScheduledAt.Format("02.01.2006 15:04"))
		} else {
			schedInput.SetValue("")
		}
		
		imagesInput.SetValue(strings.Join(p.Images, ", "))
		
		if p.Type == "thread" && len(p.Tweets) > 0 {
			var sb strings.Builder
			for i, tweet := range p.Tweets {
				if i > 0 {
					sb.WriteString("\n---\n")
				}
				sb.WriteString(tweet.Content)
			}
			bodyArea.SetValue(sb.String())
		} else {
			bodyArea.SetValue(p.Body)
		}
	}
	
	m.editorCampaign = campaignInput
	m.editorScheduledAt = schedInput
	m.editorImages = imagesInput
	m.editorBody = bodyArea
	m.editorFocus = 0
	m.isEditing = true
	
	m.updateEditorFocus()
}

// updateEditorFocus steuert den Eingabefokus der Formularfelder im Editor
func (m *Model) updateEditorFocus() {
	m.editorCampaign.Blur()
	m.editorScheduledAt.Blur()
	m.editorImages.Blur()
	m.editorBody.Blur()
	
	switch m.editorFocus {
	case 1:
		m.editorCampaign.Focus()
	case 2:
		m.editorScheduledAt.Focus()
	case 3:
		m.editorImages.Focus()
	case 4:
		m.editorBody.Focus()
	}
}

// saveEditedPost validiert die Formulareingaben und speichert den Beitrag in der SQLite-Datenbank
func (m *Model) saveEditedPost() error {
	ctx := context.Background()
	platform := m.editorPlatform
	campaign := strings.TrimSpace(m.editorCampaign.Value())
	if campaign == "" {
		campaign = "default"
	}
	
	schedStr := strings.TrimSpace(m.editorScheduledAt.Value())
	var scheduledAt *time.Time
	status := "draft"
	if schedStr != "" {
		t, err := time.ParseInLocation("02.01.2006 15:04", schedStr, time.Local)
		if err != nil {
			return fmt.Errorf("ungültiges Datumformat. Bitte verwende 'DD.MM.YYYY HH:MM'")
		}
		scheduledAt = &t
		status = "scheduled"
	}
	
	// Parse Bilder
	imagesStr := m.editorImages.Value()
	var images []string
	if strings.TrimSpace(imagesStr) != "" {
		parts := strings.Split(imagesStr, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				images = append(images, trimmed)
			}
		}
	}
	
	body := m.editorBody.Value()
	
	// ID beibehalten oder neu generieren
	id := m.editorPostID
	if id == "" {
		id = fmt.Sprintf("%s-%s-%d", campaign, platform, time.Now().UnixNano()/1e6)
	}
	
	post := models.Post{
		ID:          id,
		Platform:    platform,
		Campaign:    campaign,
		Status:      status,
		ScheduledAt: scheduledAt,
		Images:      images,
		Language:    "de",
		SourceFile:  "TUI Editor",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Falls Plattform Twitter/X, Mastodon oder Bluesky ist und '---' vorkommt, in Thread aufspalten
	if (platform == "twitter" || platform == "mastodon" || platform == "bluesky") && strings.Contains(body, "\n---\n") {
		post.Type = "thread"
		tweetParts := strings.Split(body, "\n---\n")
		for i, part := range tweetParts {
			post.Tweets = append(post.Tweets, models.Tweet{
				Index:   i + 1,
				Content: strings.TrimSpace(part),
			})
		}
	} else {
		post.Type = "single"
		post.Body = body
	}
	
	// In SQLite speichern
	if err := m.store.SavePost(ctx, &post); err != nil {
		return err
	}
	
	return nil
}

// renderEditor zeichnet die Editor-Maske im Terminal
func (m Model) renderEditor() string {
	var builder strings.Builder

	titleText := Tr("editor_title_create")
	if m.editorPostID != "" {
		titleText = Tr("editor_title_edit")
	}

	builder.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBg).
		Background(ColorSecondary).
		Padding(0, 1).
		Render(titleText))
	builder.WriteString("\n\n")

	// 1. Plattform
	platPrefix := "  "
	platStyle := lipgloss.NewStyle().Foreground(ColorText)
	if m.editorFocus == 0 {
		platPrefix = "➔ "
		platStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	}
	platformLabel := platPrefix + Tr("editor_label_platform")
	
	platSelect := ""
	platformsList := []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook"}
	for _, p := range platformsList {
		if p == m.editorPlatform {
			platSelect += lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render(" [" + strings.ToUpper(p) + "] ")
		} else {
			platSelect += "  " + strings.ToUpper(p) + "  "
		}
	}
	builder.WriteString(platStyle.Render(platformLabel) + platSelect + "\n\n")

	// 2. Kampagne
	campPrefix := "  "
	campStyle := lipgloss.NewStyle().Foreground(ColorText)
	if m.editorFocus == 1 {
		campPrefix = "➔ "
		campStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	}
	campLabel := campPrefix + Tr("editor_label_campaign")
	builder.WriteString(campStyle.Render(campLabel) + m.editorCampaign.View() + "\n\n")

	// 3. Geplantes Datum
	schedPrefix := "  "
	schedStyle := lipgloss.NewStyle().Foreground(ColorText)
	if m.editorFocus == 2 {
		schedPrefix = "➔ "
		schedStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	}
	schedLabel := schedPrefix + Tr("editor_label_schedule")
	builder.WriteString(schedStyle.Render(schedLabel) + m.editorScheduledAt.View() + "\n\n")

	// 4. Bilder
	imgPrefix := "  "
	imgStyle := lipgloss.NewStyle().Foreground(ColorText)
	if m.editorFocus == 3 {
		imgPrefix = "➔ "
		imgStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	}
	imgLabel := imgPrefix + Tr("editor_label_images")
	builder.WriteString(imgStyle.Render(imgLabel) + m.editorImages.View() + "\n\n")

	// 5. Text-Inhalt
	bodyPrefix := "  "
	bodyStyle := lipgloss.NewStyle().Foreground(ColorText)
	bodyLabel := Tr("editor_label_body")
	if m.editorPlatform == "twitter" || m.editorPlatform == "mastodon" || m.editorPlatform == "bluesky" {
		bodyLabel += Tr("editor_twitter_thread_note")
	}
	if m.editorFocus == 4 {
		bodyPrefix = "➔ "
		bodyStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	}
	builder.WriteString(bodyStyle.Render(bodyPrefix + bodyLabel) + "\n" + m.editorBody.View() + "\n\n")

	// 6. Action-Buttons
	saveLabel := Tr("editor_save")
	cancelLabel := Tr("editor_cancel")
	
	if m.editorFocus == 5 {
		saveLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorPosted).Render(saveLabel)
	} else {
		saveLabel = lipgloss.NewStyle().Foreground(ColorPosted).Render(saveLabel)
	}
	
	if m.editorFocus == 6 {
		cancelLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorFailed).Render(cancelLabel)
	} else {
		cancelLabel = lipgloss.NewStyle().Foreground(ColorFailed).Render(cancelLabel)
	}
	
	builder.WriteString("  " + saveLabel + "     " + cancelLabel + "\n\n")

	// Help footer
	helpStr := Tr("editor_help_footer")
	builder.WriteString(StyleHelp.Render(helpStr))

	return StyleBox.Width(78).Height(24).Render(builder.String())
}
