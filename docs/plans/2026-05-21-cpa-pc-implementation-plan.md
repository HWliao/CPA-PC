# Implementation Plan: CPA-PC

Source PRD: `docs/requirements/2026-05-21-redefine-prd.md`

## Overview

Build CPA-PC as a Windows-first single-process Go application. The process embeds CLIProxyAPI through its public SDK, owns a local usage service implementation, serves a migrated CPA-Manager web UI, and ships as `cpa-pc.exe` plus external `static/management.html` and `config.example.yaml`.

The plan is risk-first. Before importing a large frontend or copying usage-service code, prove that CPA-PC can embed CPA SDK, register its own routes, serve the expected management page path, and receive SDK usage records.

## Confirmed Decisions

- CPA-PC is not a script launcher.
- CPA-PC does not add `../CLIProxyAPI` or `../CPA-Manager` as subdirectories or submodules.
- CPA-PC may reference those sibling projects during initial development.
- CPA-Manager frontend source is imported once into CPA-PC without preserving CPA-Manager git history.
- CPA-Manager is not a long-term runtime, build-time, or release dependency.
- `management.html` is external in `static/`, not embedded in `cpa-pc.exe` for phase 1.
- Default Management Key follows CPA original logic; if there is no usable original default, use `123456`.
- Phase 1 targets Windows amd64 only.
- Keep CPA-Manager's original page flow, including setup wizard, and satisfy it through compatible local defaults.

## Architecture Decisions

- Use `github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy` and `github.com/router-for-me/CLIProxyAPI/v7/sdk/config` as the CPA integration surface.
- Use CLIProxyAPI SDK usage plugin as the usage data source; do not consume HTTP/RESP usage queues in CPA-PC.
- Keep the usage store and HTTP API inside CPA-PC under `internal/` packages.
- Keep the migrated frontend source under `web/` and generate `static/management.html` for release.
- Preserve CPA-Manager usage API response shapes where the migrated frontend already depends on them.
- Use SQLite via `modernc.org/sqlite`, matching CPA-Manager's CGO-free approach.

## Early Technical Constraint

CLIProxyAPI SDK already registers `GET /management.html` during default route setup. CPA-PC should not register a duplicate Gin route for the same method/path unless this is proven safe.

Preferred phase-1 approach:

- Serve `static/management.html` through CPA SDK's existing management page mechanism.
- Keep `remote-management.disable-control-panel: false`, ensure `static/management.html` exists, and avoid duplicate `/management.html` route registration.

This is a phase-1 validation item because it can affect the approved config example.

## Dependency Graph

```text
Go module foundation
  -> config loading and runtime paths
      -> embedded CPA SDK process
          -> custom route injection validation
          -> management auth compatibility
          -> usage plugin validation
              -> SQLite schema and usage store
                  -> usage API compatibility
                      -> migrated frontend integration
                          -> Windows release package
```

## Verification Commands

These commands become available as the project is created:

- Go tests: `go test ./...`
- Go build: `go build ./cmd/cpa-pc`
- Windows amd64 build: `$env:GOOS='windows'; $env:GOARCH='amd64'; go build -o dist/cpa-pc.exe ./cmd/cpa-pc`
- Windows amd64 package: `npm run package:windows -- --version dev`
- Frontend install: `npm install`
- Frontend tests: `npm test -- --run`
- Frontend build: `npm run build`

Use the concrete scripts/package commands that exist after frontend migration; update this plan if command names differ.

## Phase 0: Foundation

### Task 0.1: Create Go Module Skeleton

**Description:** Initialize CPA-PC as an independent Go module with a minimal `cmd/cpa-pc` entrypoint and internal package boundaries.

**Acceptance criteria:**

- `go.mod` exists at repo root.
- `cmd/cpa-pc/main.go` exists and supports `-config`.
- Internal package skeleton exists for `app`, `config`, `httpapi`, `store`, and `usage`.
- The program can print version/help without requiring CPA configuration.

**Verification:**

- `go test ./...`
- `go build ./cmd/cpa-pc`

**Dependencies:** None

**Files likely touched:**

- `go.mod`
- `go.sum`
- `cmd/cpa-pc/main.go`
- `internal/app/`
- `internal/config/`

**Estimated scope:** Medium

### Task 0.2: Add Config And Runtime Path Resolution

**Description:** Define CPA-PC config behavior around `config.yaml`, `config.example.yaml`, `data/`, `logs/`, and `static/` paths.

**Acceptance criteria:**

- `config.example.yaml` exists at repo root.
- Default runtime paths are relative to the executable/config directory.
- Default host and port follow CLIProxyAPI defaults: host is `""` and port is `8317`.
- Default Management Key behavior is documented and implemented as agreed.
- Usage DB defaults to `./data/usage.sqlite`.

**Verification:**

- `go test ./internal/config ./...`
- Manual check: run with missing config and confirm the error message tells the user what to do.

**Dependencies:** Task 0.1

**Files likely touched:**

- `config.example.yaml`
- `internal/config/`
- `cmd/cpa-pc/main.go`

**Estimated scope:** Medium

### Checkpoint: Foundation

- `go test ./...` passes.
- `go build ./cmd/cpa-pc` succeeds.
- No frontend or SQLite migration has started yet.

## Phase 1: Embedded CPA SDK Risk Slice

### Task 1.1: Start CLIProxyAPI SDK In CPA-PC

**Description:** Load CPA-compatible config and start CLIProxyAPI SDK from CPA-PC's app lifecycle.

**Acceptance criteria:**

- `cpa-pc.exe -config config.yaml` starts one process.
- CPA health endpoint works on the configured port.
- CPA Management API is available when `remote-management.secret-key` is configured.
- Shutdown via Ctrl+C stops the embedded CPA service cleanly.

**Verification:**

- `go test ./...`
- `go build ./cmd/cpa-pc`
- Manual check: `Invoke-WebRequest http://127.0.0.1:8317/healthz`

**Dependencies:** Task 0.2

**Files likely touched:**

- `internal/app/`
- `cmd/cpa-pc/main.go`
- `config.example.yaml`

**Estimated scope:** Medium

### Task 1.2: Validate Custom Route Injection

**Description:** Use `cliproxy.WithServerOptions` and `WithRouterConfigurator` to register CPA-PC-specific routes on the same Gin engine without breaking CPA routes.

**Acceptance criteria:**

- `GET /cpa-pc/info` returns CPA-PC service metadata.
- Existing CPA API routes continue to work.
- Existing CPA Management API routes continue to work.
- Route registration does not panic from duplicate routes.

**Verification:**

- `go test ./...`
- Manual check: `Invoke-WebRequest http://127.0.0.1:8317/cpa-pc/info`

**Dependencies:** Task 1.1

**Files likely touched:**

- `internal/app/`
- `internal/httpapi/`

**Estimated scope:** Small

### Task 1.3: Validate Management Page Serving Strategy

**Description:** Prove how CPA-PC will serve `/management.html` without duplicate Gin route conflicts.

**Acceptance criteria:**

- A placeholder `static/management.html` can be served at `GET /management.html`.
- The serving strategy does not trigger unwanted remote panel download during normal release layout.
- PRD and `config.example.yaml` agree on `disable-control-panel: false`, so CPA SDK serves the external `static/management.html`.

**Verification:**

- `go test ./...`
- Manual check: `Invoke-WebRequest http://127.0.0.1:8317/management.html`
- Manual check: delete `static/management.html` in a temp run and confirm the failure mode is acceptable.

**Dependencies:** Task 1.2

**Files likely touched:**

- `config.example.yaml`
- `static/management.html`
- `internal/app/`
- `docs/requirements/2026-05-21-redefine-prd.md` if config behavior changes

**Estimated scope:** Small

### Task 1.4: Validate Management Key Compatibility

**Description:** Ensure CPA-PC custom endpoints and embedded CPA Management API accept the same Management Key semantics.

CPA may hash plaintext `remote-management.secret-key` during config load. CPA-PC usage endpoints must not compare a user-entered key against the already-hashed string as plain text.

**Acceptance criteria:**

- A request authorized for CPA Management API is also authorized for CPA-PC usage endpoints.
- Plain default key `123456` works after CPA config loading and any CPA-side hashing behavior.
- Incorrect keys are rejected consistently.

**Verification:**

- `go test ./internal/httpapi ./internal/app ./...`
- Manual check: call one CPA Management endpoint and one CPA-PC endpoint with the same Authorization header.

**Dependencies:** Task 1.2

**Files likely touched:**

- `internal/httpapi/`
- `internal/app/`
- `internal/config/`

**Estimated scope:** Small

### Task 1.5: Validate SDK Usage Plugin Delivery

**Description:** Register a CLIProxyAPI SDK usage plugin from CPA-PC and prove records are delivered without SQLite or frontend involvement.

**Acceptance criteria:**

- CPA-PC registers a usage plugin during startup.
- A synthetic or real request emits a usage record to the plugin.
- Plugin output includes provider, model, token counts, latency, failed flag, auth metadata where available.
- No external HTTP/RESP usage queue collector is involved.

**Verification:**

- `go test ./internal/usage ./...`
- Manual check: make a proxy request and confirm the plugin logs one record.

**Dependencies:** Task 1.1

**Files likely touched:**

- `internal/usage/`
- `internal/app/`

**Estimated scope:** Medium

### Checkpoint: SDK Integration

- CPA SDK starts inside CPA-PC.
- CPA-PC can register custom routes.
- `/management.html` serving strategy is proven.
- Management auth compatibility is proven.
- SDK usage plugin delivery is proven.
- Do not import frontend code before this checkpoint passes.

## Phase 2: Usage Store And API

### Task 2.1: Define CPA-PC Usage Event Model

**Description:** Create CPA-PC's internal usage event model by adapting CPA-Manager's `usage.Event` shape to SDK plugin records.

**Acceptance criteria:**

- Event model contains fields needed by the migrated page: request ID, timestamp, provider, model, endpoint, auth fields, source fields, token fields, latency, failed flag, raw JSON.
- There is a conversion function from `sdk/cliproxy/usage.Record` to CPA-PC event.
- Event hash is deterministic enough to avoid duplicate inserts where possible.
- Sensitive source/API key material is hashed or masked before persistence.

**Verification:**

- `go test ./internal/usage`

**Dependencies:** Task 1.5

**Files likely touched:**

- `internal/usage/`

**Estimated scope:** Medium

### Task 2.2: Implement SQLite Store Schema

**Description:** Implement a local SQLite store based on CPA-Manager's schema, keeping tables needed by the migrated page.

**Acceptance criteria:**

- Store opens `usage.db-path` and creates parent directories.
- Store creates `usage_events`, `settings`, `model_prices`, `api_key_aliases`, and `dead_letter_events` tables.
- Store enables WAL, busy timeout, and foreign keys where applicable.
- Store has tests for initialization and reopen persistence.

**Verification:**

- `go test ./internal/store`

**Dependencies:** Task 2.1

**Files likely touched:**

- `internal/store/`
- `go.mod`
- `go.sum`

**Estimated scope:** Medium

### Task 2.3: Persist SDK Usage Records

**Description:** Connect the SDK usage plugin to the SQLite store.

**Acceptance criteria:**

- Usage plugin writes converted events to SQLite.
- Duplicate event hashes are ignored, not fatal.
- Store records inserted/skipped counts for status reporting.
- Store write failures are logged and do not crash the proxy path.

**Verification:**

- `go test ./internal/usage ./internal/store ./internal/app`
- Manual check: make one proxy request and inspect `data/usage.sqlite` through store tests or a debug endpoint.

**Dependencies:** Task 2.2

**Files likely touched:**

- `internal/usage/`
- `internal/store/`
- `internal/app/`

**Estimated scope:** Medium

### Task 2.4: Implement Usage Query Payload

**Description:** Implement `GET /v0/management/usage` response compatible with CPA-Manager's frontend expectations.

**Acceptance criteria:**

- Recent events are loaded by timestamp descending with `query-limit`.
- Response shape includes `total_requests`, `success_count`, `failure_count`, `total_tokens`, and nested `apis -> models -> details`.
- Empty DB returns a valid empty payload.
- Tests cover empty, success, failure, multiple model, and token aggregation cases.

**Verification:**

- `go test ./internal/usage ./internal/store ./internal/httpapi`

**Dependencies:** Task 2.3

**Files likely touched:**

- `internal/usage/`
- `internal/store/`
- `internal/httpapi/`

**Estimated scope:** Medium

### Task 2.5: Implement Usage Service Compatibility API

**Description:** Implement the minimum CPA-Manager Usage Service endpoints needed by the preserved page flow.

**Acceptance criteria:**

- `GET /usage-service/info` reports service identity, embedded mode, configured status, and started timestamp.
- `POST /setup` accepts the original setup payload and saves compatible local config defaults.
- `GET /usage-service/config` returns `ManagerConfigResponse` compatible with the frontend.
- `PUT /usage-service/config` saves compatible settings but does not start any external collector.
- `GET /status` returns DB path, event counts, dead letter counts, and collector/status fields adapted for SDK plugin mode.

**Verification:**

- `go test ./internal/httpapi ./internal/store`
- Manual check with `Invoke-WebRequest` for each endpoint.

**Dependencies:** Task 2.4

**Files likely touched:**

- `internal/httpapi/`
- `internal/store/`
- `internal/config/`

**Estimated scope:** Medium

### Task 2.6: Implement Model Prices And API Key Aliases

**Description:** Implement model price and API key alias endpoints used by CPA-Manager's monitoring UI.

**Acceptance criteria:**

- `GET /v0/management/model-prices` returns `{ "prices": ... }`.
- `PUT /v0/management/model-prices` persists prices.
- `POST /v0/management/model-prices/sync` either syncs LiteLLM prices or returns a clearly handled unsupported response if intentionally deferred.
- `GET /v0/management/api-key-aliases` returns `{ "items": ... }`.
- `PUT /v0/management/api-key-aliases` persists aliases.
- `DELETE /v0/management/api-key-aliases/:hash` removes one alias.

**Verification:**

- `go test ./internal/httpapi ./internal/store`

**Dependencies:** Task 2.5

**Files likely touched:**

- `internal/httpapi/`
- `internal/store/`

**Estimated scope:** Medium

### Task 2.7: Implement Usage Import And Export

**Description:** Port the usage import/export behavior needed by the page's monitoring tools.

**Acceptance criteria:**

- `GET /v0/management/usage/export` returns JSONL with content disposition filename.
- `POST /v0/management/usage/import` accepts JSONL and supported legacy payloads that are practical to preserve.
- Import returns added, skipped, total, failed, unsupported, warnings, and format.
- Max import size is enforced.

**Verification:**

- `go test ./internal/usage ./internal/store ./internal/httpapi`

**Dependencies:** Task 2.6

**Files likely touched:**

- `internal/usage/`
- `internal/store/`
- `internal/httpapi/`

**Estimated scope:** Medium

### Checkpoint: Usage Backend

- SDK usage records persist to SQLite.
- Usage payload endpoint returns frontend-compatible shape.
- Setup/config/status compatibility endpoints work.
- Model prices and aliases work.
- Import/export works or has an explicitly documented deferral.

## Phase 3: Frontend Migration

### Task 3.1: Import CPA-Manager Frontend Source Once

**Description:** Copy CPA-Manager frontend source into CPA-PC as a one-time import, without preserving CPA-Manager commit history.

**Acceptance criteria:**

- CPA-PC owns frontend source under `web/` or another documented frontend root.
- Imported files no longer require `../CPA-Manager` at build time.
- Package metadata is renamed from CPA-Manager naming to CPA-PC naming where minimal and safe.
- Existing source layout remains mostly intact to reduce migration risk.

**Verification:**

- Frontend dependency install succeeds using the copied package metadata.
- No import path references `../CPA-Manager`.

**Dependencies:** Usage Backend checkpoint

**Files likely touched:**

- `web/`
- `package.json` or `web/package.json`
- frontend config files copied from CPA-Manager

**Estimated scope:** Large, but mostly mechanical

### Task 3.2: Rebase Frontend Build Output To `static/management.html`

**Description:** Configure the migrated frontend build to emit a single-file management page for CPA-PC release layout.

**Acceptance criteria:**

- Frontend build command outputs a single HTML file.
- The output is copied or written to `static/management.html`.
- The build does not depend on CPA-Manager build directories.
- `static/management.html` is external and present in release layout.

**Verification:**

- `npm run build` from the frontend root.
- Manual check: `Test-Path static/management.html`.

**Dependencies:** Task 3.1

**Files likely touched:**

- frontend Vite config
- frontend package scripts
- `static/management.html`

**Estimated scope:** Medium

### Task 3.3: Configure Local Defaults For Preserved Setup Flow

**Description:** Keep the original CPA-Manager setup wizard flow but make CPA-PC's backend provide local defaults that point to the embedded service.

**Acceptance criteria:**

- First load detects CPA-PC as an embedded usage service.
- Setup wizard can default CPA base URL to same origin or `http://127.0.0.1:8317`.
- Default Management Key behavior matches `config.example.yaml`.
- Completing setup stores compatible settings in CPA-PC's SQLite settings table.
- The setup flow does not ask for a separate external Usage Service URL for the MVP path.

**Verification:**

- Frontend tests for local default resolution where practical.
- Manual browser check: new data directory, first load, complete setup wizard.

**Dependencies:** Task 3.2 and Task 2.5

**Files likely touched:**

- frontend connection/setup utilities
- `internal/httpapi/`
- `internal/store/`

**Estimated scope:** Medium

### Task 3.4: Connect Monitoring Page To CPA-PC Usage API

**Description:** Verify and adjust the migrated monitoring page so it reads usage, model prices, aliases, import/export, and status from CPA-PC same-origin APIs.

**Acceptance criteria:**

- Monitoring page loads with no console errors from missing usage endpoints.
- Empty database renders a valid empty state.
- Inserted test usage events render in the monitoring page.
- Model price save/load works.
- API key alias save/load/delete works.
- Import/export controls work or are visibly disabled only if intentionally deferred.

**Verification:**

- Frontend tests for usage service API client where practical.
- Browser check using Playwright or manual browser: open `/management.html` and inspect console/network.

**Dependencies:** Task 3.3 and Usage Backend checkpoint

**Files likely touched:**

- migrated frontend usage service client/store code
- `internal/httpapi/`
- `internal/store/`

**Estimated scope:** Medium

### Task 3.5: Remove Or Reword Obviously Wrong CPA-Manager Branding Only If Blocking

**Description:** Make only necessary text changes that prevent user confusion during MVP. Avoid broad UI redesign.

**Acceptance criteria:**

- Product-visible title/name can say CPA-PC where necessary.
- Docker-only or remote-CPA-only wording that appears in the first-run path is not misleading.
- No broad visual redesign is attempted in this phase.

**Verification:**

- Manual browser check of login/setup/monitoring/config pages.

**Dependencies:** Task 3.4

**Files likely touched:**

- migrated frontend i18n/text files
- migrated frontend page metadata

**Estimated scope:** Small

### Checkpoint: Frontend Integration

- `static/management.html` is generated from CPA-PC-owned source.
- `/management.html` loads from CPA-PC release layout.
- Setup wizard remains in place and works with local defaults.
- Monitoring page shows usage data from CPA-PC SQLite.
- Browser console has no blocking errors.

## Phase 4: Packaging And Release Layout

### Task 4.1: Add Windows amd64 Build Command

**Description:** Add a repeatable build path for Windows amd64 `cpa-pc.exe`.

**Acceptance criteria:**

- Build command produces `dist/cpa-pc.exe` or equivalent.
- Version metadata can be injected or defaults to `dev`.
- Build works on the current Windows development machine.

**Verification:**

- `$env:GOOS='windows'; $env:GOARCH='amd64'; go build -o dist/cpa-pc.exe ./cmd/cpa-pc`

**Dependencies:** Task 1.1

**Files likely touched:**

- build config or docs
- `cmd/cpa-pc/main.go`

**Estimated scope:** Small

### Task 4.2: Add Release Package Assembly

**Description:** Assemble the phase-1 release directory with executable, sample config, static management page, data directory, logs directory, README, and license.

**Acceptance criteria:**

- Output layout matches:

```text
cpa-pc_<version>_windows_amd64/
  cpa-pc.exe
  config.example.yaml
  static/
    management.html
  data/
  logs/
```

- `config.example.yaml` is included.
- `static/management.html` is included.
- Empty `data/` and `logs/` directories are represented if the packaging format preserves them, or documented if not.

**Verification:**

- Build package locally.
- Extract to a temp directory and run `cpa-pc.exe -config config.example.yaml`.

**Dependencies:** Frontend Integration checkpoint and Task 4.1

**Files likely touched:**

- package/build config
- `README.md`
- `config.example.yaml`

**Estimated scope:** Medium

### Task 4.3: Document Quick Start

**Description:** Update README for phase-1 Windows amd64 usage.

**Acceptance criteria:**

- README explains what CPA-PC is and is not.
- README documents release layout.
- README documents copying or using `config.example.yaml`.
- README documents local URLs and default Management Key behavior.
- README documents where data, auth files, logs, and SQLite are stored.

**Verification:**

- Manual doc review against PRD success criteria.

**Dependencies:** Task 4.2

**Files likely touched:**

- `README.md`
- possibly `docs/`

**Estimated scope:** Small

### Checkpoint: Release Layout

- Windows amd64 executable builds.
- Frontend builds into external `static/management.html`.
- Release directory can run from a clean temp location.
- README quick start matches actual commands and paths.

## Phase 5: End-To-End Verification

### Task 5.1: Clean Directory Smoke Test

**Description:** Run the assembled package from a clean temporary directory, not from the source tree.

**Acceptance criteria:**

- `cpa-pc.exe -config config.example.yaml` starts successfully.
- `GET /healthz` succeeds.
- `GET /management.html` returns the external static page.
- `GET /usage-service/info` succeeds.
- `GET /status` succeeds with valid auth.
- No writes escape the release directory except intentional user/system locations.

**Verification:**

- Manual test from `C:\Users\Administrator\AppData\Local\Temp\opencode` or another temp directory.

**Dependencies:** Release Layout checkpoint

**Files likely touched:** None unless smoke test exposes defects

**Estimated scope:** Small

### Task 5.2: Browser Flow Smoke Test

**Description:** Verify the preserved CPA-Manager page flow against CPA-PC from a browser.

**Acceptance criteria:**

- First load shows the expected login/setup flow.
- Local defaults are present or naturally selected.
- Completing setup succeeds.
- Monitoring page loads.
- Config page can load manager config.
- Console has no blocking errors.

**Verification:**

- Use Playwright/browser tooling or manual browser.
- Capture console errors and network failures.

**Dependencies:** Task 5.1

**Files likely touched:** None unless browser test exposes defects

**Estimated scope:** Medium

### Task 5.3: Usage Data Smoke Test

**Description:** Prove one usage event reaches SQLite and appears in the UI.

**Acceptance criteria:**

- A request that produces an SDK usage record is made.
- SQLite contains one corresponding `usage_events` row.
- `GET /v0/management/usage` includes the event.
- Monitoring UI displays the event or aggregate.

**Verification:**

- Manual request through CPA API using a configured provider or a controlled test hook if provider credentials are unavailable.
- `GET /v0/management/usage` response inspection.
- Browser monitoring page check.

**Dependencies:** Task 5.2

**Files likely touched:** None unless smoke test exposes defects

**Estimated scope:** Medium

### Checkpoint: Phase 1 Complete

- All PRD first-stage success criteria are met.
- Release package contains `.exe`, `static/management.html`, and `config.example.yaml`.
- No dependency on `../CPA-Manager` exists for runtime, build, or release.
- CPA SDK is the only long-term CPA integration dependency.

## Risks And Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Duplicate `/management.html` route conflicts with CPA SDK | High | Validate route strategy in Task 1.3 before frontend migration. Prefer SDK static asset path instead of duplicate custom route. |
| Management Key hashing breaks CPA-PC custom auth | High | Validate shared key semantics in Task 1.4 and reuse CPA-compatible auth checks where possible. |
| SDK usage plugin lacks fields the UI expects | Medium | Define conversion in Task 2.1 with graceful defaults and raw JSON preservation. Add tests for missing optional fields. |
| Migrated frontend depends on more endpoints than expected | Medium | Grep API client before migration, implement compatibility endpoints incrementally, verify with browser network logs. |
| Copying CPA-Manager code creates too much unused complexity | Medium | Copy once, then trim only when blocking. Avoid speculative rewrites in phase 1. |
| SQLite schema drifts from frontend expectations | Medium | Keep response contract tests around usage payload, model prices, aliases, import/export. |
| Packaging works only in source tree | High | Clean directory smoke test in Phase 5. |

## Parallelization Opportunities

- Frontend import can be prepared after SDK Integration checkpoint while usage store work continues, but do not wire it into release until Usage Backend checkpoint passes.
- README quick start can be drafted during packaging work, then corrected after clean directory smoke test.
- Store tests and HTTP API tests can be written in parallel after event model is defined.

## Must Stay Sequential

- Do not migrate frontend before validating SDK route and `/management.html` serving behavior.
- Do not implement full HTTP compatibility API before defining the usage event model and store schema.
- Do not package release before `static/management.html` is generated from CPA-PC-owned frontend source.

## Implementation Guardrails

- Keep changes vertical and small enough to verify after each task.
- Avoid redesigning the frontend during phase 1.
- Avoid depending on CPA-Manager after one-time code import.
- Avoid introducing scripts as the product interface; scripts/build commands are acceptable only as development or packaging helpers.
- Preserve external `static/management.html` in the release layout.
- Keep phase 1 Windows amd64 only.

## Final Acceptance Checklist

- `go test ./...` passes.
- Go build for Windows amd64 succeeds.
- Frontend build succeeds and produces `static/management.html`.
- Clean release directory starts with `cpa-pc.exe -config config.example.yaml`.
- `http://127.0.0.1:8317/management.html` loads.
- Preserved setup wizard works with local defaults.
- Same Management Key works for CPA Management API and CPA-PC usage endpoints.
- Usage record is written to SQLite and returned by `GET /v0/management/usage`.
- Monitoring page displays usage data.
- README matches actual phase-1 behavior.
