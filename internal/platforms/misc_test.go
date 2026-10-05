package platforms

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func TestGetPlatformReturnsMatchingImplementation(t *testing.T) {
	for _, name := range []string{
		models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads, models.PlatformMastodon,
		models.PlatformBluesky, models.PlatformFacebook, models.PlatformTelegram, models.PlatformDiscord,
		models.PlatformDevTo, models.PlatformReddit, models.PlatformHashnode, models.PlatformMedium,
	} {
		p, err := GetPlatform(name, nil, false)
		if err != nil || p.Name() != name {
			t.Errorf("GetPlatform(%q) = %v, %v", name, p, err)
		}
		d, err := GetPlatform(name, nil, true)
		if err != nil || d.Name() != name {
			t.Errorf("dry-run GetPlatform(%q) must keep the platform name, got %v, %v", name, d, err)
		}
		if _, ok := d.(*DryRunPlatform); !ok {
			t.Errorf("dryRun=true must return the mock for %q, got %T", name, d)
		}
	}
	if _, err := GetPlatform("myspace", nil, false); err == nil || !strings.Contains(err.Error(), "myspace") {
		t.Errorf("unknown platform: %v", err)
	}
}

func TestDryRunPlatformNeverTouchesNetworkOrSleeps(t *testing.T) {
	dryRunSleep = func(time.Duration) {}
	t.Cleanup(func() { dryRunSleep = time.Sleep })
	ctx := context.Background()
	d := NewDryRunPlatform("bluesky")

	if err := d.Auth(ctx); err != nil || !d.IsAuthenticated(ctx) {
		t.Error("dry-run is always authenticated")
	}
	single, err := d.Post(ctx, &models.Post{Body: "x", Images: []string{"a.png"}})
	thread, err2 := d.Post(ctx, &models.Post{Type: "thread", Tweets: []models.Tweet{{Index: 1, Content: "a", Image: "i.png"}, {Index: 2, Content: "b"}}})
	if err != nil || err2 != nil || !strings.HasPrefix(single, "dryrun-") || !strings.HasPrefix(thread, "dryrun-") {
		t.Errorf("ids = %q %q (%v %v)", single, thread, err, err2)
	}
	if m, err := d.UploadImage(ctx, "p"); err != nil || !strings.HasPrefix(m, "dryrun-media-") {
		t.Errorf("media id = %q %v", m, err)
	}
	a, err := d.FetchAnalytics(ctx, "pid")
	if err != nil || a.PlatformID != "pid" || a.Likes < 12 || a.Impressions <= 0 {
		t.Errorf("analytics = %+v %v", a, err)
	}
	if err := d.Delete(ctx, "pid"); err != nil {
		t.Errorf("delete = %v", err)
	}
}

// RFC 7636 appendix B test vector.
func TestPKCEChallengeMatchesRFC7636(t *testing.T) {
	if got := GenerateChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Errorf("challenge = %q", got)
	}
	a, err := GenerateVerifier()
	b, _ := GenerateVerifier()
	if err != nil || a == b || len(a) < 43 || len(a) > 128 {
		t.Errorf("verifier must be random and 43-128 chars (RFC 7636), got %q (%d)", a, len(a))
	}
}

func callbackGet(t *testing.T, query string) (int, string) {
	t.Helper()
	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ { // server starts in a goroutine
		resp, err = http.Get("http://127.0.0.1:8753/callback?" + query)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestCallbackServer(t *testing.T) {
	type result struct {
		code string
		err  error
	}
	run := func(state string, timeout time.Duration) <-chan result {
		ch := make(chan result, 1)
		go func() { c, e := StartCallbackServer(state, timeout); ch <- result{c, e} }()
		return ch
	}

	ch := run("S1", 5*time.Second)
	if status, _ := callbackGet(t, "state=S1&code=THECODE"); status != 200 {
		t.Errorf("success status = %d", status)
	}
	if r := <-ch; r.err != nil || r.code != "THECODE" {
		t.Errorf("success: %+v", r)
	}

	ch = run("S2", 5*time.Second)
	if status, _ := callbackGet(t, "state=WRONG&code=X"); status != 400 {
		t.Errorf("mismatch status = %d", status)
	}
	if r := <-ch; r.err == nil || !strings.Contains(r.err.Error(), "state mismatch") || r.code != "" {
		t.Errorf("a forged state must be rejected, got %+v", r)
	}

	ch = run("S3", 5*time.Second)
	_, body := callbackGet(t, "error=%3Cscript%3Ealert(1)%3C%2Fscript%3E")
	if strings.Contains(body, "<script>") {
		t.Errorf("provider error must be HTML-escaped in the response, got %q", body)
	}
	if r := <-ch; r.err == nil || !strings.Contains(r.err.Error(), "oauth error") {
		t.Errorf("provider error: %+v", r)
	}

	ch = run("S4", 5*time.Second)
	if status, _ := callbackGet(t, "state=S4"); status != 400 {
		t.Errorf("missing code status = %d", status)
	}
	if r := <-ch; r.err == nil || !strings.Contains(r.err.Error(), "no authorization code") {
		t.Errorf("missing code: %+v", r)
	}

	if r := <-run("S5", 100*time.Millisecond); r.err == nil || !strings.Contains(r.err.Error(), "timed out") {
		t.Errorf("timeout: %+v", r)
	}
}
