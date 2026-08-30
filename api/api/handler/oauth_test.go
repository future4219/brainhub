package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthTokenRejectsClientSecretAuthentication(t *testing.T) {
	handler, err := NewOAuthHandler(&mcpHandlerUseCaseStub{}, "https://brainhub.example/mcp", "https://brainhub.example")
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"grant_type": {"authorization_code"}, "client_id": {"client-id"},
		"code": {"code"}, "redirect_uri": {"https://claude.ai/api/mcp/auth_callback"}, "code_verifier": {strings.Repeat("a", 43)},
	}
	request := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth("client-id", "must-not-be-accepted")
	response := httptest.NewRecorder()
	handler.Token(response, request)
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"error":"invalid_client"`) {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}
