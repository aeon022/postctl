package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

// SavePost speichert einen Post (Insert oder Update via ON CONFLICT)
func (s *SQLiteStore) SavePost(ctx context.Context, post *models.Post) error {
	var content string
	if post.Type == "thread" {
		b, err := json.Marshal(post.Tweets)
		if err != nil {
			return fmt.Errorf("marshal tweets: %w", err)
		}
		content = string(b)
	} else {
		content = post.Body
	}

	var imagesStr string
	if len(post.Images) > 0 {
		b, err := json.Marshal(post.Images)
		if err != nil {
			return fmt.Errorf("marshal images: %w", err)
		}
		imagesStr = string(b)
	}

	scheduledStr := timeToNullString(post.ScheduledAt)
	postedStr := timeToNullString(post.PostedAt)

	// Falls CreatedAt/UpdatedAt nicht gesetzt sind, jetzt setzen
	now := time.Now()
	if post.CreatedAt.IsZero() {
		post.CreatedAt = now
	}
	post.UpdatedAt = now

	query := `
		INSERT INTO posts (
			id, platform, type, language, campaign, title, content, images, status,
			scheduled_at, posted_at, platform_id, error, source_file, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			platform = excluded.platform,
			type = excluded.type,
			language = excluded.language,
			campaign = excluded.campaign,
			title = excluded.title,
			content = excluded.content,
			images = excluded.images,
			status = excluded.status,
			scheduled_at = excluded.scheduled_at,
			posted_at = excluded.posted_at,
			platform_id = excluded.platform_id,
			error = excluded.error,
			source_file = excluded.source_file,
			updated_at = excluded.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		post.ID,
		post.Platform,
		post.Type,
		post.Language,
		post.Campaign,
		post.Title,
		content,
		imagesStr,
		post.Status,
		scheduledStr,
		postedStr,
		post.PlatformID,
		post.Error,
		post.SourceFile,
		post.CreatedAt.Format(time.RFC3339),
		post.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("execute save post query: %w", err)
	}

	return nil
}

// GetPost holt einen einzelnen Post anhand seiner ID
func (s *SQLiteStore) GetPost(ctx context.Context, id string) (*models.Post, error) {
	query := `
		SELECT id, platform, type, language, campaign, title, content, images, status,
		       scheduled_at, posted_at, platform_id, error, source_file, created_at, updated_at
		FROM posts
		WHERE id = ?
	`

	var post models.Post
	var content string
	var imagesStr sql.NullString
	var scheduledStr sql.NullString
	var postedStr sql.NullString
	var createdAtStr string
	var updatedAtStr string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.Platform,
		&post.Type,
		&post.Language,
		&post.Campaign,
		&post.Title,
		&content,
		&imagesStr,
		&post.Status,
		&scheduledStr,
		&postedStr,
		&post.PlatformID,
		&post.Error,
		&post.SourceFile,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	// Zeiten parsen
	post.ScheduledAt = nullStringToTime(scheduledStr)
	post.PostedAt = nullStringToTime(postedStr)
	post.CreatedAt = parseTime(createdAtStr)
	post.UpdatedAt = parseTime(updatedAtStr)

	// Content unmarshaln
	if post.Type == "thread" {
		if err := json.Unmarshal([]byte(content), &post.Tweets); err != nil {
			return nil, fmt.Errorf("unmarshal tweets: %w", err)
		}
	} else {
		post.Body = content
	}

	// Bilder unmarshaln
	if imagesStr.Valid && imagesStr.String != "" {
		if err := json.Unmarshal([]byte(imagesStr.String), &post.Images); err != nil {
			return nil, fmt.Errorf("unmarshal images: %w", err)
		}
	}

	return &post, nil
}

// ListPosts listet alle Posts auf, optional gefiltert nach platform, status und campaign
func (s *SQLiteStore) ListPosts(ctx context.Context, platform, status, campaign string) ([]models.Post, error) {
	query := `
		SELECT id, platform, type, language, campaign, title, content, images, status,
		       scheduled_at, posted_at, platform_id, error, source_file, created_at, updated_at
		FROM posts
	`
	var conditions []string
	var args []interface{}

	if platform != "" && platform != "all" {
		conditions = append(conditions, "platform = ?")
		args = append(args, platform)
	}
	if status != "" && status != "all" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if campaign != "" {
		conditions = append(conditions, "campaign = ?")
		args = append(args, campaign)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query list posts: %w", err)
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var content string
		var imagesStr sql.NullString
		var scheduledStr sql.NullString
		var postedStr sql.NullString
		var createdAtStr string
		var updatedAtStr string

		err := rows.Scan(
			&post.ID,
			&post.Platform,
			&post.Type,
			&post.Language,
			&post.Campaign,
			&post.Title,
			&content,
			&imagesStr,
			&post.Status,
			&scheduledStr,
			&postedStr,
			&post.PlatformID,
			&post.Error,
			&post.SourceFile,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("scan post row: %w", err)
		}

		post.ScheduledAt = nullStringToTime(scheduledStr)
		post.PostedAt = nullStringToTime(postedStr)
		post.CreatedAt = parseTime(createdAtStr)
		post.UpdatedAt = parseTime(updatedAtStr)

		if post.Type == "thread" {
			if err := json.Unmarshal([]byte(content), &post.Tweets); err != nil {
				return nil, fmt.Errorf("unmarshal tweets on list: %w", err)
			}
		} else {
			post.Body = content
		}

		if imagesStr.Valid && imagesStr.String != "" {
			if err := json.Unmarshal([]byte(imagesStr.String), &post.Images); err != nil {
				return nil, fmt.Errorf("unmarshal images on list: %w", err)
			}
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return posts, nil
}

// DeletePost löscht einen Post aus der Datenbank (kaskadiert die History)
func (s *SQLiteStore) DeletePost(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("execute delete query: %w", err)
	}
	return nil
}

// Hilfsfunktionen für Typ-Konvertierungen zwischen DB und Modell

func timeToNullString(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{
		String: t.Format(time.RFC3339),
		Valid:  true,
	}
}

func nullStringToTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, ns.String)
	if err != nil {
		// Fallback, falls Datumsformat abweicht
		t, err = time.Parse("2006-01-02 15:04:05", ns.String)
		if err != nil {
			return nil
		}
	}
	return &t
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
