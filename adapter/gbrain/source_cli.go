package gbrain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const (
	defaultCommandTimeout = 60 * time.Second
	killGracePeriod       = 5 * time.Second
	stdoutTailBytes       = 64 << 10
	stderrTailBytes       = 16 << 10
)

var credentialURLPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://[^:/\s]+:)[^@\s]+@`)

type CommandRunner struct {
	auditPath string
	timeout   time.Duration
	killGrace time.Duration
	redact    []string
	mu        sync.Mutex
}

type CommandResult struct {
	Argv       []string
	CWD        string
	ExitCode   int
	StdoutTail string
	StderrTail string
	Duration   time.Duration
	TimedOut   bool
}

type CommandError struct {
	Result CommandResult
}

func (e *CommandError) Error() string {
	detail := strings.TrimSpace(e.Result.StderrTail)
	if detail == "" {
		detail = strings.TrimSpace(e.Result.StdoutTail)
	}
	if detail == "" {
		detail = "no output"
	}
	if e.Result.TimedOut {
		return fmt.Sprintf("command timed out: %s", detail)
	}
	return fmt.Sprintf("command exited %d: %s", e.Result.ExitCode, detail)
}

func NewCommandRunner(auditPath string) (*CommandRunner, error) {
	if auditPath == "" {
		return nil, errors.New("audit path is required")
	}
	return &CommandRunner{
		auditPath: auditPath,
		timeout:   defaultCommandTimeout,
		killGrace: killGracePeriod,
		redact:    secretEnvironmentValues(),
	}, nil
}

func (r *CommandRunner) Run(ctx context.Context, cwd string, argv ...string) (CommandResult, error) {
	startedAt := time.Now()
	result := CommandResult{Argv: append([]string(nil), argv...), CWD: cwd, ExitCode: -1}
	if len(argv) == 0 {
		return result, errors.New("command argv is required")
	}

	commandContext, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	stdout := newTailBuffer(stdoutTailBytes)
	stderr := newTailBuffer(stderrTailBytes)
	command := exec.Command(argv[0], argv[1:]...)
	command.Dir = cwd
	command.Env = allowedChildEnvironment()
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr

	if err := command.Start(); err != nil {
		result.Duration = time.Since(startedAt)
		result.StderrTail = r.redactText(err.Error())
		if auditErr := r.appendAudit(result); auditErr != nil {
			return result, fmt.Errorf("append command audit: %w", auditErr)
		}
		return result, &CommandError{Result: result}
	}

	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()

	var waitErr error
	select {
	case waitErr = <-wait:
	case <-commandContext.Done():
		result.TimedOut = errors.Is(commandContext.Err(), context.DeadlineExceeded)
		_ = command.Process.Signal(syscall.SIGTERM)
		select {
		case waitErr = <-wait:
		case <-time.After(r.killGrace):
			_ = command.Process.Kill()
			waitErr = <-wait
		}
	}

	result.Duration = time.Since(startedAt)
	result.StdoutTail = r.redactText(stdout.String())
	result.StderrTail = r.redactText(stderr.String())
	if command.ProcessState != nil {
		result.ExitCode = command.ProcessState.ExitCode()
	}
	if err := r.appendAudit(result); err != nil {
		return result, fmt.Errorf("append command audit: %w", err)
	}
	if waitErr != nil || result.TimedOut {
		return result, &CommandError{Result: result}
	}
	return result, nil
}

func (r *CommandRunner) AddLocalSource(ctx context.Context, sourceID entity.SourceID, path string) error {
	result, err := r.Run(ctx, path, "gbrain", "sources", "add", sourceID.String(), "--path", path, "--federated")
	if err == nil {
		return nil
	}
	if strings.Contains(result.StderrTail, fmt.Sprintf(`Source id %q is already registered.`, sourceID.String())) {
		return fmt.Errorf("%w: source %q is already registered", output_port.ErrConflict, sourceID.String())
	}
	return err
}

type auditRecord struct {
	Timestamp  time.Time `json:"timestamp"`
	Argv       []string  `json:"argv"`
	CWD        string    `json:"cwd"`
	ExitCode   int       `json:"exit_code"`
	DurationMS int64     `json:"duration_ms"`
	TimedOut   bool      `json:"timed_out"`
	StdoutTail string    `json:"stdout_tail"`
	StderrTail string    `json:"stderr_tail"`
}

func (r *CommandRunner) appendAudit(result CommandResult) error {
	record, err := json.Marshal(auditRecord{
		Timestamp:  time.Now().UTC(),
		Argv:       result.Argv,
		CWD:        result.CWD,
		ExitCode:   result.ExitCode,
		DurationMS: result.Duration.Milliseconds(),
		TimedOut:   result.TimedOut,
		StdoutTail: result.StdoutTail,
		StderrTail: result.StderrTail,
	})
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	file, err := os.OpenFile(r.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(record, '\n'))
	return err
}

func (r *CommandRunner) redactText(value string) string {
	for _, secret := range r.redact {
		value = strings.ReplaceAll(value, secret, "<REDACTED>")
	}
	return credentialURLPattern.ReplaceAllString(value, `${1}<REDACTED>@`)
}

func allowedChildEnvironment() []string {
	keys := [...]string{"PATH", "HOME", "USER", "LANG", "TZ"}
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func secretEnvironmentValues() []string {
	var values []string
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || len(value) < 4 {
			continue
		}
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "KEY") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "DATABASE_URL") {
			values = append(values, value)
		}
	}
	return values
}

type tailBuffer struct {
	max       int
	data      []byte
	truncated int
}

func newTailBuffer(max int) *tailBuffer {
	return &tailBuffer{max: max}
}

func (b *tailBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	b.data = append(b.data, value...)
	if len(b.data) > b.max {
		drop := len(b.data) - b.max
		b.data = append([]byte(nil), b.data[drop:]...)
		b.truncated += drop
	}
	return originalLength, nil
}

func (b *tailBuffer) String() string {
	data := b.data
	for len(data) > 0 && !utf8.Valid(data) {
		data = data[1:]
	}
	body := strings.ToValidUTF8(string(data), "�")
	if b.truncated == 0 {
		return body
	}
	return fmt.Sprintf("[truncated %d bytes]\n%s", b.truncated, body)
}
