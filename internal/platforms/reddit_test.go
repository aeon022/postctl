package platforms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeon022/postctl/internal/models"
)

func TestRedditPlatform_Auth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		username, password, ok := r.BasicAuth()
		if ok && username == "cid" && password == "csec" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token": "token123"}`))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
		}
	}))
	defer server.Close()

	plat := NewRedditPlatform(nil, "cid", "csec", "user", "pass")
	plat.oauthURL = server.URL

	err := plat.Auth(context.Background())
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	badPlat := NewRedditPlatform(nil, "badcid", "csec", "user", "pass")
	badPlat.oauthURL = server.URL
	err = badPlat.Auth(context.Background())
	if err == nil {
		t.Errorf("expected error for bad client basic auth, got nil")
	}
}

func TestRedditPlatform_Post(t *testing.T) {
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "token123"}`))
	}))
	defer oauthServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Verification
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "bearer token123" {
			t.Errorf("expected bearer token123, got %s", r.Header.Get("Authorization"))
		}

		response := struct {
			JSON struct {
				Errors [][]interface{} `json:"errors"`
				Data   struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"data"`
			} `json:"json"`
		}{}
		response.JSON.Data.ID = "12345"
		response.JSON.Data.Name = "t3_12345"
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer apiServer.Close()

	plat := NewRedditPlatform(nil, "cid", "csec", "user", "pass")
	plat.oauthURL = oauthServer.URL
	plat.apiURL = apiServer.URL

	post := &models.Post{
		Title: "Test Post",
		Body:  "This is a reddit body.",
		Tags:  []string{"golang"},
	}

	msgID, err := plat.Post(context.Background(), post)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if msgID != "t3_12345" {
		t.Errorf("expected post name t3_12345, got: %q", msgID)
	}
}
