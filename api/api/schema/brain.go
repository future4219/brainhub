package schema

import (
	"time"

	"brainhub/domain/entity"
)

type CreateBrainRequest struct {
	SourceID    string `json:"source_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type AdoptBrainRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type BrainResponse struct {
	ID          string     `json:"id"`
	SourceID    string     `json:"source_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Visibility  string     `json:"visibility"`
	OwnerID     string     `json:"owner_id"`
	State       string     `json:"state"`
	StateReason string     `json:"state_reason"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at"`
}

type BrainErrorResponse struct {
	Error string        `json:"error"`
	Brain BrainResponse `json:"brain"`
}

func BrainResponseFromEntity(brain entity.Brain) BrainResponse {
	return BrainResponse{
		ID:          brain.ID,
		SourceID:    brain.SourceID.String(),
		Name:        brain.Name,
		Description: brain.Description,
		Visibility:  string(brain.Visibility),
		OwnerID:     brain.OwnerID,
		State:       string(brain.State),
		StateReason: brain.StateReason,
		CreatedAt:   brain.CreatedAt,
		UpdatedAt:   brain.UpdatedAt,
		ArchivedAt:  brain.ArchivedAt,
	}
}
