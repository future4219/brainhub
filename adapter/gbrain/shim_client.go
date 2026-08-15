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
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const maxShimResponseBytes = 64 << 10

type ShimClient struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

type shimResponse struct {
	Error string `json:"error"`
}

func NewShimClient(baseURL, token string) (*ShimClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("shim URL must be an absolute HTTP URL")
	}
	if token == "" {
		return nil, errors.New("SHIM_TOKEN is required")
	}
	return &ShimClient{
		endpoint:   strings.TrimRight(parsed.String(), "/") + "/internal/sources",
		token:      token,
		httpClient: &http.Client{Timeout: defaultCommandTimeout + killGracePeriod + 10*time.Second},
	}, nil
}

func (c *ShimClient) Provision(ctx context.Context, sourceID entity.SourceID) error {
	payload, err := json.Marshal(map[string]string{"id": sourceID.String()})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call source shim: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxShimResponseBytes))
	if err != nil {
		return fmt.Errorf("read source shim response: %w", err)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	var upstream shimResponse
	if err := json.Unmarshal(body, &upstream); err != nil || upstream.Error == "" {
		upstream.Error = strings.TrimSpace(string(body))
	}
	if upstream.Error == "" {
		upstream.Error = http.StatusText(response.StatusCode)
	}
	if response.StatusCode == http.StatusConflict {
		return fmt.Errorf("%w: %s", output_port.ErrConflict, upstream.Error)
	}
	return fmt.Errorf("source shim returned HTTP %d: %s", response.StatusCode, upstream.Error)
}
