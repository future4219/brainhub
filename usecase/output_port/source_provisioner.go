package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type SourceProvisioner interface {
	Provision(context.Context, entity.SourceID) error
}
