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

type PageDetailResponse struct {
	PageResponse
	CompiledTruth string `json:"compiled_truth"`
	Timeline      string `json:"timeline"`
}

func PageResponseFromEntity(page entity.Page) PageResponse {
	return PageResponse{
		Slug:      page.Slug,
		Title:     page.Title,
		Type:      page.Type,
		UpdatedAt: page.UpdatedAt,
	}
}

func PageDetailResponseFromEntity(page entity.PageDetail) PageDetailResponse {
	return PageDetailResponse{
		PageResponse:  PageResponseFromEntity(page.Page),
		CompiledTruth: page.CompiledTruth,
		Timeline:      page.Timeline,
	}
}
