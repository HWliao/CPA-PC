package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HWliao/CPA-PC/internal/usage"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type ModelPrice struct {
	Prompt        float64 `json:"prompt"`
	Completion    float64 `json:"completion"`
	Cache         float64 `json:"cache"`
	Source        string  `json:"source,omitempty"`
	SourceModelID string  `json:"sourceModelId,omitempty"`
	RawJSON       string  `json:"rawJson,omitempty"`
	UpdatedAtMS   int64   `json:"updatedAtMs,omitempty"`
	SyncedAtMS    *int64  `json:"syncedAtMs,omitempty"`
}

type APIKeyAlias struct {
	APIKeyHash  string `json:"apiKeyHash"`
	Alias       string `json:"alias"`
	UpdatedAtMS int64  `json:"updatedAtMs"`
}

type ManagerConfig struct {
	CPAConnection        ManagerCPAConnectionConfig        `json:"cpaConnection"`
	Collector            ManagerCollectorConfig            `json:"collector"`
	ExternalUsageService ManagerExternalUsageServiceConfig `json:"externalUsageService"`
	UpdatedAtMS          int64                             `json:"updatedAtMs,omitempty"`
}

type ManagerCPAConnectionConfig struct {
	CPABaseURL    string `json:"cpaBaseUrl"`
	ManagementKey string `json:"managementKey,omitempty"`
}

type ManagerCollectorConfig struct {
	Enabled        *bool  `json:"enabled,omitempty"`
	CollectorMode  string `json:"collectorMode,omitempty"`
	Queue          string `json:"queue,omitempty"`
	PopSide        string `json:"popSide,omitempty"`
	BatchSize      int    `json:"batchSize,omitempty"`
	PollIntervalMS int    `json:"pollIntervalMs,omitempty"`
	QueryLimit     int    `json:"queryLimit,omitempty"`
	TLSSkipVerify  bool   `json:"tlsSkipVerify,omitempty"`
}

type ManagerExternalUsageServiceConfig struct {
	Enabled     bool   `json:"enabled"`
	ServiceBase string `json:"serviceBase,omitempty"`
}

const managerConfigKey = "manager_config_v1"

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("usage db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create usage db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open usage db: %w", err)
	}
	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) init() error {
	statements := []string{
		`pragma journal_mode = WAL`,
		`pragma synchronous = FULL`,
		`pragma busy_timeout = 5000`,
		`pragma foreign_keys = ON`,
		`create table if not exists usage_events (
			id integer primary key autoincrement,
			request_id text,
			event_hash text not null unique,
			timestamp_ms integer not null,
			timestamp text not null,
			provider text,
			model text not null,
			endpoint text,
			method text,
			path text,
			auth_type text,
			auth_index text,
			source text,
			source_hash text,
			api_key_hash text,
			account_snapshot text,
			auth_label_snapshot text,
			auth_file_snapshot text,
			auth_provider_snapshot text,
			auth_snapshot_at_ms integer,
			input_tokens integer not null default 0,
			output_tokens integer not null default 0,
			reasoning_tokens integer not null default 0,
			cached_tokens integer not null default 0,
			cache_tokens integer not null default 0,
			total_tokens integer not null default 0,
			latency_ms integer,
			failed integer not null default 0,
			raw_json text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_usage_events_timestamp on usage_events(timestamp_ms)`,
		`create index if not exists idx_usage_events_request_id on usage_events(request_id)`,
		`create index if not exists idx_usage_events_model on usage_events(model)`,
		`create index if not exists idx_usage_events_auth_index on usage_events(auth_index)`,
		`create index if not exists idx_usage_events_endpoint on usage_events(endpoint)`,
		`create table if not exists settings (
			key text primary key,
			value text not null,
			updated_at_ms integer not null
		)`,
		`create table if not exists model_prices (
			model text primary key,
			prompt_per_1m real not null,
			completion_per_1m real not null,
			cache_per_1m real not null,
			source text,
			source_model_id text,
			raw_json text,
			updated_at_ms integer not null,
			synced_at_ms integer
		)`,
		`create table if not exists api_key_aliases (
			api_key_hash text primary key,
			alias text not null,
			updated_at_ms integer not null
		)`,
		`create table if not exists dead_letter_events (
			id integer primary key autoincrement,
			payload text not null,
			error text not null,
			created_at_ms integer not null
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("init usage store: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertEvents(ctx context.Context, events []usage.Event) (usage.InsertResult, error) {
	if len(events) == 0 {
		return usage.InsertResult{}, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return usage.InsertResult{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `insert or ignore into usage_events (
		request_id, event_hash, timestamp_ms, timestamp, provider, model, endpoint, method, path,
		auth_type, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_file_snapshot, auth_provider_snapshot, auth_snapshot_at_ms,
		input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_tokens, total_tokens,
		latency_ms, failed, raw_json, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return usage.InsertResult{}, err
	}
	defer stmt.Close()

	result := usage.InsertResult{}
	for _, event := range events {
		createdAt := event.CreatedAtMS
		if createdAt <= 0 {
			createdAt = time.Now().UnixMilli()
		}
		failed := 0
		if event.Failed {
			failed = 1
		}
		res, err := stmt.ExecContext(
			ctx,
			nullString(event.RequestID),
			event.EventHash,
			event.TimestampMS,
			event.Timestamp,
			nullString(event.Provider),
			event.Model,
			nullString(event.Endpoint),
			nullString(event.Method),
			nullString(event.Path),
			nullString(event.AuthType),
			nullString(event.AuthIndex),
			nullString(event.Source),
			nullString(event.SourceHash),
			nullString(event.APIKeyHash),
			nullString(event.AccountSnapshot),
			nullString(event.AuthLabelSnapshot),
			nullString(event.AuthFileSnapshot),
			nullString(event.AuthProviderSnapshot),
			nullPositiveInt64(event.AuthSnapshotAtMS),
			event.InputTokens,
			event.OutputTokens,
			event.ReasoningTokens,
			event.CachedTokens,
			event.CacheTokens,
			event.TotalTokens,
			nullInt64(event.LatencyMS),
			failed,
			nullString(event.RawJSON),
			createdAt,
		)
		if err != nil {
			return usage.InsertResult{}, err
		}
		affected, _ := res.RowsAffected()
		if affected > 0 {
			result.Inserted++
		} else {
			result.Skipped++
		}
	}
	if err := tx.Commit(); err != nil {
		return usage.InsertResult{}, err
	}
	return result, nil
}

func (s *Store) RecentEvents(ctx context.Context, limit int) ([]usage.Event, error) {
	if limit <= 0 {
		limit = 50000
	}
	rows, err := s.db.QueryContext(ctx, `select
		request_id, event_hash, timestamp_ms, timestamp, provider, model, endpoint, method, path,
		auth_type, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_file_snapshot, auth_provider_snapshot, auth_snapshot_at_ms,
		input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_tokens, total_tokens,
		latency_ms, failed, raw_json, created_at_ms
		from usage_events
		order by timestamp_ms desc, id desc
		limit ?`, limit)
	if err != nil {
		return nil, err
	}
	return scanUsageEvents(rows)
}

func (s *Store) ExportEvents(ctx context.Context) ([]usage.Event, error) {
	rows, err := s.db.QueryContext(ctx, `select
		request_id, event_hash, timestamp_ms, timestamp, provider, model, endpoint, method, path,
		auth_type, auth_index, source, source_hash, api_key_hash,
		account_snapshot, auth_label_snapshot, auth_file_snapshot, auth_provider_snapshot, auth_snapshot_at_ms,
		input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_tokens, total_tokens,
		latency_ms, failed, raw_json, created_at_ms
		from usage_events
		order by timestamp_ms asc, id asc`)
	if err != nil {
		return nil, err
	}
	return scanUsageEvents(rows)
}

func scanUsageEvents(rows *sql.Rows) ([]usage.Event, error) {
	defer rows.Close()

	var events []usage.Event
	for rows.Next() {
		var event usage.Event
		var requestID, provider, endpoint, method, path, authType, authIndex, source, sourceHash, apiKeyHash, accountSnapshot, authLabelSnapshot, authFileSnapshot, authProviderSnapshot, rawJSON sql.NullString
		var authSnapshotAt sql.NullInt64
		var latency sql.NullInt64
		var failed int
		if err := rows.Scan(
			&requestID,
			&event.EventHash,
			&event.TimestampMS,
			&event.Timestamp,
			&provider,
			&event.Model,
			&endpoint,
			&method,
			&path,
			&authType,
			&authIndex,
			&source,
			&sourceHash,
			&apiKeyHash,
			&accountSnapshot,
			&authLabelSnapshot,
			&authFileSnapshot,
			&authProviderSnapshot,
			&authSnapshotAt,
			&event.InputTokens,
			&event.OutputTokens,
			&event.ReasoningTokens,
			&event.CachedTokens,
			&event.CacheTokens,
			&event.TotalTokens,
			&latency,
			&failed,
			&rawJSON,
			&event.CreatedAtMS,
		); err != nil {
			return nil, err
		}
		event.RequestID = requestID.String
		event.Provider = provider.String
		event.Endpoint = endpoint.String
		event.Method = method.String
		event.Path = path.String
		event.AuthType = authType.String
		event.AuthIndex = authIndex.String
		event.Source = source.String
		event.SourceHash = sourceHash.String
		event.APIKeyHash = apiKeyHash.String
		event.AccountSnapshot = accountSnapshot.String
		event.AuthLabelSnapshot = authLabelSnapshot.String
		event.AuthFileSnapshot = authFileSnapshot.String
		event.AuthProviderSnapshot = authProviderSnapshot.String
		if authSnapshotAt.Valid {
			event.AuthSnapshotAtMS = authSnapshotAt.Int64
		}
		if latency.Valid {
			value := latency.Int64
			event.LatencyMS = &value
		}
		event.Failed = failed != 0
		event.RawJSON = rawJSON.String
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) Counts(ctx context.Context) (events int64, deadLetters int64, err error) {
	if err = s.db.QueryRowContext(ctx, `select count(*) from usage_events`).Scan(&events); err != nil {
		return 0, 0, err
	}
	if err = s.db.QueryRowContext(ctx, `select count(*) from dead_letter_events`).Scan(&deadLetters); err != nil {
		return 0, 0, err
	}
	return events, deadLetters, nil
}

func (s *Store) SaveManagerConfig(ctx context.Context, cfg ManagerConfig) error {
	cfg.UpdatedAtMS = time.Now().UnixMilli()
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`insert into settings(key, value, updated_at_ms) values(?, ?, ?)
		on conflict(key) do update set value = excluded.value, updated_at_ms = excluded.updated_at_ms`,
		managerConfigKey,
		string(data),
		cfg.UpdatedAtMS,
	)
	return err
}

func (s *Store) LoadManagerConfig(ctx context.Context) (ManagerConfig, bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `select value from settings where key = ?`, managerConfigKey).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ManagerConfig{}, false, nil
	}
	if err != nil {
		return ManagerConfig{}, false, err
	}
	var cfg ManagerConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ManagerConfig{}, false, err
	}
	return cfg, true, nil
}

func (s *Store) LoadModelPrices(ctx context.Context) (map[string]ModelPrice, error) {
	rows, err := s.db.QueryContext(ctx, `select model, prompt_per_1m, completion_per_1m, cache_per_1m, source, source_model_id, raw_json, updated_at_ms, synced_at_ms from model_prices order by model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := map[string]ModelPrice{}
	for rows.Next() {
		var model string
		var price ModelPrice
		var source, sourceModelID, rawJSON sql.NullString
		var syncedAt sql.NullInt64
		if err := rows.Scan(&model, &price.Prompt, &price.Completion, &price.Cache, &source, &sourceModelID, &rawJSON, &price.UpdatedAtMS, &syncedAt); err != nil {
			return nil, err
		}
		price.Source = source.String
		price.SourceModelID = sourceModelID.String
		price.RawJSON = rawJSON.String
		if syncedAt.Valid {
			value := syncedAt.Int64
			price.SyncedAtMS = &value
		}
		prices[model] = price
	}
	return prices, rows.Err()
}

func (s *Store) SaveModelPrices(ctx context.Context, prices map[string]ModelPrice) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, `delete from model_prices`); err != nil {
		return err
	}
	if len(prices) == 0 {
		return tx.Commit()
	}
	stmt, err := tx.PrepareContext(ctx, `insert into model_prices (model, prompt_per_1m, completion_per_1m, cache_per_1m, source, source_model_id, raw_json, updated_at_ms, synced_at_ms) values (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	now := time.Now().UnixMilli()
	for model, price := range prices {
		model = strings.TrimSpace(model)
		if err := validateModelPrice(model, price); err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, model, price.Prompt, price.Completion, price.Cache, nullString(price.Source), nullString(price.SourceModelID), nullString(price.RawJSON), now, nullInt64(price.SyncedAtMS)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) LoadAPIKeyAliases(ctx context.Context) ([]APIKeyAlias, error) {
	rows, err := s.db.QueryContext(ctx, `select api_key_hash, alias, updated_at_ms from api_key_aliases order by alias collate nocase, api_key_hash`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	aliases := []APIKeyAlias{}
	for rows.Next() {
		var alias APIKeyAlias
		if err := rows.Scan(&alias.APIKeyHash, &alias.Alias, &alias.UpdatedAtMS); err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}
	return aliases, rows.Err()
}

func (s *Store) UpsertAPIKeyAliases(ctx context.Context, aliases []APIKeyAlias) error {
	now := time.Now().UnixMilli()
	normalized := make([]APIKeyAlias, 0, len(aliases))
	seen := map[string]string{}
	for _, alias := range aliases {
		item, err := normalizeAPIKeyAlias(alias, now)
		if err != nil {
			return err
		}
		aliasKey := strings.ToLower(item.Alias)
		if existing, ok := seen[aliasKey]; ok && existing != item.APIKeyHash {
			return errors.New("api key alias already exists")
		}
		seen[aliasKey] = item.APIKeyHash
		normalized = append(normalized, item)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	stmt, err := tx.PrepareContext(ctx, `insert into api_key_aliases (api_key_hash, alias, updated_at_ms) values (?, ?, ?) on conflict(api_key_hash) do update set alias = excluded.alias, updated_at_ms = excluded.updated_at_ms`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	existingRows, err := tx.QueryContext(ctx, `select api_key_hash, alias from api_key_aliases`)
	if err != nil {
		return err
	}
	existingAliases := map[string]string{}
	for existingRows.Next() {
		var hash string
		var alias string
		if err := existingRows.Scan(&hash, &alias); err != nil {
			_ = existingRows.Close()
			return err
		}
		existingAliases[strings.ToLower(strings.TrimSpace(alias))] = hash
	}
	if err := existingRows.Close(); err != nil {
		return err
	}
	if err := existingRows.Err(); err != nil {
		return err
	}

	for _, item := range normalized {
		aliasKey := strings.ToLower(item.Alias)
		if existing, ok := existingAliases[aliasKey]; ok && existing != item.APIKeyHash {
			return errors.New("api key alias already exists")
		}
		if _, err := stmt.ExecContext(ctx, item.APIKeyHash, item.Alias, item.UpdatedAtMS); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteAPIKeyAlias(ctx context.Context, apiKeyHash string) error {
	hash := strings.ToLower(strings.TrimSpace(apiKeyHash))
	if !validAPIKeyHash(hash) {
		return errors.New("valid apiKeyHash is required")
	}
	_, err := s.db.ExecContext(ctx, `delete from api_key_aliases where api_key_hash = ?`, hash)
	return err
}

func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullPositiveInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func validateModelPrice(model string, price ModelPrice) error {
	if strings.TrimSpace(model) == "" {
		return errors.New("model is required")
	}
	if !validPriceValue(price.Prompt) || !validPriceValue(price.Completion) || !validPriceValue(price.Cache) {
		return errors.New("model price values must be non-negative finite numbers")
	}
	return nil
}

func validPriceValue(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func normalizeAPIKeyAlias(alias APIKeyAlias, now int64) (APIKeyAlias, error) {
	hash := strings.ToLower(strings.TrimSpace(alias.APIKeyHash))
	if !validAPIKeyHash(hash) {
		return APIKeyAlias{}, errors.New("valid apiKeyHash is required")
	}
	label := strings.TrimSpace(alias.Alias)
	if label == "" {
		return APIKeyAlias{}, errors.New("alias is required")
	}
	if len([]rune(label)) > 120 {
		return APIKeyAlias{}, errors.New("alias must be 120 characters or less")
	}
	if alias.UpdatedAtMS <= 0 {
		alias.UpdatedAtMS = now
	}
	alias.APIKeyHash = hash
	alias.Alias = label
	return alias, nil
}

func validAPIKeyHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}
