package entity

import "time"

// Page omits Body from docs/domain.md because the current entrypoint does not use it.
type Page struct {
	Slug      string
	Title     string
	Type      string
	UpdatedAt time.Time
}
