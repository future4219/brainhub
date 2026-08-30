package output_port

import (
	"context"
	"time"

	"brainhub/domain/entity"
)

type MCPRepository interface {
	CreateMCPClient(context.Context, entity.MCPClient) error
	FindMCPClientByID(context.Context, string) (entity.MCPClient, error)
	FindActiveMCPClientByUserName(context.Context, string, string) (entity.MCPClient, error)
	CreateMCPAuthorizationCode(context.Context, entity.MCPAuthorizationCode) (string, error)
	ConsumeMCPAuthorizationCode(context.Context, string, string, string, string, time.Time) (entity.MCPAuthorizationCode, error)
	CreateMCPTokenPair(context.Context, string, string, time.Time, time.Time) (string, string, error)
	RotateMCPRefreshToken(context.Context, string, string, time.Time, time.Time, time.Time) (entity.MCPToken, string, string, error)
	VerifyMCPAccessToken(context.Context, string, time.Time) (entity.MCPToken, error)
	RevokeMCPToken(context.Context, string, string, time.Time) error
	ListMCPVisibleBrains(context.Context, string) ([]entity.MCPVisibleBrain, error)
}

type ReaderClientRepository interface {
	CreateReaderClient(context.Context, entity.ReaderClient) error
	FindReaderClientByUser(context.Context, string) (entity.ReaderClient, error)
	ListReaderClients(context.Context) ([]entity.ReaderClient, error)
	ActivateReaderClient(context.Context, string, string, []byte, []entity.SourceID, time.Time) (entity.ReaderClient, error)
	UpdateReaderClientScope(context.Context, string, []entity.SourceID, time.Time) (entity.ReaderClient, error)
	MarkReaderClientOrphan(context.Context, string, *string, string) (entity.ReaderClient, error)
	ResetReaderClient(context.Context, string) error
}
