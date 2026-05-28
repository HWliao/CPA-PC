package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	pcstore "github.com/HWliao/CPA-PC/internal/store"
)

const (
	modelsDevSyncSource        = "model.dev"
	modelsDevAPIURL            = "https://models.dev/api.json"
	maxModelsDevAPIResponseLen = 16 * 1024 * 1024
)

type modelsDevProviderPayload struct {
	ID     string                           `json:"id"`
	Models map[string]modelsDevModelPayload `json:"models"`
}

type modelsDevModelPayload struct {
	ID   string               `json:"id"`
	Cost modelsDevCostPayload `json:"cost"`
}

type modelsDevCostPayload struct {
	Input     *float64 `json:"input"`
	Output    *float64 `json:"output"`
	CacheRead *float64 `json:"cache_read"`
}

func fetchModelsDevPrices(ctx context.Context, client *http.Client, endpoint string) (map[string]map[string]pcstore.ModelPrice, error) {
	if endpoint == "" {
		endpoint = modelsDevAPIURL
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models.dev returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxModelsDevAPIResponseLen+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxModelsDevAPIResponseLen {
		return nil, fmt.Errorf("models.dev response is too large")
	}
	return parseModelsDevPrices(data)
}

func parseModelsDevPrices(data []byte) (map[string]map[string]pcstore.ModelPrice, error) {
	var payload map[string]modelsDevProviderPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	prices := map[string]map[string]pcstore.ModelPrice{}
	for providerKey, provider := range payload {
		providerID := normalizeModelsDevID(provider.ID)
		if providerID == "" {
			providerID = normalizeModelsDevID(providerKey)
		}
		if providerID == "" || len(provider.Models) == 0 {
			continue
		}
		providerPrices := prices[providerID]
		if providerPrices == nil {
			providerPrices = map[string]pcstore.ModelPrice{}
			prices[providerID] = providerPrices
		}

		for modelKey, model := range provider.Models {
			modelID := normalizeModelsDevID(model.ID)
			if modelID == "" {
				modelID = normalizeModelsDevID(modelKey)
			}
			if modelID == "" || model.Cost.Input == nil || model.Cost.Output == nil {
				continue
			}
			input := *model.Cost.Input
			output := *model.Cost.Output
			cacheRead := input / 10
			if model.Cost.CacheRead != nil {
				cacheRead = *model.Cost.CacheRead
			}
			if !validModelsDevCost(input) || !validModelsDevCost(output) || !validModelsDevCost(cacheRead) {
				continue
			}
			providerPrices[modelID] = pcstore.ModelPrice{
				Prompt:     input,
				Completion: output,
				Cache:      cacheRead,
			}
		}
	}
	return prices, nil
}

func syncModelsDevPrices(ctx context.Context, current map[string]pcstore.ModelPrice, targets []modelPriceSyncTarget, endpoint string) (map[string]pcstore.ModelPrice, int, int, error) {
	prices := cloneModelPrices(current)
	if len(targets) == 0 {
		return prices, 0, 0, nil
	}
	catalog, err := fetchModelsDevPrices(ctx, nil, endpoint)
	if err != nil {
		return nil, 0, 0, err
	}

	now := time.Now().UnixMilli()
	imported := 0
	skipped := 0
	for _, target := range targets {
		modelKey := strings.TrimSpace(target.Model)
		modelID := normalizeModelsDevID(modelKey)
		providerID := normalizeModelPriceProvider(target.Provider, modelID)
		if modelKey == "" || modelID == "" || providerID == "" {
			skipped++
			continue
		}
		providerPrices := catalog[providerID]
		price, ok := providerPrices[modelID]
		if !ok {
			skipped++
			continue
		}
		syncedAt := now
		price.Source = modelsDevSyncSource
		price.SourceModelID = providerID + "/" + modelID
		price.SyncedAtMS = &syncedAt
		prices[modelKey] = price
		imported++
	}
	return prices, imported, skipped, nil
}

func cloneModelPrices(prices map[string]pcstore.ModelPrice) map[string]pcstore.ModelPrice {
	cloned := make(map[string]pcstore.ModelPrice, len(prices))
	for model, price := range prices {
		cloned[model] = price
	}
	return cloned
}

func normalizeModelPriceProvider(provider string, modelID string) string {
	providerID := normalizeModelsDevID(provider)
	if providerID == "codex" || strings.HasPrefix(modelID, "gpt-") {
		return "openai"
	}
	return providerID
}

func normalizeModelsDevID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validModelsDevCost(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
