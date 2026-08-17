package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"brainhub/adapter/gbrain"
	"brainhub/domain/entity"
)

func TestSourceCreatorCreatesGitRepositoryAndCleansUpFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		root := t.TempDir()
		installFakeGBrain(t, root, true)
		runner, err := gbrain.NewCommandRunner(filepath.Join(root, "audit.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		creator := &sourceCreator{root: root, runner: runner}
		if err := creator.create(context.Background(), entity.SourceID("created-brain")); err != nil {
			t.Fatal(err)
		}
		repositoryPath := filepath.Join(root, "created-brain")
		if _, err := os.Stat(filepath.Join(repositoryPath, "README.md")); err != nil {
			t.Fatal(err)
		}
		command := exec.Command("git", "rev-parse", "HEAD")
		command.Dir = repositoryPath
		if output, err := command.CombinedOutput(); err != nil || len(strings.TrimSpace(string(output))) != 40 {
			t.Fatalf("git commit = %q %v", output, err)
		}
		audit, err := os.ReadFile(filepath.Join(root, "audit.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if lines := bytes.Count(audit, []byte("\n")); lines != 4 {
			t.Fatalf("audit executions = %d; want 4\n%s", lines, audit)
		}
	})

	t.Run("gbrain failure removes directory", func(t *testing.T) {
		root := t.TempDir()
		installFakeGBrain(t, root, false)
		runner, err := gbrain.NewCommandRunner(filepath.Join(root, "audit.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		creator := &sourceCreator{root: root, runner: runner}
		if err := creator.create(context.Background(), entity.SourceID("failed-brain")); err == nil {
			t.Fatal("create unexpectedly succeeded")
		}
		if _, err := os.Stat(filepath.Join(root, "failed-brain")); !os.IsNotExist(err) {
			t.Fatalf("failed source directory remains: %v", err)
		}
	})
}

func TestSourceHandlerBoundary(t *testing.T) {
	root := t.TempDir()
	runner, err := gbrain.NewCommandRunner(filepath.Join(root, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	creator := &sourceCreator{root: root, runner: runner}
	handler := sourceHandler("test-token", creator)

	request := httptest.NewRequest(http.MethodPost, "/internal/sources", strings.NewReader(`{"id":"valid"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", recorder.Code)
	}

	for _, id := range []string{"default", "UPPER", strings.Repeat("a", 33)} {
		request = httptest.NewRequest(http.MethodPost, "/internal/sources", strings.NewReader(`{"id":`+mustJSON(t, id)+`}`))
		request.Header.Set("Authorization", "Bearer test-token")
		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("id %q status = %d", id, recorder.Code)
		}
	}

	if err := os.Mkdir(filepath.Join(root, "exists"), 0o750); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodPost, "/internal/sources", strings.NewReader(`{"id":"exists"}`))
	request.Header.Set("Authorization", "Bearer test-token")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("existing directory status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func installFakeGBrain(t *testing.T, directory string, success bool) {
	t.Helper()
	exit := "0"
	if !success {
		exit = "1"
	}
	script := filepath.Join(directory, "gbrain")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit "+exit+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+":"+os.Getenv("PATH"))
}

func mustJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
