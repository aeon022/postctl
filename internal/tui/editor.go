package tui

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

// initEditor initialisiert die Eingabefelder des Editors mit Werten eines bestehenden Beitrags oder leer
func (m *Model) initEditor(p *models.Post) {
	campaignInput := textinput.New()
	campaignInput.Placeholder = "z.B. launch-2026"

	schedInput := textinput.New()
	schedInput.Placeholder = "leer für Entwurf, 'now' für sofort, oder 'TT.MM.JJJJ HH:MM' [ctrl+d für Kalender]"

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

	schedStr := strings.TrimSpace(strings.ToLower(m.editorScheduledAt.Value()))
	var scheduledAt *time.Time
	status := "draft"
	if schedStr != "" {
		if schedStr == "now" || schedStr == "jetzt" {
			t := time.Now()
			scheduledAt = &t
			status = "scheduled"
		} else {
			t, err := time.ParseInLocation("02.01.2006 15:04", m.editorScheduledAt.Value(), time.Local)
			if err != nil {
				return fmt.Errorf("ungültiges Datumformat. Bitte verwende 'DD.MM.YYYY HH:MM' oder 'now' / 'jetzt'")
			}
			scheduledAt = &t
			status = "scheduled"
		}
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

	// Vorhandenen Titel beibehalten oder automatisch generieren
	title := ""
	if m.editorPostID != "" {
		if existing, err := m.store.GetPost(ctx, m.editorPostID); err == nil && existing != nil {
			title = existing.Title
		}
	}
	if title == "" {
		title = models.DeriveTitle(body)
	}

	post := models.Post{
		ID:          id,
		Platform:    platform,
		Campaign:    campaign,
		Title:       title,
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

	post.PrepareTweets()

	// In SQLite speichern
	if err := m.store.SavePost(ctx, &post); err != nil {
		return err
	}

	return nil
}

// renderEditor zeichnet die Editor-Maske im Terminal
func (m Model) renderEditor(w, h int) string {
	var builder strings.Builder
	var fieldAt [7]int // line where each focus target starts, to keep the focused one in view
	mark := func(i int) { fieldAt[i] = strings.Count(builder.String(), "\n") }
	label := func(focus int, text string) string {
		if m.editorFocus == focus {
			return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("➔ " + text)
		}
		return lipgloss.NewStyle().Foreground(ColorLightGray).Render("  " + text)
	}
	iw := panelRowW(w)

	titleText := Tr("editor_title_create")
	if m.editorPostID != "" {
		titleText = Tr("editor_title_edit")
	}

	// 1. Plattform (wrapped so the selected one never falls off a narrow terminal)
	mark(0)
	var platSelect []string
	for _, p := range []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook", "telegram", "discord", "devto", "reddit", "hashnode", "medium"} {
		if p == m.editorPlatform {
			platSelect = append(platSelect, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("["+strings.ToUpper(p)+"]"))
		} else {
			platSelect = append(platSelect, strings.ToUpper(p))
		}
	}
	builder.WriteString(label(0, Tr("editor_label_platform")) + "\n")
	for _, l := range wrapLines(strings.Join(platSelect, "  "), iw-4) {
		builder.WriteString("    " + l + "\n")
	}
	builder.WriteString("\n")

	// 2. Kampagne
	mark(1)
	builder.WriteString(label(1, Tr("editor_label_campaign")) + m.editorCampaign.View() + "\n\n")

	// 3. Geplantes Datum
	mark(2)
	builder.WriteString(label(2, Tr("editor_label_schedule")) + m.editorScheduledAt.View() + "\n")
	if m.editorFocus == 2 && !m.showDatePicker {
		builder.WriteString(dimStyle.Render("     "+Tr("editor_tip_now")) + "\n\n")
	} else {
		builder.WriteString("\n")
	}

	// 4. Bilder
	mark(3)
	builder.WriteString(label(3, Tr("editor_label_images")) + m.editorImages.View() + "\n")
	for _, part := range strings.Split(m.editorImages.Value(), ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		fullPath := trimmed
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fullPath = filepath.Join(config.ActiveConfig.Defaults.ImageDir, trimmed)
		}
		if _, err := os.Stat(fullPath); err != nil {
			continue
		}
		if preview := renderImageANSI(fullPath, 40, 10); preview != "" {
			builder.WriteString("     " + lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(Tr("editor_preview_label")) + "\n" + preview + "\n")
			break
		}
	}
	if m.editorFocus == 3 {
		builder.WriteString(dimStyle.Render(Tr("editor_images_help")) + "\n\n")
	} else {
		builder.WriteString("\n")
	}

	// 5. Text-Inhalt (width follows the terminal; local copy, the model keeps its own)
	mark(4)
	bodyLabel := Tr("editor_label_body")
	if m.editorPlatform == "twitter" || m.editorPlatform == "mastodon" || m.editorPlatform == "bluesky" {
		bodyLabel += Tr("editor_twitter_thread_note")
	}
	body := m.editorBody
	body.SetWidth(min(max(iw-4, 20), 100))
	builder.WriteString(label(4, bodyLabel) + "\n" + body.View() + "\n")
	if charLimitMsg, _ := m.checkCharacterLimits(); charLimitMsg != "" {
		builder.WriteString("     " + charLimitMsg + "\n\n")
	} else {
		builder.WriteString("\n")
	}

	// 6. Action-Buttons
	mark(5)
	fieldAt[6] = fieldAt[5]
	saveLabel, cancelLabel := Tr("editor_save"), Tr("editor_cancel")
	if m.editorFocus == 5 {
		saveLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBgFg).Background(ColorPosted).Render(saveLabel)
	} else {
		saveLabel = lipgloss.NewStyle().Foreground(ColorPosted).Render(saveLabel)
	}
	if m.editorFocus == 6 {
		cancelLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBgFg).Background(ColorFailed).Render(cancelLabel)
	} else {
		cancelLabel = lipgloss.NewStyle().Foreground(ColorFailed).Render(cancelLabel)
	}
	builder.WriteString("  " + saveLabel + "     " + cancelLabel)

	// scroll so the focused field (and, for the body, its text area) is visible
	room, span := max(h-2, 1), 3
	switch {
	case m.editorFocus == 4:
		span = body.Height() + 2
	case m.editorFocus >= 5:
		span = 1
	}
	off := max(fieldAt[m.editorFocus]+span-room, 0)
	page := scrollPanel(w, h, strings.TrimSpace(titleText), strings.Split(builder.String(), "\n"), off, true)

	if m.showDatePicker {
		cal := ui.Panel(min(w, 30), 11, Tr("panel_calendar"), m.renderCalendar(m.datePickerDate), true)
		return popup(page, cal, w, h)
	}
	return page
}

func (m Model) renderCalendar(selectedDate time.Time) string {
	year := selectedDate.Year()
	month := selectedDate.Month()

	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local)

	startOffset := int(firstDay.Weekday()) - 1
	if startOffset < 0 {
		startOffset = 6
	}

	numDays := lastDay.Day()

	var sb strings.Builder
	header := fmt.Sprintf("  <<< %s %d >>>  ", month.String(), year)

	sb.WriteString("  " + lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(header) + "\n")
	sb.WriteString("   " + Tr("cal_weekdays") + "\n")
	sb.WriteString("   ")

	for i := 0; i < startOffset; i++ {
		sb.WriteString("   ")
	}

	for day := 1; day <= numDays; day++ {
		if (startOffset+day-1)%7 == 0 && day > 1 {
			sb.WriteString("\n   ")
		}

		dayStr := fmt.Sprintf("%2d", day)
		if day == selectedDate.Day() {
			sb.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBgFg).
				Background(ColorSecondary).
				Render(dayStr) + " ")
		} else {
			sb.WriteString(dayStr + " ")
		}
	}

	return sb.String()
}

var editorUrlRegex = regexp.MustCompile(`https?://[^\s]+`)

// checkCharacterLimits prüft die Zeichenbeschränkungen der Plattform live im Editor
func (m Model) checkCharacterLimits() (string, bool) {
	platform := m.editorPlatform
	body := m.editorBody.Value()

	limit := platformLimit(platform)
	if limit == 0 {
		return "", true
	}

	isThread := (platform == "twitter" || platform == "mastodon" || platform == "bluesky") && strings.Contains(body, "\n---\n")

	if isThread {
		parts := strings.Split(body, "\n---\n")
		var overflowIndices []int
		var counts []string

		for idx, part := range parts {
			var count int
			if platform == "twitter" {
				processed := editorUrlRegex.ReplaceAllString(part, "12345678901234567890123")
				count = len([]rune(processed))
			} else {
				count = len([]rune(strings.TrimSpace(part)))
			}

			counts = append(counts, fmt.Sprintf("%d", count))
			if count > limit {
				overflowIndices = append(overflowIndices, idx+1)
			}
		}

		countsStr := strings.Join(counts, " | ")
		if len(overflowIndices) > 0 {
			var errParts []string
			for _, o := range overflowIndices {
				errParts = append(errParts, fmt.Sprintf("#%d", o))
			}
			var errMsg string
			if config.ActiveConfig.Defaults.Language == "de" {
				errMsg = fmt.Sprintf("⚠️ Limit überschritten in Post: %s (Längen: %s, Max: %d)", strings.Join(errParts, ", "), countsStr, limit)
			} else {
				errMsg = fmt.Sprintf("⚠️ Limit exceeded in post: %s (lengths: %s, max: %d)", strings.Join(errParts, ", "), countsStr, limit)
			}
			return lipgloss.NewStyle().Foreground(ColorFailed).Render(errMsg), false
		}

		var successMsg string
		if config.ActiveConfig.Defaults.Language == "de" {
			successMsg = fmt.Sprintf("✓ Thread-Längen okay (%s, Max: %d)", countsStr, limit)
		} else {
			successMsg = fmt.Sprintf("✓ Thread lengths okay (%s, max: %d)", countsStr, limit)
		}
		return lipgloss.NewStyle().Foreground(ColorPosted).Render(successMsg), true
	} else {
		var count int
		if platform == "twitter" {
			processed := editorUrlRegex.ReplaceAllString(body, "12345678901234567890123")
			count = len([]rune(processed))
		} else {
			count = len([]rune(strings.TrimSpace(body)))
		}

		if count > limit {
			var errMsg string
			if config.ActiveConfig.Defaults.Language == "de" {
				errMsg = fmt.Sprintf("⚠️ Limit überschritten! Zeichen: %d/%d", count, limit)
			} else {
				errMsg = fmt.Sprintf("⚠️ Limit exceeded! Chars: %d/%d", count, limit)
			}
			return lipgloss.NewStyle().Foreground(ColorFailed).Render(errMsg), false
		}

		var successMsg string
		if config.ActiveConfig.Defaults.Language == "de" {
			successMsg = fmt.Sprintf("✓ Länge okay. Zeichen: %d/%d", count, limit)
		} else {
			successMsg = fmt.Sprintf("✓ Length okay. Chars: %d/%d", count, limit)
		}
		return lipgloss.NewStyle().Foreground(ColorPosted).Render(successMsg), true
	}
}

func resizeImage(img image.Image, width, height int) image.Image {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW == 0 || srcH == 0 || width == 0 || height == 0 {
		return img
	}

	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * srcW / width
			srcY := y * srcH / height
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}
	return resized
}

func renderImageANSI(path string, maxW, maxH int) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w == 0 || h == 0 {
		return ""
	}

	aspect := float64(w) / float64(h)
	targetW := maxW
	targetH := int(float64(maxW) / aspect * 0.5)

	if targetH > maxH {
		targetH = maxH
		targetW = int(float64(maxH) * aspect * 2.0)
	}

	if targetW > maxW {
		targetW = maxW
	}
	if targetH < 1 {
		targetH = 1
	}
	if targetW < 1 {
		targetW = 1
	}

	resized := resizeImage(img, targetW, targetH*2)

	var sb strings.Builder
	for y := 0; y < targetH*2; y += 2 {
		sb.WriteString("     ")
		for x := 0; x < targetW; x++ {
			cTop := resized.At(x, y)
			cBottom := resized.At(x, y+1)

			r1, g1, b1, _ := cTop.RGBA()
			r2, g2, b2, _ := cBottom.RGBA()

			topR, topG, topB := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8)
			botR, botG, botB := uint8(r2>>8), uint8(g2>>8), uint8(b2>>8)

			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▄\x1b[0m", botR, botG, botB, topR, topG, topB))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
