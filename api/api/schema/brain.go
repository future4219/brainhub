package schema

import (
	"time"

	"brainhub/domain/entity"
)

type BrainListResponse struct {
	Sources []BrainResponse `json:"sources"`
}

type BrainResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	LocalPath  *string    `json:"local_path"`
	RemoteURL  *string    `json:"remote_url"`
	Federated  bool       `json:"federated"`
	PageCount  int        `json:"page_count"`
	LastSyncAt *time.Time `json:"last_sync_at"`
}

func BrainListResponseFromEntities(sources []entity.Source) BrainListResponse {
	response := make([]BrainResponse, len(sources))
	for i, source := range sources {
		response[i] = BrainResponse{
			ID:         source.ID,
			Name:       source.Name,
			LocalPath:  source.LocalPath,
			RemoteURL:  source.RemoteURL,
			Federated:  source.Federated,
			PageCount:  source.PageCount,
			LastSyncAt: source.LastSyncAt,
		}
	}
	return BrainListResponse{Sources: response}
}
