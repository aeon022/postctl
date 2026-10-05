package platforms

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

type qReq struct {
	Method, Path string
	Q            url.Values
	Form         url.Values
}

// qServer records requests incl. query string (Threads/Graph APIs put their
// parameters there even on POST).
func qServer(t *testing.T, h func(r qReq, n int, w http.ResponseWriter)) (*httptest.Server, *[]qReq) {
	t.Helper()
	var reqs []qReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		reqs = append(reqs, qReq{r.Method, r.URL.Path, r.URL.Query(), r.PostForm})
		w.Header().Set("Content-Type", "application/json")
		h(reqs[len(reqs)-1], len(reqs), w)
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func newThreads(t *testing.T, srvURL string, token string) *ThreadsPlatform {
	t.Helper()
	threadsRetryDelay, threadsPostGap = 0, 0
	t.Cleanup(func() { threadsRetryDelay, threadsPostGap = 3*time.Second, time.Second })
	p := NewThreadsPlatform(newTestStore(t), "app", "secret")
	p.baseURL = srvURL
	if token != "" {
		saveToken(t, p.store, models.PlatformThreads, token, "")
	}
	return p
}

func TestThreadsExchangeCodeStoresUserIDAndLongLivedToken(t *testing.T) {
	srv, reqs := qServer(t, func(r qReq, n int, w http.ResponseWriter) {
		if r.Path == "/oauth/access_token" {
			_, _ = w.Write([]byte(`{"access_token":"SHORT","user_id":1234}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"LONG","token_type":"bearer","expires_in":5184000}`))
	})
	p := newThreads(t, srv.URL, "")
	if err := p.exchangeCodeForToken(context.Background(), "C", "https://localhost:8753/callback"); err != nil {
		t.Fatal(err)
	}
	if f := (*reqs)[0].Form; f.Get("code") != "C" || f.Get("client_secret") != "secret" || f.Get("grant_type") != "authorization_code" {
		t.Errorf("short-lived request form = %v", f)
	}
	if q := (*reqs)[1].Q; (*reqs)[1].Path != "/access_token" || q.Get("grant_type") != "th_exchange_token" || q.Get("access_token") != "SHORT" {
		t.Errorf("long-lived exchange must present the short token: %s %v", (*reqs)[1].Path, q)
	}
	uid, tok, err := p.getUserIDAndToken(context.Background())
	if err != nil || uid != "1234" || tok != "LONG" {
		t.Errorf("stored uid/token = %q/%q err=%v", uid, tok, err)
	}
	_, _, exp, _ := p.store.GetToken(context.Background(), models.PlatformThreads)
	if exp == nil || time.Until(*exp) < 59*24*time.Hour {
		t.Errorf("expiry should be ~60 days, got %v", exp)
	}
	if !p.IsAuthenticated(context.Background()) || p.Name() != "threads" {
		t.Error("IsAuthenticated/Name")
	}

	fail, _ := qServer(t, func(r qReq, n int, w http.ResponseWriter) { w.WriteHeader(400); _, _ = w.Write([]byte(`x`)) })
	if err := newThreads(t, fail.URL, "").exchangeCodeForToken(context.Background(), "C", "r"); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("want status 400, got %v", err)
	}
}

func TestThreadsGetUserIDAndTokenFormat(t *testing.T) {
	p := newThreads(t, "http://x", "")
	if _, _, err := p.getUserIDAndToken(context.Background()); err == nil {
		t.Error("no token must error")
	}
	saveToken(t, p.store, models.PlatformThreads, "no-colon-token", "")
	if _, _, err := p.getUserIDAndToken(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid token format") {
		t.Errorf("malformed composite token: %v", err)
	}
}

func TestThreadsPostSingleAndThreadChain(t *testing.T) {
	srv, reqs := qServer(t, func(r qReq, n int, w http.ResponseWriter) {
		if strings.HasSuffix(r.Path, "/threads") {
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"c%d"}`, n)))
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf(`{"id":"post-of-%s"}`, r.Q.Get("creation_id"))))
	})
	p := newThreads(t, srv.URL, "77:TOK")
	ctx := context.Background()

	id, err := p.Post(ctx, &models.Post{Type: "single", Body: "hi"})
	if err != nil || id != "post-of-c1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	c := (*reqs)[0]
	if c.Path != "/v1.0/77/threads" || c.Method != "POST" || c.Q.Get("text") != "hi" || c.Q.Get("media_type") != "TEXT" || c.Q.Get("access_token") != "TOK" || c.Q.Has("reply_to_id") {
		t.Errorf("container request = %s %v", c.Path, c.Q)
	}
	if pub := (*reqs)[1]; pub.Path != "/v1.0/77/threads_publish" || pub.Q.Get("creation_id") != "c1" {
		t.Errorf("publish request = %s %v", pub.Path, pub.Q)
	}

	*reqs = nil
	id, err = p.Post(ctx, &models.Post{Type: "thread", Tweets: []models.Tweet{{Content: "a"}, {Content: "b"}}})
	if err != nil || id != "post-of-c1" {
		t.Fatalf("thread must return the first post id, got %q err=%v", id, err)
	}
	// requests: create a, publish a, create b (reply_to = published a), publish b
	if got := (*reqs)[2].Q.Get("reply_to_id"); got != "post-of-c1" {
		t.Errorf("second item must reply to the FIRST published post, got reply_to_id=%q", got)
	}
}

func TestThreadsImageByURLPassesThrough(t *testing.T) {
	srv, reqs := qServer(t, func(r qReq, n int, w http.ResponseWriter) { _, _ = w.Write([]byte(`{"id":"x"}`)) })
	p := newThreads(t, srv.URL, "1:T")
	if _, err := p.Post(context.Background(), &models.Post{Type: "single", Body: "pic", Images: []string{"https://cdn.example/p.png"}}); err != nil {
		t.Fatal(err)
	}
	if q := (*reqs)[0].Q; q.Get("media_type") != "IMAGE" || q.Get("image_url") != "https://cdn.example/p.png" {
		t.Errorf("image container params = %v", q)
	}
	if u, err := p.UploadImage(context.Background(), "http://a/b.jpg"); err != nil || u != "http://a/b.jpg" {
		t.Errorf("http(s) path must be returned unchanged, got %q %v", u, err)
	}
	if _, err := p.UploadImage(context.Background(), "/no/such/file.png"); err == nil {
		t.Error("missing local file must error")
	}
}

// Meta's "media not ready yet" (error subcode 4279009) must be retried; any
// other publish failure (e.g. expired token) must fail immediately.
func TestThreadsPublishRetriesOnlyWhenMediaNotReady(t *testing.T) {
	publishes := 0
	srv, _ := qServer(t, func(r qReq, n int, w http.ResponseWriter) {
		if strings.HasSuffix(r.Path, "/threads") {
			_, _ = w.Write([]byte(`{"id":"c"}`))
			return
		}
		publishes++
		if publishes < 3 {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"error_subcode":4279009}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"done"}`))
	})
	p := newThreads(t, srv.URL, "1:T")
	if id, err := p.Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err != nil || id != "done" || publishes != 3 {
		t.Errorf("id=%q err=%v publishes=%d, want success on 3rd attempt", id, err, publishes)
	}

	publishes = 0
	hard, _ := qServer(t, func(r qReq, n int, w http.ResponseWriter) {
		if strings.HasSuffix(r.Path, "/threads") {
			_, _ = w.Write([]byte(`{"id":"c"}`))
			return
		}
		publishes++
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":"OAuthException"}`))
	})
	if _, err := newThreads(t, hard.URL, "1:T").Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || publishes != 1 {
		t.Errorf("non-retryable error must stop after 1 attempt, publishes=%d err=%v", publishes, err)
	}

	publishes = 0
	always, _ := qServer(t, func(r qReq, n int, w http.ResponseWriter) {
		if strings.HasSuffix(r.Path, "/threads") {
			_, _ = w.Write([]byte(`{"id":"c"}`))
			return
		}
		publishes++
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`4279009`))
	})
	if _, err := newThreads(t, always.URL, "1:T").Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || !strings.Contains(err.Error(), "after 5 retries") || publishes != 5 {
		t.Errorf("must give up after 5 attempts, publishes=%d err=%v", publishes, err)
	}
}

func TestThreadsContainerErrorAndDelete(t *testing.T) {
	status := 400
	srv, reqs := qServer(t, func(r qReq, n int, w http.ResponseWriter) { w.WriteHeader(status); _, _ = w.Write([]byte(`{}`)) })
	p := newThreads(t, srv.URL, "9:TT")
	if _, err := p.Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("container error must surface, got %v", err)
	}
	if _, err := newThreads(t, srv.URL, "").Post(context.Background(), &models.Post{Body: "x"}); err == nil {
		t.Error("post without token")
	}

	status = 200
	*reqs = nil
	if err := p.Delete(context.Background(), "pid1"); err != nil {
		t.Fatal(err)
	}
	if r := (*reqs)[0]; r.Method != "DELETE" || r.Path != "/v1.0/pid1" || r.Q.Get("access_token") != "TT" {
		t.Errorf("delete request = %s %s %v", r.Method, r.Path, r.Q)
	}
	status = 403
	if err := p.Delete(context.Background(), "pid1"); err == nil {
		t.Error("failed delete must error")
	}
	if a, err := p.FetchAnalytics(context.Background(), "z"); err != nil || a.PlatformID != "z" {
		t.Errorf("analytics = %+v %v", a, err)
	}
}
