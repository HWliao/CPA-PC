package usage

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

func TestLogPluginWritesUsageRecord(t *testing.T) {
	var output bytes.Buffer
	plugin := NewLogPlugin(&output)

	plugin.HandleUsage(context.Background(), sdkusage.Record{
		Provider: "gemini",
		Model:    "gemini-test",
		AuthID:   "auth-1",
		AuthType: "oauth",
		Source:   "openai",
		Latency:  150 * time.Millisecond,
		Failed:   true,
		Detail: sdkusage.Detail{
			InputTokens:     10,
			OutputTokens:    20,
			ReasoningTokens: 3,
			CachedTokens:    4,
			TotalTokens:     37,
		},
	})

	line := output.String()
	for _, want := range []string{
		`provider="gemini"`,
		`model="gemini-test"`,
		`input_tokens=10`,
		`output_tokens=20`,
		`reasoning_tokens=3`,
		`cached_tokens=4`,
		`total_tokens=37`,
		`latency_ms=150`,
		`failed=true`,
		`auth_id="auth-1"`,
		`auth_type="oauth"`,
		`source="openai"`,
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("log line = %q, want %s", line, want)
		}
	}
}

func TestPersistPluginWritesConvertedEvent(t *testing.T) {
	store := &fakeEventStore{}
	plugin := NewPersistPlugin(store, nil)
	record := sdkusage.Record{
		Provider:    "gemini",
		Model:       "gemini-test",
		APIKey:      "secret-api-key",
		RequestedAt: time.Date(2026, 5, 21, 10, 30, 0, 0, time.UTC),
		Detail:      sdkusage.Detail{InputTokens: 1, OutputTokens: 2, TotalTokens: 3},
	}

	plugin.HandleUsage(context.Background(), record)

	if len(store.events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(store.events))
	}
	if store.events[0].Model != "gemini-test" || store.events[0].APIKeyHash == "" {
		t.Fatalf("unexpected event: %#v", store.events[0])
	}
}

func TestPersistPluginIgnoresCanceledRequestContext(t *testing.T) {
	store := &fakeEventStore{failOnCanceledCtx: true}
	plugin := NewPersistPlugin(store, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plugin.HandleUsage(ctx, sdkusage.Record{
		Provider:    "openai",
		Model:       "deepseek-v4-flash",
		APIKey:      "secret-api-key",
		RequestedAt: time.Date(2026, 6, 2, 6, 25, 29, 0, time.UTC),
	})

	if len(store.events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(store.events))
	}
	if store.ctx == nil {
		t.Fatal("store ctx is nil")
	}
	if store.ctxErrAtInsert != nil {
		t.Fatalf("store ctx err at insert = %v, want nil", store.ctxErrAtInsert)
	}
}

func TestPersistPluginLogsStoreFailure(t *testing.T) {
	wantErr := errors.New("insert failed")
	store := &fakeEventStore{err: wantErr}
	output := captureLogOutput(t)
	plugin := NewPersistPlugin(store, nil)

	plugin.HandleUsage(context.Background(), sdkusage.Record{Model: "m"})

	line := output.String()
	if !strings.Contains(line, "usage persist failed") || !strings.Contains(line, wantErr.Error()) {
		t.Fatalf("log output = %q, want usage persist failure and error", line)
	}
}

func captureLogOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	logger := log.StandardLogger()
	previousOut := logger.Out
	previousFormatter := logger.Formatter
	log.SetOutput(&output)
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	t.Cleanup(func() {
		log.SetOutput(previousOut)
		log.SetFormatter(previousFormatter)
	})
	return &output
}

type fakeEventStore struct {
	events            []Event
	err               error
	failOnCanceledCtx bool
	ctx               context.Context
	ctxErrAtInsert    error
}

func (s *fakeEventStore) InsertEvents(ctx context.Context, events []Event) (InsertResult, error) {
	s.ctx = ctx
	s.ctxErrAtInsert = ctx.Err()
	if s.err != nil {
		return InsertResult{}, s.err
	}
	if s.failOnCanceledCtx {
		if err := ctx.Err(); err != nil {
			return InsertResult{}, err
		}
	}
	s.events = append(s.events, events...)
	return InsertResult{Inserted: len(events)}, nil
}
