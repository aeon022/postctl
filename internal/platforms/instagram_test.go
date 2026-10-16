package platforms

import (
	"context"
	"net/http"
	"strings"
	"testing"

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
