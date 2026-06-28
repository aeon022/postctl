package store

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
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

// AES-256-GCM Verschlüsselungs-Hilfsfunktionen

func getEncryptionKey() []byte {
	keyStr := os.Getenv("POSTCTL_ENCRYPTION_KEY")
	if keyStr == "" {
		keyStr = defaultSaltKey
	}
	// Wir erzeugen einen SHA-256-Hash, um immer genau 32 Byte Keylänge für AES-256 zu haben
	hash := sha256.Sum256([]byte(keyStr))
	return hash[:]
}

func encrypt(plaintext string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(cryptoText string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
	if err != nil {
		return "", err
	}

	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
