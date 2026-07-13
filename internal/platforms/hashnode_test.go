package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestHashnodePlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") == "valid-token" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"me":{"username":"testuser"}}}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Unauthorized"}]}`))
		}
	}))
	defer server.Close()

	plat := NewHashnodePlatform(nil, "valid-token", "pub123")
	plat.apiURL = server.URL

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewHashnodePlatform(nil, "bad-token", "pub123")
	badPlat.apiURL = server.URL
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for bad token, got nil")
	}
}

func TestHashnodePlatform_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Verification
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "valid-token" {
			t.Errorf("expected Authorization valid-token, got %s", r.Header.Get("Authorization"))
		}

		response := struct {
			Data struct {
				PublishPost struct {
					Post struct {
						ID  string `json:"id"`
						URL string `json:"url"`
					} `json:"post"`
				} `json:"publishPost"`
			} `json:"data"`
		}{}
		response.Data.PublishPost.Post.ID = "post999"
		response.Data.PublishPost.Post.URL = "https://test.hashnode.dev/slug"
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	plat := NewHashnodePlatform(nil, "valid-token", "pub123")
	plat.apiURL = server.URL

	post := &models.Post{
		Title: "Test Article",
		Body:  "# Hashnode Integration Test\n\nThis is a test.",
		Tags:  []string{"test", "golang"},
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "post999" {
		t.Errorf("expected post ID post999, got: %q", msgID)
	}
}
