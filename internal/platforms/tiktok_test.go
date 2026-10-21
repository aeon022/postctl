package platforms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

// tiktokServer fakes the TikTok API: init returns an upload_url on the same server.
func tiktokServer(t *testing.T, initResp string) (*httptest.Server, *[]string, *[]byte) {
	t.Helper()
	var log []string
	var uploaded []byte
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		log = append(log, r.Method+" "+r.URL.Path+" "+r.Header.Get("Authorization")+" "+r.Header.Get("Content-Range"))
		switch r.URL.Path {
		case "/v2/post/publish/inbox/video/init/":
			if !strings.Contains(string(raw), `"video_size":5`) || !strings.Contains(string(raw), `"FILE_UPLOAD"`) {
				t.Errorf("init body = %s", raw)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, strings.ReplaceAll(initResp, "UPLOAD", srv.URL+"/upload"))
		case "/upload":
			uploaded = raw
			w.WriteHeader(http.StatusCreated)
		case "/v2/oauth/token/":
			form, _ := url.ParseQuery(string(raw))
			if form.Get("grant_type") != "refresh_token" || form.Get("refresh_token") != "r1" {
				t.Errorf("refresh form = %v", form)
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"access_token":"new","refresh_token":"r2","expires_in":86400}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &log, &uploaded
}

func newTikTok(t *testing.T, srvURL string) *TikTokPlatform {
	t.Helper()
	p := NewTikTokPlatform(newTestStore(t), "key", "secret")
	p.apiBase = srvURL
	return p
}

func TestTikTokPostUploadsVideoToInbox(t *testing.T) {
	srv, log, uploaded := tiktokServer(t, `{"data":{"publish_id":"v_pub_1","upload_url":"UPLOAD"},"error":{"code":"ok","message":""}}`)
	p := newTikTok(t, srv.URL)
	exp := time.Now().Add(time.Hour)
	if err := p.store.SaveToken(context.Background(), models.PlatformTikTok, "tok", "r1", &exp); err != nil {
		t.Fatal(err)
	}
	video := filepath.Join(t.TempDir(), "clip.mp4")
	os.WriteFile(video, []byte("12345"), 0o600)

	id, err := p.Post(context.Background(), &models.Post{Images: []string{video}})
	if err != nil || id != "v_pub_1" {
		t.Fatalf("Post = %q, %v", id, err)
	}
	if string(*uploaded) != "12345" {
		t.Errorf("uploaded = %q", *uploaded)
	}
	want := []string{"POST /v2/post/publish/inbox/video/init/ Bearer tok ", "PUT /upload  bytes 0-4/5"}
	for i, w := range want {
		if (*log)[i] != w {
			t.Errorf("request %d = %q, want %q", i, (*log)[i], w)
		}
	}
}

func TestTikTokPostRejectsNonVideoAndAPIErrors(t *testing.T) {
	srv, _, _ := tiktokServer(t, `{"error":{"code":"spam_risk_too_many_pending_share","message":"limit"}}`)
	p := newTikTok(t, srv.URL)
	saveToken(t, p.store, models.PlatformTikTok, "tok", "")

	if _, err := p.Post(context.Background(), &models.Post{Images: []string{"https://x/a.png"}}); err == nil || !strings.Contains(err.Error(), "Video") {
		t.Errorf("image post: err = %v", err)
	}
	video := filepath.Join(t.TempDir(), "clip.mp4")
	os.WriteFile(video, []byte("12345"), 0o600)
	if _, err := p.Post(context.Background(), &models.Post{Images: []string{video}}); err == nil || !strings.Contains(err.Error(), "spam_risk_too_many_pending_share") {
		t.Errorf("api error: err = %v", err)
	}
}

func TestTikTokRefreshesExpiringToken(t *testing.T) {
	srv, _, _ := tiktokServer(t, "")
	p := newTikTok(t, srv.URL)
	exp := time.Now().Add(time.Minute) // < 5 min left
	if err := p.store.SaveToken(context.Background(), models.PlatformTikTok, "old", "r1", &exp); err != nil {
		t.Fatal(err)
	}
	tok, err := p.getValidToken(context.Background())
	if err != nil || tok != "new" {
		t.Fatalf("getValidToken = %q, %v", tok, err)
	}
	if _, refresh, _, _ := p.store.GetToken(context.Background(), models.PlatformTikTok); refresh != "r2" {
		t.Errorf("refresh token not rotated: %q", refresh)
	}
}

func TestTikTokDeleteAndImageUploadUnsupported(t *testing.T) {
	p := newTikTok(t, "http://unused")
	if err := p.Delete(context.Background(), "v1"); err == nil {
		t.Error("Delete must explain that drafts can't be deleted via API")
	}
	if _, err := p.UploadImage(context.Background(), "a.png"); err == nil {
		t.Error("UploadImage must be unsupported")
	}
}
