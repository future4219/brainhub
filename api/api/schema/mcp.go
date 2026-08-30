package schema

import (
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type MCPClientResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MCPVisibleBrainResponse struct {
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Role     string `json:"role"`
}

type MCPReaderResponse struct {
	State       string `json:"state"`
	StateReason string `json:"state_reason"`
}

type MCPConnectionResponse struct {
	Client        *MCPClientResponse        `json:"client"`
	VisibleBrains []MCPVisibleBrainResponse `json:"visible_brains"`
	Reader        *MCPReaderResponse        `json:"reader"`
}

func MCPConnectionResponseFromInput(connection input_port.MCPConnection) MCPConnectionResponse {
	response := MCPConnectionResponse{VisibleBrains: make([]MCPVisibleBrainResponse, len(connection.VisibleBrains))}
	if connection.Client != nil {
		response.Client = &MCPClientResponse{ID: connection.Client.ID, Name: connection.Client.Name}
	}
	if connection.Reader != nil {
		response.Reader = &MCPReaderResponse{State: string(connection.Reader.State), StateReason: connection.Reader.StateReason}
	}
	for i, brain := range connection.VisibleBrains {
		response.VisibleBrains[i] = MCPVisibleBrainResponse{
			SourceID: brain.SourceID.String(), Name: brain.Name, State: brain.State, Role: brain.Role,
		}
	}
	return response
}

func MCPClientResponseFromEntity(client entity.MCPClient) MCPClientResponse {
	return MCPClientResponse{ID: client.ID, Name: client.Name}
}
