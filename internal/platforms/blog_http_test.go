package platforms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func TestDevToPostPayload(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { w.WriteHeader(201); _, _ = w.Write([]byte(`{"id":7}`)) })
	p := NewDevToPlatform(nil, "k")
	p.apiURL = srv.URL
	past, future := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)

	post := func(sched *time.Time, tags ...string) (published bool, tagList []string) {
		*reqs = nil
		if _, err := p.Post(context.Background(), &models.Post{Title: "T", Body: "B", Tags: tags, ScheduledAt: sched}); err != nil {
			t.Fatal(err)
		}
		var b struct {
			Article struct {
				Published bool
				Tags      []string
			}
		}
		_ = json.Unmarshal((*reqs)[0].Raw, &b)
		return b.Article.Published, b.Article.Tags
	}
	if pub, tags := post(nil, "a", "b", "c", "d", "e", "f"); !pub || len(tags) != 4 {
		t.Errorf("immediate post: published=%v tags=%v (Dev.to allows max 4 tags)", pub, tags)
	}
	// The scheduler calls Post when a scheduled post is DUE (ScheduledAt in the past):
	// that must publish. Only a still-future time may fall back to a draft.
	if pub, _ := post(&past, "a"); !pub {
		t.Error("a due scheduled post was saved as a Dev.to DRAFT instead of being published")
	}
	if pub, _ := post(&future, "a"); pub {
		t.Error("a future-dated post must stay a draft")
	}
}

func TestDevToErrorsAndStubs(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	})
	p := NewDevToPlatform(nil, "k")
	p.apiURL = srv.URL
	ctx := context.Background()
	if _, err := p.Post(ctx, &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "422") {
		t.Errorf("want status 422, got %v", err)
	}
	if err := p.Delete(ctx, "9"); err == nil || !strings.Contains(err.Error(), "9") {
		t.Errorf("Delete must be honest, got %v", err)
	}
	if got, _ := p.UploadImage(ctx, "u"); got != "u" || p.Name() != "devto" || !p.IsAuthenticated(ctx) || NewDevToPlatform(nil, "").IsAuthenticated(ctx) {
		t.Error("UploadImage/Name/IsAuthenticated")
	}
	if a, err := p.FetchAnalytics(ctx, "1"); err != nil || a.Likes != 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
	if err := NewDevToPlatform(nil, "").Auth(ctx); err == nil || !strings.Contains(err.Error(), "devto.api_token") {
		t.Errorf("missing token should explain configuration: %v", err)
	}
}

func TestMediumPostStatusAndTags(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Path == "/me" {
			_, _ = w.Write([]byte(`{"data":{"id":"u1"}}`))
			return
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":"post1"}}`))
	})
	p := NewMediumPlatform(nil, "tok")
	p.apiURL = srv.URL
	past, future := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)

	status := func(sched *time.Time) (st string, tags int) {
		*reqs = nil
		if _, err := p.Post(context.Background(), &models.Post{Title: "T", Body: "B", Tags: []string{"1", "2", "3", "4", "5", "6", "7"}, ScheduledAt: sched}); err != nil {
			t.Fatal(err)
		}
		var b struct {
			PublishStatus string
			Tags          []string
			ContentFormat string
		}
		_ = json.Unmarshal((*reqs)[len(*reqs)-1].Raw, &b)
		if b.ContentFormat != "markdown" {
			t.Errorf("contentFormat = %q", b.ContentFormat)
		}
		return b.PublishStatus, len(b.Tags)
	}
	if st, n := status(nil); st != "public" || n != 5 {
		t.Errorf("immediate: status=%s tags=%d (Medium allows max 5)", st, n)
	}
	if st, _ := status(&past); st != "public" {
		t.Errorf("a due scheduled post was saved as Medium %q instead of being published", st)
	}
	if st, _ := status(&future); st != "draft" {
		t.Errorf("future-dated post must be a draft, got %q", st)
	}
	ctx := context.Background()
	if err := p.Delete(ctx, "x"); err == nil {
		t.Error("Delete must be honest")
	}
	if got, _ := p.UploadImage(ctx, "u"); got != "u" || p.Name() != "medium" || !p.IsAuthenticated(ctx) {
		t.Error("UploadImage/Name/IsAuthenticated")
	}
	if a, err := p.FetchAnalytics(ctx, "1"); err != nil || a.Likes != 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
}

func TestMediumPostErrors(t *testing.T) {
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Path == "/me" {
			w.WriteHeader(401)
			return
		}
	})
	p := NewMediumPlatform(nil, "tok")
	p.apiURL = srv.URL
	if _, err := p.Post(context.Background(), &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "user ID") {
		t.Errorf("auth failure must be reported as user-id failure, got %v", err)
	}
}

func TestHashnodePostPayloadAndGraphQLErrors(t *testing.T) {
	var auth string
	respond := `{"data":{"publishPost":{"post":{"id":"hn-1","url":"u"}}}}`
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) { auth = r.Auth; _, _ = w.Write([]byte(respond)) })
	p := NewHashnodePlatform(nil, "TOKEN", "pub9")
	p.apiURL = srv.URL
	ctx := context.Background()

	id, err := p.Post(ctx, &models.Post{Title: "Titel", Body: "Body", Tags: []string{"GoLang", "CLI"}})
	if err != nil || id != "hn-1" || auth != "TOKEN" {
		t.Fatalf("id=%q err=%v auth=%q (Hashnode takes the raw token, no Bearer)", id, err, auth)
	}
	var b struct {
		Variables struct {
			Input struct {
				Title, PublicationID, ContentMarkdown string
				Tags                                  []struct{ Slug, Name string }
			}
		}
	}
	_ = json.Unmarshal((*reqs)[0].Raw, &b)
	in := b.Variables.Input
	if in.PublicationID != "pub9" || in.Title != "Titel" || len(in.Tags) != 2 || in.Tags[0].Slug != "golang" || in.Tags[0].Name != "GoLang" {
		t.Errorf("publish input = %+v (slugs must be lower-cased)", in)
	}

	// GraphQL reports failures with HTTP 200 + errors[]
	respond = `{"errors":[{"message":"publication not found"}]}`
	if _, err := p.Post(ctx, &models.Post{Body: "x"}); err == nil || !strings.Contains(err.Error(), "publication not found") {
		t.Errorf("GraphQL errors in a 200 response must fail the post, got %v", err)
	}
	if err := p.Delete(ctx, "x"); err == nil {
		t.Error("Delete must be honest")
	}
	if got, _ := p.UploadImage(ctx, "u"); got != "u" || p.Name() != "hashnode" || !p.IsAuthenticated(ctx) || NewHashnodePlatform(nil, "t", "").IsAuthenticated(ctx) {
		t.Error("UploadImage/Name/IsAuthenticated (needs token AND publication id)")
	}
	if a, err := p.FetchAnalytics(ctx, "1"); err != nil || a.Likes != 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
}

func TestRedditSubredditChoiceAndErrors(t *testing.T) {
	var submit url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "access_token") || r.URL.Path == "/" {
			_, _ = w.Write([]byte(`{"access_token":"RTOK"}`))
			return
		}
		b, _ := io.ReadAll(r.Body)
		submit, _ = url.ParseQuery(string(b))
		if r.Header.Get("Authorization") != "bearer RTOK" || !strings.Contains(r.Header.Get("User-Agent"), "/u/me") {
			w.WriteHeader(403)
			return
		}
		if submit.Get("title") == "reject" {
			_, _ = w.Write([]byte(`{"json":{"errors":[["SUBREDDIT_NOEXIST","that subreddit doesn't exist","sr"]],"data":{}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"json":{"errors":[],"data":{"id":"abc","name":"t3_abc"}}}`))
	}))
	defer srv.Close()
	p := NewRedditPlatform(nil, "cid", "csec", "me", "pw")
	p.oauthURL, p.apiURL = srv.URL, srv.URL
	ctx := context.Background()

	cases := []struct {
		name string
		post models.Post
		want string
	}{
		{"first tag wins", models.Post{Title: "t", Tags: []string{"golang", "x"}, Campaign: "camp"}, "golang"},
		{"campaign when no tags", models.Post{Title: "t", Campaign: "camp"}, "camp"},
		{"default campaign is not a subreddit", models.Post{Title: "t", Campaign: "default"}, "test"},
		{"fallback", models.Post{Title: "t"}, "test"},
	}
	for _, c := range cases {
		id, err := p.Post(ctx, &c.post)
		if err != nil || id != "t3_abc" {
			t.Fatalf("%s: id=%q err=%v", c.name, id, err)
		}
		if submit.Get("sr") != c.want || submit.Get("kind") != "self" {
			t.Errorf("%s: sr=%q kind=%q, want sr=%q", c.name, submit.Get("sr"), submit.Get("kind"), c.want)
		}
	}
	if _, err := p.Post(ctx, &models.Post{Title: "reject"}); err == nil || !strings.Contains(err.Error(), "SUBREDDIT_NOEXIST") {
		t.Errorf("reddit's in-body errors (HTTP 200) must fail the post, got %v", err)
	}
	if err := p.Delete(ctx, "t3_abc"); err == nil {
		t.Error("Delete must be honest")
	}
	if got, _ := p.UploadImage(ctx, "u"); got != "u" || p.Name() != "reddit" {
		t.Error("UploadImage/Name")
	}
	if a, err := p.FetchAnalytics(ctx, "1"); err != nil || a.Likes != 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
}
