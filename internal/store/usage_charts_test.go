package store

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/HWliao/CPA-PC/internal/usage"
)

func TestUsageChartsBuildsGlobalHourlyBuckets(t *testing.T) {
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
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularityHour,
		NowMS:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if charts.Range != usage.ChartRange1H || charts.Granularity != usage.ChartGranularityHour {
		t.Fatalf("range/granularity = %q/%q", charts.Range, charts.Granularity)
	}
	if len(charts.Global.Buckets) != 1 {
		t.Fatalf("len(global buckets) = %d, want 1", len(charts.Global.Buckets))
	}
	bucket := charts.Global.Buckets[0]
	if bucket.InputTokens != 1000 || bucket.OutputTokens != 500 || bucket.CachedTokens != 200 {
		t.Fatalf("bucket tokens = %#v", bucket)
	}
	assertFloatNear(t, bucket.TotalCost, 0.0038)
	assertFloatNear(t, bucket.TPMInput, 1000.0/60.0)
	assertFloatNear(t, bucket.TPMOutput, 500.0/60.0)
	assertFloatNear(t, bucket.TPMCached, 200.0/60.0)
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

func assertFloatNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("float = %f, want %f", got, want)
	}
}
