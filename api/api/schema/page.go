package schema

import (
	"time"

	"brainhub/domain/entity"
)

type PageResponse struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Type      string    `json:"type"`
	UpdatedAt time.Time `json:"updated_at"`
}

func PageResponseFromEntity(page entity.Page) PageResponse {
	return PageResponse{
		Slug:      page.Slug,
		Title:     page.Title,
		Type:      page.Type,
		UpdatedAt: page.UpdatedAt,
	}
}
