package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type BrainReader interface {
	AccessToken(context.Context, string, []entity.SourceID) (string, error)
	Reissue(context.Context, string, []entity.SourceID) error
}
