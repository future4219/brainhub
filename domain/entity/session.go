package entity

import "time"

// Session deliberately does not contain a raw token or token hash.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}
