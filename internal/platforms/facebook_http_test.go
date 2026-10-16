package platforms

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

func newFB(t *testing.T, srvURL string, withToken bool) *FacebookPlatform {
	t.Helper()
	f := NewFacebookPlatform(newTestStore(t), "app1", "secret1", "page42")
	f.baseURL = srvURL
	if withToken {
		saveToken(t, f.store, models.PlatformFacebook, "PAGETOK", "")
	}
	return f
}

// The 3-step login: code -> user token -> long-lived user token -> page token
// picked by pageID. Each step must hand its token to the next.
func TestFacebookExchangeCodeForPageToken(t *testing.T) {
	var queries []string
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Path+"?"+r.URL.RawQuery)
		switch {
		case r.URL.Path == "/oauth/access_token" && r.URL.Query().Get("code") == "C0DE":
			_, _ = w.Write([]byte(`{"access_token":"USERTOK"}`))
		case r.URL.Path == "/oauth/access_token" && r.URL.Query().Get("fb_exchange_token") == "USERTOK":
			_, _ = w.Write([]byte(`{"access_token":"LLTOK"}`))
		case r.URL.Path == "/me/accounts" && r.URL.Query().Get("access_token") == "LLTOK":
			_, _ = w.Write([]byte(`{"data":[{"id":"other","access_token":"NO","name":"Other"},{"id":"page42","access_token":"PAGETOK","name":"Mine"}]}`))
		default:
			w.WriteHeader(400)
		}
	})
	f := newFB(t, srv.URL, false)
	if err := f.exchangeCodeForPageToken(context.Background(), "C0DE", "https://localhost:8753/callback"); err != nil {
		t.Fatalf("login chain: %v (requests: %v)", err, queries)
	}
	if tok, _, _, _ := f.store.GetToken(context.Background(), models.PlatformFacebook); tok != "PAGETOK" {
		t.Errorf("stored token = %q, want the page token of page42", tok)
	}
	if !f.IsAuthenticated(context.Background()) || f.Name() != "facebook" {
		t.Error("IsAuthenticated/Name")
	}
}

func TestFacebookLoginWrongPageIDListsPages(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/me/accounts" {
			_, _ = w.Write([]byte(`{"data":[{"id":"111","access_token":"x","name":"Bakery"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"T"}`))
	})
	err := newFB(t, srv.URL, false).exchangeCodeForPageToken(context.Background(), "c", "r")
	if err == nil || !strings.Contains(err.Error(), "Bakery") || !strings.Contains(err.Error(), "111") {
		t.Errorf("error must list the pages the account does have, got %v", err)
	}

	fail, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(400); _, _ = w.Write([]byte(`bad`)) })
	if err := newFB(t, fail.URL, false).exchangeCodeForPageToken(context.Background(), "c", "r"); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("first step failure must surface status, got %v", err)
	}
}

func TestFacebookPostFeedAndThreadJoin(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"id":"page42_77"}`)) })
	f := newFB(t, srv.URL, true)
	id, err := f.Post(context.Background(), &models.Post{Body: "Hallo Welt"})
	if err != nil || id != "page42_77" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	r := (*reqs)[0]
	if r.Path != "/page42/feed" || r.Form.Get("message") != "Hallo Welt" || r.Form.Get("access_token") != "PAGETOK" {
		t.Errorf("feed request = %s %v", r.Path, r.Form)
	}

	// no body but a thread: tweets are joined into one post by blank lines
	if _, err := f.Post(context.Background(), &models.Post{Tweets: []models.Tweet{{Content: "eins"}, {Content: "zwei"}}}); err != nil {
		t.Fatal(err)
	}
	if got := (*reqs)[1].Form.Get("message"); got != "eins\n\nzwei" {
		t.Errorf("thread message = %q", got)
	}
}

func TestFacebookPostPhotoPrefersPostID(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"id":"photo1","post_id":"page42_88"}`))
	})
	f := newFB(t, srv.URL, true)
	img := filepath.Join(t.TempDir(), "x.jpg")
	_ = os.WriteFile(img, []byte("JPG"), 0o600)

	id, err := f.Post(context.Background(), &models.Post{Body: "mit Bild", Images: []string{img}})
	if err != nil || id != "page42_88" {
		t.Fatalf("id=%q err=%v (the post_id, not the photo id, identifies the feed post)", id, err)
	}
	r := (*reqs)[0]
	raw := string(r.Raw)
	if r.Path != "/page42/photos" || !strings.Contains(raw, `name="source"; filename="x.jpg"`) || !strings.Contains(raw, "mit Bild") || !strings.Contains(raw, "PAGETOK") {
		t.Errorf("photo upload malformed: path=%s body=%q", r.Path, raw)
	}
	if _, err := f.Post(context.Background(), &models.Post{Body: "x", Images: []string{filepath.Join(t.TempDir(), "missing.jpg")}}); err == nil {
		t.Error("missing image must error")
	}
}

func TestFacebookPostErrorsAndAnalytics(t *testing.T) {
	status := 400
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"likes":{"summary":{"total_count":5}},"comments":{"summary":{"total_count":2}},"shares":{"count":3}}`))
	})
	f := newFB(t, srv.URL, true)
	if _, err := f.Post(context.Background(), &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("want status 400, got %v", err)
	}
	if _, err := newFB(t, srv.URL, false).Post(context.Background(), &models.Post{Body: "x"}); err == nil {
		t.Error("post without token")
	}

	status = 200
	a, err := f.FetchAnalytics(context.Background(), "page42_88")
	if err != nil || a.Likes != 5 || a.Comments != 2 || a.Shares != 3 || a.Impressions != 5*15+3*60+2*25+50 {
		t.Errorf("analytics = %+v err=%v", a, err)
	}
	if r := (*reqs)[len(*reqs)-1]; r.Path != "/page42_88" {
		t.Errorf("analytics path = %s", r.Path)
	}
	status = 500
	if _, err := f.FetchAnalytics(context.Background(), "x"); err == nil {
		t.Error("non-200 analytics must error")
	}
}

func TestFacebookDeleteAndUploadAreHonest(t *testing.T) {
	f := NewFacebookPlatform(nil, "", "", "")
	if id, err := f.UploadImage(context.Background(), "x"); id != "" || err != nil {
		t.Errorf("UploadImage = %q, %v", id, err)
	}
}

func TestDeleteSendsDELETEOnThreePlatforms(t *testing.T) {
	var got []mReq
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(200) })
	_ = got
	fb := newFB(t, srv.URL, true)
	if err := fb.Delete(context.Background(), "123_456"); err != nil {
		t.Fatal(err)
	}
	li := newLI(t, srv.URL, true)
	if err := li.Delete(context.Background(), "urn:li:share:9"); err != nil {
		t.Fatal(err)
	}
	tw := newTW(t, srv.URL)
	saveTWToken(t, tw, "AT", "RT", time.Hour)
	if err := tw.Delete(context.Background(), "777"); err != nil {
		t.Fatal(err)
	}
	for i, wantPath := range []string{"/123_456", "/v2/ugcPosts/urn:li:share:9", "/2/tweets/777"} {
		if (*reqs)[i].Method != "DELETE" || (*reqs)[i].Path != wantPath {
			t.Errorf("req %d = %s %s, want DELETE %s", i, (*reqs)[i].Method, (*reqs)[i].Path, wantPath)
		}
	}
}

func TestDeleteSurfacesServerError(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(403) })
	if newFB(t, srv.URL, true).Delete(context.Background(), "1") == nil {
		t.Error("403 must be an error")
	}
}

func TestFacebookLoginAcceptsVanityNameAndStoresNumericID(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // SaveConfig must not touch the real config
	prev := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = prev })

	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			if r.URL.Query().Get("code") != "" {
				_, _ = w.Write([]byte(`{"access_token":"USERTOK"}`))
			} else {
				_, _ = w.Write([]byte(`{"access_token":"LLTOK"}`))
			}
		case "/me/accounts":
			_, _ = w.Write([]byte(`{"data":[{"id":"111","access_token":"NO","name":"Other","username":"other"},{"id":"222","access_token":"PAGETOK","name":"Die Brücke | Graz","username":"brueckegraz"}]}`))
		}
	})
	f := newFB(t, srv.URL, false)
	f.pageID = "BrueckeGraz" // as typed from facebook.com/brueckegraz
	if err := f.exchangeCodeForPageToken(context.Background(), "C", "r"); err != nil {
		t.Fatal(err)
	}
	if tok, _, _, _ := f.store.GetToken(context.Background(), models.PlatformFacebook); tok != "PAGETOK" {
		t.Errorf("token = %q", tok)
	}
	if f.pageID != "222" || config.ActiveConfig.Facebook.PageID != "222" {
		t.Errorf("numeric id not remembered: %q / %q", f.pageID, config.ActiveConfig.Facebook.PageID)
	}
}

func TestFacebookAuthWithStaticTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	prev := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = prev })

	// accounts: does /me/accounts list the page (a business-owned page is missing from it)?
	// manages: does the user token have a role on the page (page node returns access_token)?
	var f_appID string
	exchanged := false
	run := func(token string, accounts, manages bool) (*FacebookPlatform, error) {
		config.ActiveConfig.Facebook.PageToken = token
		srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {})
		srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := r.URL.Query().Get("access_token")
			switch {
			case r.URL.Path == "/222" || r.URL.Path == "/brueckegraz":
				if manages && tok == "USERTOK" {
					_, _ = w.Write([]byte(`{"id":"222","name":"Die Brücke","access_token":"PAGETOK"}`))
				} else {
					_, _ = w.Write([]byte(`{"id":"222","name":"Die Brücke"}`))
				}
			case r.URL.Path == "/oauth/access_token":
				if r.URL.Query().Get("fb_exchange_token") == "USERTOK" && f_appID != "" {
					_, _ = w.Write([]byte(`{"access_token":"USERTOK"}`)) // long-lived token (same value here)
					exchanged = true
				} else {
					w.WriteHeader(400)
				}
			case r.URL.Path == "/me/accounts":
				if accounts && tok == "USERTOK" {
					_, _ = w.Write([]byte(`{"data":[{"id":"222","name":"Die Brücke","access_token":"PAGETOK"}]}`))
				} else {
					_, _ = w.Write([]byte(`{"data":[]}`))
				}
			case r.URL.Path == "/me" && tok == "PAGETOK":
				_, _ = w.Write([]byte(`{"id":"222","name":"Die Brücke"}`))
			case r.URL.Path == "/me":
				_, _ = w.Write([]byte(`{"id":"999","name":"Gerwin"}`)) // user token: /me is the person
			}
		})
		f := newFB(t, srv.URL, false)
		f.appID, f.appSecret, f.pageID = f_appID, f_appID, "brueckegraz"
		return f, f.Auth(context.Background())
	}
	stored := func(f *FacebookPlatform) string {
		tok, _, _, _ := f.store.GetToken(context.Background(), models.PlatformFacebook)
		return tok
	}

	// user token, page found via the page node (business-owned page: /me/accounts is empty)
	f, err := run("USERTOK", false, true)
	if err != nil || stored(f) != "PAGETOK" || f.pageID != "222" || config.ActiveConfig.Facebook.PageID != "222" {
		t.Fatalf("page node: err=%v token=%q id=%q", err, stored(f), f.pageID)
	}
	if config.ActiveConfig.Facebook.PageToken != "" {
		t.Error("the pasted input token must be cleared from the config after a successful login")
	}
	// with app credentials a user token is exchanged for a long-lived one first
	f_appID = "app"
	if f, err = run("USERTOK", false, true); err != nil || !exchanged || stored(f) != "PAGETOK" {
		t.Fatalf("exchange: err=%v exchanged=%v token=%q", err, exchanged, stored(f))
	}
	f_appID = ""
	// user token, page found via /me/accounts only
	if f, err = run("USERTOK", true, false); err != nil || stored(f) != "PAGETOK" {
		t.Fatalf("me/accounts: err=%v token=%q", err, stored(f))
	}
	// a real page token is taken as is
	if f, err = run("PAGETOK", false, false); err != nil || stored(f) != "PAGETOK" {
		t.Fatalf("page token: err=%v token=%q", err, stored(f))
	}
	// a user token without a role on the page is rejected with a clear message
	if _, err = run("USERTOK", false, false); err == nil || !strings.Contains(err.Error(), "user token") {
		t.Errorf("want user-token error, got %v", err)
	}
}

func TestFacebookPhotoPostsFromURLsSingleAndMulti(t *testing.T) {
	rejectAlt := false
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		switch {
		case strings.HasSuffix(r.Path, "/photos"):
			if rejectAlt && r.Form.Get("alt_text_custom") != "" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"bad alt"}}`))
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"ph%d","post_id":"page_post%d"}`, n, n)))
		case strings.HasSuffix(r.Path, "/feed"):
			_, _ = w.Write([]byte(`{"id":"feedpost"}`))
		}
	})
	f := newFB(t, srv.URL, true)
	ctx := context.Background()

	id, err := f.Post(ctx, &models.Post{Body: "cap", Images: []string{"https://x/a.png"}, Alts: []string{"Alt A"}})
	if err != nil || id != "page_post1" {
		t.Fatalf("single: id=%q err=%v", id, err)
	}
	if c := (*reqs)[0]; c.Form.Get("url") != "https://x/a.png" || c.Form.Get("caption") != "cap" || c.Form.Get("alt_text_custom") != "Alt A" || c.Form.Get("published") != "true" {
		t.Errorf("single photo form = %v", c.Form)
	}

	*reqs = nil
	id, err = f.Post(ctx, &models.Post{Body: "week", Images: []string{"https://x/1.png", "https://x/2.png"}})
	if err != nil || id != "feedpost" {
		t.Fatalf("multi: id=%q err=%v", id, err)
	}
	r := *reqs
	if len(r) != 3 || r[0].Form.Get("published") != "false" || !strings.HasSuffix(r[2].Path, "/feed") ||
		!strings.Contains(r[2].Form.Get("attached_media[0]"), "ph1") || !strings.Contains(r[2].Form.Get("attached_media[1]"), "ph2") || r[2].Form.Get("message") != "week" {
		t.Errorf("multi requests: %+v", r)
	}

	*reqs = nil
	rejectAlt = true
	if _, err = f.Post(ctx, &models.Post{Body: "c", Images: []string{"https://x/a.png"}, Alts: []string{"bad"}}); err != nil {
		t.Fatalf("alt rejected must fall back: %v", err)
	}
	if len(*reqs) != 2 || (*reqs)[1].Form.Get("alt_text_custom") != "" {
		t.Errorf("expected a plain retry: %+v", *reqs)
	}
}
