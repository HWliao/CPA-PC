package store

import (
	"context"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/HWliao/CPA-PC/internal/usage"
)

func TestUsageChartsBuildsGlobalTenMinuteBuckets(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	inside := now - 50*60*1000
	outside := now - 2*60*60*1000

	if err := db.SaveModelPrices(context.Background(), map[string]ModelPrice{
		"gpt-test": {Prompt: 2, Completion: 4, Cache: 1},
	}); err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertEvents(context.Background(), []usage.Event{
		{
			EventHash:    "inside-event",
			TimestampMS:  inside,
			Timestamp:    time.UnixMilli(inside).UTC().Format(time.RFC3339Nano),
			Provider:     "openai",
			Model:        "gpt-test",
			Endpoint:     "SDK usage",
			InputTokens:  1000,
			OutputTokens: 500,
			CachedTokens: 200,
			CacheTokens:  200,
			TotalTokens:  1500,
			CreatedAtMS:  inside,
		},
		{
			EventHash:    "outside-event",
			TimestampMS:  outside,
			Timestamp:    time.UnixMilli(outside).UTC().Format(time.RFC3339Nano),
			Provider:     "openai",
			Model:        "gpt-test",
			Endpoint:     "SDK usage",
			InputTokens:  9000,
			OutputTokens: 9000,
			CachedTokens: 9000,
			CacheTokens:  9000,
			TotalTokens:  27000,
			CreatedAtMS:  outside,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range: usage.ChartRange1H,
		NowMS: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if charts.Range != usage.ChartRange1H || charts.Granularity != usage.ChartGranularity10Minute {
		t.Fatalf("range/granularity = %q/%q", charts.Range, charts.Granularity)
	}
	if len(charts.Global.Buckets) != 6 {
		t.Fatalf("len(global buckets) = %d, want 6", len(charts.Global.Buckets))
	}
	bucket := nonZeroBucket(t, charts.Global.Buckets)
	if bucket.InputTokens != 1000 || bucket.OutputTokens != 500 || bucket.CachedTokens != 200 {
		t.Fatalf("bucket tokens = %#v", bucket)
	}
	assertFloatNear(t, bucket.TotalCost, 0.0038)
	assertFloatNear(t, bucket.TPMInput, 1000.0/10.0)
	assertFloatNear(t, bucket.TPMOutput, 500.0/10.0)
	assertFloatNear(t, bucket.TPMCached, 200.0/10.0)
}

func TestUsageChartsReturnsEmptyBuckets(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange5H,
		Granularity: usage.ChartGranularityHour,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(charts.Global.Buckets) != 5 {
		t.Fatalf("len(global buckets) = %d, want 5", len(charts.Global.Buckets))
	}
	for _, bucket := range charts.Global.Buckets {
		if bucket.InputTokens != 0 || bucket.OutputTokens != 0 || bucket.CachedTokens != 0 || bucket.TotalCost != 0 {
			t.Fatalf("empty bucket has data: %#v", bucket)
		}
	}
}

func TestUsageChartsBuildsDimensionSeriesOptionsAndMissingPrices(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	const hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const hashB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	if err := db.SaveModelPrices(context.Background(), map[string]ModelPrice{
		"gpt-priced": {Prompt: 2, Completion: 4, Cache: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAPIKeyAliases(context.Background(), []APIKeyAlias{{
		APIKeyHash: hashA,
		Alias:      "Team A",
	}}); err != nil {
		t.Fatal(err)
	}
	_, err = db.InsertEvents(context.Background(), []usage.Event{
		{
			EventHash:         "openai-event",
			TimestampMS:       now - 20*60*1000,
			Timestamp:         time.UnixMilli(now - 20*60*1000).UTC().Format(time.RFC3339Nano),
			Provider:          "openai",
			Model:             "gpt-priced",
			AuthIndex:         "auth-a",
			AccountSnapshot:   "alice@example.com",
			AuthLabelSnapshot: "Alice",
			APIKeyHash:        hashA,
			InputTokens:       100,
			OutputTokens:      50,
			CachedTokens:      10,
			CacheTokens:       10,
			CreatedAtMS:       now - 20*60*1000,
		},
		{
			EventHash:         "gemini-event",
			TimestampMS:       now - 10*60*1000,
			Timestamp:         time.UnixMilli(now - 10*60*1000).UTC().Format(time.RFC3339Nano),
			Provider:          "gemini",
			Model:             "missing-model",
			AuthIndex:         "auth-b",
			AuthLabelSnapshot: "Bob",
			APIKeyHash:        hashB,
			InputTokens:       200,
			OutputTokens:      75,
			CachedTokens:      20,
			CacheTokens:       20,
			CreatedAtMS:       now - 10*60*1000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := accountOptionLabels(charts.Options.Accounts); !reflect.DeepEqual(got, []string{"Bob", "alice@example.com"}) {
		t.Fatalf("account labels = %#v", got)
	}
	if got := accountOptionValues(charts.Options.Accounts); !reflect.DeepEqual(got, []string{"Bob", "alice@example.com"}) {
		t.Fatalf("account values = %#v", got)
	}
	if got := apiKeyOptionLabels(charts.Options.APIKeys); !reflect.DeepEqual(got, []string{"Team A", "sha256:bbbbbbbbbbbb"}) {
		t.Fatalf("api key labels = %#v", got)
	}
	if got := modelOptionLabels(charts.Options.Models); !reflect.DeepEqual(got, []string{"gpt-priced", "missing-model"}) {
		t.Fatalf("model labels = %#v", got)
	}
	if len(charts.ByAccount.Series) != 2 || len(charts.ByAPIKey.Series) != 2 || len(charts.ByModel.Series) != 2 {
		t.Fatalf("series counts account=%d api=%d model=%d", len(charts.ByAccount.Series), len(charts.ByAPIKey.Series), len(charts.ByModel.Series))
	}
	if got := seriesLabels(charts.ByAccount.Series); !reflect.DeepEqual(got, []string{"Bob", "alice@example.com"}) {
		t.Fatalf("account series labels = %#v", got)
	}
	if got := seriesLabels(charts.ByAPIKey.Series); !reflect.DeepEqual(got, []string{"Team A", "sha256:bbbbbbbbbbbb"}) {
		t.Fatalf("api series labels = %#v", got)
	}
	if got := charts.MissingPriceModels; !reflect.DeepEqual(got, []string{"missing-model"}) {
		t.Fatalf("missing price models = %#v", got)
	}
}

func TestUsageChartsUsesAuthLabelForUUIDAccountSnapshotDisplay(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	const accountID = "550e8400-e29b-41d4-a716-446655440000"
	_, err = db.InsertEvents(context.Background(), []usage.Event{{
		EventHash:         "uuid-account-event",
		TimestampMS:       now - 20*60*1000,
		Timestamp:         time.UnixMilli(now - 20*60*1000).UTC().Format(time.RFC3339Nano),
		Provider:          "openai",
		Model:             "gpt-test",
		AuthIndex:         "auth-a",
		AccountSnapshot:   accountID,
		AuthLabelSnapshot: "Team Codex",
		InputTokens:       100,
		OutputTokens:      50,
		CachedTokens:      10,
		CreatedAtMS:       now - 20*60*1000,
	}})
	if err != nil {
		t.Fatal(err)
	}

	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(charts.Options.Accounts) != 1 {
		t.Fatalf("account options = %#v, want 1", charts.Options.Accounts)
	}
	if got := charts.Options.Accounts[0]; got.Value != accountID || got.Label != "Team Codex" || got.Account != accountID {
		t.Fatalf("account option = %#v", got)
	}
	if len(charts.ByAccount.Series) != 1 {
		t.Fatalf("account series = %#v, want 1", charts.ByAccount.Series)
	}
	if got := charts.ByAccount.Series[0]; got.Key != accountID || got.Label != "Team Codex" || got.Account != accountID {
		t.Fatalf("account series = %#v", got)
	}

	filtered, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		Account:     accountID,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.ByAccount.Series) != 1 || filtered.ByAccount.Series[0].Label != "Team Codex" {
		t.Fatalf("filtered account series = %#v", filtered.ByAccount.Series)
	}
}

func TestUsageChartsSkipsAPIKeySeriesWithoutHash(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	_, err = db.InsertEvents(context.Background(), []usage.Event{{
		EventHash:    "no-api-key-event",
		TimestampMS:  now - 20*60*1000,
		Timestamp:    time.UnixMilli(now - 20*60*1000).UTC().Format(time.RFC3339Nano),
		Provider:     "openai",
		Model:        "gpt-test",
		AuthIndex:    "auth-a",
		InputTokens:  100,
		OutputTokens: 50,
		CachedTokens: 10,
		CreatedAtMS:  now - 20*60*1000,
	}})
	if err != nil {
		t.Fatal(err)
	}

	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(charts.Options.APIKeys) != 0 || len(charts.ByAPIKey.Series) != 0 {
		t.Fatalf("api key dimension should be empty, options=%#v series=%#v", charts.Options.APIKeys, charts.ByAPIKey.Series)
	}
	if len(charts.ByAccount.Series) != 1 || len(charts.ByModel.Series) != 1 {
		t.Fatalf("non-api-key dimensions should remain populated, account=%d model=%d", len(charts.ByAccount.Series), len(charts.ByModel.Series))
	}
}

func TestUsageChartsAppliesCombinedFilters(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "usage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	const hashA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const hashB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	_, err = db.InsertEvents(context.Background(), []usage.Event{
		{
			EventHash:       "selected-event",
			TimestampMS:     now - 20*60*1000,
			Timestamp:       time.UnixMilli(now - 20*60*1000).UTC().Format(time.RFC3339Nano),
			Provider:        "openai",
			Model:           "gpt-selected",
			AuthIndex:       "auth-a",
			AccountSnapshot: "selected@example.com",
			APIKeyHash:      hashA,
			InputTokens:     100,
			OutputTokens:    50,
			CachedTokens:    10,
			CreatedAtMS:     now - 20*60*1000,
		},
		{
			EventHash:       "other-event",
			TimestampMS:     now - 10*60*1000,
			Timestamp:       time.UnixMilli(now - 10*60*1000).UTC().Format(time.RFC3339Nano),
			Provider:        "gemini",
			Model:           "gpt-other",
			AuthIndex:       "auth-b",
			AccountSnapshot: "other@example.com",
			APIKeyHash:      hashB,
			InputTokens:     200,
			OutputTokens:    75,
			CachedTokens:    20,
			CreatedAtMS:     now - 10*60*1000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	charts, err := db.UsageCharts(context.Background(), usage.ChartQuery{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		Account:     "selected@example.com",
		APIKeyHash:  hashA,
		Model:       "gpt-selected",
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	bucket := nonZeroBucket(t, charts.Global.Buckets)
	if bucket.InputTokens != 100 || bucket.OutputTokens != 50 || bucket.CachedTokens != 10 {
		t.Fatalf("filtered global bucket = %#v", bucket)
	}
	if len(charts.ByAccount.Series) != 1 || len(charts.ByAPIKey.Series) != 1 || len(charts.ByModel.Series) != 1 {
		t.Fatalf("filtered series counts account=%d api=%d model=%d", len(charts.ByAccount.Series), len(charts.ByAPIKey.Series), len(charts.ByModel.Series))
	}
	if charts.Filters.Account != "selected@example.com" || charts.Filters.APIKeyHash != hashA || charts.Filters.Model != "gpt-selected" {
		t.Fatalf("filters = %#v", charts.Filters)
	}
}

func nonZeroBucket(t *testing.T, buckets []usage.ChartMetricBucket) usage.ChartMetricBucket {
	t.Helper()
	for _, bucket := range buckets {
		if bucket.InputTokens != 0 || bucket.OutputTokens != 0 || bucket.CachedTokens != 0 || bucket.TotalCost != 0 {
			return bucket
		}
	}
	t.Fatalf("no bucket contains usage values: %#v", buckets)
	return usage.ChartMetricBucket{}
}

func assertFloatNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("float = %f, want %f", got, want)
	}
}

func accountOptionLabels(options []usage.ChartAccountOption) []string {
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	sort.Strings(labels)
	return labels
}

func accountOptionValues(options []usage.ChartAccountOption) []string {
	values := make([]string, 0, len(options))
	for _, option := range options {
		values = append(values, option.Value)
	}
	sort.Strings(values)
	return values
}

func apiKeyOptionLabels(options []usage.ChartAPIKeyOption) []string {
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	sort.Strings(labels)
	return labels
}

func modelOptionLabels(options []usage.ChartModelOption) []string {
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	sort.Strings(labels)
	return labels
}

func seriesLabels(series []usage.ChartSeries) []string {
	labels := make([]string, 0, len(series))
	for _, item := range series {
		labels = append(labels, item.Label)
	}
	sort.Strings(labels)
	return labels
}
