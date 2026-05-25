# Spec: Sticky Session Routing

## Objective

Build a CPA-PC routing mode that lets management users choose sticky session routing and causes upstream API callers from the same session to reuse the same credential when possible.

Target users:

- Management users configuring routing from the CPA-PC UI.
- Upstream API callers whose requests are proxied through CPA-PC.

Research findings:

- `../new-api` implements channel affinity in `setting/operation_setting/channel_affinity_setting.go`, `service/channel_affinity.go`, and `middleware/distributor.go`.
- `new-api` matches configurable rules by model regex, path regex, optional user-agent includes, and key sources such as Gin context values or `gjson` paths in the request body.
- `new-api` stores an in-memory or Redis TTL mapping from an affinity key to a channel ID, tries the preferred channel before normal routing, records the successful channel after a successful response, and falls back when the bound channel is disabled or unavailable unless the rule says to skip retry.
- Current CLIProxyAPI already has equivalent credential-level behavior in `sdk/cliproxy/auth.SessionAffinitySelector` with config fields `routing.session-affinity` and `routing.session-affinity-ttl`.
- CLIProxyAPI extracts session IDs from `metadata.user_id`, `X-Session-ID`, `Session_id`, `X-Amp-Thread-Id`, `X-Client-Request-Id`, `conversation_id`, and message-hash fallbacks for OpenAI, Claude, Gemini, and Responses payloads.
- CLIProxyAPI cache keys include provider, session ID, and model, so sticky routing is credential-scoped and cross-provider/model isolated.
- CLIProxyAPI failover is already built in: if the cached auth is no longer available, selection falls back to the underlying selector and updates the binding.

Recommended scope:

- Do not port `new-api` channel affinity wholesale.
- Use the existing CLIProxyAPI SDK `session-affinity` runtime support.
- In CPA-PC, expose sticky routing as a management UI option that persists to existing config fields.
- Default sticky TTL to `12h` for CPA-PC.
- Provide a CLIProxyAPI patch suggestion only if direct management APIs for session-affinity are required beyond YAML config editing.

Acceptance criteria:

- Management UI exposes a routing choice named `Sticky` or equivalent localized text.
- Selecting sticky persists `routing.session-affinity: true` and `routing.session-affinity-ttl: "12h"` when no TTL is already set.
- Non-sticky choices keep existing `round-robin` and `fill-first` behavior.
- Runtime uses the same credential for the same session while the binding is valid and the credential remains available.
- Runtime automatically switches to another credential when the bound credential is unavailable due to disablement, cooldown, quota, or repeated failure state handled by CLIProxyAPI.
- Session bindings are memory-only and expire by TTL.
- All supported proxy routes benefit from the SDK selector without route-specific rewrites.

## Tech Stack

- Backend: Go 1.26, Gin, embedded `github.com/router-for-me/CLIProxyAPI/v7` SDK.
- Runtime config: YAML loaded by both CPA-PC and CLIProxyAPI.
- Frontend: React, TypeScript, Vite, SCSS modules, `yaml` parser for visual config editing.
- Persistence: no new database table for sticky routing; use in-memory SDK session cache only.

## Commands

- Backend full test: `go test ./...`
- Backend focused test: `go test ./internal/config`
- Frontend install: `npm --prefix web ci`
- Frontend tests: `npm --prefix web test`
- Frontend lint: `npm --prefix web run lint`
- Frontend type check: `npm --prefix web run type-check`
- Frontend build and static asset generation: `npm --prefix web run build`
- Windows package with rebuilt frontend: `npm run package:windows -- --version dev --build-frontend`

## Project Structure

- `config.example.yaml` contains the documented CPA-PC default routing block and should show `session-affinity-ttl: "12h"` for sticky usage.
- `internal/app` wires the existing CLIProxyAPI SDK config through unchanged; no new standalone proxy launcher is introduced.
- `internal/httpapi` remains for CPA-PC-only routes. Add routes here only if CPA-PC needs a local wrapper API for sticky config and the SDK cannot expose it.
- `web/src/components/config/VisualConfigEditor.tsx` owns the visual routing controls.
- `web/src/hooks/useVisualConfig.ts` parses and writes YAML routing fields.
- `web/src/services/api/transformers.ts` normalizes `/config` responses into frontend config state.
- `web/src/types/config.ts` and `web/src/types/visualConfig.ts` hold typed routing fields.
- `web/src/i18n/locales/*.json` holds localized labels for the sticky option.
- `static/management.html` is generated output and must be rebuilt from `web/`, not edited by hand.

## Code Style

Keep the implementation small and map the UI concept onto existing SDK config fields rather than introducing a parallel routing system.

Example TypeScript style for the intended mapping:

```ts
type RoutingMode = 'round-robin' | 'fill-first' | 'sticky';

const toRoutingYaml = (mode: RoutingMode, ttl: string) => ({
  strategy: mode === 'fill-first' ? 'fill-first' : 'round-robin',
  'session-affinity': mode === 'sticky',
  'session-affinity-ttl': mode === 'sticky' ? ttl.trim() || '12h' : ttl.trim(),
});
```

Conventions:

- Prefer existing names: `routing`, `strategy`, `session-affinity`, and `session-affinity-ttl`.
- Keep sticky as a UI mode or derived state unless CLIProxyAPI officially accepts `routing.strategy: sticky`.
- Do not log raw session IDs; use existing SDK truncation/fingerprinting behavior.
- Preserve existing React and SCSS module patterns.
- Keep YAML key spelling compatible with CLIProxyAPI: kebab-case keys in persisted YAML.

## Testing Strategy

Backend:

- Run `go test ./...` to prove CPA-PC still loads and embeds CLIProxyAPI correctly.
- Add focused tests only if CPA-PC backend code is introduced, such as a wrapper endpoint for sticky config.
- Rely on CLIProxyAPI's existing `SessionAffinitySelector` tests for core routing semantics unless CLIProxyAPI is patched.

Frontend:

- Add or update unit tests for YAML parsing and writing in `web/src/hooks/useVisualConfig.ts` if sticky mode changes that logic.
- Add or update component tests for the visual routing control if an explicit sticky option is added.
- Run `npm --prefix web test`, `npm --prefix web run lint`, `npm --prefix web run type-check`, and `npm --prefix web run build`.

Manual verification:

- In visual config, select sticky routing and save.
- Confirm `config.yaml` contains `routing.session-affinity: true` and a TTL of `12h` unless the user provided another TTL.
- Restart or reload CPA-PC and verify the config remains visible in the UI.
- Send two requests with the same session signal, such as `X-Session-ID`, and confirm logs show the same selected auth ID while available.
- Disable or cool down the selected credential and confirm the next request reselects another available credential.

CLIProxyAPI patch verification, if needed:

- Add management handler tests for reading and writing `routing.session-affinity` and `routing.session-affinity-ttl`.
- Add config reload tests proving selector replacement when affinity or TTL changes.
- Run CLIProxyAPI's relevant tests in `sdk/cliproxy/auth` and `internal/api/handlers/management`.

## Boundaries

Always:

- Use the embedded CLIProxyAPI SDK path first.
- Keep session binding memory-only with TTL.
- Preserve automatic failover when the bound credential is unavailable.
- Preserve existing `round-robin` and `fill-first` behavior.
- Preserve CPA-PC single-process topology.
- Preserve unknown YAML pass-through fields because CLIProxyAPI consumes them.
- Rebuild `static/management.html` from `web/` after frontend changes.

Ask first:

- Add a database table or persist sticky session mappings across restarts.
- Add Redis or another external cache.
- Modify `../CLIProxyAPI` directly instead of providing a patch suggestion.
- Change CLIProxyAPI session ID extraction priority or message-hash behavior.
- Add a true random or weighted-random selector if round-robin distribution is not acceptable for new sessions.
- Change public management API shape beyond the existing YAML config behavior.

Never:

- Copy `../new-api` affinity implementation wholesale into CPA-PC.
- Add `../new-api` or `../CLIProxyAPI` as a long-term source dependency, submodule, or runtime dependency.
- Edit `static/management.html` by hand.
- Reintroduce a separate `CLIProxyAPI.exe` launcher.
- Store or expose raw API keys or raw session identifiers in logs, SQLite, or UI state.
- Register a duplicate `GET /management.html` route in CPA-PC.

## Success Criteria

- The implementation uses existing CLIProxyAPI SDK session affinity for runtime sticky routing.
- Management users can select sticky routing without editing raw YAML manually.
- Same-session requests prefer the same credential/API key/channel equivalent.
- Different sessions are distributed by the configured fallback selector for initial binding.
- Bound credentials fail over automatically when unavailable.
- Sticky TTL defaults to `12h` and is configurable.
- No persistent session mapping storage is added.
- Verification commands pass.

## Open Questions

- Should sticky mode use `round-robin` as the initial distribution fallback, or is true random initial assignment mandatory?
- Should selecting non-sticky clear `session-affinity-ttl`, or preserve the previous TTL for later sticky re-enable?
- Should CPA-PC add a small wrapper management endpoint for sticky settings now, or should this wait for a CLIProxyAPI patch that exposes `routing/session-affinity` and `routing/session-affinity-ttl` directly?

## CLIProxyAPI Patch Suggestion

Only if direct management API support is required in `../CLIProxyAPI`, add endpoints equivalent to:

- `GET /v0/management/routing/session-affinity`
- `PUT /v0/management/routing/session-affinity` with `{ "value": true | false }`
- `GET /v0/management/routing/session-affinity-ttl`
- `PUT /v0/management/routing/session-affinity-ttl` with `{ "value": "12h" }`

The patch should update `Config.Routing.SessionAffinity` and `Config.Routing.SessionAffinityTTL`, call the existing persist flow, and rely on the current reload code that rebuilds the selector when affinity or TTL changes.
