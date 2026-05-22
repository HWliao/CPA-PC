# CPA-PC

CPA-PC 是面向个人 Windows PC 的 CLI Proxy API 单体应用。它通过 CLIProxyAPI SDK 嵌入代理服务, 在同一个进程内提供本地 SQLite usage 存储和管理页面。

CPA-PC 当前目标:

- 提供 Windows amd64 可运行包。
- 使用 CLIProxyAPI SDK 集成, 不通过脚本启动独立 `CLIProxyAPI.exe`。
- 内置 CPA-Manager 兼容 usage API, 不启动独立 CPA-Manager 服务。
- 使用 `data/usage.sqlite` 持久化监控数据。
- 通过外置 `static/management.html` 提供管理页面。

CPA-PC 当前不是:

- 生产环境方案。
- 多进程编排器或脚本启动器。
- CPA-Manager 后端的运行时依赖。

## 注意事项

本项目代码主要由 AI 生成, 尚未经过严格人工 review, 请勿在生产环境使用。

## Release Layout

Windows amd64 发布目录结构:

```text
cpa-pc_<version>_windows_amd64/
  cpa-pc.exe
  config.example.yaml
  static/
    management.html
  data/
  logs/
```

`README.md` 和 `LICENSE` 会在存在时一并复制到发布目录。

## Quick Start

构建发布目录:

```powershell
pwsh -File scripts/package-windows.ps1 -Version dev
```

如果需要同时重新生成 `static/management.html`, 先安装前端依赖, 再加 `-BuildFrontend`:

```powershell
npm --prefix web ci
pwsh -File scripts/package-windows.ps1 -Version dev -BuildFrontend
```

运行发布目录:

```powershell
cd dist\cpa-pc_dev_windows_amd64
Copy-Item .\config.example.yaml .\config.yaml
.\cpa-pc.exe
```

也可以不复制配置文件, 直接显式指定示例配置:

```powershell
.\cpa-pc.exe -config .\config.example.yaml
```

启动后访问:

- Management UI: `http://127.0.0.1:8317/management.html`
- Health check: `http://127.0.0.1:8317/healthz`
- CPA-PC info: `http://127.0.0.1:8317/cpa-pc/info`

默认 Management Key 是 `123456`。仅建议本机试用, 对外开放前必须修改 `config.yaml` 中的 `remote-management.secret-key`。

## Local Data

默认路径来自 `config.example.yaml`, 相对配置文件所在目录解析:

- `data-dir`: `./data`
- `logs-dir`: `./logs`
- `static-dir`: `./static`
- `usage.db-path`: `./data/usage.sqlite`
- `auth-dir`: `~/.cli-proxy-api`

默认 `logging-to-file: false`, 开启后日志写入配置的日志目录。

## Development Checks

```powershell
go test ./...
go build ./cmd/cpa-pc
npm --prefix web test -- --run
npm --prefix web run lint
npm --prefix web run build
```
