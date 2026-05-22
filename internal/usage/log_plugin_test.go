package usage

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
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

func TestPersistPluginLogsStoreFailure(t *testing.T) {
	wantErr := errors.New("insert failed")
	store := &fakeEventStore{err: wantErr}
	var output bytes.Buffer
	plugin := NewPersistPlugin(store, &output)

	plugin.HandleUsage(context.Background(), sdkusage.Record{Model: "m"})

	if !strings.Contains(output.String(), wantErr.Error()) {
		t.Fatalf("output = %q, want error", output.String())
	}
}

type fakeEventStore struct {
	events []Event
	err    error
}

func (s *fakeEventStore) InsertEvents(_ context.Context, events []Event) (InsertResult, error) {
	if s.err != nil {
		return InsertResult{}, s.err
	}
	s.events = append(s.events, events...)
	return InsertResult{Inserted: len(events)}, nil
}
