package constructor

import (
	"errors"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/domain/validation"
)

var ErrInvalidVisibility = errors.New("visibility must be public or private")

func NewBrainCreate(id string, sourceID entity.SourceID, name, description string, visibility entconst.Visibility, ownerID string, now time.Time) (entity.Brain, error) {
	if err := validation.ValidateBrainName(name); err != nil {
		return entity.Brain{}, err
	}
	if err := validation.ValidateBrainDescription(description); err != nil {
		return entity.Brain{}, err
	}
	if visibility == "" {
		visibility = entconst.VisibilityPrivate
	}
	if visibility != entconst.VisibilityPrivate && visibility != entconst.VisibilityPublic {
		return entity.Brain{}, ErrInvalidVisibility
	}
	return entity.Brain{
		ID:          id,
		SourceID:    sourceID,
		Name:        name,
		Description: description,
		Visibility:  visibility,
		OwnerID:     ownerID,
		State:       entconst.BrainStateProvisioning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func NewBrainAdopt(id string, sourceID entity.SourceID, name, description string, visibility entconst.Visibility, ownerID string, now time.Time) (entity.Brain, error) {
	brain, err := NewBrainCreate(id, sourceID, name, description, visibility, ownerID, now)
	if err != nil {
		return entity.Brain{}, err
	}
	brain.State = entconst.BrainStateReady
	return brain, nil
}

func NewOwnerMembership(id, brainID, userID string, now time.Time) entity.Membership {
	return entity.Membership{
		ID:        id,
		BrainID:   brainID,
		UserID:    userID,
		Role:      entconst.RoleOwner,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
