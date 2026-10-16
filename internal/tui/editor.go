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
// editorPlatforms is the platform selector's order.
var editorPlatforms = []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook", "instagram", "telegram", "discord", "devto", "reddit", "hashnode", "medium"}

// editor focus targets, in tab order
const (
	focusPlatform = iota
	focusCampaign
	focusSchedule
	focusImages
	focusAlts
	focusBody
	focusSave
	focusCancel
	focusCount
)

func (m *Model) initEditor(p *models.Post) {
	campaignInput := textinput.New()
	campaignInput.Placeholder = "z.B. launch-2026"

	schedInput := textinput.New()
	schedInput.Placeholder = "leer für Entwurf, 'now' für sofort, oder 'TT.MM.JJJJ HH:MM' [ctrl+d für Kalender]"

	imagesInput := textinput.New()
	imagesInput.Placeholder = "z.B. bild1.png, bild2.png (Komma-separiert)"

	altsInput := textinput.New()
	altsInput.Placeholder = "Alt-Text je Bild, getrennt mit  |  (Barrierefreiheit, max. ~100 Zeichen)"

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
		altsInput.SetValue("")
		m.editorType = ""
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
		altsInput.SetValue(strings.Join(p.Alts, " | "))
		m.editorType = p.Type

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
	m.editorAlts = altsInput
	m.editorBody = bodyArea
	m.editorFocus = 0
	m.editorTargets = map[string]bool{}
	m.isEditing = true

	m.updateEditorFocus()
}

// updateEditorFocus steuert den Eingabefokus der Formularfelder im Editor
func (m *Model) updateEditorFocus() {
	m.editorCampaign.Blur()
	m.editorScheduledAt.Blur()
	m.editorImages.Blur()
	m.editorAlts.Blur()
	m.editorBody.Blur()

	switch m.editorFocus {
	case focusCampaign:
		m.editorCampaign.Focus()
	case focusSchedule:
		m.editorScheduledAt.Focus()
	case focusImages:
		m.editorImages.Focus()
	case focusAlts:
		m.editorAlts.Focus()
	case focusBody:
		m.editorBody.Focus()
	}
}

// saveEditedPost validiert die Formulareingaben und speichert den Beitrag in der SQLite-Datenbank
func (m *Model) saveEditedPost() error {
	targets := m.targetPlatforms()
	for i, platform := range targets {
		if err := m.saveEditedPostFor(platform, time.Duration(i)*staggerMinutes*time.Minute, len(targets) > 1); err != nil {
			return err
		}
	}
	return nil
}

// saveEditedPostFor saves the form as a post for one platform; with several targets the text is
// adapted to the platform and the planned time is staggered.
func (m *Model) saveEditedPostFor(platform string, stagger time.Duration, multi bool) error {
	ctx := context.Background()
	campaign := strings.TrimSpace(m.editorCampaign.Value())
	if campaign == "" {
		campaign = "default"
	}

	schedStr := strings.TrimSpace(strings.ToLower(m.editorScheduledAt.Value()))
	var scheduledAt *time.Time
	status := "draft"
	if schedStr != "" {
		if schedStr == "now" || schedStr == "jetzt" {
			t := time.Now().Add(stagger)
			scheduledAt = &t
			status = "scheduled"
		} else {
			t, err := time.ParseInLocation("02.01.2006 15:04", m.editorScheduledAt.Value(), time.Local)
			if err != nil {
				return fmt.Errorf("ungültiges Datumformat. Bitte verwende 'DD.MM.YYYY HH:MM' oder 'now' / 'jetzt'")
			}
			t = t.Add(stagger)
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

	var alts []string
	for _, a := range strings.Split(m.editorAlts.Value(), "|") {
		alts = append(alts, strings.TrimSpace(a))
	}
	if strings.TrimSpace(strings.Join(alts, "")) == "" {
		alts = nil
	}

	body := m.editorBody.Value()
	if multi {
		body = adaptCaption(platform, body)
	}

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
		Alts:        alts,
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
		if m.editorType == "story" { // editing must not turn a story into a feed post
			post.Type = "story"
		}
		post.Body = body
	}

	post.PrepareTweets()

	// In SQLite speichern
	if err := m.store.SavePost(ctx, &post); err != nil {
		return err
	}

	return nil
}

// renderEditorForm zeichnet die Editor-Maske im Terminal
func (m Model) renderEditorForm(w, h int) string {
	var builder strings.Builder
	var fieldAt [focusCount]int // line where each focus target starts, to keep the focused one in view
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
	mark(focusPlatform)
	var platSelect []string
	for _, p := range editorPlatforms {
		name := strings.ToUpper(p)
		if m.editorTargets[p] {
			name = "✓" + name
		}
		if p == m.editorPlatform {
			platSelect = append(platSelect, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("["+name+"]"))
		} else {
			platSelect = append(platSelect, name)
		}
	}
	builder.WriteString(label(focusPlatform, Tr("editor_label_platform")) + "\n")
	for _, l := range wrapLines(strings.Join(platSelect, "  "), iw-4) {
		builder.WriteString("    " + l + "\n")
	}
	if m.editorFocus == focusPlatform && m.editorPostID == "" {
		builder.WriteString("    " + dimStyle.Render(Tr("editor_multi_hint")) + "\n")
	}
	builder.WriteString("\n")

	// 2. Kampagne
	mark(focusCampaign)
	builder.WriteString(label(focusCampaign, Tr("editor_label_campaign")) + m.editorCampaign.View() + "\n\n")

	// 3. Geplantes Datum
	mark(focusSchedule)
	builder.WriteString(label(focusSchedule, Tr("editor_label_schedule")) + m.editorScheduledAt.View() + "\n")
	if m.editorFocus == focusSchedule && !m.showDatePicker {
		builder.WriteString(dimStyle.Render("     "+Tr("editor_tip_now")) + "\n\n")
	} else {
		builder.WriteString("\n")
	}

	// 4. Bilder
	mark(focusImages)
	builder.WriteString(label(focusImages, Tr("editor_label_images")) + m.editorImages.View() + "\n")
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
	if m.editorFocus == focusImages {
		builder.WriteString(dimStyle.Render(Tr("editor_images_help")) + "\n")
	}
	builder.WriteString("\n")

	// 4b. Alt-Texte
	mark(focusAlts)
	builder.WriteString(label(focusAlts, Tr("editor_label_alts")) + m.editorAlts.View() + "\n")
	if n := missingAlts(m.editorImages.Value(), m.editorAlts.Value()); n > 0 {
		builder.WriteString("     " + amberStyle().Render(fmt.Sprintf(Tr("editor_alt_missing"), n)) + "\n")
	}
	builder.WriteString("\n")

	// 5. Text-Inhalt (width follows the terminal; local copy, the model keeps its own)
	mark(focusBody)
	bodyLabel := Tr("editor_label_body")
	if m.editorPlatform == "twitter" || m.editorPlatform == "mastodon" || m.editorPlatform == "bluesky" {
		bodyLabel += Tr("editor_twitter_thread_note")
	}
	body := m.editorBody
	body.SetWidth(min(max(iw-4, 20), 100))
	builder.WriteString(label(focusBody, bodyLabel) + "\n" + body.View() + "\n")
	if charLimitMsg, _ := m.checkCharacterLimits(); charLimitMsg != "" {
		builder.WriteString("     " + charLimitMsg + "\n\n")
	} else {
		builder.WriteString("\n")
	}

	// 6. Action-Buttons
	mark(focusSave)
	fieldAt[focusCancel] = fieldAt[focusSave]
	saveLabel, cancelLabel := Tr("editor_save"), Tr("editor_cancel")
	if m.editorFocus == focusSave {
		saveLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBgFg).Background(ColorPosted).Render(saveLabel)
	} else {
		saveLabel = lipgloss.NewStyle().Foreground(ColorPosted).Render(saveLabel)
	}
	if m.editorFocus == focusCancel {
		cancelLabel = lipgloss.NewStyle().Bold(true).Foreground(ColorBgFg).Background(ColorFailed).Render(cancelLabel)
	} else {
		cancelLabel = lipgloss.NewStyle().Foreground(ColorFailed).Render(cancelLabel)
	}
	builder.WriteString("  " + saveLabel + "     " + cancelLabel)

	// scroll so the focused field (and, for the body, its text area) is visible
	room, span := max(h-2, 1), 3
	switch {
	case m.editorFocus == focusBody:
		span = body.Height() + 2
	case m.editorFocus >= focusSave:
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

// missingAlts counts images without an alt text.
func missingAlts(images, alts string) int {
	n := 0
	for _, i := range strings.Split(images, ",") {
		if strings.TrimSpace(i) != "" {
			n++
		}
	}
	have := 0
	for _, a := range strings.Split(alts, "|") {
		if strings.TrimSpace(a) != "" {
			have++
		}
	}
	return max(n-have, 0)
}
