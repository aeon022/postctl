// Package license keeps the cached Pro status in step with Polar. Pro is a subscription: the key stays
// valid while the subscription runs, Polar revokes it afterwards, so the cached status must be re-checked.
// Lifetime keys bought earlier never lapse and simply keep validating as "granted".
package license

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
)

const (
	// RecheckEvery is how often a granted key is validated against Polar.
	RecheckEvery = 24 * time.Hour
	// OfflineGrace is how long a granted key keeps working without a successful check (no network, Polar down).
	OfflineGrace = 21 * 24 * time.Hour
)

var apiBase = "https://api.polar.sh" // overridden by tests

// postctl Pro is sold as its own subscription. These Polar license benefits unlock it: the subscription
// benefit and the former Lifetime benefit. Keys of the missionctl Bundle also unlocked postctl Pro until the
// subscription launched, so Bundle keys created before BundleCutoff stay valid (grandfathered); a Bundle
// bought later does not include postctl Pro.
const (
	benefitSubscription = "1baf6278-efdb-457f-b2d0-15db8c5a2388"
	benefitLifetime     = "49aa53ee-5a6e-4849-b58a-ed93caf21ce2"
	benefitBundle       = "de1be860-1dfc-43da-99a8-206fb2573f09"
)

// BundleCutoff is the day postctl Pro became a separate subscription.
var BundleCutoff = time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)

// Effective turns Polar's status into the one postctl uses: a granted key only counts when its product includes postctl Pro.
func Effective(status, benefitID string, keyCreated time.Time) string {
	if status != "granted" && status != "active" {
		return status
	}
	switch benefitID {
	case "", benefitSubscription, benefitLifetime: // "" = an older Polar response without the field: keep working
		return status
	case benefitBundle:
		if !keyCreated.IsZero() && keyCreated.Before(BundleCutoff) {
			return status
		}
	}
	return "not_for_postctl"
}

type validateResp struct {
	Status    string     `json:"status"`
	BenefitID string     `json:"benefit_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// RevalidateIfDue validates the configured key when the last successful check is older than RecheckEvery
// (or the cached status is not granted, so a renewed subscription is picked up). It updates and saves the
// config and returns whether the cached status changed. Network trouble never downgrades before OfflineGrace.
func RevalidateIfDue(ctx context.Context, c *http.Client, now time.Time) (changed bool, err error) {
	cfg := &config.ActiveConfig
	key := strings.TrimSpace(cfg.LicenseKey)
	org := strings.TrimSpace(cfg.PolarOrgID)
	if key == "" || org == "" {
		return false, nil
	}
	checked, _ := time.Parse(time.RFC3339, cfg.LicenseCheckedAt)
	if config.IsPro() && !checked.IsZero() && now.Sub(checked) < RecheckEvery {
		return false, nil
	}

	body, _ := json.Marshal(map[string]string{"key": key, "organization_id": org})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/v1/customer-portal/license-keys/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return downgradeAfterGrace(cfg, checked, now), err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return downgradeAfterGrace(cfg, checked, now), nil // temporary: keep the cached status
	case resp.StatusCode == http.StatusOK:
		var v validateResp
		_ = json.Unmarshal(raw, &v)
		status := v.Status
		if status == "" {
			status = "granted"
		}
		if v.ExpiresAt != nil && v.ExpiresAt.Before(now) {
			status = "expired"
		}
		return save(cfg, Effective(status, v.BenefitID, v.CreatedAt), now), nil
	default: // 404/422...: Polar does not know the key (any more)
		return save(cfg, "invalid", now), nil
	}
}

func downgradeAfterGrace(cfg *config.Config, lastOK, now time.Time) bool {
	if config.IsPro() && !lastOK.IsZero() && now.Sub(lastOK) > OfflineGrace {
		cfg.LicenseStatus = "offline_expired"
		_ = config.SaveConfig()
		return true
	}
	return false
}

func save(cfg *config.Config, status string, now time.Time) bool {
	changed := cfg.LicenseStatus != status
	cfg.LicenseStatus = status
	cfg.LicenseCheckedAt = now.UTC().Format(time.RFC3339)
	_ = config.SaveConfig()
	return changed
}
