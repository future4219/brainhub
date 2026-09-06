package gbrain

import (
	"context"

	"brainhub/domain/entity"
)

func (s *SourceAccessService) List(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.Page, error) {
	client, err := s.sourceClient(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	return client.List(ctx, sourceID)
}

func (s *SourceAccessService) Get(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error) {
	client, err := s.sourceClient(ctx, brainID, sourceID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	return client.Get(ctx, sourceID, slug)
}
