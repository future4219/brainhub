package entity

import "time"

type Membership struct {
	ID        string
	BrainID   string
	UserID    string
	Role      Role
	InvitedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
	RevokedAt *time.Time
}

func (m Membership) IsActive() bool {
	return m.RevokedAt == nil
}

func (m Membership) CanWrite() bool {
	return m.IsActive() && (m.Role == RoleOwner || m.Role == RoleEditor)
}
