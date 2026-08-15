package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"brainhub/adapter/gbrain"
	"brainhub/domain/constructor"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const (
	listenAddress = ":8081"
	brainsRoot    = "/brains"
	maxBodyBytes  = 16 << 10
)

var errDirectoryExists = errors.New("source directory already exists")

type sourceCreator struct {
	root   string
	runner *gbrain.CommandRunner
}

type createRequest struct {
	ID string `json:"id"`
}

type response struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	token := os.Getenv("SHIM_TOKEN")
	if token == "" {
		log.Fatal("SHIM_TOKEN is required")
	}
	home := os.Getenv("HOME")
	if home == "" {
		log.Fatal("HOME is required")
	}
	auditPath := filepath.Join(home, ".gbrain", "audit", "source-shim.jsonl")
	if err := os.MkdirAll(filepath.Dir(auditPath), 0o700); err != nil {
		log.Fatal("create audit directory: ", err)
	}
	runner, err := gbrain.NewCommandRunner(auditPath)
	if err != nil {
		log.Fatal(err)
	}
	creator := &sourceCreator{root: brainsRoot, runner: runner}
	handler := sourceHandler(token, creator)
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("source shim listening on %s", listenAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func sourceHandler(token string, creator *sourceCreator) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/sources", func(w http.ResponseWriter, r *http.Request) {
		if !validAuthorization(r.Header.Get("Authorization"), token) {
			writeResponse(w, r, http.StatusUnauthorized, response{Error: "unauthorized"})
			return
		}
		var input createRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
			writeResponse(w, r, http.StatusBadRequest, response{Error: "invalid JSON request"})
			return
		}
		sourceID, err := constructor.NewSourceID(input.ID)
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, response{ID: input.ID, Error: "invalid source id"})
			return
		}
		if err := creator.create(r.Context(), sourceID); err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, errDirectoryExists) || errors.Is(err, output_port.ErrConflict) {
				status = http.StatusConflict
			} else {
				var commandError *gbrain.CommandError
				if errors.As(err, &commandError) {
					status = http.StatusBadGateway
				}
			}
			writeResponse(w, r, status, response{ID: input.ID, Error: err.Error()})
			return
		}
		writeResponse(w, r, http.StatusCreated, response{ID: input.ID, Status: "created"})
	})
	return mux
}

func (c *sourceCreator) create(ctx context.Context, sourceID entity.SourceID) (err error) {
	path := filepath.Join(c.root, sourceID.String())
	if err := os.Mkdir(path, 0o750); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s", errDirectoryExists, sourceID.String())
		}
		return fmt.Errorf("create source directory: %w", err)
	}
	created := true
	defer func() {
		if created && err != nil {
			if cleanupErr := os.RemoveAll(path); cleanupErr != nil {
				log.Printf("source cleanup failed id=%q error=%q", sourceID.String(), cleanupErr.Error())
			}
		}
	}()

	readme := []byte("# " + sourceID.String() + "\n")
	if err := os.WriteFile(filepath.Join(path, "README.md"), readme, 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	commands := [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "add", "README.md"},
		{"git", "-c", "user.name=brainhub", "-c", "user.email=brainhub@localhost", "commit", "-m", "Initialize brain"},
	}
	for _, argv := range commands {
		if _, err := c.runner.Run(ctx, path, argv...); err != nil {
			return fmt.Errorf("initialize source repository: %w", err)
		}
	}
	if err := c.runner.AddLocalSource(ctx, sourceID, path); err != nil {
		return fmt.Errorf("register GBrain source: %w", err)
	}
	created = false
	return nil
}

func validAuthorization(value, token string) bool {
	expected := "Bearer " + token
	return len(value) == len(expected) && subtle.ConstantTimeCompare([]byte(value), []byte(expected)) == 1
}

func writeResponse(w http.ResponseWriter, r *http.Request, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
	logged, _ := json.Marshal(body)
	log.Printf("request method=%q path=%q response_status=%d response=%s", r.Method, r.URL.Path, status, strings.TrimSpace(string(logged)))
}
