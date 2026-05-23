# Task List: CPA-PC Request Monitoring Charts

Status: Draft for review. Do not implement until the user explicitly says to proceed.

## Task 1: Add Global Chart Endpoint Slice

**Description:** Add the backend contract, validation, protected route, and store aggregation needed for global chart buckets. This creates one complete read path for global token, cost, and TPM charts using local SQLite.

**Acceptance criteria:**
- [x] `GET /v0/management/usage/charts` accepts `range=1h|5h|24h|7d` and `granularity=hour|day`.
- [x] Authorized requests return `global.buckets` with `inputTokens`, `outputTokens`, `cachedTokens`, `totalCost`, `tpmInput`, `tpmOutput`, and `tpmCached`.
- [x] Nil or empty store returns a valid empty chart response with buckets and empty series.
- [x] Existing `/v0/management/usage` response and behavior are unchanged.

**Verification:**
- [x] Tests pass: `go test ./internal/usage ./internal/store ./internal/httpapi`.
- [x] Route tests cover valid auth, invalid auth, invalid `range`, invalid `granularity`, and nil store.
- [x] Store tests cover fixed-range bucket generation, cost calculation, and TPM for partial buckets.

**Dependencies:** None.

**Files likely touched:**
- `internal/usage/charts.go`
- `internal/store/usage_charts.go`
- `internal/store/usage_charts_test.go`
- `internal/httpapi/info.go`
- `internal/httpapi/info_test.go`

**Estimated scope:** Medium.

## Task 2: Add Dimension Series, Filters, Aliases, And Missing Prices

**Description:** Extend the backend chart path so the same endpoint returns all provider/auth-file, API-key, and model series plus filter options. Add combinable provider, authIndex, apiKeyHash, and model filters.

**Acceptance criteria:**
- [x] Response includes all matching `byProviderAuthFile.series`, `byApiKey.series`, and `byModel.series` without Top N limiting or `Other` aggregation.
- [x] Response includes filter options for providers, auth files, API keys, and models.
- [x] API-key labels use aliases when available and never expose raw API keys.
- [x] Filters can be combined and apply consistently to global buckets and all dimension series.
- [x] `missingPriceModels` lists models that contributed events but had no stored price.

**Verification:**
- [x] Tests pass: `go test ./internal/store ./internal/httpapi ./internal/usage`.
- [x] Store tests cover all three dimensions, deterministic ordering, alias labels, missing prices, and combined filters.
- [x] Route tests cover query parameter propagation and response shape.

**Dependencies:** Task 1.

**Files likely touched:**
- `internal/usage/charts.go`
- `internal/store/usage_charts.go`
- `internal/store/usage_charts_test.go`
- `internal/httpapi/info_test.go`

**Estimated scope:** Medium.

## Task 3: Add Frontend Chart API Client And Data Loader

**Description:** Add frontend TypeScript types and a small data-loading path for `GET /v0/management/usage/charts`, following the existing usage-service base resolution and management bearer auth patterns.

**Acceptance criteria:**
- [x] `usageServiceApi` exposes a typed `getUsageCharts` method.
- [x] The client sends only supported query params and includes the management bearer token.
- [x] A chart data loader/hook resolves the embedded usage-service base consistently with existing monitoring code.
- [x] Error handling uses existing `UsageServiceApiError` behavior.

**Verification:**
- [x] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [x] API client tests verify query params and Authorization headers.

**Dependencies:** Task 1.

**Files likely touched:**
- `web/src/services/api/usageService.ts`
- `web/src/features/monitoring/charts/useUsageCharts.ts`
- `web/src/features/monitoring/charts/useUsageCharts.test.ts`

**Estimated scope:** Small.

## Task 4: Add Chart Page Shell, Route, And Monitoring Entry

**Description:** Add `/monitoring/charts` as a new page with the existing monitoring/Codex layout rhythm, then add one navigation entry beside Codex account inspection on the Request Monitoring page. The page should load chart data and render loading, error, and empty states before chart rendering is added.

**Acceptance criteria:**
- [x] `/monitoring/charts` is registered in `MainRoutes`.
- [x] `/monitoring` action bar contains a single chart-page entry beside Codex account inspection.
- [x] The new page has a header, status/action panel, and empty/error/loading states matching monitoring/Codex visual language.
- [x] No unrelated request monitoring UI behavior changes.

**Verification:**
- [x] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`.
- [x] Manual review confirms the new entry is adjacent to Codex inspection and the page shell is responsive.

**Dependencies:** Task 3.

**Files likely touched:**
- `web/src/pages/MonitoringChartsPage.tsx`
- `web/src/pages/MonitoringChartsPage.module.scss`
- `web/src/pages/MonitoringChartsPage.test.tsx`
- `web/src/router/MainRoutes.tsx`
- `web/src/pages/MonitoringCenterPage.tsx`

**Estimated scope:** Medium.

## Task 5: Add ECharts Dependency And Chart Infrastructure

**Description:** Install the direct ECharts dependency and add the thin chart host plus pure chart option builders. This keeps chart rendering testable before the page starts depending on ECharts at runtime.

**Acceptance criteria:**
- [x] `echarts` is added to `web/package.json` and `web/package-lock.json` through npm.
- [x] Chart option builders are pure and covered by unit tests.
- [x] Chart host cleans up chart instances and resizes safely.
- [x] No React wrapper dependency is added.

**Verification:**
- [x] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [x] Type check passes for chart types: `npm --prefix web run type-check`.

**Dependencies:** Task 4.

**Files likely touched:**
- `web/package.json`
- `web/package-lock.json`
- `web/src/features/monitoring/charts/chartOptions.ts`
- `web/src/features/monitoring/charts/chartOptions.test.ts`
- `web/src/features/monitoring/charts/EChartPanel.tsx`

**Estimated scope:** Medium.

## Task 6: Render Global ECharts Line Charts

**Description:** Use the chart infrastructure on the new page to render the first complete visible chart path: global token structure, global cost, and global TPM line charts from the loaded response.

**Acceptance criteria:**
- [x] Global token, cost, and TPM charts render as ECharts line charts.
- [x] Global chart panels preserve loading, error, and empty states.
- [x] Missing price models are surfaced without blocking token or TPM charts.
- [x] The layout still follows the existing monitoring/Codex page rhythm.

**Verification:**
- [x] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [x] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`.
- [x] Type check passes: `npm --prefix web run type-check`.

**Dependencies:** Task 5.

**Files likely touched:**
- `web/src/pages/MonitoringChartsPage.tsx`
- `web/src/pages/MonitoringChartsPage.module.scss`
- `web/src/pages/MonitoringChartsPage.test.tsx`
- `web/src/features/monitoring/charts/chartOptions.ts`
- `web/src/features/monitoring/charts/chartOptions.test.ts`

**Estimated scope:** Medium.

## Task 7: Add Filters And Range/Granularity Reload Behavior

**Description:** Add the required fixed time ranges, hour/day granularity, and provider/auth-file/API-key/model filters. Changing any control reloads the chart endpoint and updates the page state.

**Acceptance criteria:**
- [x] Range options are exactly last 1 hour, last 5 hours, last 24 hours, and last 7 days.
- [x] Last 1 hour is the default range.
- [x] Hour is the default granularity for 1h/5h/24h; day is the default for 7d.
- [x] No custom time range UI exists.
- [x] Provider, auth-file, API-key, and model filters are populated from response options and can be combined.

**Verification:**
- [x] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [x] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`.
- [x] Page tests verify reload params after changing range, granularity, and filters.

**Dependencies:** Task 6.

**Files likely touched:**
- `web/src/features/monitoring/charts/filters.ts`
- `web/src/features/monitoring/charts/filters.test.ts`
- `web/src/pages/MonitoringChartsPage.tsx`
- `web/src/pages/MonitoringChartsPage.module.scss`
- `web/src/pages/MonitoringChartsPage.test.tsx`

**Estimated scope:** Medium.

## Task 8: Render Dimension Chart Sections

**Description:** Add provider/auth-file, API-key, and model sections, each showing token structure, cost, and TPM line charts for all returned series.

**Acceptance criteria:**
- [ ] Provider/auth-file section renders all returned series for token, cost, and TPM metric families.
- [ ] API-key section renders all returned series for token, cost, and TPM metric families.
- [ ] Model section renders all returned series for token, cost, and TPM metric families.
- [ ] Empty dimension series show useful empty states instead of blank chart frames.
- [ ] Missing price models are surfaced without blocking token or TPM charts.

**Verification:**
- [ ] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [ ] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`.
- [ ] Manual review confirms all returned series are visible in legends or chart labels.

**Dependencies:** Task 7.

**Files likely touched:**
- `web/src/features/monitoring/charts/chartOptions.ts`
- `web/src/features/monitoring/charts/chartOptions.test.ts`
- `web/src/pages/MonitoringChartsPage.tsx`
- `web/src/pages/MonitoringChartsPage.module.scss`
- `web/src/pages/MonitoringChartsPage.test.tsx`

**Estimated scope:** Medium.

## Task 9: Add Localization, Responsive Polish, And Final Static Build

**Description:** Add all user-facing labels to existing locale files, polish responsive spacing to match monitoring/Codex pages, and run final frontend build so `static/management.html` is generated by the build pipeline.

**Acceptance criteria:**
- [ ] Chart page labels exist in all current locale files.
- [ ] Desktop and mobile layouts follow existing monitoring/Codex page patterns.
- [ ] `static/management.html` is updated only by `npm --prefix web run build`.
- [ ] All required backend and frontend verification commands pass.

**Verification:**
- [ ] Tests pass: `go test ./...`.
- [ ] Tests pass: `npm --prefix web test -- src/features/monitoring/charts`.
- [ ] Tests pass: `npm --prefix web test -- src/pages/MonitoringChartsPage.test.tsx`.
- [ ] Lint passes: `npm --prefix web run lint`.
- [ ] Type check passes: `npm --prefix web run type-check`.
- [ ] Build passes: `npm --prefix web run build`.

**Dependencies:** Task 8.

**Files likely touched:**
- `web/src/i18n/locales/*.json`
- `web/src/pages/MonitoringChartsPage.module.scss`
- `static/management.html`

**Estimated scope:** Medium.
