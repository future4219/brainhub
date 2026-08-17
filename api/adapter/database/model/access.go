package model

import (
	"time"

	"brainhub/domain/entity"
)

type Invitation struct {
	ID         string
	BrainID    string
	Email      *string
	Role       string
	TokenHash  string
	InvitedBy  string
	State      string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AcceptedAt *time.Time
	AcceptedBy *string
}

func (i Invitation) Entity() entity.Invitation {
	return entity.Invitation{
		ID: i.ID, BrainID: i.BrainID, Email: i.Email, Role: entity.Role(i.Role),
		TokenHash: i.TokenHash, InvitedBy: i.InvitedBy, State: entity.InvitationState(i.State),
		ExpiresAt: i.ExpiresAt, CreatedAt: i.CreatedAt, AcceptedAt: i.AcceptedAt, AcceptedBy: i.AcceptedBy,
	}
}

type IssuedClient struct {
	ID             string
	UserID         string
	BrainID        string
	GBrainClientID *string
	Label          string
	WriteSourceID  *string
	ReadSourceIDs  []string
	Scopes         []string
	State          string
	StateReason    string
	IssuedAt       time.Time
	LastVerifiedAt *time.Time
	RevokedAt      *time.Time
}

func (c IssuedClient) Entity() entity.IssuedClient {
	readSourceIDs := make([]entity.SourceID, len(c.ReadSourceIDs))
	for i, sourceID := range c.ReadSourceIDs {
		readSourceIDs[i] = entity.SourceID(sourceID)
	}
	var writeSourceID *entity.SourceID
	if c.WriteSourceID != nil {
		value := entity.SourceID(*c.WriteSourceID)
		writeSourceID = &value
	}
	return entity.IssuedClient{
		ID: c.ID, UserID: c.UserID, BrainID: c.BrainID, GBrainClientID: c.GBrainClientID,
		Label: c.Label, WriteSourceID: writeSourceID, ReadSourceIDs: readSourceIDs,
		Scopes: c.Scopes, State: entity.ClientState(c.State), StateReason: c.StateReason,
		IssuedAt: c.IssuedAt, LastVerifiedAt: c.LastVerifiedAt, RevokedAt: c.RevokedAt,
	}
}
