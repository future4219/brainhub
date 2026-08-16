package gbrain

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

func TestCommandRunnerEnvironmentAuditTailAndTimeout(t *testing.T) {
	t.Run("environment and audit", func(t *testing.T) {
		t.Setenv("SHIM_TEST_SECRET", "secret-value-that-must-not-appear")
		auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
		runner, err := NewCommandRunner(auditPath)
		if err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(context.Background(), t.TempDir(), "/usr/bin/env")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(result.StdoutTail, "SHIM_TEST_SECRET") || !strings.Contains(result.StdoutTail, "PATH=") {
			t.Fatalf("child environment = %q", result.StdoutTail)
		}
		audit, err := os.ReadFile(auditPath)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(audit), "secret-value-that-must-not-appear") || !strings.Contains(string(audit), `"argv":["/usr/bin/env"]`) {
			t.Fatalf("audit = %s", audit)
		}
	})

	t.Run("tail", func(t *testing.T) {
		buffer := newTailBuffer(5)
		_, _ = buffer.Write([]byte("123456789"))
		if got := buffer.String(); got != "[truncated 4 bytes]\n56789" {
			t.Fatalf("tail = %q", got)
		}
	})

	t.Run("timeout escalates", func(t *testing.T) {
		runner, err := NewCommandRunner(filepath.Join(t.TempDir(), "audit.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		runner.timeout = 20 * time.Millisecond
		runner.killGrace = 20 * time.Millisecond
		started := time.Now()
		result, err := runner.Run(context.Background(), t.TempDir(), os.Args[0], "-test.run=TestCommandRunnerHelper", "--", "ignore-term")
		if err == nil || !result.TimedOut {
			t.Fatalf("result/error = %+v %v", result, err)
		}
		if time.Since(started) > time.Second {
			t.Fatalf("timeout escalation took %v", time.Since(started))
		}
	})
}

func TestCommandRunnerMapsVerifiedGBrainDuplicate(t *testing.T) {
	directory := t.TempDir()
	script := filepath.Join(directory, "gbrain")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'Source id \"duplicate\" is already registered. Use remove first.' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+":"+os.Getenv("PATH"))
	runner, err := NewCommandRunner(filepath.Join(directory, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.AddLocalSource(context.Background(), entity.SourceID("duplicate"), directory); !errors.Is(err, output_port.ErrConflict) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestCommandRunnerHelper(t *testing.T) {
	for _, argument := range os.Args {
		if argument == "ignore-term" {
			signal.Ignore(syscall.SIGTERM)
			select {}
		}
	}
}
