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
	Tags          []string
	SupersededBy  *string
	Frontmatter   map[string]any
}

type PageType struct {
	Name      string
	Primitive string
}

type PageWrite struct {
	Slug          string
	Title         string
	Type          string
	Tags          []string
	SupersededBy  *string
	CompiledTruth string
	Timeline      string
	Frontmatter   map[string]any
}
