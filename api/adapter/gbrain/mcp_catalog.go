package gbrain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// The upstream reader catalog stays authoritative. Brainhub adds only the
// source-bound page write it authorizes; source administration stays unavailable.
func appendMCPWriteTool(response *http.Response) error {
	sources := writableCatalogSources(response.Request)
	if len(sources) == 0 || response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxResponseBytes {
		return errors.New("MCP catalog is too large")
	}
	isSSE := strings.Contains(response.Header.Get("Content-Type"), "text/event-stream")
	if isSSE {
		lines := strings.Split(string(body), "\n")
		for i, line := range lines {
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			updated, err := appendWriteToolJSON([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), sources)
			if err != nil {
				return err
			}
			lines[i] = "data: " + string(updated)
		}
		body = []byte(strings.Join(lines, "\n"))
	} else {
		body, err = appendWriteToolJSON(body, sources)
		if err != nil {
			return err
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Set("Content-Length", strconv.Itoa(len(body)))
	return nil
}

func appendWriteToolJSON(body []byte, sources []string) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if len(envelope["result"]) == 0 {
		return body, nil
	}
	if err := json.Unmarshal(envelope["result"], &result); err != nil {
		return nil, err
	}
	if _, ok := result["tools"]; !ok {
		return body, nil
	}
	var tools []json.RawMessage
	if err := json.Unmarshal(result["tools"], &tools); err != nil {
		return nil, err
	}
	// Upstream v0.46.28 put_page accepts slug/content/allow_empty. source_id is
	// Brainhub's required routing parameter, removed before calling GBrain.
	tool, err := json.Marshal(map[string]any{
		"name":        "put_page",
		"description": "Create or update a page in an editable Brainhub source. REPLACES the entire page: first get_page with include_content=true and the same source_id, preserve existing content, then write the full Markdown. Current writable sources: " + strings.Join(sources, ", ") + ".",
		"inputSchema": map[string]any{
			"type": "object", "additionalProperties": false,
			"required": []string{"source_id", "slug", "content"},
			"properties": map[string]any{
				"source_id":   map[string]any{"type": "string", "description": "Explicit target source ID; must be currently editable. Never __all__."},
				"slug":        map[string]any{"type": "string", "description": "Page slug"},
				"content":     map[string]any{"type": "string", "description": "Full Markdown content with YAML frontmatter"},
				"allow_empty": map[string]any{"type": "boolean", "description": "Allow intentionally overwriting a non-empty page with empty content (default false)."},
			},
		},
		"annotations": map[string]any{"readOnlyHint": false, "destructiveHint": true, "openWorldHint": false},
	})
	if err != nil {
		return nil, err
	}
	tools = append(tools, tool)
	result["tools"], err = json.Marshal(tools)
	if err != nil {
		return nil, err
	}
	envelope["result"], err = json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}
