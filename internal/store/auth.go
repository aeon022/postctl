package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
)

// Default-Key für die AES-Verschlüsselung, falls kein ENV-Key gesetzt ist
const defaultSaltKey = "postctl-system-encryption-salt-key-2026"

// SaveToken verschlüsselt und speichert die OAuth-Tokens für eine Plattform
func (s *SQLiteStore) SaveToken(ctx context.Context, platform, token, refresh string, expiresAt *time.Time) error {
	// Pro-Feature Check: Maximale Plattform-Anzahl in Core auf 2 limitieren
	if !config.IsPro() {
		var exists int
		err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth_tokens WHERE platform = ?", platform).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check platform existence: %w", err)
		}
		if exists == 0 {
			var count int
			err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth_tokens").Scan(&count)
			if err != nil {
				return fmt.Errorf("count active platforms: %w", err)
			}
			if count >= 2 {
				return fmt.Errorf("💡 Pro Feature: Mit der kostenlosen Core-Version von postctl kannst du bis zu 2 Social-Media-Accounts gleichzeitig verbinden. Um unbegrenzt viele Plattformen zu nutzen, aktiviere bitte postctl Pro mit einem Lizenzschlüssel.")
			}
		}
	}

	if KeySource() == "legacy" {
		var n int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth_tokens").Scan(&n); err == nil && n == 0 {
			_ = EnsureKeyFile() // fresh install: start with a random per-installation key
		}
	}

	encToken, err := encrypt(token)
	if err != nil {
		return fmt.Errorf("encrypt token: %w", err)
	}

	var encRefresh sql.NullString
	if refresh != "" {
		encryptedRef, err := encrypt(refresh)
		if err != nil {
			return fmt.Errorf("encrypt refresh token: %w", err)
		}
		encRefresh = sql.NullString{String: encryptedRef, Valid: true}
	}

	expiresStr := timeToNullString(expiresAt)

	query := `
		INSERT INTO auth_tokens (platform, token, refresh, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(platform) DO UPDATE SET
			token = excluded.token,
			refresh = excluded.refresh,
			expires_at = excluded.expires_at
	`

	_, err = s.db.ExecContext(ctx, query, platform, encToken, encRefresh, expiresStr)
	if err != nil {
		return fmt.Errorf("execute save token query: %w", err)
	}

	return nil
}

// GetToken holt und entschlüsselt die OAuth-Tokens für eine Plattform
func (s *SQLiteStore) GetToken(ctx context.Context, platform string) (token, refresh string, expiresAt *time.Time, err error) {
	query := `
		SELECT token, refresh, expires_at
		FROM auth_tokens
		WHERE platform = ?
	`

	var encToken string
	var encRefresh sql.NullString
	var expiresStr sql.NullString

	err = s.db.QueryRowContext(ctx, query, platform).Scan(&encToken, &encRefresh, &expiresStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", nil, fmt.Errorf("token not found for platform %s: %w", platform, err)
		}
		return "", "", nil, fmt.Errorf("query token: %w", err)
	}

	token, err = decrypt(encToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("decrypt token: %w", err)
	}

	if encRefresh.Valid && encRefresh.String != "" {
		refresh, err = decrypt(encRefresh.String)
		if err != nil {
			return "", "", nil, fmt.Errorf("decrypt refresh token: %w", err)
		}
	}

	expiresAt = nullStringToTime(expiresStr)
	return token, refresh, expiresAt, nil
}

// DeleteToken löscht die Tokens einer Plattform aus der Datenbank
func (s *SQLiteStore) DeleteToken(ctx context.Context, platform string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE platform = ?", platform)
	if err != nil {
		return fmt.Errorf("execute delete token query: %w", err)
	}
	return nil
}

// AES-256-GCM token encryption.
//
// Key order: POSTCTL_ENCRYPTION_KEY, then the per-installation key file (~/.config/postctl/encryption.key,
// random, 0600, never inside a synced data_dir), then — for installations that predate the key file —
// the old built-in default. Fresh installs get a key file automatically; existing ones switch with
// `postctl security migrate` (and must copy the key file to their other devices).

func keyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "postctl", "encryption.key")
}

func hashKey(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func readKeyFile() []byte {
	b, err := os.ReadFile(keyFilePath())
	if err != nil || len(strings.TrimSpace(string(b))) < 32 {
		return nil
	}
	return hashKey(strings.TrimSpace(string(b)))
}

// KeySource reports which key currently encrypts new tokens: "env", "keyfile" or "legacy".
func KeySource() string {
	switch {
	case os.Getenv("POSTCTL_ENCRYPTION_KEY") != "":
		return "env"
	case readKeyFile() != nil:
		return "keyfile"
	}
	return "legacy"
}

// KeyFilePath is where the per-installation key lives.
func KeyFilePath() string { return keyFilePath() }

// EnsureKeyFile creates the per-installation key if it does not exist yet.
func EnsureKeyFile() error {
	p := keyFilePath()
	if p == "" || readKeyFile() != nil {
		return nil
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(hex.EncodeToString(raw)+"\n"), 0o600)
}

func activeKey() []byte {
	if env := os.Getenv("POSTCTL_ENCRYPTION_KEY"); env != "" {
		return hashKey(env)
	}
	if k := readKeyFile(); k != nil {
		return k
	}
	return hashKey(defaultSaltKey)
}

// candidateKeys are tried in order when decrypting, so tokens written under an older scheme still open.
func candidateKeys() [][]byte {
	keys := [][]byte{activeKey()}
	for _, k := range [][]byte{readKeyFile(), hashKey(defaultSaltKey)} {
		if k == nil {
			continue
		}
		dup := false
		for _, e := range keys {
			if string(e) == string(k) {
				dup = true
			}
		}
		if !dup {
			keys = append(keys, k)
		}
	}
	return keys
}

func encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(activeKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

func decrypt(cryptoText string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}
	var last error = errors.New("no key")
	for _, key := range candidateKeys() {
		block, err := aes.NewCipher(key)
		if err != nil {
			last = err
			continue
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			last = err
			continue
		}
		n := gcm.NonceSize()
		if len(ciphertext) < n {
			return "", errors.New("ciphertext too short")
		}
		if plain, err := gcm.Open(nil, ciphertext[:n], ciphertext[n:], nil); err == nil {
			return string(plain), nil
		} else {
			last = err
		}
	}
	return "", fmt.Errorf("cannot decrypt (wrong or missing encryption key — copy ~/.config/postctl/encryption.key from your other device): %w", last)
}

// MigrateTokenKey creates the key file (if needed) and re-encrypts every stored token with it.
func (s *SQLiteStore) MigrateTokenKey(ctx context.Context) (int, error) {
	if err := EnsureKeyFile(); err != nil {
		return 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT platform, token, refresh FROM auth_tokens")
	if err != nil {
		return 0, err
	}
	type row struct {
		platform, token string
		refresh         sql.NullString
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.platform, &r.token, &r.refresh); err != nil {
			rows.Close()
			return 0, err
		}
		all = append(all, r)
	}
	rows.Close()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, r := range all {
		tok, err := decrypt(r.token)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", r.platform, err)
		}
		encTok, err := encrypt(tok)
		if err != nil {
			return 0, err
		}
		encRef := r.refresh
		if r.refresh.Valid && r.refresh.String != "" {
			ref, err := decrypt(r.refresh.String)
			if err != nil {
				return 0, fmt.Errorf("%s refresh: %w", r.platform, err)
			}
			e, err := encrypt(ref)
			if err != nil {
				return 0, err
			}
			encRef = sql.NullString{String: e, Valid: true}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE auth_tokens SET token = ?, refresh = ? WHERE platform = ?", encTok, encRef, r.platform); err != nil {
			return 0, err
		}
	}
	return len(all), tx.Commit()
}
