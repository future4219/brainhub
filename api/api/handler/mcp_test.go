package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type mcpHandlerUseCaseStub struct {
	error     error
	tool      string
	requested *string
}

func (s *mcpHandlerUseCaseStub) Connection(context.Context, string) (input_port.MCPConnection, error) {
	return input_port.MCPConnection{}, nil
}
func (s *mcpHandlerUseCaseStub) IssueClient(context.Context, string) (entity.MCPClient, error) {
	return entity.MCPClient{}, nil
}
func (s *mcpHandlerUseCaseStub) IssueCLIToken(context.Context, string, string) (input_port.IssuedCLIToken, error) {
	return input_port.IssuedCLIToken{}, nil
}
func (s *mcpHandlerUseCaseStub) RevokeCLIToken(context.Context, string, string) error { return nil }
func (s *mcpHandlerUseCaseStub) ValidateAuthorization(context.Context, input_port.OAuthAuthorizationRequest, string) (input_port.OAuthAuthorization, error) {
	return input_port.OAuthAuthorization{}, nil
}
func (s *mcpHandlerUseCaseStub) ApproveAuthorization(context.Context, input_port.OAuthAuthorizationRequest, string) (string, error) {
	return "", nil
}
func (s *mcpHandlerUseCaseStub) ExchangeAuthorizationCode(context.Context, string, string, string, string) (input_port.OAuthTokenPair, error) {
	return input_port.OAuthTokenPair{}, nil
}
func (s *mcpHandlerUseCaseStub) RefreshAccessToken(context.Context, string, string) (input_port.OAuthTokenPair, error) {
	return input_port.OAuthTokenPair{}, nil
}
func (s *mcpHandlerUseCaseStub) RevokeToken(context.Context, string, string) error { return nil }
func (s *mcpHandlerUseCaseStub) AuthorizeCall(_ context.Context, token, tool string, requested *string) (input_port.MCPCallAuthorization, error) {
	if token != "brainhub-token" {
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPUnauthorized
	}
	s.tool, s.requested = tool, requested
	if s.error != nil {
		return input_port.MCPCallAuthorization{}, s.error
	}
	if tool == "search" {
		return input_port.MCPCallAuthorization{GBrainToken: "gbrain-reader-token"}, nil
	}
	source := "brain-a"
	return input_port.MCPCallAuthorization{GBrainToken: "gbrain-reader-token", SourceID: &source}, nil
}
func (s *mcpHandlerUseCaseStub) ReissueReader(context.Context, string) error { return nil }
func (s *mcpHandlerUseCaseStub) ReconcileReaders(context.Context) error      { return nil }

func TestMCPHandlerInjectsSourceReplacesAuthorizationAndStreamsResponse(t *testing.T) {
	useCase := &mcpHandlerUseCaseStub{}
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gbrain-reader-token" {
			t.Errorf("upstream authorization = %q", r.Header.Get("Authorization"))
		}
		var body struct {
			Params struct {
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Params.Arguments["source_id"] != "brain-a" {
			t.Errorf("source_id = %v", body.Params.Arguments["source_id"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: message\ndata: ok\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	})
	handler := NewMCPHandler(useCase, proxy, "https://brainhub.example/.well-known/oauth-protected-resource/mcp")
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_page","arguments":{"slug":"people/a","source_id":"brain-a"}}}`))
	request.Header.Set("Authorization", "Bearer brainhub-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/event-stream" || !strings.Contains(response.Body.String(), "data: ok") {
		t.Fatalf("response = %d %q %q", response.Code, response.Header().Get("Content-Type"), response.Body.String())
	}
	if useCase.tool != "get_page" || useCase.requested == nil || *useCase.requested != "brain-a" {
		t.Fatalf("tool/source = %q %v", useCase.tool, useCase.requested)
	}
}

func TestMCPHandlerProxiesSearchAndRejectsLegacyTokenAs401(t *testing.T) {
	useCase := &mcpHandlerUseCaseStub{}
	called := false
	handler := NewMCPHandler(useCase, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		var body struct {
			Params struct {
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body.Params.Arguments["source_id"]; ok {
			t.Fatal("search must rely on the reader grant, not inject source_id")
		}
	}), "https://brainhub.example/.well-known/oauth-protected-resource/mcp")
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"search","arguments":{"query":"secret"}}}`))
	request.Header.Set("Authorization", "Bearer brainhub-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !called || useCase.tool != "search" || useCase.requested != nil {
		t.Fatalf("search response/called = %d %q %t", response.Code, response.Body.String(), called)
	}

	legacy := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":8,"method":"tools/list"}`))
	legacy.Header.Set("Authorization", "Bearer old-gbrain-token")
	legacyResponse := httptest.NewRecorder()
	handler.ServeHTTP(legacyResponse, legacy)
	if legacyResponse.Code != http.StatusUnauthorized {
		t.Fatalf("legacy status = %d; want 401", legacyResponse.Code)
	}
	if got := legacyResponse.Header().Get("WWW-Authenticate"); got != `Bearer resource_metadata="https://brainhub.example/.well-known/oauth-protected-resource/mcp"` {
		t.Fatalf("WWW-Authenticate = %q", got)
	}
}

func TestMCPHandlerExplainsExpiredCLIToken(t *testing.T) {
	useCase := &mcpHandlerUseCaseStub{error: input_port.ErrMCPTokenExpired}
	handler := NewMCPHandler(useCase, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("expired token must not reach GBrain")
	}), "https://brainhub.example/.well-known/oauth-protected-resource/mcp")
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"list_pages","arguments":{}}}`))
	request.Header.Set("Authorization", "Bearer brainhub-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"error":"token_expired"`) || !strings.Contains(response.Body.String(), "Issue a new CLI token") {
		t.Fatalf("expired response = %d %q", response.Code, response.Body.String())
	}
	if header := response.Header().Get("WWW-Authenticate"); !strings.Contains(header, `error="invalid_token"`) || !strings.Contains(header, "CLI token expired") {
		t.Fatalf("WWW-Authenticate = %q", header)
	}
}
