package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

// AddHistoryEntry fügt einen neuen Eintrag in die Historie ein
func (s *SQLiteStore) AddHistoryEntry(ctx context.Context, entry *models.HistoryEntry) error {
	if entry.ID == "" {
		entry.ID = GenerateUUID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	var errorStr sql.NullString
	if entry.Error != "" {
		errorStr = sql.NullString{String: entry.Error, Valid: true}
	}

	var platformID sql.NullString
	if entry.PlatformID != "" {
		platformID = sql.NullString{String: entry.PlatformID, Valid: true}
	}

	query := `
		INSERT INTO history (id, post_id, action, platform_id, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		entry.ID,
		entry.PostID,
		entry.Action,
		platformID,
		errorStr,
		entry.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("execute add history query: %w", err)
	}

	return nil
}

// GetHistory liest die Historien-Einträge absteigend sortiert nach Erstellungszeitpunkt aus
func (s *SQLiteStore) GetHistory(ctx context.Context, limit int) ([]models.HistoryEntry, error) {
	query := `
		SELECT id, post_id, action, platform_id, error, created_at
		FROM history
		ORDER BY created_at DESC
	`
	var args []interface{}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []models.HistoryEntry
	for rows.Next() {
		var entry models.HistoryEntry
		var platformID sql.NullString
		var errorStr sql.NullString
		var createdAtStr string

		err := rows.Scan(
			&entry.ID,
			&entry.PostID,
			&entry.Action,
			&platformID,
			&errorStr,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan history row: %w", err)
		}

		if platformID.Valid {
			entry.PlatformID = platformID.String
		}
		if errorStr.Valid {
			entry.Error = errorStr.String
		}
		entry.CreatedAt = parseTime(createdAtStr)

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history rows iteration error: %w", err)
	}

	return entries, nil
}
