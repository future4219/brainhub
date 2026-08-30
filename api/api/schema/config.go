package schema

type ConfigResponse struct {
	MCPURL string `json:"mcp_url"`
	WebURL string `json:"web_url"`
}
