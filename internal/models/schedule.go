package models

import "time"

// ScheduledPost repräsentiert einen geplanten Post für den Scheduler
type ScheduledPost struct {
	PostID      string    `json:"post_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}
