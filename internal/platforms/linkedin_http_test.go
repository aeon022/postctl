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

	"github.com/aeon022/postctl/internal/config"
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
}

func TestLinkedInAuthorIsOrganizationWhenConfigured(t *testing.T) {
	config.ActiveConfig.LinkedIn.OrganizationID = "424242"
	defer func() { config.ActiveConfig.LinkedIn.OrganizationID = "" }()
	got, err := NewLinkedInPlatform(nil, "", "").authorURN(context.Background(), "tok")
	if err != nil || got != "urn:li:organization:424242" {
		t.Errorf("author = %q, %v", got, err)
	}
}

func TestLinkedInCompanyPagePostsViaPostsAPI(t *testing.T) {
	config.ActiveConfig.LinkedIn.OrganizationID = "424242"
	defer func() { config.ActiveConfig.LinkedIn.OrganizationID = "" }()

	var srvURL string
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		switch {
		case strings.HasPrefix(r.Path, "/rest/images"):
			_, _ = w.Write([]byte(`{"value":{"uploadUrl":"` + srvURL + `/upload","image":"urn:li:image:IMG1"}}`))
		case r.Path == "/rest/posts" && r.Method == "POST":
			w.Header().Set("x-restli-id", "urn:li:share:99")
			w.WriteHeader(201)
		}
	})
	srvURL = srv.URL
	l := newLI(t, srv.URL, true)
	ctx := context.Background()

	// text only
	id, err := l.Post(ctx, &models.Post{Body: "hello page"})
	if err != nil || id != "urn:li:share:99" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	var sent map[string]any
	r := (*reqs)[0]
	_ = json.Unmarshal(r.Raw, &sent)
	if r.Method != "POST" || r.Path != "/rest/posts" || sent["author"] != "urn:li:organization:424242" || sent["commentary"] != "hello page" {
		t.Errorf("request: %s %s %v", r.Method, r.Path, sent)
	}

	// one image: initializeUpload -> PUT bytes -> post with content.media.id
	*reqs = nil
	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Post(ctx, &models.Post{Body: "pic", Images: []string{img}}); err != nil {
		t.Fatal(err)
	}
	got := *reqs
	if len(got) != 3 || got[1].Method != "PUT" || string(got[1].Raw) != "PNGDATA" {
		t.Fatalf("expected init, PUT, post; got %+v", got)
	}
	_ = json.Unmarshal(got[2].Raw, &sent)
	media := sent["content"].(map[string]any)["media"].(map[string]any)
	if media["id"] != "urn:li:image:IMG1" {
		t.Errorf("media = %v", media)
	}

	// several images are rejected, not silently dropped
	if _, err := l.Post(ctx, &models.Post{Body: "x", Images: []string{"a.png", "b.png"}}); err == nil {
		t.Error("multiple images must be rejected")
	}

	// delete goes through /rest/posts
	*reqs = nil
	if err := l.Delete(ctx, "urn:li:share:99"); err != nil {
		t.Fatal(err)
	}
	if (*reqs)[0].Method != "DELETE" || (*reqs)[0].Path != "/rest/posts/urn:li:share:99" {
		t.Errorf("delete: %+v", (*reqs)[0])
	}
}

func TestLinkedInScopes(t *testing.T) {
	if got := linkedinScopes(); got != "w_member_social openid profile" {
		t.Errorf("person scopes = %q", got)
	}
	config.ActiveConfig.LinkedIn.OrganizationID = "1"
	defer func() { config.ActiveConfig.LinkedIn.OrganizationID = "" }()
	if got := linkedinScopes(); got != "w_organization_social" {
		t.Errorf("org scopes = %q", got)
	}
}
