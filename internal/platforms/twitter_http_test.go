package platforms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func newTW(t *testing.T, srvURL string) *TwitterPlatform {
	t.Helper()
	p := NewTwitterPlatform(newTestStore(t), "cid", "csecret")
	p.apiURL, p.uploadURL = srvURL, srvURL
	return p
}

func saveTWToken(t *testing.T, p *TwitterPlatform, access, refresh string, exp time.Duration) {
	t.Helper()
	at := time.Now().Add(exp)
	if err := p.store.SaveToken(context.Background(), models.PlatformTwitter, access, refresh, &at); err != nil {
		t.Fatal(err)
	}
}

func TestTwitterExchangeCodeUsesBasicAuthAndPKCE(t *testing.T) {
	var auth string
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		auth = r.Auth
		_, _ = w.Write([]byte(`{"access_token":"AT","refresh_token":"RT","expires_in":7200}`))
	})
	p := newTW(t, srv.URL)
	if err := p.exchangeCodeForToken(context.Background(), "CODE", "VERIFIER", "http://localhost:8753/callback"); err != nil {
		t.Fatal(err)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte("cid:csecret")); auth != want {
		t.Errorf("auth = %q, want %q", auth, want)
	}
	if f := (*reqs)[0].Form; (*reqs)[0].Path != "/2/oauth2/token" || f.Get("code_verifier") != "VERIFIER" || f.Get("grant_type") != "authorization_code" || f.Get("code") != "CODE" {
		t.Errorf("token form = %v", f)
	}
	tok, refresh, exp, err := p.store.GetToken(context.Background(), models.PlatformTwitter)
	if err != nil || tok != "AT" || refresh != "RT" || exp == nil || time.Until(*exp) < time.Hour {
		t.Errorf("stored = %q/%q/%v err=%v", tok, refresh, exp, err)
	}
	if !p.IsAuthenticated(context.Background()) || p.Name() != "twitter" {
		t.Error("IsAuthenticated/Name")
	}

	bad, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(401); _, _ = w.Write([]byte(`no`)) })
	if err := newTW(t, bad.URL).exchangeCodeForToken(context.Background(), "c", "v", "r"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("want status 401, got %v", err)
	}
}

func TestTwitterGetValidTokenRefreshesOnlyNearExpiry(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"access_token":"NEW","refresh_token":"RT2","expires_in":7200}`))
	})
	p := newTW(t, srv.URL)
	ctx := context.Background()

	if _, err := p.getValidToken(ctx); err == nil {
		t.Error("no stored token must error")
	}

	saveTWToken(t, p, "FRESH", "RT", time.Hour)
	if tok, err := p.getValidToken(ctx); err != nil || tok != "FRESH" || len(*reqs) != 0 {
		t.Errorf("valid token must be used as-is: %q err=%v requests=%d", tok, err, len(*reqs))
	}

	// expires in 30s: inside the 1-minute safety window → refresh
	saveTWToken(t, p, "OLD", "RT", 30*time.Second)
	tok, err := p.getValidToken(ctx)
	if err != nil || tok != "NEW" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
	if f := (*reqs)[0].Form; f.Get("grant_type") != "refresh_token" || f.Get("refresh_token") != "RT" {
		t.Errorf("refresh form = %v", f)
	}
	if stored, refresh, _, _ := p.store.GetToken(ctx, models.PlatformTwitter); stored != "NEW" || refresh != "RT2" {
		t.Errorf("refreshed pair must be persisted (refresh tokens rotate), got %q/%q", stored, refresh)
	}

	saveTWToken(t, p, "OLD", "", -time.Hour)
	if _, err := p.getValidToken(ctx); err == nil || !strings.Contains(err.Error(), "no refresh token") {
		t.Errorf("expired without refresh token: %v", err)
	}
}

func TestTwitterPostSingleThreadAndMedia(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Path == "/1.1/media/upload.json" {
			_, _ = w.Write([]byte(`{"media_id_string":"M1"}`))
			return
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"data":{"id":"t%d"}}`, n)))
	})
	p := newTW(t, srv.URL)
	saveTWToken(t, p, "AT", "RT", time.Hour)
	ctx := context.Background()

	id, err := p.Post(ctx, &models.Post{Type: "single", Body: "hallo"})
	if err != nil || id != "t1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	var tw struct {
		Text  string
		Reply map[string]string
		Media struct{ Media_ids []string }
	}
	_ = json.Unmarshal((*reqs)[0].Raw, &tw)
	if (*reqs)[0].Path != "/2/tweets" || (*reqs)[0].Auth != "Bearer AT" || tw.Text != "hallo" || tw.Reply != nil {
		t.Errorf("tweet request = %s auth=%q body=%s", (*reqs)[0].Path, (*reqs)[0].Auth, (*reqs)[0].Raw)
	}

	*reqs = nil
	id, err = p.Post(ctx, &models.Post{Type: "thread", Tweets: []models.Tweet{{Index: 1, Content: "a"}, {Index: 2, Content: "b"}, {Index: 3, Content: "c"}}})
	if err != nil || id != "t1" {
		t.Fatalf("thread must return first tweet id, got %q err=%v", id, err)
	}
	for i, want := range []string{"", "t1", "t2"} {
		var b struct{ Reply map[string]string }
		_ = json.Unmarshal((*reqs)[i].Raw, &b)
		if b.Reply["in_reply_to_tweet_id"] != want {
			t.Errorf("tweet %d replies to %q, want %q", i+1, b.Reply["in_reply_to_tweet_id"], want)
		}
	}

	img := filepath.Join(t.TempDir(), "p.png")
	_ = os.WriteFile(img, []byte("PNG"), 0o600)
	*reqs = nil
	if _, err := p.Post(ctx, &models.Post{Type: "single", Body: "pic", Images: []string{img}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string((*reqs)[0].Raw), `name="media"; filename="p.png"`) || (*reqs)[0].Auth != "Bearer AT" {
		t.Errorf("media upload malformed: %q", (*reqs)[0].Raw)
	}
	if !strings.Contains(string((*reqs)[1].Raw), `"media_ids":["M1"]`) {
		t.Errorf("tweet must reference uploaded media: %s", (*reqs)[1].Raw)
	}
}

func TestTwitterPostErrors(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"title":"Forbidden"}`))
	})
	p := newTW(t, srv.URL)
	saveTWToken(t, p, "AT", "RT", time.Hour)
	if _, err := p.Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || !strings.Contains(err.Error(), "403") {
		t.Errorf("want status 403, got %v", err)
	}
	if _, err := p.Post(context.Background(), &models.Post{Type: "thread"}); err == nil || !strings.Contains(err.Error(), "no tweets") {
		t.Errorf("empty thread: %v", err)
	}
	if _, err := p.UploadImage(context.Background(), filepath.Join(t.TempDir(), "nope.png")); err == nil {
		t.Error("missing image")
	}
	if _, err := newTW(t, srv.URL).Post(context.Background(), &models.Post{Body: "x"}); err == nil {
		t.Error("post without token")
	}
}

// Regression: Delete used to return nil without doing anything, so
// `postctl delete` removed the local record of a tweet that was still live.
func TestTwitterDeleteIsHonest(t *testing.T) {
	err := NewTwitterPlatform(nil, "", "").Delete(context.Background(), "12345")
	if err == nil || !strings.Contains(err.Error(), "12345") {
		t.Errorf("Delete must not pretend to succeed, got %v", err)
	}
}

func TestParseCookies(t *testing.T) {
	got := parseCookies(" auth_token=abc; ct0=def=g ;; junk ; k = v ")
	want := map[string]string{"auth_token": "abc", "ct0": "def=g", "k": "v"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("cookie %s = %q, want %q (values may contain '=')", k, got[k], v)
		}
	}
}
