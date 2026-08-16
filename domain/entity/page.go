package entity

import "time"

type Page struct {
	Slug      string
	Title     string
	Type      string
	UpdatedAt time.Time
}

type PageDetail struct {
	Page
	CompiledTruth string
	Timeline      string
}
