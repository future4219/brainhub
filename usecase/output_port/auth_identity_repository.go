package output_port

import (
	"context"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
)

type AuthIdentityRepository interface {
	CreateAuthIdentity(context.Context, entity.AuthIdentity) error
	FindAuthIdentity(context.Context, entconst.AuthProvider, string) (entity.AuthIdentity, error)
}
