package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestMediumPlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") == "Bearer valid-token" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"user123","username":"testuser"}}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Unauthorized"}]}`))
		}
	}))
	defer server.Close()

	plat := NewMediumPlatform(nil, "valid-token")
	plat.apiURL = server.URL

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewMediumPlatform(nil, "bad-token")
	badPlat.apiURL = server.URL
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for bad token, got nil")
	}
}

func TestMediumPlatform_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Handle GET /me
		if r.Method == "GET" && r.URL.Path == "/me" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"id":"user123"}}`))
			return
		}

		// Handle POST /users/user123/posts
		if r.Method == "POST" && r.URL.Path == "/users/user123/posts" {
			if r.Header.Get("Authorization") != "Bearer valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			w.WriteHeader(http.StatusCreated)
			response := struct {
				Data struct {
					ID string `json:"id"`
				} `json:"data"`
			}{}
			response.Data.ID = "medium-post-1"
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	plat := NewMediumPlatform(nil, "valid-token")
	plat.apiURL = server.URL

	post := &models.Post{
		Title: "Test Article",
		Body:  "# Medium Integration Test\n\nThis is a test.",
		Tags:  []string{"test", "golang"},
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "medium-post-1" {
		t.Errorf("expected post ID medium-post-1, got: %q", msgID)
	}
}
