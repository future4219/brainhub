package gbrain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"brainhub/domain/entity"
)

const (
	pageLimit        = 100
	maxResponseBytes = 8 << 20
	tokenRefreshSkew = 30 * time.Second
)

type Client struct {
	baseURL      *url.URL
	httpClient   *http.Client
	clientID     string
	clientSecret string

	tokenMu       sync.Mutex
	tokenEndpoint string
	accessToken   string
	tokenExpiry   time.Time
}

type authorizationServerMetadata struct {
	TokenEndpoint string `json:"token_endpoint"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Scope       string `json:"scope"`
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  rpcCallParams `json:"params"`
}

type rpcCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type rpcResponse struct {
	Result *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type upstreamPage struct {
	Slug      string    `json:"slug"`
	SourceID  string    `json:"source_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

type upstreamSourcesResponse struct {
	Sources []struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		LocalPath  *string    `json:"local_path"`
		RemoteURL  *string    `json:"remote_url"`
		Federated  bool       `json:"federated"`
		PageCount  int        `json:"page_count"`
		LastSyncAt *time.Time `json:"last_sync_at"`
	} `json:"sources"`
}

func NewClient(baseURL, clientID, clientSecret string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("GBrain base URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, errors.New("GBrain base URL must include a host")
	}
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("GBrain OAuth client credentials are required")
	}

	return &Client{
		baseURL:      parsed,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

func (c *Client) List(ctx context.Context, sourceID entity.SourceID) ([]entity.Page, error) {
	var pages []entity.Page
	for offset := 0; ; offset += pageLimit {
		batch, err := c.listPageBatch(ctx, offset)
		if err != nil {
			return nil, err
		}
		for _, page := range batch {
			if page.SourceID != sourceID.String() {
				continue
			}
			pages = append(pages, entity.Page{
				Slug:      page.Slug,
				Title:     page.Title,
				Type:      page.Type,
				UpdatedAt: page.UpdatedAt,
			})
		}
		if len(batch) < pageLimit {
			return pages, nil
		}
	}
}

func (c *Client) listPageBatch(ctx context.Context, offset int) ([]upstreamPage, error) {
	response, err := c.callTool(ctx, offset+1, "list_pages", map[string]any{
		"limit":  pageLimit,
		"offset": offset,
	})
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, fmt.Errorf("GBrain list_pages failed: %s", response.Error.Message)
	}
	if response.Result == nil || len(response.Result.Content) == 0 {
		return nil, errors.New("GBrain list_pages returned no content")
	}

	var pages []upstreamPage
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &pages); err != nil {
		return nil, fmt.Errorf("decode GBrain pages: %w", err)
	}
	return pages, nil
}

func (c *Client) ListSources(ctx context.Context) ([]entity.Source, error) {
	response, err := c.callTool(ctx, 1, "sources_list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return nil, fmt.Errorf("GBrain sources_list failed: %s", response.Error.Message)
	}
	if response.Result == nil || len(response.Result.Content) == 0 {
		return nil, errors.New("GBrain sources_list returned no content")
	}

	var upstream upstreamSourcesResponse
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &upstream); err != nil {
		return nil, fmt.Errorf("decode GBrain sources: %w", err)
	}
	sources := make([]entity.Source, len(upstream.Sources))
	for i, source := range upstream.Sources {
		sources[i] = entity.Source{
			ID:         source.ID,
			Name:       source.Name,
			LocalPath:  source.LocalPath,
			RemoteURL:  source.RemoteURL,
			Federated:  source.Federated,
			PageCount:  source.PageCount,
			LastSyncAt: source.LastSyncAt,
		}
	}
	return sources, nil
}

func (c *Client) callTool(ctx context.Context, id int, name string, arguments map[string]any) (*rpcResponse, error) {
	payload, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: rpcCallParams{
			Name:      name,
			Arguments: arguments,
		},
	})
	if err != nil {
		return nil, err
	}
	return c.callMCP(ctx, payload)
}

func (c *Client) callMCP(ctx context.Context, payload []byte) (*rpcResponse, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.token(ctx)
		if err != nil {
			return nil, err
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolve("/mcp"), bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")

		response, err := c.httpClient.Do(request)
		if err != nil {
			return nil, err
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 {
			_ = response.Body.Close()
			c.invalidateToken(token)
			continue
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("GBrain MCP returned HTTP %d", response.StatusCode)
		}
		return decodeRPCResponse(response.Body, response.Header.Get("Content-Type"))
	}
	return nil, errors.New("GBrain authentication failed after token refresh")
}

func (c *Client) token(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Add(tokenRefreshSkew).Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	if c.tokenEndpoint == "" {
		endpoint, err := c.discoverTokenEndpoint(ctx)
		if err != nil {
			return "", err
		}
		c.tokenEndpoint = endpoint
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {"read"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("GBrain token endpoint returned HTTP %d", response.StatusCode)
	}

	var token tokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&token); err != nil {
		return "", fmt.Errorf("decode GBrain token response: %w", err)
	}
	if token.AccessToken == "" || token.ExpiresIn <= 0 {
		return "", errors.New("GBrain token response is missing access_token or expires_in")
	}

	c.accessToken = token.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func (c *Client) discoverTokenEndpoint(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolve("/.well-known/oauth-authorization-server"), nil)
	if err != nil {
		return "", err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("GBrain OAuth discovery returned HTTP %d", response.StatusCode)
	}

	var metadata authorizationServerMetadata
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&metadata); err != nil {
		return "", fmt.Errorf("decode GBrain OAuth discovery: %w", err)
	}
	endpoint, err := url.Parse(metadata.TokenEndpoint)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return "", errors.New("GBrain OAuth discovery returned an invalid token endpoint")
	}
	return endpoint.String(), nil
}

func (c *Client) invalidateToken(token string) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.accessToken == token {
		c.accessToken = ""
		c.tokenExpiry = time.Time{}
	}
}

func (c *Client) resolve(path string) string {
	reference := &url.URL{Path: path}
	return c.baseURL.ResolveReference(reference).String()
}

func decodeRPCResponse(reader io.Reader, contentType string) (*rpcResponse, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxResponseBytes {
		return nil, errors.New("GBrain MCP response is too large")
	}

	candidates := [][]byte{body}
	if strings.Contains(contentType, "text/event-stream") {
		candidates = candidates[:0]
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "data:") {
				candidates = append(candidates, []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))))
			}
		}
	}

	for _, candidate := range candidates {
		var response rpcResponse
		if len(candidate) == 0 || json.Unmarshal(candidate, &response) != nil {
			continue
		}
		return &response, nil
	}
	return nil, errors.New("GBrain MCP returned no valid JSON-RPC response")
}
