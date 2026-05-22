package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	pcstore "github.com/HWliao/CPA-PC/internal/store"
	"github.com/HWliao/CPA-PC/internal/usage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const serviceID = "cpa-pc"

const (
	modelPriceSyncSource = "embedded"
	usageImportFormat    = "usage_service_jsonl"
	maxUsageImportBytes  = 64 * 1024 * 1024
)

type Info struct {
	Service    string    `json:"service"`
	Mode       string    `json:"mode,omitempty"`
	Version    string    `json:"version"`
	StartedAt  int64     `json:"startedAt,omitempty"`
	Configured bool      `json:"configured"`
	CPA        CPAInfo   `json:"cpa"`
	Usage      UsageInfo `json:"usage"`
}

type CPAInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type UsageInfo struct {
	Enabled    bool   `json:"enabled"`
	DBPath     string `json:"dbPath,omitempty"`
	QueryLimit int    `json:"queryLimit,omitempty"`
}

type UsageStore interface {
	RecentEvents(ctx context.Context, limit int) ([]usage.Event, error)
	ExportEvents(ctx context.Context) ([]usage.Event, error)
	InsertEvents(ctx context.Context, events []usage.Event) (usage.InsertResult, error)
	Counts(ctx context.Context) (events int64, deadLetters int64, err error)
	LoadManagerConfig(ctx context.Context) (pcstore.ManagerConfig, bool, error)
	SaveManagerConfig(ctx context.Context, cfg pcstore.ManagerConfig) error
	LoadModelPrices(ctx context.Context) (map[string]pcstore.ModelPrice, error)
	SaveModelPrices(ctx context.Context, prices map[string]pcstore.ModelPrice) error
	LoadAPIKeyAliases(ctx context.Context) ([]pcstore.APIKeyAlias, error)
	UpsertAPIKeyAliases(ctx context.Context, aliases []pcstore.APIKeyAlias) error
	DeleteAPIKeyAlias(ctx context.Context, apiKeyHash string) error
}

type managerConfigRequest struct {
	Config pcstore.ManagerConfig `json:"config"`
}

type setupRequest struct {
	CPAUpstreamURL           string `json:"cpaBaseUrl"`
	ManagementKey            string `json:"managementKey"`
	CollectorMode            string `json:"collectorMode"`
	Queue                    string `json:"queue"`
	PopSide                  string `json:"popSide"`
	BatchSize                int    `json:"batchSize"`
	PollIntervalMS           int    `json:"pollIntervalMs"`
	QueryLimit               int    `json:"queryLimit"`
	TLSSkipVerify            bool   `json:"tlsSkipVerify"`
	RequestMonitoringEnabled *bool  `json:"requestMonitoringEnabled"`
}

type modelPricesRequest struct {
	Prices map[string]pcstore.ModelPrice `json:"prices"`
}

type apiKeyAliasesRequest struct {
	Items []pcstore.APIKeyAlias `json:"items"`
}

type RouteOptions struct {
	Info          Info
	Store         UsageStore
	Config        *pcconfig.Config
	ManagementKey string
	StartedAt     time.Time
}

func RegisterRoutes(engine *gin.Engine, info Info) {
	RegisterRoutesWithOptions(engine, RouteOptions{Info: info})
}

func RegisterRoutesWithOptions(engine *gin.Engine, opts RouteOptions) {
	if engine == nil {
		return
	}
	info := opts.Info
	if info.Service == "" {
		info.Service = "cpa-pc"
	}
	if info.Mode == "" {
		info.Mode = "embedded"
	}
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.StartedAt == 0 && !opts.StartedAt.IsZero() {
		info.StartedAt = opts.StartedAt.UnixMilli()
	}
	if opts.Config != nil {
		info.Configured = true
		info.Usage.Enabled = opts.Config.Usage.Enabled
		info.Usage.DBPath = opts.Config.Runtime.UsageDBPath
		info.Usage.QueryLimit = opts.Config.Usage.QueryLimit
	}

	engine.GET("/cpa-pc/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, info)
	})
	engine.GET("/usage-service/info", func(c *gin.Context) {
		configured, err := usageServiceConfigured(c.Request.Context(), opts.Store, info.Configured)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage service setup"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"service":    serviceID,
			"mode":       "embedded",
			"startedAt":  info.StartedAt,
			"configured": configured,
		})
	})

	protected := func(c *gin.Context) bool {
		allowed, status, message := authenticateManagementRequest(c, opts)
		if allowed {
			return true
		}
		c.AbortWithStatusJSON(status, gin.H{"error": message})
		return false
	}

	engine.GET("/v0/management/usage", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if opts.Store == nil {
			c.JSON(http.StatusOK, usage.BuildPayload(nil))
			return
		}
		limit := info.Usage.QueryLimit
		if limit <= 0 {
			limit = pcconfig.DefaultUsageQueryLimit
		}
		events, err := opts.Store.RecentEvents(c.Request.Context(), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage events"})
			return
		}
		c.JSON(http.StatusOK, usage.BuildPayload(events))
	})
	engine.GET("/v0/management/usage/export", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		handleUsageExport(c, opts.Store)
	})
	engine.POST("/v0/management/usage/import", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		handleUsageImport(c, opts.Store)
	})

	engine.GET("/v0/management/model-prices", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if opts.Store == nil {
			c.JSON(http.StatusOK, gin.H{"prices": map[string]pcstore.ModelPrice{}})
			return
		}
		prices, err := opts.Store.LoadModelPrices(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model prices"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"prices": prices})
	})
	engine.PUT("/v0/management/model-prices", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if !requireStore(c, opts.Store) {
			return
		}
		var req modelPricesRequest
		if err := decodeJSONBody(c, &req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		if req.Prices == nil {
			writeAPIError(c, http.StatusBadRequest, "prices_required", "prices are required")
			return
		}
		if err := opts.Store.SaveModelPrices(c.Request.Context(), req.Prices); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		prices, err := opts.Store.LoadModelPrices(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model prices"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"prices": prices})
	})
	engine.POST("/v0/management/model-prices/sync", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if opts.Store == nil {
			c.JSON(http.StatusOK, gin.H{
				"source":   modelPriceSyncSource,
				"imported": 0,
				"skipped":  0,
				"prices":   map[string]pcstore.ModelPrice{},
			})
			return
		}
		prices, err := opts.Store.LoadModelPrices(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load model prices"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"source":   modelPriceSyncSource,
			"imported": 0,
			"skipped":  0,
			"prices":   prices,
		})
	})

	engine.GET("/v0/management/api-key-aliases", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if opts.Store == nil {
			c.JSON(http.StatusOK, gin.H{"items": []pcstore.APIKeyAlias{}})
			return
		}
		aliases, err := opts.Store.LoadAPIKeyAliases(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load api key aliases"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": aliases})
	})
	engine.PUT("/v0/management/api-key-aliases", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if !requireStore(c, opts.Store) {
			return
		}
		var req apiKeyAliasesRequest
		if err := decodeJSONBody(c, &req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		if req.Items == nil {
			writeAPIError(c, http.StatusBadRequest, "api_key_aliases_required", "api key aliases are required")
			return
		}
		if err := opts.Store.UpsertAPIKeyAliases(c.Request.Context(), req.Items); err != nil {
			writeAPIError(c, http.StatusBadRequest, "api_key_alias_duplicate", err.Error())
			return
		}
		aliases, err := opts.Store.LoadAPIKeyAliases(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load api key aliases"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": aliases})
	})
	engine.DELETE("/v0/management/api-key-aliases/:apiKeyHash", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if !requireStore(c, opts.Store) {
			return
		}
		if err := opts.Store.DeleteAPIKeyAlias(c.Request.Context(), c.Param("apiKeyHash")); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	engine.GET("/usage-service/config", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		response, err := managerConfigResponse(c.Request.Context(), opts.Store, opts.Config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load manager config"})
			return
		}
		c.JSON(http.StatusOK, response)
	})
	engine.PUT("/usage-service/config", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		if !requireStore(c, opts.Store) {
			return
		}
		var req managerConfigRequest
		if err := decodeJSONBody(c, &req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		cfg := mergeManagerConfig(defaultManagerConfig(opts.Config), req.Config)
		if err := opts.Store.SaveManagerConfig(c.Request.Context(), cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save manager config"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"config": cfg, "source": "db"})
	})
	engine.POST("/setup", func(c *gin.Context) {
		var req setupRequest
		if err := decodeJSONBody(c, &req); err != nil {
			writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
			return
		}
		if allowed, status, message := authenticateManagementRequest(c, opts, req.ManagementKey); !allowed {
			c.AbortWithStatusJSON(status, gin.H{"error": message, "code": "invalid_management_key"})
			return
		}
		managerCfg := managerConfigFromSetup(req, opts.Config, opts.ManagementKey)
		if opts.Store != nil {
			if err := opts.Store.SaveManagerConfig(c.Request.Context(), managerCfg); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setup"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "upstream": managerCfg.CPAConnection.CPABaseURL})
	})
	engine.GET("/status", func(c *gin.Context) {
		if !protected(c) {
			return
		}
		var events int64
		var deadLetters int64
		if opts.Store != nil {
			var err error
			events, deadLetters, err = opts.Store.Counts(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage status"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"service":     serviceID,
			"dbPath":      info.Usage.DBPath,
			"events":      events,
			"deadLetters": deadLetters,
			"collector": gin.H{
				"collector":   "sdk-plugin",
				"mode":        "embedded",
				"transport":   "sdk-plugin",
				"deadLetters": deadLetters,
			},
		})
	})
}

func managementKeyFromRequest(c *gin.Context) string {
	provided := strings.TrimSpace(c.GetHeader("Authorization"))
	if provided != "" {
		parts := strings.SplitN(provided, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			provided = strings.TrimSpace(parts[1])
		}
	}
	if provided == "" {
		provided = strings.TrimSpace(c.GetHeader("X-Management-Key"))
	}
	return provided
}

func authenticateManagementRequest(c *gin.Context, opts RouteOptions, fallbackKeys ...string) (bool, int, string) {
	providedKeys := append([]string{managementKeyFromRequest(c)}, fallbackKeys...)
	configuredKey := strings.TrimSpace(opts.ManagementKey)
	envSecret := strings.TrimSpace(os.Getenv("MANAGEMENT_PASSWORD"))
	allowRemote := envSecret != ""
	if opts.Config != nil && opts.Config.RemoteManagement.AllowRemote {
		allowRemote = true
	}

	clientIP := c.ClientIP()
	localClient := clientIP == "127.0.0.1" || clientIP == "::1"
	if !localClient && !allowRemote {
		return false, http.StatusForbidden, "remote management disabled"
	}
	if configuredKey == "" && envSecret == "" {
		return false, http.StatusForbidden, "remote management key not set"
	}

	hasProvided := false
	for _, provided := range providedKeys {
		provided = strings.TrimSpace(provided)
		if provided == "" {
			continue
		}
		hasProvided = true
		if managementPlainKeyEqual(provided, envSecret) || managementKeyEqual(provided, configuredKey) {
			return true, 0, ""
		}
	}
	if !hasProvided {
		return false, http.StatusUnauthorized, "missing management key"
	}
	return false, http.StatusUnauthorized, "invalid management key"
}

func managementPlainKeyEqual(provided, configured string) bool {
	provided = strings.TrimSpace(provided)
	configured = strings.TrimSpace(configured)
	if provided == "" || configured == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1
}

func managementKeyEqual(provided, configured string) bool {
	provided = strings.TrimSpace(provided)
	configured = strings.TrimSpace(configured)
	if provided == "" || configured == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1 {
		return true
	}
	return bcrypt.CompareHashAndPassword([]byte(configured), []byte(provided)) == nil
}

func decodeJSONBody(c *gin.Context, value any) error {
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(value)
}

func requireStore(c *gin.Context, store UsageStore) bool {
	if store != nil {
		return true
	}
	writeAPIError(c, http.StatusServiceUnavailable, "usage_service_not_configured", "usage store is not available")
	return false
}

func writeAPIError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, gin.H{
		"error": message,
		"code":  code,
	})
}

func usageServiceConfigured(ctx context.Context, store UsageStore, fallback bool) (bool, error) {
	if store == nil {
		return fallback, nil
	}
	cfg, ok, err := store.LoadManagerConfig(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return strings.TrimSpace(cfg.CPAConnection.CPABaseURL) != "" && strings.TrimSpace(cfg.CPAConnection.ManagementKey) != "", nil
}

func managerConfigResponse(ctx context.Context, store UsageStore, cfg *pcconfig.Config) (gin.H, error) {
	defaultConfig := defaultManagerConfig(cfg)
	if store == nil {
		return gin.H{"config": defaultConfig, "source": "embedded"}, nil
	}
	stored, ok, err := store.LoadManagerConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return gin.H{"config": defaultConfig, "source": "embedded"}, nil
	}
	return gin.H{"config": mergeManagerConfig(defaultConfig, stored), "source": "db"}, nil
}

func defaultManagerConfig(cfg *pcconfig.Config) pcstore.ManagerConfig {
	queryLimit := pcconfig.DefaultUsageQueryLimit
	enabled := true
	if cfg != nil {
		if cfg.Usage.QueryLimit > 0 {
			queryLimit = cfg.Usage.QueryLimit
		}
		enabled = cfg.Usage.Enabled
	}
	return pcstore.ManagerConfig{
		CPAConnection: pcstore.ManagerCPAConnectionConfig{
			CPABaseURL: defaultCPABaseURL(cfg),
		},
		Collector: pcstore.ManagerCollectorConfig{
			Enabled:        boolPtr(enabled),
			CollectorMode:  "sdk-plugin",
			Queue:          "sdk-plugin",
			PopSide:        "none",
			BatchSize:      1,
			PollIntervalMS: 0,
			QueryLimit:     queryLimit,
			TLSSkipVerify:  false,
		},
		ExternalUsageService: pcstore.ManagerExternalUsageServiceConfig{
			Enabled:     false,
			ServiceBase: "",
		},
	}
}

func mergeManagerConfig(base pcstore.ManagerConfig, next pcstore.ManagerConfig) pcstore.ManagerConfig {
	if next.CPAConnection.CPABaseURL != "" {
		base.CPAConnection.CPABaseURL = strings.TrimSpace(next.CPAConnection.CPABaseURL)
	}
	if next.CPAConnection.ManagementKey != "" {
		base.CPAConnection.ManagementKey = strings.TrimSpace(next.CPAConnection.ManagementKey)
	}
	if next.Collector.Enabled != nil {
		base.Collector.Enabled = next.Collector.Enabled
	}
	if next.Collector.CollectorMode != "" {
		base.Collector.CollectorMode = strings.TrimSpace(next.Collector.CollectorMode)
	}
	if next.Collector.Queue != "" {
		base.Collector.Queue = strings.TrimSpace(next.Collector.Queue)
	}
	if next.Collector.PopSide != "" {
		base.Collector.PopSide = strings.TrimSpace(next.Collector.PopSide)
	}
	if next.Collector.BatchSize > 0 {
		base.Collector.BatchSize = next.Collector.BatchSize
	}
	if next.Collector.PollIntervalMS > 0 {
		base.Collector.PollIntervalMS = next.Collector.PollIntervalMS
	}
	if next.Collector.QueryLimit > 0 {
		base.Collector.QueryLimit = next.Collector.QueryLimit
	}
	base.Collector.TLSSkipVerify = next.Collector.TLSSkipVerify
	base.ExternalUsageService = next.ExternalUsageService
	base.UpdatedAtMS = next.UpdatedAtMS
	return base
}

func managerConfigFromSetup(req setupRequest, cfg *pcconfig.Config, managementKey string) pcstore.ManagerConfig {
	managerCfg := defaultManagerConfig(cfg)
	if strings.TrimSpace(req.CPAUpstreamURL) != "" {
		managerCfg.CPAConnection.CPABaseURL = strings.TrimSpace(req.CPAUpstreamURL)
	}
	if strings.TrimSpace(req.ManagementKey) != "" {
		managerCfg.CPAConnection.ManagementKey = strings.TrimSpace(req.ManagementKey)
	} else {
		managerCfg.CPAConnection.ManagementKey = strings.TrimSpace(managementKey)
	}
	if strings.TrimSpace(req.CollectorMode) != "" {
		managerCfg.Collector.CollectorMode = strings.TrimSpace(req.CollectorMode)
	}
	if strings.TrimSpace(req.Queue) != "" {
		managerCfg.Collector.Queue = strings.TrimSpace(req.Queue)
	}
	if strings.TrimSpace(req.PopSide) != "" {
		managerCfg.Collector.PopSide = strings.TrimSpace(req.PopSide)
	}
	if req.BatchSize > 0 {
		managerCfg.Collector.BatchSize = req.BatchSize
	}
	if req.PollIntervalMS > 0 {
		managerCfg.Collector.PollIntervalMS = req.PollIntervalMS
	}
	if req.QueryLimit > 0 {
		managerCfg.Collector.QueryLimit = req.QueryLimit
	}
	managerCfg.Collector.TLSSkipVerify = req.TLSSkipVerify
	if req.RequestMonitoringEnabled != nil {
		managerCfg.Collector.Enabled = req.RequestMonitoringEnabled
	}
	return managerCfg
}

func defaultCPABaseURL(cfg *pcconfig.Config) string {
	host := "127.0.0.1"
	port := pcconfig.DefaultPort
	if cfg != nil {
		port = cfg.Port
		trimmedHost := strings.TrimSpace(cfg.Host)
		if trimmedHost != "" && trimmedHost != "0.0.0.0" && trimmedHost != "::" {
			host = trimmedHost
		}
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func boolPtr(value bool) *bool {
	return &value
}

func handleUsageExport(c *gin.Context, store UsageStore) {
	if !requireStore(c, store) {
		return
	}
	events, err := store.ExportEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load usage events"})
		return
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode usage export"})
			return
		}
	}
	c.Writer.Header().Set("Content-Type", "application/x-ndjson")
	c.Writer.Header().Set("Content-Disposition", `attachment; filename="usage-events.jsonl"`)
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(output.Bytes())
}

func handleUsageImport(c *gin.Context, store UsageStore) {
	if !requireStore(c, store) {
		return
	}
	reader := http.MaxBytesReader(c.Writer, c.Request.Body, maxUsageImportBytes)
	data, err := io.ReadAll(reader)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeAPIError(c, http.StatusRequestEntityTooLarge, "request_failed", err.Error())
			return
		}
		writeAPIError(c, http.StatusBadRequest, "request_failed", err.Error())
		return
	}
	parsed, err := parseUsageJSONL(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       err.Error(),
			"format":      usageImportFormat,
			"failed":      parsed.failed,
			"unsupported": 0,
			"warnings":    []string{},
		})
		return
	}
	result, err := store.InsertEvents(c.Request.Context(), parsed.events)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import usage events"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"format":      usageImportFormat,
		"added":       result.Inserted,
		"skipped":     result.Skipped,
		"total":       len(parsed.events),
		"failed":      parsed.failed,
		"unsupported": 0,
		"warnings":    []string{},
	})
}

type usageJSONLParseResult struct {
	events []usage.Event
	failed int
}

func parseUsageJSONL(data []byte) (usageJSONLParseResult, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return usageJSONLParseResult{}, errors.New("empty usage import payload")
	}

	result := usageJSONLParseResult{}
	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event usage.Event
		if err := json.Unmarshal(line, &event); err != nil {
			result.failed++
			continue
		}
		normalized, err := normalizeImportedEvent(event)
		if err != nil {
			result.failed++
			continue
		}
		result.events = append(result.events, normalized)
	}
	if err := scanner.Err(); err != nil {
		return result, err
	}
	if len(result.events) == 0 && result.failed > 0 {
		return result, errors.New("usage import contains no valid events")
	}
	return result, nil
}

func normalizeImportedEvent(event usage.Event) (usage.Event, error) {
	if strings.TrimSpace(event.EventHash) == "" {
		return usage.Event{}, errors.New("event_hash is required")
	}
	event.EventHash = strings.TrimSpace(event.EventHash)
	if strings.TrimSpace(event.Model) == "" {
		event.Model = "-"
	}
	if strings.TrimSpace(event.Endpoint) == "" {
		event.Endpoint = "-"
	}
	if event.TimestampMS <= 0 && strings.TrimSpace(event.Timestamp) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(event.Timestamp))
		if err == nil {
			event.TimestampMS = parsed.UTC().UnixMilli()
		}
	}
	if strings.TrimSpace(event.Timestamp) == "" && event.TimestampMS > 0 {
		event.Timestamp = time.UnixMilli(event.TimestampMS).UTC().Format(time.RFC3339Nano)
	}
	if event.TimestampMS <= 0 || strings.TrimSpace(event.Timestamp) == "" {
		return usage.Event{}, errors.New("timestamp is required")
	}
	if event.TotalTokens <= 0 {
		event.TotalTokens = event.InputTokens + event.OutputTokens + event.ReasoningTokens + event.CachedTokens
	}
	if event.CreatedAtMS <= 0 {
		event.CreatedAtMS = time.Now().UnixMilli()
	}
	return event, nil
}
