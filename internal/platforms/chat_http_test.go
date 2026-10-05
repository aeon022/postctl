package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func writeImgs(t *testing.T, names ...string) []string {
	t.Helper()
	var out []string
	for _, n := range names {
		p := filepath.Join(t.TempDir(), n)
		if err := os.WriteFile(p, []byte("IMG-"+n), 0o600); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func newTG(t *testing.T, h func(r mReq, n int, w http.ResponseWriter)) (*TelegramPlatform, *[]mReq) {
	t.Helper()
	srv, reqs := mastoServer(t, h)
	p := NewTelegramPlatform(nil, "TOK", "chat7")
	p.apiURL = srv.URL
	return p, reqs
}

func TestTelegramPostRoutesByImageCount(t *testing.T) {
	p, reqs := newTG(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":55}}`))
	})
	ctx := context.Background()

	id, err := p.Post(ctx, &models.Post{Body: "*fett* text"})
	if err != nil || id != "55" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	var msg map[string]any
	_ = json.Unmarshal((*reqs)[0].Raw, &msg)
	if (*reqs)[0].Path != "/botTOK/sendMessage" || msg["chat_id"] != "chat7" || msg["text"] != "*fett* text" || msg["parse_mode"] != "Markdown" {
		t.Errorf("text message = %s %v", (*reqs)[0].Path, msg)
	}

	one := writeImgs(t, "a.jpg")
	if _, err := p.Post(ctx, &models.Post{Body: "cap", Images: one}); err != nil {
		t.Fatal(err)
	}
	raw := string((*reqs)[1].Raw)
	if (*reqs)[1].Path != "/botTOK/sendPhoto" || !strings.Contains(raw, `name="photo"; filename="a.jpg"`) || !strings.Contains(raw, "cap") || !strings.Contains(raw, "chat7") {
		t.Errorf("photo message = %s %q", (*reqs)[1].Path, raw)
	}

	two := writeImgs(t, "a.jpg", "b.jpg")
	if _, err := p.Post(ctx, &models.Post{Body: "album", Images: two}); err != nil {
		t.Fatal(err)
	}
	raw = string((*reqs)[2].Raw)
	if (*reqs)[2].Path != "/botTOK/sendMediaGroup" || !strings.Contains(raw, `name="photo_0"; filename="a.jpg"`) || !strings.Contains(raw, `name="photo_1"; filename="b.jpg"`) {
		t.Errorf("media group parts missing: %s", (*reqs)[2].Path)
	}
	i := strings.Index(raw, `name="media"`)
	if i < 0 {
		t.Fatal("no media field")
	}
	rest := raw[i:]
	rest = rest[strings.Index(rest, "[") : strings.LastIndex(rest, "]")+1]
	var media []map[string]string
	if err := json.Unmarshal([]byte(rest), &media); err != nil || len(media) != 2 {
		t.Fatalf("media json %q: %v", rest, err)
	}
	if media[0]["media"] != "attach://photo_0" || media[0]["caption"] != "album" || media[1]["media"] != "attach://photo_1" || media[1]["caption"] != "" {
		t.Errorf("only the first album item may carry the caption: %v", media)
	}
}

func TestTelegramErrorsDeleteAndMisc(t *testing.T) {
	status := 400
	p, reqs := newTG(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"ok":false}`))
	})
	ctx := context.Background()
	if _, err := p.Post(ctx, &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("want status 400, got %v", err)
	}
	if _, err := p.Post(ctx, &models.Post{Body: "x", Images: []string{filepath.Join(t.TempDir(), "no.jpg")}}); err == nil {
		t.Error("missing photo must error")
	}
	if err := p.Delete(ctx, "not-a-number"); err == nil {
		t.Error("non-numeric message id must be rejected before any request")
	}
	if len(*reqs) != 1 {
		t.Errorf("invalid id / missing file must not hit the API (requests=%d)", len(*reqs))
	}
	if err := p.Delete(ctx, "55"); err == nil {
		t.Error("API failure on delete must surface")
	}
	status = 200
	if err := p.Delete(ctx, "55"); err != nil {
		t.Fatal(err)
	}
	var del map[string]any
	_ = json.Unmarshal((*reqs)[len(*reqs)-1].Raw, &del)
	if del["message_id"] != float64(55) || del["chat_id"] != "chat7" {
		t.Errorf("delete payload = %v", del)
	}
	if !p.IsAuthenticated(ctx) || NewTelegramPlatform(nil, "TOK", "").IsAuthenticated(ctx) {
		t.Error("IsAuthenticated needs token AND chat id")
	}
	if got, _ := p.UploadImage(ctx, "/x.png"); got != "/x.png" || p.Name() != "telegram" {
		t.Error("UploadImage/Name")
	}
}

func newDC(t *testing.T, h func(r mReq, n int, w http.ResponseWriter)) (*DiscordPlatform, *[]mReq) {
	t.Helper()
	srv, reqs := mastoServer(t, h)
	return NewDiscordPlatform(nil, srv.URL+"/webhooks/1/tok"), reqs
}

func TestDiscordPostTextAndFiles(t *testing.T) {
	p, reqs := newDC(t, func(r mReq, n int, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"id":"msg-1"}`)) })
	ctx := context.Background()

	id, err := p.Post(ctx, &models.Post{Body: "hallo discord"})
	if err != nil || id != "msg-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	var body map[string]string
	_ = json.Unmarshal((*reqs)[0].Raw, &body)
	if (*reqs)[0].Path != "/webhooks/1/tok" || body["content"] != "hallo discord" {
		t.Errorf("text post = %s %v", (*reqs)[0].Path, body)
	}

	imgs := writeImgs(t, "a.png", "b.png")
	if _, err := p.Post(ctx, &models.Post{Body: "mit bildern", Images: imgs}); err != nil {
		t.Fatal(err)
	}
	raw := string((*reqs)[1].Raw)
	if !strings.Contains(raw, `name="payload_json"`) || !strings.Contains(raw, "mit bildern") || !strings.Contains(raw, `name="files[0]"; filename="a.png"`) || !strings.Contains(raw, `name="files[1]"; filename="b.png"`) {
		t.Errorf("multipart body malformed: %q", raw)
	}
	if _, err := p.Post(ctx, &models.Post{Body: "x", Images: []string{filepath.Join(t.TempDir(), "no.png")}}); err == nil {
		t.Error("missing image must error")
	}
}

func TestDiscordAuthStatusAndErrors(t *testing.T) {
	status := 200
	p, _ := newDC(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(status); _, _ = w.Write([]byte(`{}`)) })
	ctx := context.Background()
	if err := p.Auth(ctx); err != nil {
		t.Errorf("valid webhook: %v", err)
	}
	status = 404
	if err := p.Auth(ctx); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("unknown webhook: %v", err)
	}
	if _, err := p.Post(ctx, &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("post error must carry status: %v", err)
	}
	if err := NewDiscordPlatform(nil, "").Auth(ctx); err == nil || !strings.Contains(err.Error(), "discord.webhook_url") {
		t.Errorf("missing webhook should explain configuration: %v", err)
	}
	if p.Name() != "discord" || !p.IsAuthenticated(ctx) || NewDiscordPlatform(nil, "").IsAuthenticated(ctx) {
		t.Error("Name/IsAuthenticated")
	}
	if a, err := p.FetchAnalytics(ctx, "x"); err != nil || a.Likes != 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
}

// A webhook POST without ?wait=true answers 204 and no message id, so the post
// could never be deleted remotely later. Posting must ask for the message.
func TestDiscordPostAsksForMessageID(t *testing.T) {
	var query string
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"id":"m9"}`))
	})
	p := NewDiscordPlatform(nil, srv.URL+"/webhooks/1/tok")
	if id, err := p.Post(context.Background(), &models.Post{Body: "x"}); err != nil || id != "m9" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if !strings.Contains(query, "wait=true") {
		t.Errorf("webhook must be called with wait=true to get the message id back, query=%q", query)
	}
}

func TestDiscordDelete(t *testing.T) {
	status := 204
	p, reqs := newDC(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(status) })
	ctx := context.Background()
	if err := p.Delete(ctx, "msg-1"); err != nil {
		t.Fatal(err)
	}
	if r := (*reqs)[0]; r.Method != "DELETE" || r.Path != "/webhooks/1/tok/messages/msg-1" {
		t.Errorf("delete request = %s %s", r.Method, r.Path)
	}
	status = 404
	if err := p.Delete(ctx, "msg-1"); err == nil {
		t.Error("failed delete must surface")
	}
	// Legacy/unknown id: nothing was deleted, so claiming success would let
	// postctl drop the local record of a message that is still live.
	if err := p.Delete(ctx, "webhook-posted"); err == nil {
		t.Error("Delete of a post without a message id must not report success")
	}
}
