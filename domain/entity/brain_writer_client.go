package entity

import "time"

type WriterClientState string

const (
	WriterClientStateIssuing WriterClientState = "issuing"
	WriterClientStateActive  WriterClientState = "active"
	WriterClientStateOrphan  WriterClientState = "orphan"
)

type BrainWriterClient struct {
	BrainID                string
	WriteSourceID          SourceID
	GBrainClientID         *string
	ClientSecretCiphertext []byte
	State                  WriterClientState
	StateReason            string
	IssuedAt               *time.Time
	LastVerifiedAt         *time.Time
}
