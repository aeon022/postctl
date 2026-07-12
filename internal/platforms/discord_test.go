package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestDiscordPlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name": "test-webhook"}`))
	}))
	defer server.Close()

	plat := NewDiscordPlatform(nil, server.URL)

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewDiscordPlatform(nil, "")
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for empty webhook url, got nil")
	}
}

func TestDiscordPlatform_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Verification
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		response := struct {
			ID string `json:"id"`
		}{
			ID: "7777",
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	plat := NewDiscordPlatform(nil, server.URL)

	post := &models.Post{
		Body: "Hello Discord!",
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "7777" {
		t.Errorf("expected message ID 7777, got: %q", msgID)
	}
}
