package gbrain

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

func TestShimClientProvision(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		want       error
		wantBearer string
	}{
		{name: "created", status: http.StatusCreated, wantBearer: "Bearer shim-secret"},
		{name: "conflict", status: http.StatusConflict, want: output_port.ErrConflict, wantBearer: "Bearer shim-secret"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/internal/sources" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != test.wantBearer {
					t.Errorf("Authorization = %q", got)
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["id"] != "accounting" {
					t.Errorf("body = %v %v", body, err)
				}
				w.WriteHeader(test.status)
				if test.status != http.StatusCreated {
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "source already exists"})
				}
			}))
			defer server.Close()
			client, err := NewShimClient(server.URL, "shim-secret")
			if err != nil {
				t.Fatal(err)
			}
			err = client.Provision(context.Background(), entity.SourceID("accounting"))
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v; want %v", err, test.want)
			}
		})
	}
}
