package platforms

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

// igPosts drops the status polls so tests can assert on the creating/publishing calls.
func igPosts(r []mReq) (out []mReq) {
	for _, x := range r {
		if x.Method != "GET" {
			out = append(out, x)
		}
	}
	return
}

func newIG(srvURL string) *InstagramPlatform {
	p := NewInstagramPlatform("TOK", "IG1")
	p.baseURL, p.pollEvery = srvURL, time.Millisecond
	return p
}

func TestInstagramPostCreatesContainerThenPublishes(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"status_code":"FINISHED"}`))
			return
		}
		if strings.HasSuffix(r.Path, "/media") {
			_, _ = w.Write([]byte(`{"id":"C1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"M9"}`))
	})
	p := newIG(srv.URL)
	id, err := p.Post(context.Background(), &models.Post{Body: "hi", Images: []string{"https://x.test/a.jpg"}})
	if err != nil || id != "M9" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	r := igPosts(*reqs)
	if len(r) != 2 || r[0].Path != "/IG1/media" || r[0].Form.Get("image_url") != "https://x.test/a.jpg" ||
		r[1].Path != "/IG1/media_publish" || r[1].Form.Get("creation_id") != "C1" {
		t.Errorf("unexpected requests: %+v", r)
	}
}

func TestInstagramRejectsTextOnlyAndLocalImages(t *testing.T) {
	p := NewInstagramPlatform("TOK", "IG1")
	for _, imgs := range [][]string{nil, {"./local.png"}} {
		if _, err := p.Post(context.Background(), &models.Post{Body: "x", Images: imgs}); err == nil {
			t.Errorf("images %v should be rejected", imgs)
		}
	}
	if p.Delete(context.Background(), "M1") == nil {
		t.Error("Delete must be an honest error")
	}
}

func TestGetPlatformInstagramHostFollowsAPISetting(t *testing.T) {
	config.ActiveConfig.Instagram.API = "instagram"
	defer func() { config.ActiveConfig.Instagram.API = "" }()
	p, err := GetPlatform(models.PlatformInstagram, nil, false)
	if err != nil || !strings.HasPrefix(p.(*InstagramPlatform).baseURL, "https://graph.instagram.com/") {
		t.Fatalf("got %v %v", p, err)
	}
	config.ActiveConfig.Instagram.API = ""
	p, _ = GetPlatform(models.PlatformInstagram, nil, false)
	if !strings.HasPrefix(p.(*InstagramPlatform).baseURL, "https://graph.facebook.com/") {
		t.Error("default must stay on graph.facebook.com")
	}
}

func TestInstagramCarouselAndStory(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"status_code":"FINISHED"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"X` + string(rune('0'+n)) + `"}`))
	})
	p := newIG(srv.URL)
	ctx := context.Background()

	if _, err := p.Post(ctx, &models.Post{Body: "c", Images: []string{"https://x/1.png", "https://x/2.png"}}); err != nil {
		t.Fatal(err)
	}
	r := igPosts(*reqs)
	if len(r) != 4 || r[0].Form.Get("is_carousel_item") != "true" || r[2].Form.Get("media_type") != "CAROUSEL" ||
		r[2].Form.Get("children") == "" || r[3].Path != "/IG1/media_publish" {
		t.Errorf("carousel requests: %+v", r)
	}

	*reqs = nil
	if _, err := p.Post(ctx, &models.Post{Type: "story", Body: "ignored", Images: []string{"https://x/s.png"}}); err != nil {
		t.Fatal(err)
	}
	r = igPosts(*reqs)
	if r[0].Form.Get("media_type") != "STORIES" || r[0].Form.Get("caption") != "" {
		t.Errorf("story container: %+v", r[0])
	}
	if _, err := p.Post(ctx, &models.Post{Type: "story", Images: []string{"https://x/1.png", "https://x/2.png"}}); err == nil {
		t.Error("a story with two images must be rejected")
	}
}

func TestInstagramWaitsUntilContainerIsFinished(t *testing.T) {
	polls := 0
	srv, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Method == "GET" {
			polls++
			if polls < 3 {
				_, _ = w.Write([]byte(`{"status_code":"IN_PROGRESS"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status_code":"FINISHED"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"M1"}`))
	})
	p := newIG(srv.URL)
	if _, err := p.Post(context.Background(), &models.Post{Body: "x", Images: []string{"https://x/a.png"}}); err != nil || polls != 3 {
		t.Fatalf("polls=%d err=%v", polls, err)
	}

	bad, _ := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"status_code":"ERROR","status":"bad image"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"M1"}`))
	})
	if _, err := newIG(bad.URL).Post(context.Background(), &models.Post{Body: "x", Images: []string{"https://x/a.png"}}); err == nil || !strings.Contains(err.Error(), "bad image") {
		t.Errorf("ERROR status must surface, got %v", err)
	}
}

func TestInstagramSendsAltTextAndLocationAndFallsBack(t *testing.T) {
	reject := false
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"status_code":"FINISHED"}`))
			return
		}
		if reject && strings.HasSuffix(r.Path, "/media") && (r.Form.Get("alt_text") != "" || r.Form.Get("location_id") != "") {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":{"message":"bad optional field"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"C1"}`))
	})
	p := newIG(srv.URL)
	p.locationID = "555"
	post := &models.Post{Body: "x", Images: []string{"https://x/a.png"}, Alts: []string{"Plakat"}}

	if _, err := p.Post(context.Background(), post); err != nil {
		t.Fatal(err)
	}
	c := igPosts(*reqs)[0]
	if c.Form.Get("alt_text") != "Plakat" || c.Form.Get("location_id") != "555" {
		t.Errorf("container form = %v", c.Form)
	}

	// Instagram rejects the optional fields: the post must still go out, without them
	*reqs = nil
	reject = true
	if _, err := p.Post(context.Background(), post); err != nil {
		t.Fatalf("fallback must succeed: %v", err)
	}
	r := igPosts(*reqs)
	if len(r) != 3 || r[1].Form.Get("alt_text") != "" || r[1].Form.Get("location_id") != "" || r[1].Form.Get("image_url") == "" {
		t.Errorf("expected rejected attempt, plain retry, publish; got %+v", r)
	}
}
