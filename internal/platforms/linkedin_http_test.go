package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func newLI(t *testing.T, srvURL string, withToken bool) *LinkedInPlatform {
	t.Helper()
	l := NewLinkedInPlatform(newTestStore(t), "cid", "csecret")
	l.authURL, l.apiURL = srvURL, srvURL
	if withToken {
		saveToken(t, l.store, models.PlatformLinkedIn, "LITOK", "")
	}
	return l
}

func TestLinkedInExchangeCodeStoresTokenWithExpiry(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"access_token":"LITOK","expires_in":3600}`))
	})
	l := newLI(t, srv.URL, false)
	if err := l.exchangeCodeForToken(context.Background(), "abc", "http://localhost:8753/callback"); err != nil {
		t.Fatal(err)
	}
	f := (*reqs)[0].Form
	if (*reqs)[0].Path != "/oauth/v2/accessToken" || f.Get("grant_type") != "authorization_code" || f.Get("code") != "abc" || f.Get("client_secret") != "csecret" {
		t.Errorf("token request = %v", f)
	}
	tok, _, exp, err := l.store.GetToken(context.Background(), models.PlatformLinkedIn)
	if err != nil || tok != "LITOK" || exp == nil {
		t.Fatalf("token=%q exp=%v err=%v", tok, exp, err)
	}
	if d := time.Until(*exp); d < 59*time.Minute || d > 61*time.Minute {
		t.Errorf("expiry should be ~1h from now, got %v", d)
	}
	if !l.IsAuthenticated(context.Background()) || l.Name() != "linkedin" {
		t.Error("IsAuthenticated/Name")
	}

	bad, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(401); _, _ = w.Write([]byte(`nope`)) })
	if err := newLI(t, bad.URL, false).exchangeCodeForToken(context.Background(), "x", "y"); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("want status 401, got %v", err)
	}
}

func liServer(t *testing.T, ugcStatus int) (*LinkedInPlatform, *[]mReq) {
	t.Helper()
	var srvURL string
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		switch r.Path {
		case "/v2/userinfo":
			_, _ = w.Write([]byte(`{"sub":"member123"}`))
		case "/v2/assets":
			_, _ = w.Write([]byte(`{"value":{"asset":"urn:li:digitalmediaAsset:A1","uploadMechanism":{"com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest":{"uploadUrl":"` + srvURL + `/upload/A1"}}}}`))
		case "/upload/A1":
			w.WriteHeader(201)
		case "/v2/ugcPosts":
			w.WriteHeader(ugcStatus)
			_, _ = w.Write([]byte(`{"id":"urn:li:share:999"}`))
		}
	})
	srvURL = srv.URL
	return newLI(t, srv.URL, true), reqs
}

func TestLinkedInPostText(t *testing.T) {
	l, reqs := liServer(t, 201)
	id, err := l.Post(context.Background(), &models.Post{Body: "Hallo LinkedIn"})
	if err != nil || id != "urn:li:share:999" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	var ugc mReq
	for _, r := range *reqs {
		if r.Path == "/v2/ugcPosts" {
			ugc = r
		}
	}
	if ugc.Auth != "Bearer LITOK" || ugc.CT != "application/json" {
		t.Errorf("auth=%q ct=%q", ugc.Auth, ugc.CT)
	}
	var body struct {
		Author, LifecycleState string
		SpecificContent        map[string]struct {
			ShareCommentary    struct{ Text string }
			ShareMediaCategory string
			Media              []any
		}
		Visibility map[string]string
	}
	if err := json.Unmarshal(ugc.Raw, &body); err != nil {
		t.Fatal(err)
	}
	sc := body.SpecificContent["com.linkedin.ugc.ShareContent"]
	if body.Author != "urn:li:person:member123" || body.LifecycleState != "PUBLISHED" || sc.ShareCommentary.Text != "Hallo LinkedIn" || sc.ShareMediaCategory != "NONE" || len(sc.Media) != 0 {
		t.Errorf("ugc body = %+v", body)
	}
	if body.Visibility["com.linkedin.ugc.MemberNetworkVisibility"] != "PUBLIC" {
		t.Errorf("visibility = %v", body.Visibility)
	}
}

func TestLinkedInPostWithImageRegistersUploadsThenPosts(t *testing.T) {
	l, reqs := liServer(t, 201)
	img := filepath.Join(t.TempDir(), "i.png")
	_ = os.WriteFile(img, []byte("PNGDATA"), 0o600)

	if _, err := l.Post(context.Background(), &models.Post{Body: "mit Bild", Images: []string{img}}); err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, r := range *reqs {
		order = append(order, r.Method+" "+r.Path)
	}
	// userinfo (Post) → userinfo (UploadImage) → register → PUT → ugcPosts
	want := "GET /v2/userinfo,GET /v2/userinfo,POST /v2/assets,PUT /upload/A1,POST /v2/ugcPosts"
	if got := strings.Join(order, ","); got != want {
		t.Errorf("order = %s\nwant    %s", got, want)
	}
	for _, r := range *reqs {
		if r.Path == "/upload/A1" && string(r.Raw) != "PNGDATA" {
			t.Errorf("uploaded bytes = %q", r.Raw)
		}
		if r.Path == "/v2/ugcPosts" && (!strings.Contains(string(r.Raw), `"shareMediaCategory":"IMAGE"`) || !strings.Contains(string(r.Raw), "urn:li:digitalmediaAsset:A1")) {
			t.Errorf("ugc post must reference the uploaded asset: %s", r.Raw)
		}
	}
}

func TestLinkedInThreadIsJoinedAndErrorsSurface(t *testing.T) {
	l, reqs := liServer(t, 201)
	if _, err := l.Post(context.Background(), &models.Post{Tweets: []models.Tweet{{Content: "a"}, {Content: "b"}}}); err != nil {
		t.Fatal(err)
	}
	if last := (*reqs)[len(*reqs)-1]; !strings.Contains(string(last.Raw), `"text":"a\n\nb"`) {
		t.Errorf("thread body = %s", last.Raw)
	}

	bad, _ := liServer(t, 422)
	if _, err := bad.Post(context.Background(), &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "422") {
		t.Errorf("want status 422, got %v", err)
	}
	if _, err := newLI(t, "http://127.0.0.1:1", false).Post(context.Background(), &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "token") {
		t.Errorf("no token: %v", err)
	}
	if _, err := l.UploadImage(context.Background(), filepath.Join(t.TempDir(), "nope.png")); err == nil {
		t.Error("missing image must error")
	}
}

func TestLinkedInAnalyticsAndDelete(t *testing.T) {
	l := NewLinkedInPlatform(nil, "", "")
	if a, err := l.FetchAnalytics(context.Background(), "u"); err != nil || a.PlatformID != "u" {
		t.Errorf("analytics = %+v err=%v", a, err)
	}
	if err := l.Delete(context.Background(), "urn:x"); err == nil || !strings.Contains(err.Error(), "urn:x") {
		t.Errorf("Delete must be an honest error, got %v", err)
	}
}
