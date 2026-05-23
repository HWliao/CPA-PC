package store

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"strings"

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
	aliases, err := s.LoadAPIKeyAliases(ctx)
	if err != nil {
		return usage.ChartsResponse{}, err
	}
	aliasLabels := apiKeyAliasMap(aliases)

	statement, args := usageChartQuerySQL(response.StartMS, response.EndMS, query)
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return usage.ChartsResponse{}, err
	}
	defer rows.Close()

	providers := map[string]struct{}{}
	authFiles := map[string]usage.ChartAuthFileOption{}
	apiKeys := map[string]usage.ChartAPIKeyOption{}
	models := map[string]struct{}{}
	missingPriceModels := map[string]struct{}{}
	providerSeries := map[string]*usage.ChartSeries{}
	apiKeySeries := map[string]*usage.ChartSeries{}
	modelSeries := map[string]*usage.ChartSeries{}

	for rows.Next() {
		var timestampMS int64
		var provider, authIndex, authLabel, apiKeyHash sql.NullString
		var model string
		var inputTokens, outputTokens, cachedTokens, cacheTokens int64
		if err := rows.Scan(&timestampMS, &provider, &model, &authIndex, &authLabel, &apiKeyHash, &inputTokens, &outputTokens, &cachedTokens, &cacheTokens); err != nil {
			return usage.ChartsResponse{}, err
		}
		providerText := strings.TrimSpace(provider.String)
		model = strings.TrimSpace(model)
		authIndexText := strings.TrimSpace(authIndex.String)
		authLabelText := strings.TrimSpace(authLabel.String)
		apiKeyHashText := strings.ToLower(strings.TrimSpace(apiKeyHash.String))

		bucket := bucketForTimestamp(response.Global.Buckets, timestampMS)
		if bucket == nil {
			continue
		}
		if cacheTokens > cachedTokens {
			cachedTokens = cacheTokens
		}
		if providerText != "" {
			providers[providerText] = struct{}{}
		}
		if model != "" {
			models[model] = struct{}{}
			if _, ok := prices[model]; !ok {
				missingPriceModels[model] = struct{}{}
			}
		}
		if authIndexText != "" {
			authFiles[authIndexText] = usage.ChartAuthFileOption{
				AuthIndex: authIndexText,
				Label:     defaultChartLabel(authLabelText, authIndexText),
				Provider:  providerText,
			}
		}
		if apiKeyHashText != "" {
			apiKeys[apiKeyHashText] = usage.ChartAPIKeyOption{
				APIKeyHash: apiKeyHashText,
				Label:      apiKeyDisplayLabel(apiKeyHashText, aliasLabels),
			}
		}

		cost := chartCost(model, inputTokens, outputTokens, cachedTokens, prices)
		bucket.InputTokens += inputTokens
		bucket.OutputTokens += outputTokens
		bucket.CachedTokens += cachedTokens
		bucket.TotalCost += cost

		providerKey := providerText + "\x00" + authIndexText
		providerLabel := providerAuthFileLabel(providerText, authIndexText, authLabelText)
		providerEntry := getChartSeries(providerSeries, providerKey, providerLabel, response, func(series *usage.ChartSeries) {
			series.Provider = providerText
			series.AuthIndex = authIndexText
		})
		addToSeries(providerEntry, timestampMS, inputTokens, outputTokens, cachedTokens, cost)

		apiKeyEntry := getChartSeries(apiKeySeries, apiKeyHashText, apiKeyDisplayLabel(apiKeyHashText, aliasLabels), response, func(series *usage.ChartSeries) {
			series.APIKeyHash = apiKeyHashText
		})
		addToSeries(apiKeyEntry, timestampMS, inputTokens, outputTokens, cachedTokens, cost)

		modelEntry := getChartSeries(modelSeries, model, defaultChartLabel(model, "-"), response, func(series *usage.ChartSeries) {
			series.Model = model
		})
		addToSeries(modelEntry, timestampMS, inputTokens, outputTokens, cachedTokens, cost)
	}
	if err := rows.Err(); err != nil {
		return usage.ChartsResponse{}, err
	}

	computeTPM(response.Global.Buckets)
	response.ByProviderAuthFile.Series = finishSeries(providerSeries)
	response.ByAPIKey.Series = finishSeries(apiKeySeries)
	response.ByModel.Series = finishSeries(modelSeries)
	response.Options.Providers = sortedKeys(providers)
	response.Options.AuthFiles = sortedAuthFileOptions(authFiles)
	response.Options.APIKeys = sortedAPIKeyOptions(apiKeys)
	response.Options.Models = sortedKeys(models)
	response.MissingPriceModels = sortedKeys(missingPriceModels)
	return response, nil
}

func usageChartQuerySQL(startMS int64, endMS int64, query usage.ChartQuery) (string, []any) {
	statement := `select timestamp_ms, provider, model, auth_index, auth_label_snapshot, api_key_hash, input_tokens, output_tokens, cached_tokens, cache_tokens
		from usage_events
		where timestamp_ms >= ? and timestamp_ms <= ?`
	args := []any{startMS, endMS}
	if query.Provider != "" {
		statement += ` and provider = ?`
		args = append(args, query.Provider)
	}
	if query.AuthIndex != "" {
		statement += ` and auth_index = ?`
		args = append(args, query.AuthIndex)
	}
	if query.APIKeyHash != "" {
		statement += ` and lower(api_key_hash) = ?`
		args = append(args, query.APIKeyHash)
	}
	if query.Model != "" {
		statement += ` and model = ?`
		args = append(args, query.Model)
	}
	statement += ` order by timestamp_ms asc, id asc`
	return statement, args
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

func getChartSeries(seriesMap map[string]*usage.ChartSeries, key string, label string, response usage.ChartsResponse, assign func(*usage.ChartSeries)) *usage.ChartSeries {
	if key == "" {
		key = "-"
	}
	series := seriesMap[key]
	if series == nil {
		series = &usage.ChartSeries{
			Key:     key,
			Label:   defaultChartLabel(label, key),
			Buckets: usage.BuildChartBuckets(response.StartMS, response.EndMS, response.BucketMS, response.Granularity),
		}
		assign(series)
		seriesMap[key] = series
	}
	return series
}

func addToSeries(series *usage.ChartSeries, timestampMS int64, inputTokens int64, outputTokens int64, cachedTokens int64, cost float64) {
	if series == nil {
		return
	}
	bucket := bucketForTimestamp(series.Buckets, timestampMS)
	if bucket == nil {
		return
	}
	bucket.InputTokens += inputTokens
	bucket.OutputTokens += outputTokens
	bucket.CachedTokens += cachedTokens
	bucket.TotalCost += cost
}

func finishSeries(seriesMap map[string]*usage.ChartSeries) []usage.ChartSeries {
	series := make([]usage.ChartSeries, 0, len(seriesMap))
	for _, item := range seriesMap {
		computeTPM(item.Buckets)
		series = append(series, *item)
	}
	sort.Slice(series, func(i, j int) bool {
		if series[i].Label == series[j].Label {
			return series[i].Key < series[j].Key
		}
		return series[i].Label < series[j].Label
	})
	return series
}

func apiKeyAliasMap(aliases []APIKeyAlias) map[string]string {
	result := map[string]string{}
	for _, alias := range aliases {
		hash := strings.ToLower(strings.TrimSpace(alias.APIKeyHash))
		label := strings.TrimSpace(alias.Alias)
		if hash != "" && label != "" {
			result[hash] = label
		}
	}
	return result
}

func apiKeyDisplayLabel(apiKeyHash string, aliases map[string]string) string {
	hash := strings.ToLower(strings.TrimSpace(apiKeyHash))
	if hash == "" {
		return "-"
	}
	if alias := strings.TrimSpace(aliases[hash]); alias != "" {
		return alias
	}
	if len(hash) > 12 {
		return "sha256:" + hash[:12]
	}
	return "sha256:" + hash
}

func providerAuthFileLabel(provider string, authIndex string, authLabel string) string {
	if strings.TrimSpace(authLabel) != "" {
		return strings.TrimSpace(authLabel)
	}
	if strings.TrimSpace(authIndex) != "" {
		return strings.TrimSpace(authIndex)
	}
	return defaultChartLabel(provider, "-")
}

func defaultChartLabel(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func sortedAuthFileOptions(values map[string]usage.ChartAuthFileOption) []usage.ChartAuthFileOption {
	items := make([]usage.ChartAuthFileOption, 0, len(values))
	for _, item := range values {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return items[i].AuthIndex < items[j].AuthIndex
		}
		return items[i].Label < items[j].Label
	})
	return items
}

func sortedAPIKeyOptions(values map[string]usage.ChartAPIKeyOption) []usage.ChartAPIKeyOption {
	items := make([]usage.ChartAPIKeyOption, 0, len(values))
	for _, item := range values {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return items[i].APIKeyHash < items[j].APIKeyHash
		}
		return items[i].Label < items[j].Label
	})
	return items
}
