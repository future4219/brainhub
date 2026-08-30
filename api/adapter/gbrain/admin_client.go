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
	"slices"
	"sync"
	"time"

	"brainhub/usecase/output_port"
)

const adminCookieName = "gbrain_admin"

// AdminClient permanently holds GBrain's full bootstrap-admin authority.
// Its token and session cookie must never leave this adapter or be logged.
type AdminClient struct {
	baseURL        *url.URL
	httpClient     *http.Client
	bootstrapToken string

	sessionMu     sync.Mutex
	sessionCookie string
	sessionExpiry time.Time
}

func NewAdminClient(baseURL, bootstrapToken string) (*AdminClient, error) {
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
	if bootstrapToken == "" {
		return nil, errors.New("GBrain admin bootstrap token is required")
	}
	return &AdminClient{
		baseURL: parsed, httpClient: &http.Client{Timeout: 15 * time.Second}, bootstrapToken: bootstrapToken,
	}, nil
}

func (c *AdminClient) RegisterClient(ctx context.Context, input output_port.RegisterGBrainClientInput) (output_port.RegisteredGBrainClient, error) {
	payload := struct {
		Name                    string   `json:"name"`
		Scopes                  []string `json:"scopes"`
		Source                  *string  `json:"source,omitempty"`
		FederatedRead           []string `json:"federatedRead"`
		GrantTypes              []string `json:"grantTypes"`
		RedirectURIs            []string `json:"redirectUris"`
		TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
	}{
		Name: input.Name, Scopes: input.Scopes, Source: input.Source,
		FederatedRead: input.FederatedRead, GrantTypes: input.GrantTypes,
		RedirectURIs: input.RedirectURIs, TokenEndpointAuthMethod: input.TokenEndpointAuthMethod,
	}
	var response struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := c.doJSON(ctx, "/admin/api/register-client", payload, &response); err != nil {
		return output_port.RegisteredGBrainClient{}, err
	}
	if response.ClientID == "" {
		return output_port.RegisteredGBrainClient{}, errors.New("GBrain register-client response is missing clientId")
	}
	return output_port.RegisteredGBrainClient{ID: response.ClientID, Secret: response.ClientSecret}, nil
}

func (c *AdminClient) RevokeClient(ctx context.Context, clientID string) error {
	var response struct {
		Revoked bool `json:"revoked"`
	}
	if err := c.doJSON(ctx, "/admin/api/revoke-client", map[string]string{"clientId": clientID}, &response); err != nil {
		return err
	}
	if !response.Revoked {
		return errors.New("GBrain revoke-client did not confirm revocation")
	}
	return nil
}

func (c *AdminClient) RescopeClient(ctx context.Context, clientID string, federatedRead []string) error {
	if clientID == "" || len(federatedRead) == 0 {
		return errors.New("GBrain rescope requires a client ID and at least one source")
	}
	var response struct {
		ClientID      string   `json:"clientId"`
		FederatedRead []string `json:"federatedRead"`
	}
	if err := c.doJSON(ctx, "/admin/api/rescope-client", map[string]any{
		"clientId": clientID, "federatedRead": federatedRead,
	}, &response); err != nil {
		return err
	}
	if response.ClientID != clientID || !slices.Equal(response.FederatedRead, federatedRead) {
		return errors.New("GBrain rescope-client did not confirm the requested scope")
	}
	return nil
}

func (c *AdminClient) doJSON(ctx context.Context, path string, payload, destination any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 2; attempt++ {
		cookie, err := c.adminSession(ctx, attempt > 0)
		if err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolve(path), bytes.NewReader(body))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		// GBrain marks this cookie Secure for its public HTTPS origin. The
		// backend talks over Compose's trusted HTTP network, so attach the
		// captured cookie explicitly instead of relying on a cookie jar.
		request.Header.Set("Cookie", cookie)
		response, err := c.httpClient.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 {
			_ = response.Body.Close()
			c.invalidateSession(cookie)
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			message := adminError(response.Body)
			_ = response.Body.Close()
			return fmt.Errorf("GBrain admin %s returned HTTP %d: %s", path, response.StatusCode, message)
		}
		err = json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(destination)
		_ = response.Body.Close()
		if err != nil {
			return fmt.Errorf("decode GBrain admin %s response: %w", path, err)
		}
		return nil
	}
	return errors.New("GBrain admin authentication failed after one retry")
}

func (c *AdminClient) adminSession(ctx context.Context, force bool) (string, error) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if !force && c.sessionCookie != "" && time.Now().Add(tokenRefreshSkew).Before(c.sessionExpiry) {
		return c.sessionCookie, nil
	}
	payload, err := json.Marshal(map[string]string{"token": c.bootstrapToken})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.resolve("/admin/login"), bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("GBrain admin login returned HTTP %d", response.StatusCode)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name != adminCookieName || cookie.Value == "" {
			continue
		}
		expiry := cookie.Expires
		if cookie.MaxAge > 0 {
			expiry = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
		}
		if expiry.IsZero() {
			return "", errors.New("GBrain admin cookie is missing an expiry")
		}
		c.sessionCookie = cookie.Name + "=" + cookie.Value
		c.sessionExpiry = expiry
		return c.sessionCookie, nil
	}
	return "", errors.New("GBrain admin login response is missing gbrain_admin cookie")
}

func (c *AdminClient) invalidateSession(cookie string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	if c.sessionCookie == cookie {
		c.sessionCookie = ""
		c.sessionExpiry = time.Time{}
	}
}

func (c *AdminClient) resolve(path string) string {
	return c.baseURL.ResolveReference(&url.URL{Path: path}).String()
}

func adminError(reader io.Reader) string {
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.NewDecoder(io.LimitReader(reader, maxResponseBytes)).Decode(&body) != nil {
		return "request failed"
	}
	if body.Message != "" {
		return body.Message
	}
	if body.Error != "" {
		return body.Error
	}
	return "request failed"
}
