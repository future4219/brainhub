package gbrain

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

type proxyReaderStub struct{}

func (proxyReaderStub) AccessToken(context.Context, string, []entity.SourceID) (string, error) {
	return "reader-token", nil
}

type proxyWriterStub struct{}

func (proxyWriterStub) AccessToken(context.Context, string, entity.SourceID) (string, error) {
	return "writer-token", nil
}

func TestProxyRequiresBoundServerAuthorization(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("invalid grant reached upstream") }))
	defer upstream.Close()
	proxy, err := NewProxy(upstream.URL, proxyReaderStub{}, proxyWriterStub{})
	if err != nil {
		t.Fatal(err)
	}
	source := "brain-a"
	for _, tc := range []struct {
		name, tool, source string
		grant              *input_port.MCPCallAuthorization
		status             int
	}{
		{"missing", "put_page", "brain-a", nil, 403},
		{"empty", "put_page", "brain-a", &input_port.MCPCallAuthorization{}, 403},
		{"reader cannot write", "put_page", "brain-a", &input_port.MCPCallAuthorization{UserID: "user", ToolName: "put_page", ReadableSources: []entity.SourceID{"brain-a"}}, 403},
		{"writer source mismatch", "put_page", "brain-b", &input_port.MCPCallAuthorization{UserID: "user", ToolName: "put_page", WriteTarget: &input_port.MCPWriteTarget{BrainID: "a", SourceID: "brain-a"}}, 403},
		{"writer used for management", "sources_add", "brain-a", &input_port.MCPCallAuthorization{UserID: "user", ToolName: "sources_add", WriteTarget: &input_port.MCPWriteTarget{BrainID: "a", SourceID: "brain-a"}}, 403},
		{"read source mismatch", "get_page", "brain-b", &input_port.MCPCallAuthorization{UserID: "user", ToolName: "get_page", SourceID: &source, ReadableSources: []entity.SourceID{"brain-a"}}, 403},
		{"no visible sources", "search", "", &input_port.MCPCallAuthorization{UserID: "user", ToolName: "search"}, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]any{"method": "tools/call", "params": map[string]any{"name": tc.tool, "arguments": map[string]any{"source_id": tc.source}}}
			body, _ := json.Marshal(payload)
			r := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(body)))
			r.Header.Set("Authorization", "Bearer caller-token")
			r.Header.Set("X-Brainhub-Write-Source", "brain-a")
			if tc.grant != nil {
				r = r.WithContext(input_port.WithAuthorizedMCPRequest(r.Context(), input_port.AuthorizedMCPRequest{Request: payload, Authorization: *tc.grant}))
			}
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
		})
	}
}

func TestProxyTranslatesReadSourcesWithoutChangingRPCID(t *testing.T) {
	for _, source := range []string{"", "brain-a"} {
		t.Run(source, func(t *testing.T) {
			wantSource := source
			if wantSource == "" {
				wantSource = "__all__"
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer reader-token" {
					t.Error("caller token leaked")
				}
				var request struct {
					ID     json.Number
					Params struct{ Arguments map[string]any }
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				if request.ID.String() != "9007199254740993" || request.Params.Arguments["source_id"] != wantSource {
					t.Errorf("changed request: %+v", request)
				}
				w.Header().Set("Mcp-Session-Id", "session")
				w.WriteHeader(http.StatusAccepted)
				_, _ = io.WriteString(w, "upstream response")
			}))
			defer upstream.Close()
			proxy, err := NewProxy(upstream.URL, proxyReaderStub{}, proxyWriterStub{})
			if err != nil {
				t.Fatal(err)
			}
			payload := map[string]any{"jsonrpc": "2.0", "id": json.Number("9007199254740993"), "method": "tools/call", "params": map[string]any{"name": "get_page", "arguments": map[string]any{"slug": "notes/test", "source_id": source}}}
			body, _ := json.Marshal(payload)
			r := httptest.NewRequest("POST", "/mcp", strings.NewReader(string(body)))
			r.Header.Set("Authorization", "Bearer caller-token")
			r = r.WithContext(input_port.WithAuthorizedMCPRequest(r.Context(), input_port.AuthorizedMCPRequest{Request: payload, Authorization: input_port.MCPCallAuthorization{UserID: "user", ToolName: "get_page", SourceID: &wantSource, ReadableSources: []entity.SourceID{"brain-a"}}}))
			w := httptest.NewRecorder()
			proxy.ServeHTTP(w, r)
			if w.Code != http.StatusAccepted || w.Header().Get("Mcp-Session-Id") != "session" || w.Body.String() != "upstream response" {
				t.Fatalf("response changed: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
