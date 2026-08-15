package gbrain

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientListsEveryPageAndRefreshesToken(t *testing.T) {
	var tokenRequests int32
	var discoveryRequests int32
	var offsetsMu sync.Mutex
	var offsets []int
	var upstream *httptest.Server

	allPages := make([]upstreamPage, 140)
	for i := range allPages {
		sourceID := "brainhub"
		if i == len(allPages)-1 {
			sourceID = "other"
		}
		allPages[i] = upstreamPage{
			Slug:      fmt.Sprintf("pages/%03d", i),
			SourceID:  sourceID,
			Type:      "decision",
			Title:     fmt.Sprintf("Page %03d", i),
			UpdatedAt: time.Date(2026, 8, 15, 0, 0, i, 0, time.UTC),
		}
	}

	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			atomic.AddInt32(&discoveryRequests, 1)
			_ = json.NewEncoder(w).Encode(map[string]string{"token_endpoint": upstream.URL + "/token"})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse token form: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("scope") != "read" {
				t.Errorf("unexpected token request: %v", r.Form)
			}
			if r.Form.Get("client_id") != "client-id" || r.Form.Get("client_secret") != "client-secret" {
				t.Error("client credentials were not sent with client_secret_post")
			}
			n := atomic.AddInt32(&tokenRequests, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": fmt.Sprintf("token-%d", n),
				"token_type":   "bearer",
				"expires_in":   3600,
				"scope":        "read",
			})
		case "/mcp":
			var request struct {
				Params struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode MCP request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if request.Params.Name == "sources_list" {
				if len(request.Params.Arguments) != 0 {
					t.Errorf("sources_list arguments = %v; want none", request.Params.Arguments)
				}
				sourceJSON, _ := json.Marshal(map[string]any{"sources": []map[string]any{
					{"id": "default", "name": "default", "local_path": nil, "remote_url": nil, "federated": true, "page_count": 0, "last_sync_at": nil},
					{"id": "brainhub", "name": "brainhub", "local_path": "/brain", "remote_url": nil, "federated": true, "page_count": 140, "last_sync_at": "2026-08-15T00:00:00Z"},
				}})
				envelope, _ := json.Marshal(map[string]any{
					"jsonrpc": "2.0",
					"id":      1,
					"result": map[string]any{
						"content": []map[string]string{{"type": "text", "text": string(sourceJSON)}},
					},
				})
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", envelope)
				return
			}
			if request.Params.Name != "list_pages" {
				t.Errorf("unexpected tool: %s", request.Params.Name)
			}
			if len(request.Params.Arguments) != 2 {
				t.Errorf("list_pages arguments = %v; want only limit and offset", request.Params.Arguments)
			}
			if _, ok := request.Params.Arguments["sort"]; ok {
				t.Error("list_pages must not send sort")
			}
			offset := int(request.Params.Arguments["offset"].(float64))
			if int(request.Params.Arguments["limit"].(float64)) != pageLimit {
				t.Errorf("limit = %v; want %d", request.Params.Arguments["limit"], pageLimit)
			}
			offsetsMu.Lock()
			offsets = append(offsets, offset)
			offsetsMu.Unlock()

			if offset == pageLimit && r.Header.Get("Authorization") == "Bearer token-1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" && got != "Bearer token-2" {
				t.Errorf("Authorization = %q", got)
			}

			end := offset + pageLimit
			if end > len(allPages) {
				end = len(allPages)
			}
			pageJSON, _ := json.Marshal(allPages[offset:end])
			envelope, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      offset + 1,
				"result": map[string]any{
					"content": []map[string]string{{"type": "text", "text": string(pageJSON)}},
				},
			})
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", envelope)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	repository, err := NewClient(upstream.URL, "client-id", "client-secret")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := repository.List(context.Background(), "brainhub")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 139 {
		t.Fatalf("pages = %d; want 139", len(pages))
	}
	sources, err := repository.ListSources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 || sources[1].ID != "brainhub" || sources[1].PageCount != 140 {
		t.Fatalf("sources = %+v", sources)
	}
	if got := atomic.LoadInt32(&discoveryRequests); got != 1 {
		t.Errorf("discovery requests = %d; want 1", got)
	}
	if got := atomic.LoadInt32(&tokenRequests); got != 2 {
		t.Errorf("token requests = %d; want 2", got)
	}
	offsetsMu.Lock()
	gotOffsets := append([]int(nil), offsets...)
	offsetsMu.Unlock()
	if want := []int{0, 100, 100}; !reflect.DeepEqual(gotOffsets, want) {
		t.Errorf("offsets = %v; want %v", gotOffsets, want)
	}
}

func TestProxyForwardsRequestAndFlushesSSE(t *testing.T) {
	release := make(chan struct{})
	requestSeen := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/test" {
			_, _ = io.WriteString(w, `{"ok":true}`)
			return
		}
		if r.Method != http.MethodPatch || r.URL.RequestURI() != "/mcp/stream?x=1" {
			t.Errorf("method/path = %s %s", r.Method, r.URL.RequestURI())
		}
		if r.Header.Get("Authorization") != "Bearer caller-token" || r.Header.Get("X-Test") != "unchanged" {
			t.Errorf("forwarded headers = %v", r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, []byte("raw-body")) {
			t.Errorf("body = %q", body)
		}
		requestSeen <- struct{}{}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "data: second\n\n")
	}))
	defer upstream.Close()

	proxy, err := NewProxy(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()

	request, err := http.NewRequest(http.MethodPatch, server.URL+"/mcp/stream?x=1", bytes.NewReader([]byte("raw-body")))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer caller-token")
	request.Header.Set("X-Test", "unchanged")
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	select {
	case <-requestSeen:
	case <-time.After(time.Second):
		t.Fatal("upstream did not receive request")
	}
	line := make(chan string, 1)
	go func() {
		got, _ := bufio.NewReader(response.Body).ReadString('\n')
		line <- got
	}()
	select {
	case got := <-line:
		if got != "data: first\n" {
			t.Errorf("first SSE line = %q", got)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("SSE response was buffered")
	}
	close(release)

	metadataResponse, err := http.Get(server.URL + "/.well-known/test")
	if err != nil {
		t.Fatal(err)
	}
	defer metadataResponse.Body.Close()
	if metadataResponse.StatusCode != http.StatusOK {
		t.Fatalf("well-known status = %d", metadataResponse.StatusCode)
	}
}
