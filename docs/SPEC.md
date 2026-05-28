# Spec: Model Price Sync Source Selection And Field Labels

Status: Spec confirmed by user. Implementation planning not started.

## Objective

Improve the model pricing settings in the request monitoring page so administrators can understand pricing fields correctly and choose how one-click price sync resolves model prices.

Target users:

- CPA-PC management UI administrators who review request monitoring costs.
- Administrators maintaining local model prices for cost estimates by account, model, caller key, and live request rows.

What changes:

- Rename the three visible model price fields from prompt/completion/cache terminology to input/output/input-cache terminology.
- Keep exactly three price fields: input price, output price, and input cache price.
- Add a second modal after clicking one-click sync so the user chooses the sync source.
- Allow the user to choose `embedded` or `model.dev` as the sync source.
- For `model.dev`, match prices by provider plus model.
- Treat Codex-sourced requests and `gpt-*` model IDs as OpenAI provider matches for `model.dev` pricing.

What stays the same:

- Existing request monitoring page and price settings modal remain the entry point.
- Existing `/v0/management/model-prices` endpoint remains the manual save/load endpoint.
- Existing `/v0/management/model-prices/sync` endpoint remains the one-click sync endpoint.
- Existing persisted price fields remain `prompt`, `completion`, and `cache` unless implementation proves a schema change is required and it is approved separately.
- Existing cost formula remains three-way: non-cached input, cached input, output.
- No output-cache price field is added.

Field meaning:

| UI label | Existing API/storage field | Meaning | Cost source |
| --- | --- | --- | --- |
| Input price / 输入价格 | `prompt` | Non-cached input token price per 1M tokens | `input_tokens - cached_tokens` |
| Output price / 输出价格 | `completion` | Output token price per 1M tokens | `output_tokens` |
| Input cache price / 输入缓存价格 | `cache` | Cached input token price per 1M tokens | `cached_tokens` / `cache_tokens` |

Acceptance criteria:

- Chinese UI no longer says `提示价格`, `补全价格`, or generic `缓存价格` in the model price settings UI.
- English UI no longer says `Prompt price`, `Completion price`, or generic `Cache price` in the model price settings UI.
- The price editor and saved-price table show exactly three price columns: input, output, and input cache.
- Clicking one-click sync opens a secondary modal instead of immediately calling the sync API.
- The secondary modal has source choices for `embedded` and `model.dev`.
- Choosing `embedded` preserves the current embedded sync behavior unless an embedded price source is added in a later approved change.
- Choosing `model.dev` fetches pricing from the public models.dev API and imports matching prices into local SQLite.
- `model.dev` matching uses provider plus model, not model-only matching, except for the explicitly approved OpenAI inference rules.
- If a models.dev model has `cost.input` and `cost.output` but no `cost.cache_read`, the imported input cache price defaults to one tenth of the input price.
- Codex provider/source rows match against the OpenAI provider in models.dev.
- Any model whose ID starts with `gpt-` matches against the OpenAI provider in models.dev even if its source/provider metadata is missing or Codex-like.
- Synced prices update request monitoring cost estimates after the sync result is loaded.
- Unmatched models do not delete existing manual prices.
- Sync results report imported and skipped counts clearly.

## Tech Stack

- Backend: Go `1.26.0`, Gin, embedded CLIProxyAPI SDK, `modernc.org/sqlite`.
- Storage: existing local SQLite database opened by `internal/store`; no external database or service.
- Frontend: React `19.2.1`, TypeScript `5.9.3`, Vite `8.0.10`, SCSS modules, Vitest `4.1.5`.
- HTTP pricing source: standard Go `net/http`; no new dependency should be added for fetching `https://models.dev/api.json`.
- Routing: existing hash-router management UI; do not switch to browser-router URLs.
- Packaging asset: `static/management.html` is generated from `web/`; do not edit it by hand.

## Commands

Focused backend tests for model price routes and store behavior:

```powershell
go test ./internal/httpapi ./internal/store
```

Full backend tests when sync parsing or route contracts change:

```powershell
go test ./...
```

Focused frontend tests if page or helper tests are added or changed:

```powershell
npm --prefix web test -- src/pages/MonitoringCenterPage.test.tsx
```

Frontend type check:

```powershell
npm --prefix web run type-check
```

Frontend lint:

```powershell
npm --prefix web run lint
```

Frontend production build if generated management asset verification is required:

```powershell
npm --prefix web run build
```

Local app run for manual smoke testing:

```powershell
go run ./cmd/cpa-pc -config .\config.example.yaml
```

## Project Structure

Relevant files:

```text
docs/SPEC.md                                      -> this active feature spec
internal/httpapi/info.go                          -> management routes, including model price sync
internal/httpapi/info_test.go                     -> route/auth/sync tests
internal/store/store.go                           -> ModelPrice shape and SQLite persistence
internal/store/store_test.go                      -> model price persistence tests
internal/store/usage_charts.go                    -> backend chart cost formula
web/src/pages/MonitoringCenterPage.tsx            -> request monitoring price modal and sync flow
web/src/features/monitoring/hooks/useUsageData.ts -> model price load/save/sync hook
web/src/services/api/usageService.ts              -> frontend usage-service API types/client
web/src/utils/usage.ts                            -> ModelPrice type and frontend cost formula
web/src/i18n/locales/en.json                      -> English price labels
web/src/i18n/locales/zh-CN.json                   -> Simplified Chinese price labels
web/src/i18n/locales/zh-TW.json                   -> Traditional Chinese price labels
web/src/i18n/locales/ru.json                      -> Russian price labels
```

Expected implementation scope:

- Update user-facing labels for the three existing price fields.
- Extend frontend sync flow to collect provider/model targets and ask the user for a source.
- Extend sync request/response types to include source selection and skipped/imported details if needed.
- Implement a backend `model.dev` sync source that fetches models.dev data, matches provider+model targets, maps prices, and saves imported prices.
- Preserve existing manual price persistence and cost calculation semantics.

## Code Style

Backend route code should keep request parsing, source dispatch, and external response parsing small and testable. External API responses must be treated as untrusted data.

Example Go shape:

```go
type modelPriceSyncRequest struct {
	Source string                 `json:"source"`
	Models []modelPriceSyncTarget `json:"models"`
}

type modelPriceSyncTarget struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func normalizePricingProvider(provider string, model string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.ToLower(strings.TrimSpace(model))
	if provider == "codex" || strings.HasPrefix(model, "gpt-") {
		return "openai"
	}
	return provider
}
```

Frontend code should keep the sync modal state explicit and avoid hiding side effects inside render helpers.

Example TypeScript shape:

```ts
type ModelPriceSyncSource = 'embedded' | 'model.dev';

type ModelPriceSyncTarget = {
  provider: string;
  model: string;
};

const modelPriceColumns = [
  'model_price_input',
  'model_price_output',
  'model_price_input_cache',
] as const;
```

Style conventions:

- Prefer minimal changes over broad refactors.
- Keep persisted JSON fields camelCase where new frontend-facing fields are required.
- Keep existing `prompt`, `completion`, and `cache` field names internally unless explicitly approved to migrate them.
- Keep SQL parameterized; never interpolate model names or providers into SQL strings.
- Keep user-facing text in i18n files.
- Do not introduce abstractions for future price sources beyond the two requested choices.

## Testing Strategy

Backend unit tests:

- Route tests for `POST /v0/management/model-prices/sync` with `source: "embedded"` preserve current stored-price behavior.
- Route or helper tests for `source: "model.dev"` use an injected test HTTP server or injectable fetcher; tests must not call the live models.dev network endpoint.
- Parser tests cover models.dev pricing fields: `cost.input`, `cost.output`, and `cost.cache_read`.
- Mapping tests cover `input -> prompt`, `output -> completion`, `cache_read -> cache`, and missing `cache_read -> input / 10`.
- Matching tests cover provider+model exact matches, Codex-to-OpenAI provider normalization, and `gpt-*` model-to-OpenAI fallback.
- Sync tests prove unmatched models are skipped and existing prices are preserved.

Frontend tests:

- Price label tests or snapshots prove the UI uses input/output/input-cache terminology.
- Sync flow tests prove clicking one-click sync opens the source selection modal before making the API request.
- API client or hook tests prove selected source and provider/model targets are sent to the sync endpoint if such tests already fit the existing test setup.

Manual smoke checks:

- Open request monitoring, open model price settings, verify the three labels.
- Click one-click sync, verify the secondary modal appears.
- Choose `embedded`, verify current behavior and notification remain coherent.
- Choose `model.dev` with known OpenAI/Codex/GPT models, verify imported count and saved prices update.

Verification expectations:

- Run focused backend tests after backend route or sync changes.
- Run frontend type check after TypeScript API or component changes.
- Run lint if frontend files are changed.
- Run `npm --prefix web run build` only if updating the generated `static/management.html` is explicitly required.

## Boundaries

Always do:

- Preserve the single-process CPA-PC app with embedded CLIProxyAPI SDK.
- Keep model pricing stored locally in SQLite when the usage store is available.
- Keep management authorization on model price endpoints.
- Treat models.dev responses as untrusted external data and validate fields before using them.
- Preserve manual prices for models that a sync source cannot match.
- Keep the cost formula consistent between frontend request monitoring and backend charts.
- Use i18n keys for all new user-facing labels.
- Use tests for provider/model normalization and price mapping.

Ask first:

- Renaming persisted `prompt`, `completion`, or `cache` fields.
- Adding a fourth price field such as output cache or cache write.
- Adding database schema migrations for model prices.
- Adding new third-party dependencies.
- Adding scheduled/background automatic price sync.
- Adding broad provider alias rules beyond the requested Codex/OpenAI and `gpt-*` behavior.
- Calling models.dev directly from the browser instead of the backend.
- Rebuilding and committing `static/management.html`.

Never do:

- Edit `static/management.html` by hand.
- Remove or weaken management authentication.
- Delete existing manual model prices just because a sync source cannot match them.
- Make live network calls in unit tests.
- Store secrets, API keys, or request payloads from external services in logs.
- Reintroduce a separate `CLIProxyAPI.exe` launcher.
- Refactor unrelated monitoring charts, quota panels, config merging, packaging scripts, or Windows management scripts.

## Success Criteria

- Administrators see clear input/output/input-cache price labels instead of prompt/completion/cache wording.
- The one-click sync action is gated by a source selection modal.
- `embedded` remains available as a selectable source.
- `model.dev` sync imports prices from models.dev into local model prices.
- `model.dev` sync matches prices using provider+model and the requested OpenAI special cases.
- Missing models.dev `cost.cache_read` values import as one tenth of `cost.input`.
- Existing cost estimates continue to use exactly three price values.
- Focused backend tests, relevant frontend tests, type check, and lint pass for changed areas.

## Resolved Decisions

1. The UI source label should be `model.dev`; the implementation may fetch `https://models.dev/api.json`.
2. If models.dev has `cost.input` and `cost.output` but no `cost.cache_read`, input cache price defaults to `cost.input / 10`.
3. `embedded` remains visible and keeps the current behavior of returning existing stored prices with `imported = 0` unless a real embedded price table is approved later.
