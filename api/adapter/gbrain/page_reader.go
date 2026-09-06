package gbrain

import (
	"context"

	"brainhub/domain/entity"
)

func (s *WriterService) writerClient(ctx context.Context, brainID string, sourceID entity.SourceID) (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := brainID + ":" + sourceID.String()
	if client := s.clients[key]; client != nil {
		return client, nil
	}
	clientID, secret, err := s.credentialsFor(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	client, err := newClient(s.baseURL, clientID, secret, "read write")
	if err == nil {
		s.clients[key] = client
	}
	return client, err
}

func (s *WriterService) List(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.Page, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	return client.List(ctx, sourceID)
}

func (s *WriterService) Get(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	return client.Get(ctx, sourceID, slug)
}
