package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type UserRepository interface {
	CreateUser(context.Context, entity.User) error
	FindUserByID(context.Context, string) (entity.User, error)
}
