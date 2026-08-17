package model

import (
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
)

type User struct {
	ID        string
	Email     string
	Name      string
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) Entity() entity.User {
	return entity.User{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		State:     entconst.UserState(u.State),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

type AuthIdentity struct {
	ID         string
	UserID     string
	Provider   string
	Identifier string
	SecretHash *string
	CreatedAt  time.Time
}

func (a AuthIdentity) Entity() entity.AuthIdentity {
	return entity.AuthIdentity{
		ID:         a.ID,
		UserID:     a.UserID,
		Provider:   entconst.AuthProvider(a.Provider),
		Identifier: a.Identifier,
		SecretHash: a.SecretHash,
		CreatedAt:  a.CreatedAt,
	}
}

type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

func (s Session) Entity() entity.Session {
	return entity.Session{
		ID:        s.ID,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
		RevokedAt: s.RevokedAt,
	}
}
