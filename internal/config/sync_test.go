package config

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestBackupCryptoRoundTripAndLegacy(t *testing.T) {
	plain := []byte(`{"secret":"x"}`)
	enc, err := encrypt(plain, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(enc, backupMagic) {
		t.Fatal("new backups carry the v2 header")
	}
	if got, err := decrypt(enc, "pw"); err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if _, err := decrypt(enc, "wrong"); err == nil {
		t.Fatal("wrong password must fail")
	}
	// a backup made by the old version (key = SHA-256(password), no header) still restores
	sum := sha256.Sum256([]byte("pw"))
	gcm, _ := gcmFor(sum[:])
	nonce := bytes.Repeat([]byte{1}, gcm.NonceSize())
	legacy := gcm.Seal(nonce, nonce, plain, nil)
	if got, err := decrypt(legacy, "pw"); err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("legacy restore: %q %v", got, err)
	}
}
