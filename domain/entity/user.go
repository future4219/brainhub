package entity

import (
	"time"

	"brainhub/domain/entconst"
)

type User struct {
	ID        string
	Email     string
	Name      string
	State     entconst.UserState
	CreatedAt time.Time
	UpdatedAt time.Time
}
