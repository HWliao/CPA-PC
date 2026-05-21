package usage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

type Event struct {
	RequestID            string `json:"request_id,omitempty"`
	EventHash            string `json:"event_hash"`
	TimestampMS          int64  `json:"timestamp_ms"`
	Timestamp            string `json:"timestamp"`
	Provider             string `json:"provider,omitempty"`
	Model                string `json:"model"`
	Endpoint             string `json:"endpoint,omitempty"`
	Method               string `json:"method,omitempty"`
	Path                 string `json:"path,omitempty"`
	AuthType             string `json:"auth_type,omitempty"`
	AuthIndex            string `json:"auth_index,omitempty"`
	Source               string `json:"source,omitempty"`
	SourceHash           string `json:"source_hash,omitempty"`
	APIKeyHash           string `json:"api_key_hash,omitempty"`
	AccountSnapshot      string `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string `json:"auth_label_snapshot,omitempty"`
	AuthFileSnapshot     string `json:"auth_file_snapshot,omitempty"`
	AuthProviderSnapshot string `json:"auth_provider_snapshot,omitempty"`
	AuthSnapshotAtMS     int64  `json:"auth_snapshot_at_ms,omitempty"`
	InputTokens          int64  `json:"input_tokens"`
	OutputTokens         int64  `json:"output_tokens"`
	ReasoningTokens      int64  `json:"reasoning_tokens"`
	CachedTokens         int64  `json:"cached_tokens"`
	CacheTokens          int64  `json:"cache_tokens"`
	TotalTokens          int64  `json:"total_tokens"`
	LatencyMS            *int64 `json:"latency_ms,omitempty"`
	Failed               bool   `json:"failed"`
	RawJSON              string `json:"raw_json,omitempty"`
	CreatedAtMS          int64  `json:"created_at_ms"`
}

type InsertResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

type Tokens struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	CacheTokens     int64 `json:"cache_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

type Detail struct {
	Timestamp            string `json:"timestamp"`
	Source               string `json:"source"`
	AuthIndex            string `json:"auth_index,omitempty"`
	APIKeyHash           string `json:"api_key_hash,omitempty"`
	AccountSnapshot      string `json:"account_snapshot,omitempty"`
	AuthLabelSnapshot    string `json:"auth_label_snapshot,omitempty"`
	AuthFileSnapshot     string `json:"auth_file_snapshot,omitempty"`
	AuthProviderSnapshot string `json:"auth_provider_snapshot,omitempty"`
	AuthSnapshotAtMS     int64  `json:"auth_snapshot_at_ms,omitempty"`
	LatencyMS            *int64 `json:"latency_ms,omitempty"`
	Tokens               Tokens `json:"tokens"`
	Failed               bool   `json:"failed"`
}

type ModelAggregate struct {
	Details []Detail `json:"details"`
}

type APIAggregate struct {
	Models map[string]*ModelAggregate `json:"models"`
}

type Payload struct {
	TotalRequests int64                    `json:"total_requests"`
	SuccessCount  int64                    `json:"success_count"`
	FailureCount  int64                    `json:"failure_count"`
	TotalTokens   int64                    `json:"total_tokens"`
	APIs          map[string]*APIAggregate `json:"apis"`
}

func BuildPayload(events []Event) Payload {
	payload := Payload{APIs: map[string]*APIAggregate{}}
	for _, event := range events {
		payload.TotalRequests++
		if event.Failed {
			payload.FailureCount++
		} else {
			payload.SuccessCount++
		}
		payload.TotalTokens += event.TotalTokens

		endpoint := defaultString(event.Endpoint, "-")
		apiEntry := payload.APIs[endpoint]
		if apiEntry == nil {
			apiEntry = &APIAggregate{Models: map[string]*ModelAggregate{}}
			payload.APIs[endpoint] = apiEntry
		}

		model := defaultString(event.Model, "-")
		modelEntry := apiEntry.Models[model]
		if modelEntry == nil {
			modelEntry = &ModelAggregate{}
			apiEntry.Models[model] = modelEntry
		}
		modelEntry.Details = append(modelEntry.Details, Detail{
			Timestamp:            event.Timestamp,
			Source:               event.Source,
			AuthIndex:            event.AuthIndex,
			APIKeyHash:           event.APIKeyHash,
			AccountSnapshot:      event.AccountSnapshot,
			AuthLabelSnapshot:    event.AuthLabelSnapshot,
			AuthFileSnapshot:     event.AuthFileSnapshot,
			AuthProviderSnapshot: event.AuthProviderSnapshot,
			AuthSnapshotAtMS:     event.AuthSnapshotAtMS,
			LatencyMS:            event.LatencyMS,
			Failed:               event.Failed,
			Tokens: Tokens{
				InputTokens:     event.InputTokens,
				OutputTokens:    event.OutputTokens,
				ReasoningTokens: event.ReasoningTokens,
				CachedTokens:    event.CachedTokens,
				CacheTokens:     event.CacheTokens,
				TotalTokens:     event.TotalTokens,
			},
		})
	}
	return payload
}

func EventFromSDKRecord(record sdkusage.Record) Event {
	requestedAt := record.RequestedAt
	if requestedAt.IsZero() {
		requestedAt = time.Now()
	}
	requestedAt = requestedAt.UTC()

	latencyMS := record.Latency.Milliseconds()
	if latencyMS < 0 {
		latencyMS = 0
	}

	totalTokens := record.Detail.TotalTokens
	if totalTokens <= 0 {
		totalTokens = record.Detail.InputTokens + record.Detail.OutputTokens + record.Detail.ReasoningTokens + record.Detail.CachedTokens
	}

	authIndex := strings.TrimSpace(record.AuthIndex)
	if authIndex == "" {
		authIndex = strings.TrimSpace(record.AuthID)
	}

	event := Event{
		TimestampMS:          requestedAt.UnixMilli(),
		Timestamp:            requestedAt.Format(time.RFC3339Nano),
		Provider:             strings.TrimSpace(record.Provider),
		Model:                defaultString(record.Model, "-"),
		Endpoint:             defaultString("SDK usage", "-"),
		AuthType:             strings.TrimSpace(record.AuthType),
		AuthIndex:            authIndex,
		Source:               maskSource(record.Source),
		SourceHash:           hashString(record.Source),
		APIKeyHash:           hashString(record.APIKey),
		AuthProviderSnapshot: strings.TrimSpace(record.Provider),
		InputTokens:          record.Detail.InputTokens,
		OutputTokens:         record.Detail.OutputTokens,
		ReasoningTokens:      record.Detail.ReasoningTokens,
		CachedTokens:         record.Detail.CachedTokens,
		CacheTokens:          record.Detail.CachedTokens,
		TotalTokens:          totalTokens,
		LatencyMS:            &latencyMS,
		Failed:               record.Failed,
		CreatedAtMS:          time.Now().UnixMilli(),
	}
	event.RequestID = buildRequestID(event)
	event.EventHash = buildEventHash(event)
	event.RawJSON = buildRawJSON(record, event)
	return event
}

func buildRawJSON(record sdkusage.Record, event Event) string {
	raw := map[string]any{
		"provider":         event.Provider,
		"model":            event.Model,
		"source":           event.Source,
		"source_hash":      event.SourceHash,
		"api_key_hash":     event.APIKeyHash,
		"auth_id_hash":     hashString(record.AuthID),
		"auth_index":       event.AuthIndex,
		"auth_type":        event.AuthType,
		"requested_at":     event.Timestamp,
		"latency_ms":       valueOrZero(event.LatencyMS),
		"failed":           event.Failed,
		"input_tokens":     event.InputTokens,
		"output_tokens":    event.OutputTokens,
		"reasoning_tokens": event.ReasoningTokens,
		"cached_tokens":    event.CachedTokens,
		"total_tokens":     event.TotalTokens,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	return string(data)
}

func buildRequestID(event Event) string {
	parts := []string{
		event.Timestamp,
		event.Provider,
		event.Model,
		event.AuthIndex,
		event.SourceHash,
		strconv.FormatInt(event.InputTokens, 10),
		strconv.FormatInt(event.OutputTokens, 10),
		strconv.FormatInt(event.ReasoningTokens, 10),
		strconv.FormatInt(event.CachedTokens, 10),
		strconv.FormatBool(event.Failed),
	}
	return "sdk:" + shortHash(strings.Join(parts, "|"))
}

func buildEventHash(event Event) string {
	parts := []string{
		event.RequestID,
		event.Timestamp,
		event.Endpoint,
		event.Provider,
		event.Model,
		event.AuthIndex,
		event.SourceHash,
		event.APIKeyHash,
		strconv.FormatInt(event.InputTokens, 10),
		strconv.FormatInt(event.OutputTokens, 10),
		strconv.FormatInt(event.ReasoningTokens, 10),
		strconv.FormatInt(event.CachedTokens, 10),
		strconv.FormatBool(event.Failed),
	}
	if event.LatencyMS != nil {
		parts = append(parts, strconv.FormatInt(*event.LatencyMS, 10))
	}
	return hashString(strings.Join(parts, "|"))
}

func hashString(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func shortHash(value string) string {
	hash := hashString(value)
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16]
}

func maskSource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "@") {
		parts := strings.SplitN(trimmed, "@", 2)
		prefix := parts[0]
		if len(prefix) > 3 {
			prefix = prefix[:3]
		}
		return prefix + "***@" + parts[1]
	}
	if looksSecret(trimmed) {
		if len(trimmed) <= 8 {
			return "m:****"
		}
		return fmt.Sprintf("m:%s...%s", trimmed[:4], trimmed[len(trimmed)-4:])
	}
	return trimmed
}

func looksSecret(value string) bool {
	if strings.ContainsAny(value, " /\\") {
		return false
	}
	return strings.HasPrefix(value, "sk-") || strings.HasPrefix(value, "AIza") || len(value) >= 32
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
