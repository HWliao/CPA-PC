package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseModelsDevPricesMapsCosts(t *testing.T) {
	prices, err := parseModelsDevPrices([]byte(`{
		"openai": {
			"id": "openai",
			"models": {
				"gpt-test": {"id": "gpt-test", "cost": {"input": 2.5, "output": 10, "cache_read": 1.25}},
				"gpt-no-cache": {"id": "gpt-no-cache", "cost": {"input": 3, "output": 12}}
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	gptTest := prices["openai"]["gpt-test"]
	if gptTest.Prompt != 2.5 || gptTest.Completion != 10 || gptTest.Cache != 1.25 {
		t.Fatalf("gpt-test price = %#v", gptTest)
	}
	gptNoCache := prices["openai"]["gpt-no-cache"]
	if gptNoCache.Prompt != 3 || gptNoCache.Completion != 12 || gptNoCache.Cache != 0.3 {
		t.Fatalf("gpt-no-cache price = %#v", gptNoCache)
	}
}

func TestParseModelsDevPricesRejectsInvalidNumericCosts(t *testing.T) {
	_, err := parseModelsDevPrices([]byte(`{
		"openai": {
			"id": "openai",
			"models": {
				"gpt-test": {"id": "gpt-test", "cost": {"input": "free", "output": 10}}
			}
		}
	}`))
	if err == nil {
		t.Fatal("parseModelsDevPrices accepted invalid numeric cost")
	}
}

func TestFetchModelsDevPricesUsesProvidedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api.json" {
			t.Fatalf("path = %q, want /api.json", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"openai":{"id":"openai","models":{"gpt-test":{"id":"gpt-test","cost":{"input":1,"output":2,"cache_read":0.1}}}}}`))
	}))
	defer server.Close()

	prices, err := fetchModelsDevPrices(context.Background(), server.Client(), server.URL+"/api.json")
	if err != nil {
		t.Fatal(err)
	}
	price := prices["openai"]["gpt-test"]
	if price.Prompt != 1 || price.Completion != 2 || price.Cache != 0.1 {
		t.Fatalf("price = %#v", price)
	}
}
