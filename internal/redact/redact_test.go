package redact

import (
	"strings"
	"testing"
)

func TestSecrets(t *testing.T) {
	in := `Get "https://graph.facebook.com/v19.0/me?fields=id&access_token=EAAB123secret&x=1": dial tcp: timeout; Authorization: Bearer abcdefghijklmnop123456`
	out := Secrets(in)
	for _, leak := range []string{"EAAB123secret", "abcdefghijklmnop123456"} {
		if strings.Contains(out, leak) {
			t.Fatalf("secret %q leaked: %s", leak, out)
		}
	}
	if !strings.Contains(out, "fields=id") || !strings.Contains(out, "x=1") {
		t.Fatalf("harmless parameters must stay: %s", out)
	}
}
