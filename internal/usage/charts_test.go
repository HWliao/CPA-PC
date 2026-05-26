package usage

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeChartQueryLinksGranularityToRange(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 30, 0, 0, time.UTC).UnixMilli()
	tests := []struct {
		name        string
		chartRange  ChartRange
		granularity ChartGranularity
		bucketMS    int64
		bucketCount int
	}{
		{
			name:        "one hour uses ten minute buckets",
			chartRange:  ChartRange1H,
			granularity: ChartGranularity10Minute,
			bucketMS:    int64((10 * time.Minute) / time.Millisecond),
			bucketCount: 6,
		},
		{
			name:        "five hours uses hourly buckets",
			chartRange:  ChartRange5H,
			granularity: ChartGranularityHour,
			bucketMS:    int64(time.Hour / time.Millisecond),
			bucketCount: 5,
		},
		{
			name:        "twenty four hours uses hourly buckets",
			chartRange:  ChartRange24H,
			granularity: ChartGranularityHour,
			bucketMS:    int64(time.Hour / time.Millisecond),
			bucketCount: 24,
		},
		{
			name:        "seven days uses daily buckets",
			chartRange:  ChartRange7D,
			granularity: ChartGranularityDay,
			bucketMS:    int64((24 * time.Hour) / time.Millisecond),
			bucketCount: 7,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query, err := NormalizeChartQuery(ChartQuery{
				Range:       tc.chartRange,
				Granularity: ChartGranularityHour,
				NowMS:       now,
			})
			if err != nil {
				t.Fatal(err)
			}
			if query.Granularity != tc.granularity {
				t.Fatalf("granularity = %q, want %q", query.Granularity, tc.granularity)
			}

			startMS, endMS, bucketMS := ChartWindow(query)
			if bucketMS != tc.bucketMS {
				t.Fatalf("bucketMS = %d, want %d", bucketMS, tc.bucketMS)
			}
			if buckets := BuildChartBuckets(startMS, endMS, bucketMS, query.Granularity); len(buckets) != tc.bucketCount {
				t.Fatalf("bucket count = %d, want %d", len(buckets), tc.bucketCount)
			}
		})
	}
}

func TestParseChartQueryUsesAccountFilter(t *testing.T) {
	query, err := ParseChartQuery(url.Values{
		"account":    {" Team Codex "},
		"apiKeyHash": {" ABCDEF "},
		"model":      {" gpt-test "},
	})
	if err != nil {
		t.Fatal(err)
	}

	if query.Account != "Team Codex" {
		t.Fatalf("account = %q, want %q", query.Account, "Team Codex")
	}
	if query.APIKeyHash != "abcdef" {
		t.Fatalf("api key hash = %q, want %q", query.APIKeyHash, "abcdef")
	}
	if query.Model != "gpt-test" {
		t.Fatalf("model = %q, want %q", query.Model, "gpt-test")
	}
}

func TestEmptyChartsResponseUsesAccountContract(t *testing.T) {
	response := EmptyChartsResponse(ChartQuery{Account: "Team Codex"})

	if response.Filters.Account != "Team Codex" {
		t.Fatalf("filters.account = %q, want %q", response.Filters.Account, "Team Codex")
	}
	if len(response.Options.Accounts) != 0 {
		t.Fatalf("accounts options = %#v, want empty", response.Options.Accounts)
	}
	if len(response.ByAccount.Series) != 0 {
		t.Fatalf("account series = %#v, want empty", response.ByAccount.Series)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(encoded)
	for _, staleField := range []string{"byProvider", "providers", "provider"} {
		if strings.Contains(jsonText, staleField) {
			t.Fatalf("response JSON contains stale provider field %q: %s", staleField, jsonText)
		}
	}
}
