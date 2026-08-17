package entity

import (
	"time"

	"brainhub/domain/entconst"
)

type Brain struct {
	ID          string
	SourceID    SourceID
	Name        string
	Description string
	Visibility  entconst.Visibility
	OwnerID     string
	State       entconst.BrainState
	StateReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

func (b Brain) IsReadable() bool {
	return b.State == entconst.BrainStateReady
}

func (b Brain) IsArchived() bool {
	return b.ArchivedAt != nil
}
