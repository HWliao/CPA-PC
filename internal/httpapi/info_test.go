package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	pcstore "github.com/HWliao/CPA-PC/internal/store"
	"github.com/HWliao/CPA-PC/internal/usage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestRegisterRoutesServesInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine, Info{
		Version:     "test-version",
		BuildDate:   "2026-05-22T00:00:00Z",
		CLIProxyAPI: CLIProxyAPIInfo{Version: "v7.1.20"},
		CPA:         CPAInfo{Host: "", Port: 8317},
		Usage:       UsageInfo{Enabled: true},
	})

	req := httptest.NewRequest(http.MethodGet, "/cpa-pc/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got Info
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Service != "cpa-pc" {
		t.Fatalf("Service = %q, want %q", got.Service, "cpa-pc")
	}
	if got.Version != "test-version" {
		t.Fatalf("Version = %q, want %q", got.Version, "test-version")
	}
	if got.BuildDate != "2026-05-22T00:00:00Z" {
		t.Fatalf("BuildDate = %q, want %q", got.BuildDate, "2026-05-22T00:00:00Z")
	}
	if got.CLIProxyAPI.Version != "v7.1.20" {
		t.Fatalf("CLIProxyAPI.Version = %q, want %q", got.CLIProxyAPI.Version, "v7.1.20")
	}
	if got.CPA.Port != 8317 {
		t.Fatalf("CPA.Port = %d, want 8317", got.CPA.Port)
	}
	if !got.Usage.Enabled {
		t.Fatal("Usage.Enabled = false, want true")
	}
}

func TestRegisterRoutesServesUsageServiceInfo(t *testing.T) {
	g := newTestRouter(nil)

	rec := performRequest(g, http.MethodGet, "/usage-service/info", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["service"] != serviceID || got["mode"] != "embedded" || got["configured"] != true {
		t.Fatalf("info = %#v", got)
	}
}

func TestRegisterRoutesUsageServiceInfoUsesStoredSetupState(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)

	rec := performRequest(g, http.MethodGet, "/usage-service/info", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["configured"] != true {
		t.Fatalf("configured = %#v, want true", got["configured"])
	}

	store.managerConfig = pcstore.ManagerConfig{CPAConnection: pcstore.ManagerCPAConnectionConfig{CPABaseURL: "http://127.0.0.1:8317", ManagementKey: "123456"}}
	store.hasManagerConfig = true
	rec = performRequest(g, http.MethodGet, "/usage-service/info", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["configured"] != true {
		t.Fatalf("configured = %#v, want true", got["configured"])
	}
}

func TestRegisterRoutesUsageServiceInfoReportsUnconfiguredWithoutEmbeddedConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutesWithOptions(engine, RouteOptions{
		Info:  Info{Version: "test", CPA: CPAInfo{Port: 8317}},
		Store: &fakeUsageStore{},
	})

	rec := performRequest(engine, http.MethodGet, "/usage-service/info", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["configured"] != false {
		t.Fatalf("configured = %#v, want false", got["configured"])
	}
}

func TestRegisterRoutesServesUsagePayload(t *testing.T) {
	store := &fakeUsageStore{events: []usage.Event{{
		Timestamp:   "2026-05-21T10:00:00Z",
		Endpoint:    "SDK usage",
		Model:       "gemini-test",
		TotalTokens: 3,
	}}}
	g := newTestRouter(store)

	rec := performRequest(g, http.MethodGet, "/v0/management/usage", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload usage.Payload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TotalRequests != 1 || payload.TotalTokens != 3 {
		t.Fatalf("payload = %#v", payload)
	}
	if store.limit != 123 {
		t.Fatalf("limit = %d, want 123", store.limit)
	}
}

func TestRegisterRoutesServesUsageCharts(t *testing.T) {
	store := &fakeUsageStore{charts: usage.ChartsResponse{
		Range:       usage.ChartRange1H,
		Granularity: usage.ChartGranularity10Minute,
		StartMS:     1_779_000_000_000,
		EndMS:       1_779_000_600_000,
		BucketMS:    int64((10 * time.Minute) / time.Millisecond),
		Global: usage.ChartBucketGroup{Buckets: []usage.ChartMetricBucket{{
			StartMS:      1_779_000_000_000,
			EndMS:        1_779_000_600_000,
			Label:        "10:00",
			InputTokens:  1000,
			OutputTokens: 500,
			CachedTokens: 200,
			TotalCost:    0.0038,
			TPMInput:     1000.0 / 10.0,
			TPMOutput:    500.0 / 10.0,
			TPMCached:    200.0 / 10.0,
		}},
		},
	}}
	g := newTestRouter(store)

	const hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rec := performRequest(g, http.MethodGet, "/v0/management/usage/charts?range=1h&granularity=10m&account=Team%20Codex&apiKeyHash="+hash+"&model=gpt-test", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload usage.ChartsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Range != usage.ChartRange1H || payload.Granularity != usage.ChartGranularity10Minute {
		t.Fatalf("payload range/granularity = %#v", payload)
	}
	if len(payload.Global.Buckets) != 1 || payload.Global.Buckets[0].InputTokens != 1000 {
		t.Fatalf("payload global = %#v", payload.Global)
	}
	if store.chartQuery.Range != usage.ChartRange1H || store.chartQuery.Granularity != usage.ChartGranularity10Minute {
		t.Fatalf("chart query = %#v", store.chartQuery)
	}
	if store.chartQuery.Account != "Team Codex" || store.chartQuery.APIKeyHash != hash || store.chartQuery.Model != "gpt-test" {
		t.Fatalf("chart filters = %#v", store.chartQuery)
	}
}

func TestRegisterRoutesPassesAuthMetadataToUsageCharts(t *testing.T) {
	store := &fakeUsageStore{}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutesWithOptions(engine, RouteOptions{
		Info:  Info{Version: "test", CPA: CPAInfo{Port: 8317}},
		Store: store,
		Config: &pcconfig.Config{
			Usage: pcconfig.Usage{Enabled: true, QueryLimit: 123},
		},
		ManagementKey: "123456",
		ChartAuthMetadataProvider: func(context.Context) []usage.ChartAuthMetadata {
			return []usage.ChartAuthMetadata{{
				AuthIndex: "auth-index-1",
				Account:   "alice@example.com",
				Label:     "Alice OAuth",
				AuthFile:  "alice-auth.json",
			}}
		},
	})

	rec := performRequest(engine, http.MethodGet, "/v0/management/usage/charts", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(store.chartQuery.AuthMetadata) != 1 || store.chartQuery.AuthMetadata[0].Account != "alice@example.com" {
		t.Fatalf("chart auth metadata = %#v", store.chartQuery.AuthMetadata)
	}
}

func TestRegisterRoutesServesEmptyUsageChartsWithoutStore(t *testing.T) {
	g := newTestRouter(nil)

	rec := performRequest(g, http.MethodGet, "/v0/management/usage/charts", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload usage.ChartsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Range != usage.ChartRange1H || payload.Granularity != usage.ChartGranularity10Minute {
		t.Fatalf("payload = %#v", payload)
	}
	if len(payload.Global.Buckets) != 6 {
		t.Fatalf("len(global buckets) = %d, want 6", len(payload.Global.Buckets))
	}
}

func TestRegisterRoutesRejectsInvalidUsageChartsQuery(t *testing.T) {
	g := newTestRouter(&fakeUsageStore{})

	for _, path := range []string{
		"/v0/management/usage/charts?range=2h",
		"/v0/management/usage/charts?granularity=minute",
	} {
		t.Run(path, func(t *testing.T) {
			rec := performRequest(g, http.MethodGet, path, "123456")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["code"] != "request_failed" {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func TestRegisterRoutesRejectsInvalidManagementKey(t *testing.T) {
	g := newTestRouter(nil)

	for _, path := range []string{"/v0/management/usage", "/v0/management/usage/charts"} {
		t.Run(path, func(t *testing.T) {
			rec := performRequest(g, http.MethodGet, path, "wrong")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestProtectedUsageServiceRoutesAcceptBcryptHashedManagementKey(t *testing.T) {
	store := &fakeUsageStore{events: []usage.Event{{
		EventHash:   "existing-hash",
		TimestampMS: 1_779_000_000_000,
		Timestamp:   "2026-05-21T00:00:00Z",
		Model:       "gemini-test",
		Endpoint:    "SDK usage",
		TotalTokens: 3,
		CreatedAtMS: 1_779_000_000_001,
	}}}
	g := newTestRouterWithManagementKey(t, store, "123456")
	const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "usage", method: http.MethodGet, path: "/v0/management/usage"},
		{name: "usage charts", method: http.MethodGet, path: "/v0/management/usage/charts"},
		{name: "usage export", method: http.MethodGet, path: "/v0/management/usage/export"},
		{name: "usage import", method: http.MethodPost, path: "/v0/management/usage/import", body: []byte(`{"event_hash":"imported-hash","timestamp_ms":1779000000000,"timestamp":"2026-05-21T00:00:00Z","model":"gemini-test","endpoint":"SDK usage","total_tokens":1,"created_at_ms":1779000000001}`)},
		{name: "model prices", method: http.MethodGet, path: "/v0/management/model-prices"},
		{name: "save model prices", method: http.MethodPut, path: "/v0/management/model-prices", body: []byte(`{"prices":{"gpt-test":{"prompt":1}}}`)},
		{name: "sync model prices", method: http.MethodPost, path: "/v0/management/model-prices/sync", body: []byte(`{}`)},
		{name: "api key aliases", method: http.MethodGet, path: "/v0/management/api-key-aliases"},
		{name: "save api key aliases", method: http.MethodPut, path: "/v0/management/api-key-aliases", body: []byte(`{"items":[{"apiKeyHash":"` + hash + `","alias":"Team A"}]}`)},
		{name: "delete api key alias", method: http.MethodDelete, path: "/v0/management/api-key-aliases/" + hash},
		{name: "manager config", method: http.MethodGet, path: "/usage-service/config"},
		{name: "save manager config", method: http.MethodPut, path: "/usage-service/config", body: []byte(`{"config":{"cpaConnection":{"cpaBaseUrl":"http://127.0.0.1:8317","managementKey":"123456"}}}`)},
		{name: "status", method: http.MethodGet, path: "/status"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := performRequestWithBody(g, tc.method, tc.path, "123456", tc.body)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

func TestRegisterRoutesSetupAcceptsBcryptHashedManagementKeyFromBody(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouterWithManagementKey(t, store, "123456")
	body := []byte(`{"cpaBaseUrl":"http://127.0.0.1:8317","managementKey":"123456"}`)

	rec := performRequestWithBody(g, http.MethodPost, "/setup", "", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRegisterRoutesRejectsRemoteManagementWhenDisabled(t *testing.T) {
	g := newTestRouter(nil)

	rec := performRequestFrom(g, http.MethodGet, "/status", "123456", nil, "203.0.113.10:1234")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestRegisterRoutesAcceptsManagementPasswordEnv(t *testing.T) {
	t.Setenv("MANAGEMENT_PASSWORD", "env-secret")
	g := newTestRouter(nil)

	rec := performRequestFrom(g, http.MethodGet, "/status", "env-secret", nil, "203.0.113.10:1234")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRegisterRoutesServesStatus(t *testing.T) {
	store := &fakeUsageStore{countEvents: 2, countDeadLetters: 1}
	g := newTestRouter(store)

	rec := performRequest(g, http.MethodGet, "/status", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["events"] != float64(2) || got["deadLetters"] != float64(1) || got["service"] != serviceID {
		t.Fatalf("status payload = %#v", got)
	}
}

func TestRegisterRoutesPersistsManagerConfig(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)
	body := []byte(`{"config":{"cpaConnection":{"cpaBaseUrl":"http://127.0.0.1:8317","managementKey":"secret"},"collector":{"collectorMode":"sdk-plugin","queue":"sdk-plugin","popSide":"none","batchSize":1,"queryLimit":42},"externalUsageService":{"enabled":false}}}`)

	rec := performRequestWithBody(g, http.MethodPut, "/usage-service/config", "123456", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !store.hasManagerConfig || store.managerConfig.Collector.QueryLimit != 42 {
		t.Fatalf("manager config not persisted: %#v", store.managerConfig)
	}

	rec = performRequest(g, http.MethodGet, "/usage-service/config", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got struct {
		Config pcstore.ManagerConfig `json:"config"`
		Source string                `json:"source"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "db" || got.Config.CPAConnection.CPABaseURL != "http://127.0.0.1:8317" || got.Config.Collector.QueryLimit != 42 {
		t.Fatalf("config response = %#v", got)
	}
}

func TestRegisterRoutesManagerConfigIncludesEmbeddedUsageStatus(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)

	rec := performRequest(g, http.MethodGet, "/usage-service/config", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got struct {
		Config   pcstore.ManagerConfig `json:"config"`
		Source   string                `json:"source"`
		CPAUsage struct {
			UsageStatisticsEnabled          bool `json:"usageStatisticsEnabled"`
			RedisUsageQueueRetentionSeconds int  `json:"redisUsageQueueRetentionSeconds"`
			RetentionSourceDefault          bool `json:"retentionSourceDefault"`
		} `json:"cpaUsage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "embedded" || got.Config.Collector.Enabled == nil || !*got.Config.Collector.Enabled {
		t.Fatalf("manager config = %#v", got)
	}
	if !got.CPAUsage.UsageStatisticsEnabled || got.CPAUsage.RedisUsageQueueRetentionSeconds != 60 || !got.CPAUsage.RetentionSourceDefault {
		t.Fatalf("cpa usage = %#v", got.CPAUsage)
	}
}

func TestRegisterRoutesSetupPersistsManagerConfig(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)
	body := []byte(`{"cpaBaseUrl":"http://127.0.0.1:8317","managementKey":"123456","collectorMode":"sdk-plugin","queue":"sdk-plugin","popSide":"none","batchSize":1,"queryLimit":55,"requestMonitoringEnabled":true}`)

	rec := performRequestWithBody(g, http.MethodPost, "/setup", "", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !store.hasManagerConfig || store.managerConfig.CPAConnection.ManagementKey != "123456" || store.managerConfig.Collector.QueryLimit != 55 {
		t.Fatalf("setup config = %#v", store.managerConfig)
	}
}

func TestRegisterRoutesSetupRejectsInvalidManagementKey(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)
	body := []byte(`{"cpaBaseUrl":"http://127.0.0.1:8317","managementKey":"wrong"}`)

	rec := performRequestWithBody(g, http.MethodPost, "/setup", "", body)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	if store.hasManagerConfig {
		t.Fatalf("setup unexpectedly persisted: %#v", store.managerConfig)
	}
}

func TestRegisterRoutesServesModelPrices(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)
	body := []byte(`{"prices":{"gpt-test":{"prompt":1.25,"completion":2.5,"cache":0.1}}}`)

	rec := performRequestWithBody(g, http.MethodPut, "/v0/management/model-prices", "123456", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	rec = performRequest(g, http.MethodGet, "/v0/management/model-prices", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got struct {
		Prices map[string]pcstore.ModelPrice `json:"prices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	price := got.Prices["gpt-test"]
	if price.Prompt != 1.25 || price.Completion != 2.5 || price.Cache != 0.1 {
		t.Fatalf("prices = %#v", got.Prices)
	}
}

func TestRegisterRoutesSyncModelPricesReturnsStoredPrices(t *testing.T) {
	store := &fakeUsageStore{modelPrices: map[string]pcstore.ModelPrice{"gpt-test": {Prompt: 1}}}
	g := newTestRouter(store)

	rec := performRequestWithBody(g, http.MethodPost, "/v0/management/model-prices/sync", "123456", []byte(`{}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got struct {
		Source   string                        `json:"source"`
		Imported int                           `json:"imported"`
		Skipped  int                           `json:"skipped"`
		Prices   map[string]pcstore.ModelPrice `json:"prices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "embedded" || got.Imported != 0 || got.Skipped != 0 || got.Prices["gpt-test"].Prompt != 1 {
		t.Fatalf("sync response = %#v", got)
	}
}

func TestRegisterRoutesServesAPIKeyAliases(t *testing.T) {
	store := &fakeUsageStore{}
	g := newTestRouter(store)
	const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	body := []byte(`{"items":[{"apiKeyHash":"` + hash + `","alias":"Team A"}]}`)

	rec := performRequestWithBody(g, http.MethodPut, "/v0/management/api-key-aliases", "123456", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	rec = performRequest(g, http.MethodGet, "/v0/management/api-key-aliases", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got struct {
		Items []pcstore.APIKeyAlias `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].APIKeyHash != hash || got.Items[0].Alias != "Team A" {
		t.Fatalf("aliases = %#v", got.Items)
	}

	rec = performRequest(g, http.MethodDelete, "/v0/management/api-key-aliases/"+hash, "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(store.aliases) != 0 {
		t.Fatalf("aliases after delete = %#v", store.aliases)
	}
}

func TestRegisterRoutesExportsAndImportsUsageJSONL(t *testing.T) {
	store := &fakeUsageStore{events: []usage.Event{{
		RequestID:   "req-1",
		EventHash:   "event-hash-1",
		TimestampMS: 1_779_000_000_000,
		Timestamp:   "2026-05-21T00:00:00Z",
		Model:       "gemini-test",
		Endpoint:    "SDK usage",
		TotalTokens: 3,
		CreatedAtMS: 1_779_000_000_001,
	}}}
	g := newTestRouter(store)

	rec := performRequest(g, http.MethodGet, "/v0/management/usage/export", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "application/x-ndjson" {
		t.Fatalf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}

	store.events = nil
	rec = performRequestWithBody(g, http.MethodPost, "/v0/management/usage/import", "123456", rec.Body.Bytes())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got struct {
		Format  string `json:"format"`
		Added   int    `json:"added"`
		Skipped int    `json:"skipped"`
		Total   int    `json:"total"`
		Failed  int    `json:"failed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Format != "usage_service_jsonl" || got.Added != 1 || got.Total != 1 || got.Failed != 0 || len(store.events) != 1 {
		t.Fatalf("import response = %#v events=%#v", got, store.events)
	}
}

func newTestRouter(store UsageStore) *gin.Engine {
	return newTestRouterWithOptions(store, "123456")
}

func newTestRouterWithManagementKey(t *testing.T, store UsageStore, key string) *gin.Engine {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	return newTestRouterWithOptions(store, string(hashed))
}

func newTestRouterWithOptions(store UsageStore, managementKey string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutesWithOptions(engine, RouteOptions{
		Info:      Info{Version: "test", CPA: CPAInfo{Port: 8317}},
		Store:     store,
		StartedAt: time.UnixMilli(1_779_000_000_000),
		Config: &pcconfig.Config{
			Usage: pcconfig.Usage{Enabled: true, QueryLimit: 123},
			Runtime: pcconfig.RuntimePaths{
				UsageDBPath: "test.sqlite",
			},
		},
		ManagementKey: managementKey,
	})
	return engine
}

func performRequest(engine *gin.Engine, method string, path string, key string) *httptest.ResponseRecorder {
	return performRequestWithBody(engine, method, path, key, nil)
}

func performRequestWithBody(engine *gin.Engine, method string, path string, key string, body []byte) *httptest.ResponseRecorder {
	return performRequestFrom(engine, method, path, key, body, "127.0.0.1:1234")
}

func performRequestFrom(engine *gin.Engine, method string, path string, key string, body []byte, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.RemoteAddr = remoteAddr
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

type fakeUsageStore struct {
	events           []usage.Event
	limit            int
	countEvents      int64
	countDeadLetters int64
	managerConfig    pcstore.ManagerConfig
	hasManagerConfig bool
	modelPrices      map[string]pcstore.ModelPrice
	aliases          []pcstore.APIKeyAlias
	charts           usage.ChartsResponse
	chartQuery       usage.ChartQuery
}

func (s *fakeUsageStore) RecentEvents(_ context.Context, limit int) ([]usage.Event, error) {
	s.limit = limit
	return s.events, nil
}

func (s *fakeUsageStore) UsageCharts(_ context.Context, query usage.ChartQuery) (usage.ChartsResponse, error) {
	s.chartQuery = query
	if s.charts.Range == "" {
		return usage.EmptyChartsResponse(query), nil
	}
	return s.charts, nil
}

func (s *fakeUsageStore) ExportEvents(context.Context) ([]usage.Event, error) {
	return s.events, nil
}

func (s *fakeUsageStore) Counts(context.Context) (int64, int64, error) {
	return s.countEvents, s.countDeadLetters, nil
}

func (s *fakeUsageStore) InsertEvents(_ context.Context, events []usage.Event) (usage.InsertResult, error) {
	result := usage.InsertResult{}
	seen := map[string]struct{}{}
	for _, event := range s.events {
		seen[event.EventHash] = struct{}{}
	}
	for _, event := range events {
		if _, ok := seen[event.EventHash]; ok {
			result.Skipped++
			continue
		}
		seen[event.EventHash] = struct{}{}
		s.events = append(s.events, event)
		result.Inserted++
	}
	return result, nil
}

func (s *fakeUsageStore) LoadManagerConfig(context.Context) (pcstore.ManagerConfig, bool, error) {
	return s.managerConfig, s.hasManagerConfig, nil
}

func (s *fakeUsageStore) SaveManagerConfig(_ context.Context, cfg pcstore.ManagerConfig) error {
	s.managerConfig = cfg
	s.hasManagerConfig = true
	return nil
}

func (s *fakeUsageStore) LoadModelPrices(context.Context) (map[string]pcstore.ModelPrice, error) {
	if s.modelPrices == nil {
		return map[string]pcstore.ModelPrice{}, nil
	}
	return s.modelPrices, nil
}

func (s *fakeUsageStore) SaveModelPrices(_ context.Context, prices map[string]pcstore.ModelPrice) error {
	s.modelPrices = prices
	return nil
}

func (s *fakeUsageStore) LoadAPIKeyAliases(context.Context) ([]pcstore.APIKeyAlias, error) {
	return s.aliases, nil
}

func (s *fakeUsageStore) UpsertAPIKeyAliases(_ context.Context, aliases []pcstore.APIKeyAlias) error {
	s.aliases = aliases
	return nil
}

func (s *fakeUsageStore) DeleteAPIKeyAlias(_ context.Context, apiKeyHash string) error {
	remaining := s.aliases[:0]
	for _, alias := range s.aliases {
		if alias.APIKeyHash != apiKeyHash {
			remaining = append(remaining, alias)
		}
	}
	s.aliases = remaining
	return nil
}
