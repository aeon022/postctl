package platforms

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

// GenerateVerifier generiert einen kryptografischen Code Verifier für PKCE
func GenerateVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateChallenge berechnet die SHA256 Code Challenge für PKCE
func GenerateChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// OpenBrowser öffnet den Standardbrowser des Betriebssystems mit der angegebenen URL
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // Linux und andere Unix-Systeme
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

// StartCallbackServer startet einen temporären Webserver auf Port 8753,
// um den OAuth 2.0 Authorization Code abzufangen.
func StartCallbackServer(expectedState string, timeout time.Duration) (string, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    "127.0.0.1:8753",
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		authErr := r.URL.Query().Get("error")

		if authErr != "" {
			errChan <- fmt.Errorf("oauth error from provider: %s", authErr)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("<h1>Authentifizierung fehlgeschlagen</h1><p>Fehler: " + authErr + "</p>"))
			return
		}

		if state != expectedState {
			errChan <- fmt.Errorf("state mismatch: got %q, want %q", state, expectedState)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("<h1>Ungültiger State</h1>"))
			return
		}

		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("<h1>Fehlender Code</h1>"))
			return
		}

		// Erfolg
		codeChan <- code
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Erfolgreich Authentifiziert</title>
				<style>
					body { font-family: -apple-system, sans-serif; text-align: center; padding: 50px; background-color: #1a202c; color: #f7fafc; }
					.card { max-width: 500px; margin: auto; padding: 40px; background: #2d3748; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
					h1 { color: #00f5d4; }
				</style>
			</head>
			<body>
				<div class="card">
					<h1>Verbindung erfolgreich!</h1>
					<p>Du kannst dieses Browserfenster jetzt schließen und zum Terminal zurückkehren.</p>
				</div>
			</body>
			</html>
		`))
	})

	// Server im Hintergrund starten
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Auf Code, Fehler oder Timeout warten
	var code string
	var err error

	select {
	case code = <-codeChan:
		// Erfolgreich empfangen
	case err = <-errChan:
		// Fehler aufgetreten
	case <-time.After(timeout):
		err = fmt.Errorf("authentication timed out after %v", timeout)
	}

	// Server herunterfahren
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	return code, err
}
