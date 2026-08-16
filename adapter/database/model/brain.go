package model

import (
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
)

type Brain struct {
	ID          string
	SourceID    string
	Name        string
	Description string
	Visibility  string
	OwnerID     string
	State       string
	StateReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
}

func (b Brain) Entity() entity.Brain {
	return entity.Brain{
		ID:          b.ID,
		SourceID:    entity.SourceID(b.SourceID),
		Name:        b.Name,
		Description: b.Description,
		Visibility:  entconst.Visibility(b.Visibility),
		OwnerID:     b.OwnerID,
		State:       entconst.BrainState(b.State),
		StateReason: b.StateReason,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
		ArchivedAt:  b.ArchivedAt,
	}
}

type Membership struct {
	ID        string
	BrainID   string
	UserID    string
	Role      string
	InvitedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
	RevokedAt *time.Time
}

func (m Membership) Entity() entity.Membership {
	return entity.Membership{
		ID:        m.ID,
		BrainID:   m.BrainID,
		UserID:    m.UserID,
		Role:      entity.Role(m.Role),
		InvitedBy: m.InvitedBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		RevokedAt: m.RevokedAt,
	}
}
