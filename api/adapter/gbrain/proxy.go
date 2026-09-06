package gbrain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

// Credential access is private to this adapter boundary.
type readerTokenProvider interface {
	AccessToken(context.Context, string, []entity.SourceID) (string, error)
}
type writerTokenProvider interface {
	AccessToken(context.Context, string, entity.SourceID) (string, error)
}

func NewProxy(baseURL string, reader readerTokenProvider, writer writerTokenProvider) (http.Handler, error) {
	target, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, errors.New("GBrain base URL must use http or https")
	}
	if target.Host == "" {
		return nil, errors.New("GBrain base URL must include a host")
	}
	if reader == nil || writer == nil {
		return nil, errors.New("GBrain reader and writer are required")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		if len(writableCatalogSources(r)) > 0 {
			r.Header.Set("Accept-Encoding", "identity")
		}
	}
	proxy.ModifyResponse = appendMCPWriteTool
	proxy.FlushInterval = -1
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorized, ok := input_port.MCPRequestFromContext(r.Context())
		if !ok || !validMCPAuthorization(authorized, r.Method) {
			http.Error(w, "invalid internal MCP authorization", http.StatusForbidden)
			return
		}
		grant := authorized.Authorization
		var token string
		var err error
		if grant.WriteTarget != nil {
			token, err = writer.AccessToken(r.Context(), grant.WriteTarget.BrainID, grant.WriteTarget.SourceID)
		} else if len(grant.ReadableSources) == 0 {
			// Preserve the reader preparation failure when no sources are visible.
			err = errors.New("no visible sources")
		} else {
			token, err = reader.AccessToken(r.Context(), grant.UserID, grant.ReadableSources)
		}
		if err != nil || token == "" {
			http.Error(w, "MCP connection is unavailable; open brainhub connection settings and follow the recovery message", http.StatusServiceUnavailable)
			return
		}
		if grant.SourceID != nil || grant.WriteTarget != nil {
			params := authorized.Request["params"].(map[string]any)
			arguments, _ := params["arguments"].(map[string]any)
			if arguments == nil {
				arguments = make(map[string]any)
				params["arguments"] = arguments
			}
			if grant.WriteTarget != nil {
				delete(arguments, "source_id")
			} else {
				arguments["source_id"] = *grant.SourceID
			}
			body, err := json.Marshal(authorized.Request)
			if err != nil {
				http.Error(w, "failed to prepare MCP request", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
		}
		// Never forward the caller's Brainhub token or fall back to a wider grant.
		r.Header.Set("Authorization", "Bearer "+token)
		proxy.ServeHTTP(w, r)
	}), nil
}

// Check that the server's decision still matches the request being forwarded.
// Membership and consent are decided once by the use case, not reconstructed here.
func validMCPAuthorization(request input_port.AuthorizedMCPRequest, method string) bool {
	grant := request.Authorization
	if grant.UserID == "" {
		return false
	}
	var tool string
	params, _ := request.Request["params"].(map[string]any)
	if request.Request["method"] == "tools/call" {
		if method != http.MethodPost {
			return false
		}
		tool, _ = params["name"].(string)
	}
	if tool != grant.ToolName {
		return false
	}
	arguments, _ := params["arguments"].(map[string]any)
	source, _ := arguments["source_id"].(string)
	if target := grant.WriteTarget; target != nil {
		return tool == "put_page" && grant.SourceID == nil && target.BrainID != "" &&
			target.SourceID != "" && target.SourceID != "__all__" && source == target.SourceID.String()
	}
	if tool == "put_page" {
		return false
	}
	if tool == "query" || tool == "list_pages" || tool == "get_page" {
		if source == "" {
			source = "__all__"
		}
		return grant.SourceID != nil && *grant.SourceID == source &&
			(source == "__all__" || slices.Contains(grant.ReadableSources, entity.SourceID(source)))
	}
	return grant.SourceID == nil
}

func writableCatalogSources(r *http.Request) []string {
	request, ok := input_port.MCPRequestFromContext(r.Context())
	if !ok || r.Method != http.MethodPost || request.Request["method"] != "tools/list" {
		return nil
	}
	return request.Authorization.WritableSources
}
