// Package redact strips credentials from text before it is logged or stored: Go's HTTP errors print
// the full request URL, and several platform APIs take tokens as query parameters.
package redact

import "regexp"

var (
	query  = regexp.MustCompile(`(?i)\b(access_token|client_secret|app_secret|fb_exchange_token|refresh_token|token|api_key|password|code)=[^&\s"')]+`)
	bearer = regexp.MustCompile(`(?i)\b(bearer\s+)[A-Za-z0-9._~+/=%-]{12,}`)
)

// Secrets replaces token-like values in s with REDACTED.
func Secrets(s string) string {
	return bearer.ReplaceAllString(query.ReplaceAllString(s, "${1}=REDACTED"), "${1}REDACTED")
}
