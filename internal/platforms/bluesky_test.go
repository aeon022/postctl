package platforms

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIsJWTExpired(t *testing.T) {
	createTestToken := func(exp time.Time, rawUrlEncode bool) string {
		header := `{"alg":"HS256","typ":"JWT"}`
		headerB64 := base64.RawURLEncoding.EncodeToString([]byte(header))

		claims := struct {
			Exp int64 `json:"exp"`
		}{
			Exp: exp.Unix(),
		}
		claimsBytes, _ := json.Marshal(claims)
		
		var claimsB64 string
		if rawUrlEncode {
			claimsB64 = base64.RawURLEncoding.EncodeToString(claimsBytes)
		} else {
			// standard padded url encoding
			claimsB64 = base64.URLEncoding.EncodeToString(claimsBytes)
		}

		signature := "dummy-signature"
		signatureB64 := base64.RawURLEncoding.EncodeToString([]byte(signature))

		return headerB64 + "." + claimsB64 + "." + signatureB64
	}

	tests := []struct {
		name     string
		token    string
		expected bool
	}{
		{
			name:     "Invalid token structure",
			token:    "invalid-token",
			expected: true,
		},
		{
			name:     "Invalid base64 payload",
			token:    "abc.invalid_base64_payload.xyz",
			expected: true,
		},
		{
			name:     "Invalid JSON payload",
			token:    "abc." + base64.RawURLEncoding.EncodeToString([]byte("{invalid-json")) + ".xyz",
			expected: true,
		},
		{
			name:     "Unexpired Token (Raw URLEncode)",
			token:    createTestToken(time.Now().Add(1*time.Hour), true),
			expected: false,
		},
		{
			name:     "Unexpired Token (Padded URLEncode)",
			token:    createTestToken(time.Now().Add(1*time.Hour), false),
			expected: false,
		},
		{
			name:     "Expired Token (Raw URLEncode)",
			token:    createTestToken(time.Now().Add(-1*time.Hour), true),
			expected: true,
		},
		{
			name:     "Expired Token (Padded URLEncode)",
			token:    createTestToken(time.Now().Add(-1*time.Hour), false),
			expected: true,
		},
		{
			name:     "Token near expiration (less than 10 seconds)",
			token:    createTestToken(time.Now().Add(5*time.Second), true),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJWTExpired(tt.token)
			// Remove any padding checks if needed or verify
			if got != tt.expected {
				// Let's print the token payload details for debugging if it fails
				parts := strings.Split(tt.token, ".")
				if len(parts) == 3 {
					raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
					t.Errorf("isJWTExpired(%s) = %v; want %v. Decoded payload: %s", tt.name, got, tt.expected, string(raw))
				} else {
					t.Errorf("isJWTExpired(%s) = %v; want %v", tt.name, got, tt.expected)
				}
			}
		})
	}
}
