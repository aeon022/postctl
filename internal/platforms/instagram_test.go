package platforms

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

func TestInstagramPostCreatesContainerThenPublishes(t *testing.T) {
	srv, reqs := mastoServer(t, func(r mReq, n int, w http.ResponseWriter) {
		if strings.HasSuffix(r.Path, "/media") {
			_, _ = w.Write([]byte(`{"id":"C1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"M9"}`))
	})
	p := NewInstagramPlatform("TOK", "IG1")
	p.baseURL = srv.URL
	id, err := p.Post(context.Background(), &models.Post{Body: "hi", Images: []string{"https://x.test/a.jpg"}})
	if err != nil || id != "M9" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	r := *reqs
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
		_, _ = w.Write([]byte(`{"id":"X` + string(rune('0'+n)) + `"}`))
	})
	p := NewInstagramPlatform("TOK", "IG1")
	p.baseURL = srv.URL
	ctx := context.Background()

	if _, err := p.Post(ctx, &models.Post{Body: "c", Images: []string{"https://x/1.png", "https://x/2.png"}}); err != nil {
		t.Fatal(err)
	}
	r := *reqs
	if len(r) != 4 || r[0].Form.Get("is_carousel_item") != "true" || r[2].Form.Get("media_type") != "CAROUSEL" ||
		r[2].Form.Get("children") == "" || r[3].Path != "/IG1/media_publish" {
		t.Errorf("carousel requests: %+v", r)
	}

	*reqs = nil
	if _, err := p.Post(ctx, &models.Post{Type: "story", Body: "ignored", Images: []string{"https://x/s.png"}}); err != nil {
		t.Fatal(err)
	}
	r = *reqs
	if r[0].Form.Get("media_type") != "STORIES" || r[0].Form.Get("caption") != "" {
		t.Errorf("story container: %+v", r[0])
	}
	if _, err := p.Post(ctx, &models.Post{Type: "story", Images: []string{"https://x/1.png", "https://x/2.png"}}); err == nil {
		t.Error("a story with two images must be rejected")
	}
}
