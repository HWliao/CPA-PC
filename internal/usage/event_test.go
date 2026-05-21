package usage

import (
	"strings"
	"testing"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestEventFromSDKRecordMapsAndMasksFields(t *testing.T) {
	requestedAt := time.Date(2026, 5, 21, 10, 30, 0, 123000000, time.UTC)
	record := sdkusage.Record{
		Provider:    "gemini",
		Model:       "gemini-2.5-pro",
		APIKey:      "sk-secret-api-key-that-must-not-be-stored",
		AuthID:      "auth-id-1",
		AuthIndex:   "auth-index-1",
		AuthType:    "oauth",
		Source:      "alice@example.com",
		RequestedAt: requestedAt,
		Latency:     150 * time.Millisecond,
		Failed:      true,
		Detail: sdkusage.Detail{
			InputTokens:     10,
			OutputTokens:    20,
			ReasoningTokens: 3,
			CachedTokens:    4,
		},
	}

	event := EventFromSDKRecord(record)

	if event.RequestID == "" || !strings.HasPrefix(event.RequestID, "sdk:") {
		t.Fatalf("RequestID = %q, want sdk prefix", event.RequestID)
	}
	if event.EventHash == "" {
		t.Fatal("EventHash is empty")
	}
	if event.TimestampMS != requestedAt.UnixMilli() {
		t.Fatalf("TimestampMS = %d, want %d", event.TimestampMS, requestedAt.UnixMilli())
	}
	if event.Provider != "gemini" || event.Model != "gemini-2.5-pro" || event.Endpoint != "SDK usage" {
		t.Fatalf("unexpected mapped event: %#v", event)
	}
	if event.AuthIndex != "auth-index-1" || event.AuthType != "oauth" {
		t.Fatalf("auth fields not mapped: %#v", event)
	}
	if event.Source != "ali***@example.com" {
		t.Fatalf("Source = %q, want masked email", event.Source)
	}
	if event.SourceHash == "" || event.APIKeyHash == "" {
		t.Fatalf("hashes missing: source=%q api=%q", event.SourceHash, event.APIKeyHash)
	}
	if strings.Contains(event.RawJSON, record.APIKey) || strings.Contains(event.RawJSON, record.AuthID) {
		t.Fatalf("RawJSON contains sensitive material: %s", event.RawJSON)
	}
	if event.TotalTokens != 37 {
		t.Fatalf("TotalTokens = %d, want fallback total 37", event.TotalTokens)
	}
	if event.LatencyMS == nil || *event.LatencyMS != 150 {
		t.Fatalf("LatencyMS = %v, want 150", event.LatencyMS)
	}
	if !event.Failed {
		t.Fatal("Failed = false, want true")
	}
}

func TestEventFromSDKRecordBuildsStableHash(t *testing.T) {
	record := sdkusage.Record{
		Provider:    "codex",
		Model:       "gpt-5",
		APIKey:      "sk-stable",
		AuthIndex:   "auth-index",
		Source:      "sk-stable",
		RequestedAt: time.Date(2026, 5, 21, 10, 30, 0, 0, time.UTC),
		Latency:     time.Second,
		Detail:      sdkusage.Detail{InputTokens: 1, OutputTokens: 2, TotalTokens: 3},
	}

	first := EventFromSDKRecord(record)
	second := EventFromSDKRecord(record)

	if first.EventHash != second.EventHash {
		t.Fatalf("EventHash not stable: %q != %q", first.EventHash, second.EventHash)
	}
	if first.RequestID != second.RequestID {
		t.Fatalf("RequestID not stable: %q != %q", first.RequestID, second.RequestID)
	}
	if first.Source != "m:sk-s...able" {
		t.Fatalf("Source = %q, want masked secret", first.Source)
	}
}

func TestBuildPayloadAggregatesEvents(t *testing.T) {
	latency := int64(42)
	payload := BuildPayload([]Event{
		{
			Timestamp:       "2026-05-21T10:00:00Z",
			Endpoint:        "SDK usage",
			Model:           "gemini-test",
			Source:          "ali***@example.com",
			AuthIndex:       "auth-1",
			APIKeyHash:      "api-hash",
			InputTokens:     1,
			OutputTokens:    2,
			ReasoningTokens: 3,
			CachedTokens:    4,
			CacheTokens:     4,
			TotalTokens:     10,
			LatencyMS:       &latency,
		},
		{
			Timestamp:   "2026-05-21T10:01:00Z",
			Endpoint:    "SDK usage",
			Model:       "gemini-test",
			Failed:      true,
			TotalTokens: 5,
		},
		{
			Timestamp:   "2026-05-21T10:02:00Z",
			Endpoint:    "SDK usage",
			Model:       "codex-test",
			TotalTokens: 7,
		},
	})

	if payload.TotalRequests != 3 || payload.SuccessCount != 2 || payload.FailureCount != 1 || payload.TotalTokens != 22 {
		t.Fatalf("payload totals = %#v", payload)
	}
	apiEntry := payload.APIs["SDK usage"]
	if apiEntry == nil {
		t.Fatal("missing SDK usage aggregate")
	}
	gemini := apiEntry.Models["gemini-test"]
	if gemini == nil || len(gemini.Details) != 2 {
		t.Fatalf("gemini aggregate = %#v", gemini)
	}
	first := gemini.Details[0]
	if first.Tokens.TotalTokens != 10 || first.APIKeyHash != "api-hash" || first.LatencyMS == nil || *first.LatencyMS != latency {
		t.Fatalf("detail = %#v", first)
	}
	if apiEntry.Models["codex-test"] == nil {
		t.Fatal("missing codex model aggregate")
	}
}
