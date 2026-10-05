package platforms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

type bskyReq struct {
	Path, Auth, CT string
	Query          string
	Body           []byte
}

// bskyServer records every request and answers via h(path, n) where n is how
// many times that path has been hit before (1-based).
func bskyServer(t *testing.T, h func(r bskyReq, n int, w http.ResponseWriter)) (*httptest.Server, *[]bskyReq) {
	t.Helper()
	var mu sync.Mutex
	var reqs []bskyReq
	counts := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		counts[r.URL.Path]++
		n := counts[r.URL.Path]
		q := bskyReq{r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type"), r.URL.RawQuery, b}
		reqs = append(reqs, q)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		h(q, n, w)
	}))
	t.Cleanup(srv.Close)
	return srv, &reqs
}

func newBsky(t *testing.T, srvURL string) *BlueskyPlatform {
	t.Helper()
	b := NewBlueskyPlatform(newTestStore(t), "alice", "app-pass")
	b.baseURL = srvURL
	return b
}

func TestBlueskyAuth(t *testing.T) {
	jwt := testJWT(time.Hour)
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"accessJwt":"` + jwt + `","did":"did:plc:abc","handle":"alice.bsky.social"}`))
	})
	b := newBsky(t, srv.URL)
	if b.handle != "alice.bsky.social" {
		t.Errorf("bare handle must get .bsky.social appended, got %q", b.handle)
	}
	if b.IsAuthenticated(context.Background()) {
		t.Error("must not be authenticated before Auth")
	}
	if err := b.Auth(context.Background()); err != nil {
		t.Fatal(err)
	}
	var sent map[string]string
	_ = json.Unmarshal((*reqs)[0].Body, &sent)
	if sent["identifier"] != "alice.bsky.social" || sent["password"] != "app-pass" {
		t.Errorf("session body = %v", sent)
	}
	tok, did, _, err := b.store.GetToken(context.Background(), models.PlatformBluesky)
	if err != nil || tok != jwt || did != "did:plc:abc" {
		t.Errorf("stored token/did = %q/%q err=%v", tok, did, err)
	}
	if !b.IsAuthenticated(context.Background()) || b.Name() != "bluesky" {
		t.Error("IsAuthenticated/Name after Auth")
	}
}

func TestBlueskyAuthFailures(t *testing.T) {
	srv, _ := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"error":"AuthenticationRequired"}`))
	})
	if err := newBsky(t, srv.URL).Auth(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("want status 401 in error, got %v", err)
	}
	b := NewBlueskyPlatform(newTestStore(t), "", "")
	if err := b.Auth(context.Background()); err == nil || !strings.Contains(err.Error(), "bluesky.handle") {
		t.Errorf("missing config must explain how to configure, got %v", err)
	}
}

func TestBlueskyPostSingleAndThread(t *testing.T) {
	jwt := testJWT(time.Hour)
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		if r.Path == "/xrpc/com.atproto.repo.createRecord" {
			_, _ = w.Write([]byte(`{"uri":"at://did:plc:abc/app.bsky.feed.post/r` + string(rune('0'+n)) + `","cid":"cid` + string(rune('0'+n)) + `"}`))
		}
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, jwt, "did:plc:abc")

	uri, err := b.Post(context.Background(), &models.Post{Type: "single", Body: "hallo bluesky"})
	if err != nil || !strings.HasSuffix(uri, "/r1") {
		t.Fatalf("single post uri=%q err=%v", uri, err)
	}
	var rec struct {
		Repo, Collection string
		Record           map[string]any
	}
	_ = json.Unmarshal((*reqs)[0].Body, &rec)
	if rec.Repo != "did:plc:abc" || rec.Collection != "app.bsky.feed.post" || rec.Record["text"] != "hallo bluesky" || rec.Record["$type"] != "app.bsky.feed.post" {
		t.Errorf("record = %+v", rec)
	}
	if (*reqs)[0].Auth != "Bearer "+jwt || (*reqs)[0].CT != "application/json" {
		t.Errorf("headers auth=%q ct=%q", (*reqs)[0].Auth, (*reqs)[0].CT)
	}
	if _, has := rec.Record["reply"]; has {
		t.Error("first post must not carry reply refs")
	}

	// thread: items 2 and 3 reply with root = first, parent = previous
	*reqs = nil
	uri, err = b.Post(context.Background(), &models.Post{Type: "thread", Tweets: []models.Tweet{{Index: 1, Content: "a"}, {Index: 2, Content: "b"}, {Index: 3, Content: "c"}}})
	if err != nil || !strings.HasSuffix(uri, "/r2") {
		t.Fatalf("thread must return the FIRST post's uri (r2 here), got %q err=%v", uri, err)
	}
	reply := func(i int) (root, parent string) {
		var r struct {
			Record struct {
				Reply struct{ Root, Parent struct{ URI string } }
			}
		}
		_ = json.Unmarshal((*reqs)[i].Body, &r)
		return r.Record.Reply.Root.URI, r.Record.Reply.Parent.URI
	}
	if root, parent := reply(1); !strings.HasSuffix(root, "/r2") || !strings.HasSuffix(parent, "/r2") {
		t.Errorf("item 2 reply root/parent = %q/%q", root, parent)
	}
	if root, parent := reply(2); !strings.HasSuffix(root, "/r2") || !strings.HasSuffix(parent, "/r3") {
		t.Errorf("item 3 reply root/parent = %q/%q, want root=first parent=second", root, parent)
	}
}

func TestBlueskyPostErrors(t *testing.T) {
	jwt := testJWT(time.Hour)
	srv, _ := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":"InvalidRequest","message":"too long"}`))
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, jwt, "did:plc:abc")
	if _, err := b.Post(context.Background(), &models.Post{Type: "single", Body: "x"}); err == nil || !strings.Contains(err.Error(), "400") {
		t.Errorf("want status in error, got %v", err)
	}
	if _, err := b.Post(context.Background(), &models.Post{Type: "thread"}); err == nil || !strings.Contains(err.Error(), "no content") {
		t.Errorf("empty thread must be rejected, got %v", err)
	}
}

// An ExpiredToken response must trigger a fresh createSession and a retry
// carrying the NEW token — the stored token was valid by its exp claim, so
// only the server's answer reveals the expiry.
func TestBlueskyExpiredTokenReauthsAndRetries(t *testing.T) {
	oldJWT, newJWT := testJWT(time.Hour), testJWT(2*time.Hour)
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		switch r.Path {
		case "/xrpc/com.atproto.server.createSession":
			_, _ = w.Write([]byte(`{"accessJwt":"` + newJWT + `","did":"did:plc:abc"}`))
		case "/xrpc/com.atproto.repo.createRecord":
			if r.Auth != "Bearer "+newJWT {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":"ExpiredToken","message":"Token has expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"uri":"at://did:plc:abc/app.bsky.feed.post/ok","cid":"c"}`))
		}
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, oldJWT, "did:plc:abc")

	uri, err := b.Post(context.Background(), &models.Post{Type: "single", Body: "x"})
	if err != nil || !strings.HasSuffix(uri, "/ok") {
		t.Fatalf("uri=%q err=%v", uri, err)
	}
	var paths []string
	for _, r := range *reqs {
		paths = append(paths, strings.TrimPrefix(r.Path, "/xrpc/com.atproto."))
	}
	if got, want := strings.Join(paths, " "), "repo.createRecord server.createSession repo.createRecord"; got != want {
		t.Errorf("request order = %q, want %q", got, want)
	}
	if tok, _, _, _ := b.store.GetToken(context.Background(), models.PlatformBluesky); tok != newJWT {
		t.Error("refreshed token was not persisted")
	}
}

func TestBlueskyUploadImage(t *testing.T) {
	jwt := testJWT(time.Hour)
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"blob":{"$type":"blob","ref":{"$link":"bafk"},"mimeType":"image/png","size":4}}`))
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, jwt, "did:plc:abc")

	dir := t.TempDir()
	png, jpg := filepath.Join(dir, "a.PNG"), filepath.Join(dir, "b.jpg")
	_ = os.WriteFile(png, []byte("PNG!"), 0o600)
	_ = os.WriteFile(jpg, []byte("JPG!"), 0o600)

	blob, err := b.UploadImage(context.Background(), png)
	if err != nil || !strings.Contains(blob, "bafk") {
		t.Fatalf("blob=%q err=%v", blob, err)
	}
	if (*reqs)[0].CT != "image/png" || string((*reqs)[0].Body) != "PNG!" {
		t.Errorf("png upload ct=%q body=%q (extension match must be case-insensitive)", (*reqs)[0].CT, (*reqs)[0].Body)
	}
	_, _ = b.UploadImage(context.Background(), jpg)
	if (*reqs)[1].CT != "image/jpeg" {
		t.Errorf("jpg ct=%q", (*reqs)[1].CT)
	}
	if _, err := b.UploadImage(context.Background(), filepath.Join(dir, "missing.png")); err == nil {
		t.Error("missing file must error")
	}
}

func TestBlueskyFetchAnalytics(t *testing.T) {
	jwt := testJWT(time.Hour)
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		_, _ = w.Write([]byte(`{"thread":{"post":{"likeCount":3,"repostCount":2,"replyCount":1}}}`))
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, jwt, "did:plc:abc")

	uri := "at://did:plc:abc/app.bsky.feed.post/xyz"
	a, err := b.FetchAnalytics(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if a.Likes != 3 || a.Shares != 2 || a.Comments != 1 || a.PlatformID != uri || a.Impressions != 3*8+2*35+1*12+10 {
		t.Errorf("analytics = %+v", a)
	}
	if !strings.Contains((*reqs)[0].Query, "uri=at%3A%2F%2Fdid%3Aplc%3Aabc") {
		t.Errorf("uri must be query-escaped, got %q", (*reqs)[0].Query)
	}
}

func TestBlueskyDelete(t *testing.T) {
	jwt := testJWT(time.Hour)
	status := 200
	srv, reqs := bskyServer(t, func(r bskyReq, n int, w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{}`))
	})
	b := newBsky(t, srv.URL)
	saveToken(t, b.store, models.PlatformBluesky, jwt, "did:plc:abc")

	if err := b.Delete(context.Background(), "at://did:plc:abc/app.bsky.feed.post/3kabc"); err != nil {
		t.Fatal(err)
	}
	var sent map[string]string
	_ = json.Unmarshal((*reqs)[0].Body, &sent)
	if sent["rkey"] != "3kabc" || sent["repo"] != "did:plc:abc" || sent["collection"] != "app.bsky.feed.post" {
		t.Errorf("delete body = %v", sent)
	}
	status = 500
	if err := b.Delete(context.Background(), "x/3kabc"); err == nil {
		t.Error("a failed delete must surface an error, or the local record is removed while the post stays live")
	}
}
