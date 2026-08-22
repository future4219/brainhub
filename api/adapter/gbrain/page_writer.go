package gbrain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const (
	timelineDelimiter = "<!-- timeline -->"
	stateLinkType     = "superseded_by"
	stateLinkSource   = "brainhub-web"
)

type upstreamLink struct {
	ToSlug     string `json:"to_slug"`
	LinkType   string `json:"link_type"`
	LinkSource string `json:"link_source"`
}

type schemaGraph struct {
	Pack  string `json:"pack"`
	Nodes []struct {
		Name      string `json:"name"`
		Primitive string `json:"primitive"`
	} `json:"nodes"`
}

type putPageResult struct {
	Status       string `json:"status"`
	WriteThrough *struct {
		Written   bool   `json:"written"`
		Committed bool   `json:"committed"`
		Skipped   string `json:"skipped"`
		Error     string `json:"error"`
	} `json:"write_through"`
}

func (s *WriterService) writerClient(ctx context.Context, brainID string, sourceID entity.SourceID) (*Client, error) {
	clientID, secret, err := s.credentialsFor(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	return newClient(s.baseURL, clientID, secret, "read write")
}

func (s *WriterService) List(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.Page, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	return client.List(ctx, sourceID)
}

func (s *WriterService) GetEditable(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	page, err := client.Get(ctx, sourceID, slug)
	if err != nil {
		return entity.PageDetail{}, err
	}
	links, err := client.links(ctx, slug)
	if err != nil {
		return entity.PageDetail{}, err
	}
	// Current-state authority follows the three-layer resolve contract: an
	// explicit brainhub-web graph edge is truth, frontmatter is its write-through
	// representation, and body status claims are ignored. If edge and
	// frontmatter disagree, resolve from the edge.
	for _, link := range links {
		if link.LinkType == stateLinkType && link.LinkSource == stateLinkSource {
			target := link.ToSlug
			page.SupersededBy = &target
			break
		}
	}
	return page, nil
}

func (s *WriterService) ListTypes(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.PageType, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return nil, err
	}
	var graph schemaGraph
	if err := client.toolJSON(ctx, "schema_graph", map[string]any{}, &graph); err != nil {
		return nil, err
	}
	types := make([]entity.PageType, len(graph.Nodes))
	for i, node := range graph.Nodes {
		types[i] = entity.PageType{Name: node.Name, Primitive: node.Primitive}
	}
	return types, nil
}

func (s *WriterService) Put(ctx context.Context, brainID string, sourceID entity.SourceID, page entity.PageWrite) error {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return err
	}
	content, err := pageContent(page)
	if err != nil {
		return err
	}
	var result putPageResult
	if err := client.toolJSON(ctx, "put_page", map[string]any{"slug": page.Slug, "content": content}, &result); err != nil {
		return err
	}
	if result.WriteThrough == nil {
		return errors.New("GBrain put_page did not report write-through status")
	}
	if result.WriteThrough.Error != "" {
		return fmt.Errorf("GBrain write-through failed: %s", result.WriteThrough.Error)
	}
	if !result.WriteThrough.Written {
		return fmt.Errorf("GBrain write-through skipped: %s", result.WriteThrough.Skipped)
	}
	if !result.WriteThrough.Committed {
		return errors.New("GBrain wrote the page but did not commit it to git")
	}
	return client.syncStateLink(ctx, page.Slug, page.SupersededBy)
}

func pageContent(page entity.PageWrite) (string, error) {
	frontmatter := make(map[string]any, len(page.Frontmatter)+4)
	for key, value := range page.Frontmatter {
		frontmatter[key] = value
	}
	frontmatter["title"] = page.Title
	frontmatter["type"] = page.Type
	frontmatter["tags"] = page.Tags
	if page.SupersededBy == nil {
		delete(frontmatter, "superseded_by")
	} else {
		frontmatter["superseded_by"] = *page.SupersededBy
	}
	encoded, err := json.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("encode page frontmatter: %w", err)
	}
	return "---\n" + string(encoded) + "\n---\n\n" + page.CompiledTruth + "\n\n" + timelineDelimiter + "\n\n" + page.Timeline, nil
}

func (c *Client) links(ctx context.Context, slug string) ([]upstreamLink, error) {
	var links []upstreamLink
	if err := c.toolJSON(ctx, "get_links", map[string]any{"slug": slug}, &links); err != nil {
		return nil, err
	}
	return links, nil
}

func (c *Client) syncStateLink(ctx context.Context, slug string, target *string) error {
	links, err := c.links(ctx, slug)
	if err != nil {
		return err
	}
	found := false
	for _, link := range links {
		if link.LinkType != stateLinkType || link.LinkSource != stateLinkSource {
			continue
		}
		if target != nil && link.ToSlug == *target && !found {
			found = true
			continue
		}
		var ignored map[string]any
		if err := c.toolJSON(ctx, "remove_link", map[string]any{
			"from": slug, "to": link.ToSlug, "link_type": stateLinkType, "link_source": stateLinkSource,
		}, &ignored); err != nil {
			return err
		}
	}
	if target != nil && !found {
		var ignored map[string]any
		return c.toolJSON(ctx, "add_link", map[string]any{
			"from": slug, "to": *target, "link_type": stateLinkType, "link_source": stateLinkSource,
		}, &ignored)
	}
	return nil
}

func (c *Client) toolJSON(ctx context.Context, name string, arguments map[string]any, destination any) error {
	response, err := c.callTool(ctx, 1, name, arguments)
	if err != nil {
		return err
	}
	if response.Error != nil {
		return fmt.Errorf("GBrain %s failed: %s", name, response.Error.Message)
	}
	if response.Result == nil || len(response.Result.Content) == 0 {
		return fmt.Errorf("GBrain %s returned no content", name)
	}
	content := response.Result.Content[0].Text
	if response.Result.IsError {
		var upstreamError struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(content), &upstreamError) == nil && upstreamError.Error == "page_not_found" {
			return output_port.ErrNotFound
		}
		message := strings.TrimSpace(upstreamError.Message)
		if message == "" {
			message = strings.TrimSpace(content)
		}
		return fmt.Errorf("GBrain %s failed: %s", name, message)
	}
	if destination == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(content), destination); err != nil {
		return fmt.Errorf("decode GBrain %s response: %w", name, err)
	}
	return nil
}
