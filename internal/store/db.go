package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/models"
	_ "modernc.org/sqlite"
)

// Store definiert die Schnittstelle für die Datenhaltung
type Store interface {
	// Posts
	SavePost(ctx context.Context, post *models.Post) error
	GetPost(ctx context.Context, id string) (*models.Post, error)
	ListPosts(ctx context.Context, platform, status, campaign string) ([]models.Post, error)
	DeletePost(ctx context.Context, id string) error

	// History
	AddHistoryEntry(ctx context.Context, entry *models.HistoryEntry) error
	GetHistory(ctx context.Context, limit int) ([]models.HistoryEntry, error)

	// Auth Tokens
	SaveToken(ctx context.Context, platform, token, refresh string, expiresAt *time.Time) error
	GetToken(ctx context.Context, platform string) (token, refresh string, expiresAt *time.Time, err error)
	DeleteToken(ctx context.Context, platform string) error

	// Close schließt die Datenbankverbindung
	Close() error
}

// SQLiteStore ist die konkrete Implementierung des Store-Interfaces mit SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore erstellt oder öffnet die SQLite-Datenbank und führt Migrationen aus
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Tilde expandieren (z. B. ~/.config/postctl/postctl.db)
	expandedPath, err := expandPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("expand db path: %w", err)
	}

	// Verzeichnis erstellen, falls es nicht existiert
	dbDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// Verbindung öffnen
	db, err := sql.Open("sqlite", expandedPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Verbindung testen
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Migrationen ausführen
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close schließt die Datenbankverbindung
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// expandPath expandiert den Home-Directory-Tilde-Pfad
func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return filepath.Clean(path), nil
}

// migrate erstellt die Tabellen, falls diese noch nicht existieren
func (s *SQLiteStore) migrate() error {
	queries := []string{
		`PRAGMA foreign_keys = ON;`,
		
		`CREATE TABLE IF NOT EXISTS posts (
			id          TEXT PRIMARY KEY,
			platform    TEXT NOT NULL,
			type        TEXT NOT NULL,
			language    TEXT DEFAULT 'en',
			campaign    TEXT,
			title       TEXT,
			content     TEXT NOT NULL,     -- JSON bei Threads, sonst Plaintext
			images      TEXT,              -- JSON-Array von Pfaden
			status      TEXT DEFAULT 'draft',
			scheduled_at TEXT,
			posted_at   TEXT,
			platform_id TEXT,
			error       TEXT,
			source_file TEXT,
			created_at  TEXT DEFAULT (datetime('now')),
			updated_at  TEXT DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS history (
			id          TEXT PRIMARY KEY,
			post_id     TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			action      TEXT NOT NULL,
			platform_id TEXT,
			error       TEXT,
			created_at  TEXT DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS auth_tokens (
			platform    TEXT PRIMARY KEY,
			token       TEXT NOT NULL,
			refresh     TEXT,
			expires_at  TEXT
		);`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("exec query: %w\nQuery: %s", err, query)
		}
	}

	return nil
}
