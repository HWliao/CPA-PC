package usage

import (
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
