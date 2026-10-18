package license

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
)

func setup(t *testing.T, status string, checked time.Time, handler http.HandlerFunc) (*int32, time.Time) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	old := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = old })
	saved := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = saved })
	config.ActiveConfig.LicenseKey = "a83-postctl-ABC"
	config.ActiveConfig.PolarOrgID = "org-1"
	config.ActiveConfig.LicenseStatus = status
	config.ActiveConfig.LicenseCheckedAt = ""
	if !checked.IsZero() {
		config.ActiveConfig.LicenseCheckedAt = checked.UTC().Format(time.RFC3339)
	}
	return &calls, time.Now()
}

func reply(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, `{"status":%q}`, status) }
}

func TestFreshGrantedKeyIsNotRevalidated(t *testing.T) {
	calls, now := setup(t, "granted", time.Now().Add(-2*time.Hour), reply("granted"))
	if changed, err := RevalidateIfDue(context.Background(), http.DefaultClient, now); changed || err != nil || *calls != 0 {
		t.Errorf("checked 2 h ago: no call expected, got changed=%v err=%v calls=%d", changed, err, *calls)
	}
}

func TestDueKeyIsValidatedAndTimestampUpdated(t *testing.T) {
	calls, now := setup(t, "granted", time.Now().Add(-48*time.Hour), reply("granted"))
	if _, err := RevalidateIfDue(context.Background(), http.DefaultClient, now); err != nil || *calls != 1 {
		t.Fatalf("a due key is validated once: err=%v calls=%d", err, *calls)
	}
	if got := config.ActiveConfig.LicenseCheckedAt; got == "" || !config.IsPro() {
		t.Errorf("still Pro, checked_at refreshed: %q pro=%v", got, config.IsPro())
	}
}

func TestEndedSubscriptionLosesPro(t *testing.T) {
	_, now := setup(t, "granted", time.Now().Add(-48*time.Hour), reply("revoked"))
	changed, err := RevalidateIfDue(context.Background(), http.DefaultClient, now)
	if err != nil || !changed || config.IsPro() {
		t.Errorf("a revoked key must drop Pro: changed=%v err=%v pro=%v status=%q", changed, err, config.IsPro(), config.ActiveConfig.LicenseStatus)
	}
}

func TestRenewedSubscriptionIsPickedUp(t *testing.T) {
	calls, now := setup(t, "revoked", time.Now().Add(-1*time.Hour), reply("granted"))
	if changed, _ := RevalidateIfDue(context.Background(), http.DefaultClient, now); !changed || !config.IsPro() || *calls != 1 {
		t.Errorf("a lapsed key is re-checked right away and regains Pro: pro=%v calls=%d", config.IsPro(), *calls)
	}
}

func TestUnknownKeyBecomesInvalid(t *testing.T) {
	_, now := setup(t, "granted", time.Now().Add(-48*time.Hour), func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	RevalidateIfDue(context.Background(), http.DefaultClient, now)
	if config.IsPro() || config.ActiveConfig.LicenseStatus != "invalid" {
		t.Errorf("404 → invalid, got %q", config.ActiveConfig.LicenseStatus)
	}
}

func TestOfflineKeepsProUntilGraceEnds(t *testing.T) {
	_, now := setup(t, "granted", time.Now().Add(-10*24*time.Hour), reply("granted"))
	apiBase = "http://127.0.0.1:1" // nothing listens
	if changed, _ := RevalidateIfDue(context.Background(), &http.Client{Timeout: time.Second}, now); changed || !config.IsPro() {
		t.Errorf("10 days offline is within the grace period")
	}
	config.ActiveConfig.LicenseCheckedAt = now.Add(-30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	if changed, _ := RevalidateIfDue(context.Background(), &http.Client{Timeout: time.Second}, now); !changed || config.IsPro() {
		t.Errorf("30 days without a successful check ends Pro: status=%q", config.ActiveConfig.LicenseStatus)
	}
}

func TestServerErrorsAndOverridesDoNotDowngrade(t *testing.T) {
	_, now := setup(t, "granted", time.Now().Add(-48*time.Hour), func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 503) })
	RevalidateIfDue(context.Background(), http.DefaultClient, now)
	if !config.IsPro() {
		t.Error("a 503 from Polar must not remove Pro")
	}
	calls, now := setup(t, "active", time.Time{}, reply("revoked"))
	config.ActiveConfig.LicenseKey = "postctl-pro-family"
	if changed, _ := RevalidateIfDue(context.Background(), http.DefaultClient, now); changed || *calls != 0 || !config.IsPro() {
		t.Error("the local developer/family override is never sent to Polar")
	}
}

func TestOnlyPostctlProductsUnlockPro(t *testing.T) {
	before := BundleCutoff.Add(-24 * time.Hour)
	after := BundleCutoff.Add(24 * time.Hour)
	for _, tc := range []struct {
		name, status, benefit string
		created               time.Time
		want                  string
	}{
		{"subscription key", "granted", benefitSubscription, after, "granted"},
		{"former lifetime key", "granted", benefitLifetime, before, "granted"},
		{"Bundle bought before the cutoff stays valid (grandfathered)", "granted", benefitBundle, before, "granted"},
		{"Bundle bought after the cutoff does not include postctl Pro", "granted", benefitBundle, after, "not_for_postctl"},
		{"some other product's key", "granted", "00000000-0000-0000-0000-000000000000", before, "not_for_postctl"},
		{"older response without benefit_id keeps working", "granted", "", time.Time{}, "granted"},
		{"revoked stays revoked", "revoked", benefitSubscription, after, "revoked"},
	} {
		if got := Effective(tc.status, tc.benefit, tc.created); got != tc.want {
			t.Errorf("%s: Effective = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRevalidateAppliesTheProductCheck(t *testing.T) {
	_, now := setup(t, "granted", time.Now().Add(-48*time.Hour), func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"granted","benefit_id":%q,"created_at":%q}`, benefitBundle, BundleCutoff.Add(time.Hour).Format(time.RFC3339))
	})
	RevalidateIfDue(context.Background(), http.DefaultClient, now)
	if config.IsPro() || config.ActiveConfig.LicenseStatus != "not_for_postctl" {
		t.Errorf("a Bundle key from after the cutoff must not unlock Pro, got %q", config.ActiveConfig.LicenseStatus)
	}
}
