package store

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeon022/missionctl-core/syncdir"
	_ "modernc.org/sqlite"
)

// SQLiteStore ist die konkrete Implementierung des Store-Interfaces mit SQLite
type SQLiteStore struct {
	db     *sql.DB
	path   string
	shared bool
}

// No cross-process file lock: SQLite's own locking (busy timeout below) is enough for several
// local processes, and a TUI left open must never stop the scheduler daemon from publishing.
// Double publishing is prevented per post by TryLockPost.

// NewSQLiteStore erstellt oder öffnet die SQLite-Datenbank und führt Migrationen aus.
// shared must reflect whether dbPath is a user-configured (possibly
// folder-synced) directory rather than postctl's own default/legacy path
// — see config.Shared.
func NewSQLiteStore(dbPath string, shared bool) (*SQLiteStore, error) {
	// :memory: (used by tests) has no on-disk file at all — none of the
	// sync-safety machinery below applies, and appending DSN params or
	// trying to flock a literal ":memory:" path would be actively wrong.
	if dbPath == ":memory:" {
		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			return nil, fmt.Errorf("open sqlite db: %w", err)
		}
		if err := db.Ping(); err != nil {
			db.Close()
			return nil, fmt.Errorf("ping db: %w", err)
		}
		store := &SQLiteStore{db: db}
		if err := store.migrate(); err != nil {
			db.Close()
			return nil, fmt.Errorf("run migrations: %w", err)
		}
		return store, nil
	}

	// Tilde expandieren (z. B. ~/.config/postctl/postctl.db)
	expandedPath, err := expandPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("expand db path: %w", err)
	}

	if isPlaceholder, placeholder := syncdir.ICloudPlaceholder(expandedPath); isPlaceholder {
		return nil, fmt.Errorf("%s hasn't finished downloading from iCloud yet (found %s) — open Finder and download it, or disable \"Optimize Mac Storage\" for this folder", expandedPath, placeholder)
	}

	// Verzeichnis erstellen, falls es nicht existiert
	dbDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	// Verbindung öffnen
	db, err := sql.Open("sqlite", expandedPath+"?_journal="+syncdir.JournalMode(shared)+"&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Verbindung testen
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	store := &SQLiteStore{db: db, path: expandedPath, shared: shared}
	for _, f := range []string{expandedPath, expandedPath + "-wal", expandedPath + "-shm"} {
		_ = os.Chmod(f, 0o600) // posts and encrypted tokens: owner only (no-op if the file does not exist)
	}

	// Migrationen ausführen
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Suspend closes the connection and releases the lock so another process (for example a CLI
// wizard started from the TUI) can use the database; Resume takes it back. Both are no-ops
// for in-memory stores.
func (s *SQLiteStore) Suspend() error {
	if s.path == "" {
		return nil
	}
	return s.db.Close()
}

// Resume reopens the connection after Suspend.
func (s *SQLiteStore) Resume() error {
	if s.path == "" {
		return nil
	}
	db, err := sql.Open("sqlite", s.path+"?_journal="+syncdir.JournalMode(s.shared)+"&_timeout=5000")
	if err == nil {
		err = db.Ping()
	}
	if err != nil {
		return fmt.Errorf("reopen database: %w", err)
	}
	s.db = db
	return nil
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

// GenerateUUID erzeugt eine Version 4 UUID mit crypto/rand
func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
