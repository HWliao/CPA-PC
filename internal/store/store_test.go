package store

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"

	"github.com/HWliao/CPA-PC/internal/usage"
	_ "modernc.org/sqlite"
)

func TestOpenCreatesExpectedTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "usage.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	for _, table := range []string{"usage_events", "settings", "model_prices", "api_key_aliases", "dead_letter_events"} {
		var name string
		err := sqlDB.QueryRow(`select name from sqlite_master where type = 'table' and name = ?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestStoreModelPrices(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	prices := map[string]ModelPrice{
		"gpt-test": {Prompt: 1.25, Completion: 2.5, Cache: 0.1, Source: "manual"},
	}
	if err := db.SaveModelPrices(context.Background(), prices); err != nil {
		t.Fatal(err)
	}

	loaded, err := db.LoadModelPrices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	price := loaded["gpt-test"]
	if price.Prompt != 1.25 || price.Completion != 2.5 || price.Cache != 0.1 || price.Source != "manual" || price.UpdatedAtMS <= 0 {
		t.Fatalf("price = %#v", price)
	}

	if err := db.SaveModelPrices(context.Background(), map[string]ModelPrice{"bad": {Prompt: math.NaN()}}); err == nil {
		t.Fatal("SaveModelPrices accepted NaN")
	}
	if err := db.SaveModelPrices(context.Background(), map[string]ModelPrice{"": {Prompt: 1}}); err == nil {
		t.Fatal("SaveModelPrices accepted empty model")
	}
}

func TestStoreAPIKeyAliases(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := db.UpsertAPIKeyAliases(context.Background(), []APIKeyAlias{{APIKeyHash: hash, Alias: " Team A "}}); err != nil {
		t.Fatal(err)
	}
	aliases, err := db.LoadAPIKeyAliases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 || aliases[0].APIKeyHash != hash || aliases[0].Alias != "Team A" || aliases[0].UpdatedAtMS <= 0 {
		t.Fatalf("aliases = %#v", aliases)
	}

	const otherHash = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	if err := db.UpsertAPIKeyAliases(context.Background(), []APIKeyAlias{{APIKeyHash: otherHash, Alias: " team a "}}); err == nil || err.Error() != "api key alias already exists" {
		t.Fatalf("duplicate alias error = %v", err)
	}
	if err := db.DeleteAPIKeyAlias(context.Background(), hash); err != nil {
		t.Fatal(err)
	}
	aliases, err = db.LoadAPIKeyAliases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 0 {
		t.Fatalf("aliases after delete = %#v", aliases)
	}
}

func TestStoreManagerConfig(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, ok, err := db.LoadManagerConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("LoadManagerConfig found unexpected config")
	}

	cfg := ManagerConfig{Collector: ManagerCollectorConfig{CollectorMode: "sdk-plugin", QueryLimit: 123}}
	if err := db.SaveManagerConfig(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := db.LoadManagerConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ok || loaded.Collector.CollectorMode != "sdk-plugin" || loaded.Collector.QueryLimit != 123 || loaded.UpdatedAtMS <= 0 {
		t.Fatalf("loaded = %#v ok=%t", loaded, ok)
	}
}

func TestInsertEventsDeduplicatesByEventHash(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	latency := int64(123)
	event := usage.Event{
		RequestID:            "req-1",
		EventHash:            "event-hash-1",
		TimestampMS:          1_779_000_000_000,
		Timestamp:            "2026-05-21T00:00:00Z",
		Provider:             "gemini",
		Model:                "gemini-test",
		Endpoint:             "SDK usage",
		AuthType:             "oauth",
		AuthIndex:            "auth-1",
		Source:               "ali***@example.com",
		SourceHash:           "source-hash",
		APIKeyHash:           "api-key-hash",
		AuthProviderSnapshot: "gemini",
		InputTokens:          1,
		OutputTokens:         2,
		ReasoningTokens:      3,
		CachedTokens:         4,
		CacheTokens:          4,
		TotalTokens:          10,
		LatencyMS:            &latency,
		Failed:               true,
		RawJSON:              `{"ok":true}`,
		CreatedAtMS:          1_779_000_000_001,
	}

	result, err := db.InsertEvents(context.Background(), []usage.Event{event, event})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 1 || result.Skipped != 1 {
		t.Fatalf("InsertEvents result = %#v, want inserted=1 skipped=1", result)
	}

	events, err := db.RecentEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	got := events[0]
	if got.EventHash != event.EventHash || got.Model != event.Model || got.APIKeyHash != event.APIKeyHash {
		t.Fatalf("stored event = %#v", got)
	}
	if got.LatencyMS == nil || *got.LatencyMS != latency {
		t.Fatalf("LatencyMS = %v, want %d", got.LatencyMS, latency)
	}
	if !got.Failed {
		t.Fatal("Failed = false, want true")
	}
}

func TestStoreReopenPersistsEventsAndCounts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "usage.sqlite")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertEvents(context.Background(), []usage.Event{{
		EventHash:   "event-hash-1",
		TimestampMS: 1_779_000_000_000,
		Timestamp:   "2026-05-21T00:00:00Z",
		Model:       "model-a",
		CreatedAtMS: 1_779_000_000_001,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	events, deadLetters, err := reopened.Counts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if events != 1 || deadLetters != 0 {
		t.Fatalf("Counts = (%d, %d), want (1, 0)", events, deadLetters)
	}
}

func TestExportEventsReturnsAllEventsOldestFirst(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.InsertEvents(context.Background(), []usage.Event{
		{EventHash: "event-new", TimestampMS: 3, Timestamp: "2026-05-21T00:00:03Z", Model: "model", CreatedAtMS: 3},
		{EventHash: "event-old", TimestampMS: 1, Timestamp: "2026-05-21T00:00:01Z", Model: "model", CreatedAtMS: 1},
		{EventHash: "event-middle", TimestampMS: 2, Timestamp: "2026-05-21T00:00:02Z", Model: "model", CreatedAtMS: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	recent, err := db.RecentEvents(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(recent))
	}

	exported, err := db.ExportEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(exported) != 3 {
		t.Fatalf("len(exported) = %d, want 3", len(exported))
	}
	if exported[0].EventHash != "event-old" || exported[1].EventHash != "event-middle" || exported[2].EventHash != "event-new" {
		t.Fatalf("export order = %#v", exported)
	}
}
