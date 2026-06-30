# CPA SDK 7.2.20 Upgrade Report

Status: SDK upgraded and verified.

Spec: `docs/SPEC.md`

Plan: `docs/tasks/plan.md`

Tasks: `docs/tasks/todo.md`

## Baseline

- CPA-PC branch: `main`
- Current CPA-PC SDK dependency: `github.com/router-for-me/CLIProxyAPI/v7 v7.1.39`
- Target CPA SDK dependency: `github.com/router-for-me/CLIProxyAPI/v7 v7.2.20`
- SDK reference source: `D:\develop\workspace\CLIProxyApi`
- SDK reference branch: `main`
- SDK reference head: `041a065b`
- SDK baseline tag: `v7.1.39` at `f353979e`
- SDK target tag: `v7.2.20` at `041a065b`
- SDK comparison range: `v7.1.39..v7.2.20` (`205` commits)
- SDK reference status: untracked local artifacts only (`.codegraph/`, `server.heapsnapshot`, `tui.heapsnapshot`); tag-based comparison will avoid relying on local working-tree files.
- CPA-PC worktree note: existing `.opencode/` deletions/untracked skill files were present at implementation start and are outside this upgrade scope.

## Scope

Analyze and apply only changes relevant to CPA-PC's embedded CLIProxyAPI SDK integration:

- App startup and SDK embedding
- Config loading and pass-through behavior
- Management routes and route ownership
- Management authentication
- Static management UI serving
- Usage plugin hooks and local persistence integration
- Logging and runtime path behavior
- Public APIs used by CPA-PC
- Dependency changes required by SDK `v7.2.20`

## Impact Summary

- Upstream diff size is large: `550` files, `68842` insertions, `16994` deletions.
- CPA-PC's direct SDK import surface is narrow: `sdk/api`, `sdk/api/handlers`, `sdk/cliproxy`, `sdk/config`, `sdk/translator/builtin`, and `sdk/cliproxy/usage`.
- The upgrade is expected to be mostly non-breaking for CPA-PC code because current CPA-PC imports do not use the removed Amp or Gemini CLI exported SDK aliases/routes.
- `config.example.yaml` needs a targeted upstream merge because SDK `7.2.20` changed example config semantics around plugins, image-generation mode, video auth cache TTL, Claude cloak mode, auth cooldown, Gemini CLI endpoint removal, and Amp removal.
- New upstream management/API capabilities may require frontend sync if official UI exposes them, especially plugin-related management and API-key disable controls.

## Additions

- Plugin runtime support across SDK services:
  - `cliproxy.Builder.WithPluginHost`.
  - Plugin auth parsing hooks.
  - Plugin executor/model registration flows.
  - Plugin API/ABI packages (`sdk/pluginapi`, `sdk/pluginabi`).
  - Plugin management/store examples and delete endpoint support.
- Named usage plugin registration:
  - `usage.Manager.RegisterNamed`.
  - `usage.RegisterNamedPlugin` for default manager registration.
- Config and runtime additions:
  - `plugins` config block in upstream example.
  - `transient-error-cooldown-seconds`.
  - `disable-claude-cloak-mode`.
  - `disable-image-generation: "passthrough"` mode.
  - `video-result-auth-cache-ttl`.
- Auth/runtime additions:
  - Persistent cooldown state management.
  - Retryable `home_unavailable` classification for transport errors.
  - API-key disable management handling.
- Request/translator/API additions:
  - Direct image API proxying and `gpt-image-1.5` support.
  - Video result credential pinning behavior.
  - OpenAI response conversion improvements for structured `tool_choice`, `input_image`, namespace tools, service tier normalization, and object schema properties.
  - Plugin request/stream/model-router interception surfaces.

## Modifications

- `sdk/cliproxy/service.go` changed substantially to support plugin runtime sync, plugin executor registration, auth refresh callbacks, and model registration concurrency.
- `sdk/cliproxy/builder.go` now wires plugin host and runtime auth sync behavior.
- `sdk/cliproxy/types.go` adds watcher/runtime dispatch helpers, plugin auth parser hooks, and persisted auth update dispatch.
- `sdk/cliproxy/usage/manager.go` supports replacing plugins by name while preserving existing unnamed registration.
- `sdk/api/handlers/handlers.go` adds model execution/router and plugin interception behavior.
- `internal/api/server.go` adds config reload and plugin host options upstream; CPA-PC embeds through `cliproxy.NewBuilder`, so compile verification will determine whether any adapter changes are needed.
- Upstream config parsing removed legacy migration code and adds plugin config normalization.
- Upstream example config removes Gemini CLI endpoint and Amp sections, broadens model alias/exclusion examples for plugin providers, and updates session key comments.
- `go.mod` in upstream makes `golang.org/x/sys v0.38.0` a direct dependency.

## Breaking Changes

- Upstream breaking change: Amp integration support was removed (`feat!`).
- Upstream removed Gemini CLI-related translator/packages and management token route surface.
- Upstream `sdk/config.Config` no longer exports the `AmpCode` type alias.
- CPA-PC impact classification: potentially breaking only for users who relied on upstream Amp/Gemini CLI example config or behavior through CPA-PC pass-through config. Current CPA-PC code does not directly import those removed symbols.
- Expected CPA-PC user-facing status after targeted config merge: non-breaking for existing CPA-PC usage pages, local persistence, SDK embedding, and management auth.

Final CPA-PC impact:

- Backend SDK dependency now runs against CLIProxyAPI `v7.2.20`.
- Existing CPA-PC backend tests pass without Go source adapter changes.
- Example config now reflects SDK `7.2.20` additions/removals while preserving CPA-PC-specific config.
- Frontend now exposes plugin-management capabilities when the upgraded SDK reports plugin support.

Classification will use:

- Non-breaking: no behavior or integration contract break for CPA-PC users/maintainers.
- Potentially breaking: could affect configuration, routing, auth, persistence, packaging, or management UI behavior depending on deployment.
- Breaking: requires user/admin action or changes an existing CPA-PC contract.

## SDK Diff Findings

Comparison source: `D:\develop\workspace\CLIProxyApi` range `v7.1.39..v7.2.20`.

Relevant changed files and areas:

- Root/config: `config.example.yaml`, `go.mod`.
- SDK server/API: `sdk/api/management.go`, `sdk/api/handlers/handlers.go`, OpenAI image/video/websocket handlers.
- SDK service: `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`, `sdk/cliproxy/types.go`, `sdk/cliproxy/usage/manager.go`, `sdk/cliproxy/watcher.go`.
- SDK config: `sdk/config/config.go`, upstream `internal/config/config.go`, `internal/config/sdk_config.go`, `internal/config/parse.go`.
- Management/API internals: upstream `internal/api/server.go`, management config/API-key handlers, management asset updater.
- Translator/runtime: `sdk/translator/*`, OpenAI/Codex/Kimi/xAI/auth related internals.

CPA-PC direct imports checked:

- `internal/app/cliproxy.go` imports `sdk/api`, `sdk/api/handlers`, `sdk/cliproxy`, `sdk/config`, and `sdk/translator/builtin`.
- `internal/usage/*` imports `sdk/cliproxy/usage` and implements `usage.Plugin` through `HandleUsage(context.Context, usage.Record)`.
- No CPA-PC source imports removed Amp or Gemini CLI SDK symbols.

## CPA-PC Integration Changes

Applied changes:

- Updated root `go.mod` from `github.com/router-for-me/CLIProxyAPI/v7 v7.1.39` to `v7.2.20`.
- Updated `go.sum` checksums for `github.com/router-for-me/CLIProxyAPI/v7 v7.2.20`.
- Updated `config.example.yaml` to reflect SDK `7.2.20` example config changes while preserving CPA-PC-specific keys.
- No Go source compatibility edits were required.

Verified compatibility points:

- `cliproxy.NewBuilder().WithConfig(...).WithConfigPath(...).WithServerOptions(...).Build()` compiles.
- `api.WithRouterConfigurator` still accepts CPA-PC route registration callback.
- `handlers.BaseAPIHandler` still exposes `AuthManager.List()` data used by chart metadata snapshots.
- `service.RegisterUsagePlugin(pcusage.NewPersistPlugin(...))` still compiles with the SDK usage plugin interface.
- CPA-PC-owned backend tests still pass against SDK `v7.2.20`.

Config merge details:

- Added the upstream `plugins` example block.
- Added `save-cooldown-status`.
- Added `transient-error-cooldown-seconds`.
- Added `disable-claude-cloak-mode`.
- Updated `disable-image-generation` comments to include `"passthrough"`.
- Added `video-result-auth-cache-ttl`.
- Removed the retired `enable-gemini-cli-endpoint` example.
- Removed the retired Amp example section.
- Updated session-affinity comments to remove Amp-specific session ID wording.
- Updated OAuth alias/excluded-model examples to remove `gemini-cli` and include `xai`/plugin-provider examples.
- Preserved CPA-PC-specific keys: `data-dir`, `logs-dir`, `static-dir`, `usage`, and `remote-management.secret-key: "123456"`.

## Verification

Commands run:

```powershell
go get github.com/router-for-me/CLIProxyAPI/v7@v7.2.20
go mod tidy
go test ./...
```

Results:

- `go get`: passed, upgraded `github.com/router-for-me/CLIProxyAPI/v7 v7.1.39 => v7.2.20`.
- `go mod tidy`: passed with no output.
- `go test ./...`: passed after SDK upgrade.
- `go test ./...`: passed again after `config.example.yaml` merge.
- `go test ./...`: passed in final verification after UI sync/build.
- `go test ./internal/config`: passed after adding the missing `save-cooldown-status` example key.
- `npm run package:windows -- --build-frontend`: passed after version bump; created `dist\cpa-pc_0.2.11_windows_amd64` and `dist\cpa-pc_0.2.11_windows_amd64.zip`.
- `dist\cpa-pc_0.2.11_windows_amd64\cpa-pc.exe -version`: returned `cpa-pc 0.2.11`.
- `static\management.html`: contains frontend version `1.1.8` and no longer contains `1.1.7`.

Passing packages:

- `github.com/HWliao/CPA-PC/cmd/cpa-pc`
- `github.com/HWliao/CPA-PC/internal/app`
- `github.com/HWliao/CPA-PC/internal/config`
- `github.com/HWliao/CPA-PC/internal/httpapi`
- `github.com/HWliao/CPA-PC/internal/store`
- `github.com/HWliao/CPA-PC/internal/usage`

## Residual Risks

- Users with old config files containing removed upstream Amp/Gemini CLI keys may need to review upstream `7.2.20` migration behavior; CPA-PC does not add a separate migration layer.
- Plugin support depends on SDK runtime/plugin configuration; plugin UI is hidden unless the backend advertises support.
- Official UI provider workbench changes were not applied to avoid a broad, potentially disruptive replacement of CPA-PC provider pages.
