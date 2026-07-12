package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestTelegramPlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bottoken123/getMe" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":123,"is_bot":true,"first_name":"TestBot","username":"test_bot"}}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
		}
	}))
	defer server.Close()

	plat := NewTelegramPlatform(nil, "token123", "chat123")
	plat.apiURL = server.URL

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewTelegramPlatform(nil, "badtoken", "chat123")
	badPlat.apiURL = server.URL
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for bad token, got nil")
	}
}

func TestTelegramPlatform_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Verification
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		response := struct {
			Ok     bool `json:"ok"`
			Result struct {
				MessageID int `json:"message_id"`
			} `json:"result"`
		}{
			Ok: true,
		}
		response.Result.MessageID = 456
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	plat := NewTelegramPlatform(nil, "token123", "chat123")
	plat.apiURL = server.URL

	post := &models.Post{
		Body: "Hello Telegram!",
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "456" {
		t.Errorf("expected message ID 456, got: %q", msgID)
	}
}
