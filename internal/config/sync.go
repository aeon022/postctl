package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupPackage kapselt alle Daten, die für einen Transfer auf andere Geräte nötig sind
type BackupPackage struct {
	ConfigYAML []byte `json:"config_yaml"`
	Database   []byte `json:"database"`
	CreatedAt  int64  `json:"created_at"`
}

// ExportConfig verpackt die Konfiguration und die SQLite-Datenbank,
// verschlüsselt das Paket mit AES-256-GCM und speichert es ab.
func ExportConfig(password string, outputPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	configPath := filepath.Join(home, ".config", "postctl", "config.yaml")
	dbPath := GetDBPath()

	// 1. Dateien einlesen
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	// Datenbank ist optional, falls das Tool noch nie gestartet wurde
	var dbBytes []byte
	if _, err := os.Stat(dbPath); err == nil {
		dbBytes, err = os.ReadFile(dbPath)
		if err != nil {
			return fmt.Errorf("read database file: %w", err)
		}
	}

	// 2. Paket erstellen
	pkg := BackupPackage{
		ConfigYAML: configBytes,
		Database:   dbBytes,
		CreatedAt:  time.Now().Unix(),
	}

	// 3. Serialisieren
	pkgBytes, err := json.Marshal(pkg)
	if err != nil {
		return fmt.Errorf("serialize backup package: %w", err)
	}

	// 4. Verschlüsseln
	encryptedBytes, err := encrypt(pkgBytes, password)
	if err != nil {
		return fmt.Errorf("encrypt backup package: %w", err)
	}

	// 5. In Ausgabedatei schreiben
	if err := os.WriteFile(outputPath, encryptedBytes, 0600); err != nil {
		return fmt.Errorf("write encrypted backup file: %w", err)
	}

	return nil
}

// ImportConfig entschlüsselt das Paket, stellt die config.yaml sowie die
// postctl.db wieder an den korrekten Stellen her und lädt die Konfiguration neu.
func ImportConfig(password string, inputPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	// 1. Datei einlesen
	encryptedBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read encrypted backup file: %w", err)
	}

	// 2. Entschlüsseln
	pkgBytes, err := decrypt(encryptedBytes, password)
	if err != nil {
		return fmt.Errorf("decrypt backup package (falsches Passwort?): %w", err)
	}

	// 3. Deserialisieren
	var pkg BackupPackage
	if err := json.Unmarshal(pkgBytes, &pkg); err != nil {
		return fmt.Errorf("deserialize backup package: %w", err)
	}

	// 4. Verzeichnisse sicherstellen
	configDir := filepath.Join(home, ".config", "postctl")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// 5. config.yaml wiederherstellen
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, pkg.ConfigYAML, 0644); err != nil {
		return fmt.Errorf("restore config file: %w", err)
	}

	// 6. postctl.db wiederherstellen
	if len(pkg.Database) > 0 {
		dbPath := GetDBPath()
		// Sicherstellen, dass das Datenbankverzeichnis existiert
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return fmt.Errorf("create database dir: %w", err)
		}
		if err := os.WriteFile(dbPath, pkg.Database, 0644); err != nil {
			return fmt.Errorf("restore database file: %w", err)
		}
	}

	// 7. Konfiguration in der laufenden Instanz neu laden
	return LoadConfig()
}

// encrypt verschlüsselt Daten mit AES-256-GCM (Schlüssel abgeleitet über SHA-256 des Passworts)
func encrypt(plaintext []byte, password string) ([]byte, error) {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	key := hasher.Sum(nil)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Ciphertext erzeugen, Nonce wird vorne angehängt
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt entschlüsselt Daten mit AES-256-GCM
func decrypt(ciphertext []byte, password string) ([]byte, error) {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	key := hasher.Sum(nil)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm decryption failed")
	}

	return plaintext, nil
}
