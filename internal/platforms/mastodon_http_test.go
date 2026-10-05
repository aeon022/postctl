package platforms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

type mReq struct {
	Method, Path, Auth string
	Form               url.Values
	Raw                []byte
	CT                 string
}

func mastoServer(t *testing.T, h func(r mReq, n int, w http.ResponseWriter)) (*httptest.Server, *[]mReq) {
	t.Helper()
	var reqs []mReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		q := mReq{r.Method, r.URL.Path, r.Header.Get("Authorization"), nil, raw, r.Header.Get("Content-Type")}
		if strings.HasPrefix(q.CT, "application/x-www-form-urlencoded") {
			q.Form, _ = url.ParseQuery(string(raw))
		}
		reqs = append(reqs, q)
		w.Header().Set("Content-Type", "application/json")
		h(q, len(reqs), w)
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func newMasto(t *testing.T, srvURL string, withToken bool) *MastodonPlatform {
	t.Helper()
	m := NewMastodonPlatform(newTestStore(t), srvURL+"/", "cid", "csecret")
	if withToken {
		saveToken(t, m.store, models.PlatformMastodon, "tok", "")
	}
	return m
}

func TestMastodonInstanceURLNormalised(t *testing.T) {
	if m := NewMastodonPlatform(nil, "", "", ""); m.instanceURL != "https://mastodon.social" {
		t.Errorf("default instance = %q", m.instanceURL)
	}
	if m := NewMastodonPlatform(nil, "https://x.example/", "", ""); m.instanceURL != "https://x.example" {
		t.Errorf("trailing slash must be trimmed, got %q", m.instanceURL)
	}
}

func TestMastodonExchangeCodeStoresToken(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"access_token":"fresh"}`))
	})
	m := newMasto(t, srv.URL, false)
	if m.IsAuthenticated(context.Background()) {
		t.Fatal("authenticated before login")
	}
	if err := m.exchangeCodeForToken(context.Background(), "the-code", "http://127.0.0.1:8753/callback"); err != nil {
		t.Fatal(err)
	}
	f := (*reqs)[0].Form
	if (*reqs)[0].Path != "/oauth/token" || f.Get("grant_type") != "authorization_code" || f.Get("code") != "the-code" || f.Get("client_id") != "cid" || f.Get("client_secret") != "csecret" {
		t.Errorf("token request = %v", f)
	}
	if tok, _, _, _ := m.store.GetToken(context.Background(), models.PlatformMastodon); tok != "fresh" || !m.IsAuthenticated(context.Background()) {
		t.Errorf("token = %q", tok)
	}

	bad, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})
	if err := newMasto(t, bad.URL, false).exchangeCodeForToken(context.Background(), "x", "y"); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("want status in error, got %v", err)
	}
}

func TestMastodonRequiresToken(t *testing.T) {
	m := newMasto(t, "http://127.0.0.1:1", false)
	ctx := context.Background()
	if _, err := m.Post(ctx, &models.Post{Body: "x"}); err == nil {
		t.Error("Post without token")
	}
	if _, err := m.UploadImage(ctx, "x.png"); err == nil {
		t.Error("UploadImage without token")
	}
	if _, err := m.FetchAnalytics(ctx, "1"); err == nil {
		t.Error("FetchAnalytics without token")
	}
	if err := m.Delete(ctx, "1"); err == nil {
		t.Error("Delete without token")
	}
}

func TestMastodonPostSingleThreadAndMedia(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Path == "/api/v1/media" {
			_, _ = w.Write([]byte(`{"id":"media-9"}`))
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"s%d"}`, n)))
	})
	m := newMasto(t, srv.URL, true)
	ctx := context.Background()

	id, err := m.Post(ctx, &models.Post{Type: "single", Body: "toot"})
	if err != nil || id != "s1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	f := (*reqs)[0].Form
	if f.Get("status") != "toot" || f.Get("visibility") != "public" || (*reqs)[0].Auth != "Bearer tok" || f.Has("in_reply_to_id") {
		t.Errorf("single status form = %v auth=%q", f, (*reqs)[0].Auth)
	}

	*reqs = nil
	id, err = m.Post(ctx, &models.Post{Type: "thread", Tweets: []models.Tweet{{Content: "a"}, {Content: "b"}, {Content: "c"}}})
	if err != nil || id != "s1" {
		t.Fatalf("thread must return the first status id, got %q err=%v", id, err)
	}
	if (*reqs)[0].Form.Has("in_reply_to_id") || (*reqs)[1].Form.Get("in_reply_to_id") != "s1" || (*reqs)[2].Form.Get("in_reply_to_id") != "s2" {
		t.Errorf("thread chaining wrong: %v / %v / %v", (*reqs)[0].Form, (*reqs)[1].Form, (*reqs)[2].Form)
	}

	// single post with an image: media first, then status referencing it
	img := filepath.Join(t.TempDir(), "p.png")
	_ = os.WriteFile(img, []byte("PNG"), 0o600)
	*reqs = nil
	if _, err := m.Post(ctx, &models.Post{Type: "single", Body: "pic", Images: []string{img}}); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Path != "/api/v1/media" || !strings.HasPrefix((*reqs)[0].CT, "multipart/form-data") || !strings.Contains(string((*reqs)[0].Raw), `name="file"; filename="p.png"`) {
		t.Errorf("media upload request wrong: path=%s ct=%s", (*reqs)[0].Path, (*reqs)[0].CT)
	}
	if got := (*reqs)[1].Form["media_ids[]"]; len(got) != 1 || got[0] != "media-9" {
		t.Errorf("status media_ids[] = %v", got)
	}
}

func TestMastodonPostErrors(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"error":"too long"}`))
	})
	m := newMasto(t, srv.URL, true)
	if _, err := m.Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || !strings.Contains(err.Error(), "422") {
		t.Errorf("want status 422, got %v", err)
	}
	if _, err := m.Post(context.Background(), &models.Post{Type: "thread"}); err == nil || !strings.Contains(err.Error(), "no content") {
		t.Errorf("empty thread: %v", err)
	}
	if _, err := m.UploadImage(context.Background(), filepath.Join(t.TempDir(), "nope.png")); err == nil {
		t.Error("missing image file must error")
	}
}

func TestMastodonAnalyticsAndDelete(t *testing.T) {
	status := 200
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"favourites_count":4,"reblogs_count":2,"replies_count":1}`))
	})
	m := newMasto(t, srv.URL, true)
	ctx := context.Background()

	a, err := m.FetchAnalytics(ctx, "abc")
	if err != nil || a.Likes != 4 || a.Shares != 2 || a.Comments != 1 || a.Impressions != 4*10+2*50+20 || a.PlatformID != "abc" {
		t.Errorf("analytics = %+v err=%v", a, err)
	}
	if (*reqs)[0].Method != "GET" || (*reqs)[0].Path != "/api/v1/statuses/abc" {
		t.Errorf("analytics request = %s %s", (*reqs)[0].Method, (*reqs)[0].Path)
	}
	if err := m.Delete(ctx, "abc"); err != nil {
		t.Fatal(err)
	}
	if r := (*reqs)[1]; r.Method != "DELETE" || r.Path != "/api/v1/statuses/abc" {
		t.Errorf("delete request = %s %s", r.Method, r.Path)
	}
	status = 404
	if err := m.Delete(ctx, "abc"); err == nil {
		t.Error("failed delete must error")
	}
	if _, err := m.FetchAnalytics(ctx, "abc"); err == nil {
		t.Error("analytics on non-200 must error")
	}
}
