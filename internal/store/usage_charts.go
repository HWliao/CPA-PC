package store

import (
	"context"
	"math"

	"github.com/HWliao/CPA-PC/internal/usage"
)

const chartTokensPerPriceUnit = 1_000_000

func (s *Store) UsageCharts(ctx context.Context, query usage.ChartQuery) (usage.ChartsResponse, error) {
	query, err := usage.NormalizeChartQuery(query)
	if err != nil {
		return usage.ChartsResponse{}, err
	}

	response := usage.EmptyChartsResponse(query)
	prices, err := s.LoadModelPrices(ctx)
	if err != nil {
		return usage.ChartsResponse{}, err
	}

	rows, err := s.db.QueryContext(ctx, `select timestamp_ms, model, input_tokens, output_tokens, cached_tokens, cache_tokens
		from usage_events
		where timestamp_ms >= ? and timestamp_ms <= ?
		order by timestamp_ms asc, id asc`, response.StartMS, response.EndMS)
	if err != nil {
		return usage.ChartsResponse{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var timestampMS int64
		var model string
		var inputTokens, outputTokens, cachedTokens, cacheTokens int64
		if err := rows.Scan(&timestampMS, &model, &inputTokens, &outputTokens, &cachedTokens, &cacheTokens); err != nil {
			return usage.ChartsResponse{}, err
		}
		bucket := bucketForTimestamp(response.Global.Buckets, timestampMS)
		if bucket == nil {
			continue
		}
		if cacheTokens > cachedTokens {
			cachedTokens = cacheTokens
		}
		bucket.InputTokens += inputTokens
		bucket.OutputTokens += outputTokens
		bucket.CachedTokens += cachedTokens
		bucket.TotalCost += chartCost(model, inputTokens, outputTokens, cachedTokens, prices)
	}
	if err := rows.Err(); err != nil {
		return usage.ChartsResponse{}, err
	}

	computeTPM(response.Global.Buckets)
	return response, nil
}

func bucketForTimestamp(buckets []usage.ChartMetricBucket, timestampMS int64) *usage.ChartMetricBucket {
	for index := range buckets {
		bucket := &buckets[index]
		if timestampMS >= bucket.StartMS && timestampMS < bucket.EndMS {
			return bucket
		}
		if index == len(buckets)-1 && timestampMS == bucket.EndMS {
			return bucket
		}
	}
	return nil
}

func chartCost(model string, inputTokens int64, outputTokens int64, cachedTokens int64, prices map[string]ModelPrice) float64 {
	price, ok := prices[model]
	if !ok {
		return 0
	}
	promptTokens := inputTokens - cachedTokens
	if promptTokens < 0 {
		promptTokens = 0
	}
	total := (float64(promptTokens)/chartTokensPerPriceUnit)*price.Prompt +
		(float64(cachedTokens)/chartTokensPerPriceUnit)*price.Cache +
		(float64(outputTokens)/chartTokensPerPriceUnit)*price.Completion
	if math.IsNaN(total) || math.IsInf(total, 0) || total < 0 {
		return 0
	}
	return total
}

func computeTPM(buckets []usage.ChartMetricBucket) {
	for index := range buckets {
		bucket := &buckets[index]
		minutes := float64(bucket.EndMS-bucket.StartMS) / float64(60_000)
		if minutes <= 0 {
			continue
		}
		bucket.TPMInput = float64(bucket.InputTokens) / minutes
		bucket.TPMOutput = float64(bucket.OutputTokens) / minutes
		bucket.TPMCached = float64(bucket.CachedTokens) / minutes
	}
}
