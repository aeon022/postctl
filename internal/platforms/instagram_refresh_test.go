package platforms

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
)

func TestRefreshInstagramTokenParsesNewToken(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"access_token":"NEW","token_type":"bearer","expires_in":5184000}`))
	})
	tok, err := refreshInstagramToken(context.Background(), srv.Client(), srv.URL+"/refresh_access_token", "OLD")
	if err != nil || tok != "NEW" {
		t.Fatalf("tok=%q err=%v", tok, err)
	}
	if r := (*reqs)[0]; r.Path != "/refresh_access_token" {
		t.Errorf("path = %s", r.Path)
	}
	bad, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(400) })
	if _, err := refreshInstagramToken(context.Background(), bad.Client(), bad.URL, "OLD"); err == nil {
		t.Error("an error response must surface")
	}
}

func TestRefreshInstagramIfDueOnlyWhenOldAndInstagramLogin(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	prev := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = prev })
	ig := &config.ActiveConfig.Instagram

	ig.API, ig.AccessToken, ig.TokenIssued = "instagram", "TOK", time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	if ok, err := RefreshInstagramIfDue(context.Background()); ok || err != nil {
		t.Errorf("a 5-day-old token must not be refreshed (ok=%v err=%v)", ok, err)
	}
	ig.API = "facebook"
	ig.TokenIssued = time.Now().AddDate(0, 0, -50).Format("2006-01-02")
	if ok, _ := RefreshInstagramIfDue(context.Background()); ok {
		t.Error("only Instagram-Login tokens are refreshed this way")
	}
	ig.API, ig.TokenIssued = "instagram", ""
	if ok, _ := RefreshInstagramIfDue(context.Background()); ok {
		t.Error("an unknown issue date is left alone")
	}
}
