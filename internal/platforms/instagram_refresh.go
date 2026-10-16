package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aeon022/postctl/internal/config"
)

// Instagram-Login tokens last 60 days and can be refreshed once they are 24 hours old (and
// before they expire). Refreshing after 30 days keeps a scheduler-run profile working forever.
const (
	instagramRefreshAfter = 30 * 24 * time.Hour
	instagramRefreshURL   = "https://graph.instagram.com/refresh_access_token"
)

// refreshInstagramToken asks Instagram for a fresh 60-day token.
func refreshInstagramToken(ctx context.Context, client *http.Client, endpoint, token string) (string, error) {
	q := url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var r struct {
		AccessToken string `json:"access_token"`
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(b, &r) != nil || r.AccessToken == "" {
		return "", fmt.Errorf("instagram token refresh failed (status %d): %s", resp.StatusCode, b)
	}
	return r.AccessToken, nil
}

// RefreshInstagramIfDue refreshes the configured Instagram-Login token when it is older than
// 30 days and stores the new token in the config. Returns whether it refreshed.
func RefreshInstagramIfDue(ctx context.Context) (bool, error) {
	ig := &config.ActiveConfig.Instagram
	if ig.API != "instagram" || ig.AccessToken == "" || ig.TokenIssued == "" {
		return false, nil
	}
	issued, err := time.ParseInLocation("2006-01-02", ig.TokenIssued, time.Local)
	if err != nil || time.Since(issued) < instagramRefreshAfter {
		return false, nil
	}
	tok, err := refreshInstagramToken(ctx, &http.Client{Timeout: 20 * time.Second}, instagramRefreshURL, ig.AccessToken)
	if err != nil {
		return false, err
	}
	ig.AccessToken, ig.TokenIssued = tok, time.Now().Format("2006-01-02")
	return true, config.SaveConfig()
}
