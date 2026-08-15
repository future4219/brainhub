package entity

import (
	"time"

	"brainhub/domain/entconst"
)

type Membership struct {
	ID        string
	BrainID   string
	UserID    string
	Role      entconst.Role
	InvitedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
	RevokedAt *time.Time
}

func (m Membership) IsActive() bool {
	return m.RevokedAt == nil
}
