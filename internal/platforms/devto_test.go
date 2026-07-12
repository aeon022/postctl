package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestDevToPlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("api-key") == "valid-token" && r.URL.Path == "/users/me" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"username": "testuser"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "Unauthorized"}`))
		}
	}))
	defer server.Close()

	plat := NewDevToPlatform(nil, "valid-token")
	plat.apiURL = server.URL

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewDevToPlatform(nil, "bad-token")
	badPlat.apiURL = server.URL
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for bad token, got nil")
	}
}

func TestDevToPlatform_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		// Verification
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.Header.Get("api-key") != "valid-token" {
			t.Errorf("expected api-key valid-token, got %s", r.Header.Get("api-key"))
		}

		response := struct {
			ID int `json:"id"`
		}{
			ID: 88888,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	plat := NewDevToPlatform(nil, "valid-token")
	plat.apiURL = server.URL

	post := &models.Post{
		Title: "Test Article",
		Body:  "# Dev.to Integration Test\n\nThis is a test.",
		Tags:  []string{"test", "golang"},
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "88888" {
		t.Errorf("expected article ID 88888, got: %q", msgID)
	}
}
