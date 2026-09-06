package schema

import (
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type MCPClientResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MCPVisibleBrainResponse struct {
	CanWrite bool   `json:"can_write"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Role     string `json:"role"`
}

type MCPReaderResponse struct {
	State       string `json:"state"`
	StateReason string `json:"state_reason"`
}

type MCPCLITokenResponse struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateMCPCLITokenRequest struct {
	Label string `json:"label"`
}

type CreatedMCPCLITokenResponse struct {
	MCPCLITokenResponse
	Token string `json:"token"`
}

type MCPConnectionResponse struct {
	Client        *MCPClientResponse        `json:"client"`
	CodexClient   *MCPClientResponse        `json:"codex_client"`
	CLITokens     []MCPCLITokenResponse     `json:"cli_tokens"`
	VisibleBrains []MCPVisibleBrainResponse `json:"visible_brains"`
	Reader        *MCPReaderResponse        `json:"reader"`
}

func MCPConnectionResponseFromInput(connection input_port.MCPConnection) MCPConnectionResponse {
	response := MCPConnectionResponse{
		CLITokens:     make([]MCPCLITokenResponse, len(connection.CLITokens)),
		VisibleBrains: make([]MCPVisibleBrainResponse, len(connection.VisibleBrains)),
	}
	if connection.Client != nil {
		response.Client = &MCPClientResponse{ID: connection.Client.ID, Name: connection.Client.Name}
	}
	if connection.CodexClient != nil {
		response.CodexClient = &MCPClientResponse{ID: connection.CodexClient.ID, Name: connection.CodexClient.Name}
	}
	if connection.Reader != nil {
		response.Reader = &MCPReaderResponse{State: string(connection.Reader.State), StateReason: connection.Reader.StateReason}
	}
	for i, token := range connection.CLITokens {
		response.CLITokens[i] = MCPCLITokenResponseFromEntity(token)
	}
	for i, brain := range connection.VisibleBrains {
		response.VisibleBrains[i] = MCPVisibleBrainResponse{
			SourceID: brain.SourceID.String(), Name: brain.Name, State: brain.State, Role: brain.Role, CanWrite: brain.CanWrite(),
		}
	}
	return response
}

func MCPCLITokenResponseFromEntity(token entity.MCPToken) MCPCLITokenResponse {
	label := ""
	if token.Label != nil {
		label = *token.Label
	}
	return MCPCLITokenResponse{
		ID: token.ID, Label: label, ExpiresAt: token.ExpiresAt, CreatedAt: token.CreatedAt,
	}
}

func MCPClientResponseFromEntity(client entity.MCPClient) MCPClientResponse {
	return MCPClientResponse{ID: client.ID, Name: client.Name}
}
