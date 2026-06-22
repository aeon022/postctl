package models

import "time"

// AnalyticsData hält die Social-Media-Engagement-Kennzahlen eines Posts
type AnalyticsData struct {
	PlatformID  string    `json:"platform_id"`
	Likes       int       `json:"likes"`
	Shares      int       `json:"shares"`      // Reposts / Retweets / Reshares
	Comments    int       `json:"comments"`
	Impressions int       `json:"impressions"` // Views / Reach
	FetchedAt   time.Time `json:"fetched_at"`
}
