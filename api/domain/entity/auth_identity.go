package entity

import (
	"time"

	"brainhub/domain/entconst"
)

type AuthIdentity struct {
	ID         string
	UserID     string
	Provider   entconst.AuthProvider
	Identifier string
	SecretHash *string
	CreatedAt  time.Time
}
