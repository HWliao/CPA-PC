# Spec: Monitoring Charts Account Dimension Adjustment

Status: Spec corrected; implementation not started.

## Confirmed Assumptions

- This is a modification to the existing monitoring charts page, not a new page.
- Target users are backend administrators using the CPA-PC management UI.
- "账号" should match the request monitoring page concept. The fallback order is: current auth-file metadata account/email/label, event account snapshot, current auth label or event auth label/file snapshot, source display name, auth index. If a row still lacks an account, request monitoring groups by auth label or source as a final display fallback. Provider is not part of the account fallback.
- "API key" is a user-facing naming issue. UI copy should say "调用方密钥" / "Caller key", while existing technical identifiers such as `apiKeyHash` may remain unless implementation proves a rename is necessary.
- Unit tests are required for changed behavior. Full integration testing is out of scope for this spec unless requested later.

## Objective

Adjust the existing monitoring charts feature so administrators can analyze usage by account rather than provider.

What changes:

- Remove provider from chart dimensions.
- Remove provider from chart filters.
- Add account to chart dimensions.
- Add account to chart filters.
- Rename user-facing API key labels on the chart page to "调用方密钥" in Chinese and "Caller key" or equivalent in English.

What stays the same:

- Existing `/monitoring/charts` route.
- Existing chart metric families: token usage, cumulative token usage, cost, cumulative cost, and TPM.
- Existing fixed ranges and linked granularity rules: `1h` -> `10m`, `5h` and `24h` -> `hour`, `7d` -> `day`.
- Existing model dimension and model filter.
- Existing caller key grouping by API-key hash and alias.

Acceptance criteria:

- Chart dimension options are exactly global total, account, caller key, and model. Provider is not visible.
- Filter controls include account, caller key, and model. Provider is not visible.
- Selecting account as the chart dimension hides and omits the account filter, matching the current rule for filters that are already used as the active dimension.
- Selecting caller key as the chart dimension hides and omits the caller-key filter.
- Selecting model as the chart dimension hides and omits the model filter.
- Account options and account series labels are consistent with request monitoring account labels wherever persisted data allows.
- Backend chart data groups account series by stable account identity, not by provider.
- Backend chart filtering supports account filtering and no longer depends on provider filtering for the chart UI.
- User-facing chart copy no longer says "Provider" or "提供方" for dimensions or filters.
- User-facing chart copy no longer says "API key" / "API Key" for this page where it refers to the caller's key; it says "Caller key" / "调用方密钥" instead.
- Relevant frontend and backend unit tests pass.

## Tech Stack

- Backend: Go `1.26.0`, Gin, embedded CLIProxyAPI SDK, `modernc.org/sqlite`.
- Storage: existing local SQLite database opened by `internal/store`; no external database or service.
- Frontend: React `19.2.1`, TypeScript `5.9.3`, Vite `8.0.10`, SCSS modules, Vitest `4.1.5`.
- Charts: existing direct `echarts` dependency.
- Routing: existing hash-router management UI; do not switch to browser-router URLs.
- Packaging asset: `static/management.html` is generated from `web/`; do not edit it by hand.

## Commands

Focused backend unit tests:

```powershell
go test ./internal/usage ./internal/store ./internal/httpapi
```

Focused frontend unit tests:

```powershell
npm --prefix web test -- src/features/monitoring/charts/filters.test.ts src/pages/MonitoringChartsPage.test.tsx
```

Frontend type check:

```powershell
npm --prefix web run type-check
```

Frontend lint:

```powershell
npm --prefix web run lint
```

Full backend test suite when backend chart contract changes:

```powershell
go test ./...
```

Frontend production build if UI labels or generated static assets need verification:

```powershell
npm --prefix web run build
```

Local app run for manual smoke testing:

```powershell
go run ./cmd/cpa-pc -config .\config.example.yaml
```

Frontend dev server for manual UI inspection:

```powershell
npm --prefix web run dev
```

## Project Structure

Relevant existing files:

```text
docs/SPEC.md                                      -> this draft feature spec
docs/specs/2026-05-25-cpa-pc-request-monitoring-charts.md
                                                  -> existing implemented chart feature record
internal/usage/charts.go                          -> chart query, response structs, validation, empty response
internal/store/usage_charts.go                    -> SQLite-backed chart aggregation and filter handling
internal/store/usage_charts_test.go               -> backend chart aggregation unit tests
internal/httpapi/info.go                          -> management chart route registration
internal/httpapi/info_test.go                     -> chart route/auth/query unit tests
web/src/pages/MonitoringChartsPage.tsx            -> chart page controls, labels, dimensions, chart rendering
web/src/pages/MonitoringChartsPage.test.tsx       -> page-level chart behavior tests
web/src/features/monitoring/charts/filters.ts     -> chart dimension/filter state and query-param builder
web/src/features/monitoring/charts/filters.test.ts
                                                  -> filter and dimension unit tests
web/src/features/monitoring/charts/chartOptions.ts
                                                  -> chart series option generation
web/src/features/monitoring/charts/useUsageCharts.ts
                                                  -> chart data loading hook
web/src/services/api/usageService.ts              -> frontend chart API types/client
web/src/pages/MonitoringCenterPage.tsx            -> request monitoring account/caller-key naming reference
web/src/features/monitoring/hooks/useMonitoringData.ts
                                                  -> request monitoring account identity/display reference
web/src/i18n/locales/en.json                      -> English chart labels
web/src/i18n/locales/zh-CN.json                   -> Simplified Chinese chart labels
web/src/i18n/locales/zh-TW.json                   -> Traditional Chinese chart labels
web/src/i18n/locales/ru.json                      -> Russian chart labels
```

Expected implementation scope:

- Update chart query/response structures from provider-oriented options and series to account-oriented options and series.
- Update SQLite aggregation to build account options and account series from usage event account/auth snapshots.
- Update frontend filter state and query building to use account instead of provider.
- Update chart page labels, option mapping, active dimension resolution, and tests.
- Update i18n labels for the chart page.
- Avoid unrelated request monitoring page refactors.

## Code Style

Backend code should keep query validation in `internal/usage`, SQL aggregation in `internal/store`, and route concerns in `internal/httpapi`.

Example Go style:

```go
type ChartQuery struct {
	Range       ChartRange
	Granularity ChartGranularity
	Account     string
	APIKeyHash  string
	Model       string
	NowMS       int64
}

func NormalizeChartQuery(query ChartQuery) (ChartQuery, error) {
	if query.Range == "" {
		query.Range = ChartRange1H
	}
	if !validChartRange(query.Range) {
		return ChartQuery{}, errors.New("invalid chart range")
	}

	query.Granularity = defaultChartGranularity(query.Range)
	query.Account = strings.TrimSpace(query.Account)
	query.APIKeyHash = strings.ToLower(strings.TrimSpace(query.APIKeyHash))
	query.Model = strings.TrimSpace(query.Model)
	return query, nil
}
```

Frontend code should keep dimension/filter transformations in pure helpers and keep the page component focused on rendering state.

Example TypeScript style:

```ts
export type UsageChartsDimension = 'global' | 'account' | 'apiKey' | 'model';

export type UsageChartsFilterState = {
  range: UsageChartsRange;
  dimension: UsageChartsDimension;
  account: string;
  apiKeyHash: string;
  model: string;
};

export function buildUsageChartsQueryParams(state: UsageChartsFilterState): UsageChartsQueryParams {
  const params: UsageChartsQueryParams = {
    range: state.range,
    granularity: resolveDefaultUsageChartsGranularity(state.range),
  };

  if (!shouldDisableUsageChartsFilter('account', state.dimension)) {
    appendNonEmptyParam(params, 'account', state.account);
  }
  if (!shouldDisableUsageChartsFilter('apiKey', state.dimension)) {
    appendNonEmptyParam(params, 'apiKeyHash', state.apiKeyHash);
  }
  if (!shouldDisableUsageChartsFilter('model', state.dimension)) {
    appendNonEmptyParam(params, 'model', state.model);
  }
  return params;
}
```

Style conventions:

- Prefer the smallest direct change that satisfies the spec.
- Keep user-facing terminology in i18n files, not hard-coded in components except existing test defaults.
- Use camelCase for frontend API types and JSON response fields.
- Keep SQL parameterized; never interpolate filter values into SQL strings.
- Do not introduce new abstractions for one-off label changes.

## Testing Strategy

Backend unit tests:

- `internal/usage/charts.go`: parsing and normalization should accept `account`, reject invalid range/granularity, and no longer require provider logic for chart filters.
- `internal/store/usage_charts_test.go`: chart aggregation should produce account options and `byAccount` series; account filters should constrain global, account, caller-key, and model series; provider-only expectations should be removed or replaced.
- `internal/httpapi/info_test.go`: `/v0/management/usage/charts` should pass account query params through to the store and continue returning authorized responses and HTTP 400 for invalid queries.

Frontend unit tests:

- `web/src/features/monitoring/charts/filters.test.ts`: dimensions should include account and exclude provider; query params should include account and omit it when account is the active dimension.
- `web/src/pages/MonitoringChartsPage.test.tsx`: the page should render account and caller-key labels, hide the active-dimension filter, and switch chart panels to account/caller-key/model series.
- Existing chart option tests should continue to pass without provider-specific assumptions.

Verification expectations:

- Run focused backend and frontend unit tests after implementation.
- Run `npm --prefix web run type-check` because chart API types are likely to change.
- Run lint if frontend files are changed.
- Run `go test ./...` if the backend response contract is changed beyond local chart structs.

## Boundaries

Always do:

- Keep the app as a single-process CPA-PC app with embedded CLIProxyAPI SDK.
- Preserve existing `/monitoring/charts` route and existing chart metric behavior unless directly required by this spec.
- Match request monitoring terminology for account and caller key.
- Keep chart SQL read-only and parameterized.
- Add or update unit tests for behavior changed by this spec.
- Rebuild `static/management.html` only through `npm --prefix web run build` when a generated asset update is required.

Ask first:

- Changing the public endpoint path away from `/v0/management/usage/charts`.
- Adding dependencies.
- Adding or changing persistent database schema.
- Changing request monitoring page behavior beyond using it as a terminology/reference source.
- Adding backward compatibility fields for provider if an external consumer is discovered.
- Changing chart metrics, time ranges, granularity, or pricing formulas.

Never do:

- Reintroduce a separate `CLIProxyAPI.exe` launcher.
- Edit `static/management.html` by hand.
- Add external collectors, queues, databases, or services for usage data.
- Remove or weaken management authorization.
- Commit secrets or real API keys.
- Refactor unrelated monitoring tables, quota panels, Codex inspection logic, packaging scripts, or config merging.

## Success Criteria

- Administrators can use the monitoring charts page without seeing provider as a dimension or filter.
- Administrators can choose account as a chart dimension and filter, with labels consistent with request monitoring where data is available.
- Administrators see caller-key terminology instead of API-key terminology on this chart page.
- Existing chart metrics still render for global, account, caller-key, and model dimensions.
- Focused unit tests for changed backend and frontend behavior pass.

## Resolved Decisions

- Account labels and grouping should follow the request monitoring page logic: current auth-file metadata account/email/label, event account snapshot, current auth label or event auth label/file snapshot, source display name, auth index, then auth label/source as final row grouping fallback.
- Provider should not be used as the account fallback.
- English UI should use "Caller key" and Chinese UI should use "调用方密钥".
