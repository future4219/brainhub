package model

import (
	"time"

	"brainhub/domain/entity"
)

type BrainWriterClient struct {
	BrainID                string
	WriteSourceID          string
	GBrainClientID         *string
	ClientSecretCiphertext []byte
	State                  string
	StateReason            string
	IssuedAt               *time.Time
	LastVerifiedAt         *time.Time
}

func (c BrainWriterClient) Entity() entity.BrainWriterClient {
	return entity.BrainWriterClient{
		BrainID: c.BrainID, WriteSourceID: entity.SourceID(c.WriteSourceID),
		GBrainClientID: c.GBrainClientID, ClientSecretCiphertext: c.ClientSecretCiphertext,
		State: entity.WriterClientState(c.State), StateReason: c.StateReason,
		IssuedAt: c.IssuedAt, LastVerifiedAt: c.LastVerifiedAt,
	}
}
