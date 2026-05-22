# Spec: Merge Complete CPA Example Config Into CPA-PC

Status: Approved and implemented.

## Objective

Improve `config.example.yaml` as a deployment configuration template for CPA-PC users.

The target users are people packaging or running CPA-PC on Windows who need a complete example configuration that documents the upstream CLIProxyAPI options while keeping CPA-PC-specific defaults and runtime paths.

Success means `config.example.yaml` is updated from `../CLIProxyApi/config.example.yaml`, but CPA-PC-specific differences are preserved instead of blindly overwriting them.

Acceptance criteria:

- `config.example.yaml` includes the complete upstream CPA example sections from `../CLIProxyApi/config.example.yaml`.
- CPA-PC-specific settings remain present: `data-dir`, `logs-dir`, `static-dir`, and `usage`.
- CPA-PC deployment default `remote-management.secret-key: "123456"` is preserved instead of using the upstream empty value.
- If an upstream CPA option conflicts with CPA-PC behavior or dependencies, stop and ask before deciding.
- Only `docs/specs/2026-05-22-merge-cpa-config-example.md` and `config.example.yaml` are in scope for this change.

Confirmed decisions:

- Keep `remote-management.secret-key: "123456"` for CPA-PC deployment convenience.
- Keep upstream CPA fields that are not owned by `internal/config.Config`; they are SDK pass-through options loaded by CLIProxyAPI.

## Tech Stack

- Go module: `github.com/HWliao/CPA-PC`, Go `1.26.0`.
- Embedded upstream SDK: `github.com/router-for-me/CLIProxyAPI/v7`.
- Config format: YAML parsed with `gopkg.in/yaml.v3`.
- Frontend: React `19.2.1`, TypeScript `5.9.3`, Vite `8.0.10`, Vitest `4.1.5`.
- Packaging: root npm scripts plus `scripts/package-windows.ts`.

## Commands

Repository verification:

```powershell
go test ./...
go build ./cmd/cpa-pc
npm --prefix web run lint
npm --prefix web run build
```

Focused config verification:

```powershell
go test ./internal/config ./internal/app
```

Windows package build:

```powershell
npm install
npm run package:windows -- --version dev
```

Windows package build with frontend rebuild:

```powershell
npm --prefix web ci
npm run package:windows -- --version dev --build-frontend
```

Run with the example config:

```powershell
go run ./cmd/cpa-pc -config .\config.example.yaml
```

## Project Structure

```text
config.example.yaml                 -> CPA-PC deployment config template to update
../CLIProxyApi/config.example.yaml   -> upstream CPA source example to merge from
internal/config/                     -> CPA-PC config loader, defaults, and config tests
internal/app/                        -> embedded CLIProxyAPI service wiring
cmd/cpa-pc/                          -> CPA-PC executable entry point
web/                                 -> management UI source, build, lint, and tests
scripts/                             -> Windows packaging scripts
```

Important behavior:

- `internal/config.Load` reads CPA-PC-owned fields and ignores unknown YAML fields.
- `internal/app.loadCPAConfig` also loads the same config path through the CLIProxyAPI SDK, so upstream CPA fields in `config.example.yaml` are meaningful.
- CPA-PC path fields are resolved relative to the config file: `data-dir`, `logs-dir`, `static-dir`, and `usage.db-path`.

## Code Style

YAML should use the upstream CPA example's comment-heavy style while keeping CPA-PC additions clearly grouped and minimally changed.

Example style:

```yaml
# CPA-PC runtime paths are resolved relative to this config file.
data-dir: "./data"
logs-dir: "./logs"
static-dir: "./static"

# Management API settings
remote-management:
  allow-remote: false
  secret-key: "123456"
  disable-control-panel: false

# CPA-PC local usage database
usage:
  enabled: true
  db-path: "./data/usage.sqlite"
  query-limit: 50000
```

Conventions:

- Keep YAML indentation at two spaces.
- Prefer quoted strings where the existing config uses quoted strings.
- Preserve upstream comments unless they are incorrect for CPA-PC.
- Keep placeholders example-only; do not introduce real API keys, tokens, secrets, certificates, or local absolute paths.
- Do not reformat unrelated files.

## Testing Strategy

Primary validation:

- Run `go test ./...` after changing `config.example.yaml` because `internal/config/config_test.go` loads the example config.
- Run `go build ./cmd/cpa-pc` to confirm the executable still builds.

Secondary validation when frontend/package impact is possible:

- Run `npm --prefix web run lint` if frontend source is touched.
- Run `npm --prefix web run build` if frontend or package output is affected.
- Run `npm run package:windows -- --version dev` when release layout behavior needs verification.

Manual review:

- Compare the updated `config.example.yaml` against `../CLIProxyApi/config.example.yaml` and confirm upstream sections were not accidentally omitted.
- Confirm CPA-PC-specific keys are still present and documented.
- Confirm no real credentials or machine-specific paths were copied.

## Boundaries

Always:

- Modify only `docs/specs/2026-05-22-merge-cpa-config-example.md` during specification and only `config.example.yaml` during implementation unless the user approves more files.
- Preserve CPA-PC-specific config fields and deployment defaults.
- Use `../CLIProxyApi/config.example.yaml` as the upstream source.
- Run focused config tests after implementation.
- Keep example values safe placeholders.

Ask first:

- Any conflict between upstream CPA defaults and CPA-PC defaults.
- Any dependency, SDK version, packaging, or runtime code change.
- Any change to `README.md`, tests, Go code, frontend code, or packaging scripts.
- Any decision to remove an existing CPA-PC config key.

Never:

- Modify real local config files such as `config.yaml`.
- Commit secrets, API keys, tokens, certificates, or environment-specific paths.
- Change dependency versions without explicit approval.
- Delete failing tests or weaken validation to make tests pass.
- Replace CPA-PC-specific defaults with upstream values silently.

## Success Criteria

- `docs/specs/2026-05-22-merge-cpa-config-example.md` was reviewed and approved before implementation began.
- `config.example.yaml` is a merged deployment template: complete upstream CPA example plus CPA-PC-specific sections.
- Focused verification passed with `go test ./internal/config ./internal/app`.
- Full verification passed with `go test ./...`.
- Build verification passed with `go build ./cmd/cpa-pc`.

## Open Questions

- None.
