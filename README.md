# CPA-PC

## 概述

CPA-PC 是面向个人 Windows PC 的 CLI Proxy API 单体应用。它通过 CLIProxyAPI SDK 在同一个 `cpa-pc.exe` 进程内启动代理服务、本地 usage 存储和管理页面，目标是提供一个可直接解压运行的 Windows amd64 包。

本文档面向希望运行 CPA-PC 的用户，以及需要了解项目结构、构建流程和运维脚本的开发者。CPA-PC 当前定位是本机个人使用和试用交付，不是生产环境方案，也不需要通过脚本编排外部 `CLIProxyAPI.exe`。

## 功能列表

- 单进程代理服务：内嵌 CLIProxyAPI SDK 代理能力，不需要单独启动或编排 `CLIProxyAPI.exe`。
- 本地 Management UI：通过 `http://127.0.0.1:8317/management.html` 管理基础配置、API Keys、AI Providers、Auth Files、OAuth、Quota、日志和系统信息。
- 请求监控：内置本地 usage API，使用 `data/usage.sqlite` 持久化请求、token、cost、TPM、API-key alias 和模型单价等监控数据。
- 监控图表：提供 `/monitoring/charts` 页面，支持最近 1 小时、5 小时、24 小时、7 天固定窗口；粒度随范围联动为 10 分钟、小时或天。
- 多维度图表：支持全局总量、Provider、API-key 和模型维度，展示 token 用量/累计 token、cost/累计 cost 和 TPM 趋势；token 与 cost 图在同一行内用 tab 切换合计与累计视图。
- 安全展示：API-key 图表和筛选项只展示 alias 或哈希前缀，不展示原始 API key；模型无单价时按 0 cost 处理并在页面提示。
- 本地接口：提供 `/healthz`、`/cpa-pc/info`、`/v0/management/usage*`、模型单价和 API-key alias 等 CPA-PC 扩展接口。
- 外置配置：使用 `config.yaml`/`config.example.yaml` 管理运行配置，数据、日志、静态资源和 usage DB 路径相对配置文件解析。
- 外置静态管理页：通过 `static/management.html` 提供管理页面，前端可独立构建并在打包时复制。
- Windows 运维脚本：提供计划任务管理脚本，可注册登录自启动、启动、停止、查看状态和删除任务。
- Windows 发布包：打包生成发布目录和同名 zip，包含可执行文件、示例配置、管理页、Windows 脚本和必要文档。

## Quick Start

安装根目录依赖：

```powershell
npm install
```

构建 Windows 发布目录和 zip：

```powershell
npm run package:windows -- --version dev
```

如果需要同时重新生成 `static/management.html`，先安装前端依赖，再传入 `--build-frontend`：

```powershell
npm --prefix web ci
npm run package:windows -- --version dev --build-frontend
```

运行发布目录：

```powershell
cd dist\cpa-pc_dev_windows_amd64
Copy-Item .\config.example.yaml .\config.yaml
.\cpa-pc.exe
```

也可以不复制配置文件，直接显式指定示例配置：

```powershell
.\cpa-pc.exe -config .\config.example.yaml
```

启动后访问：

- Management UI: `http://127.0.0.1:8317/management.html`
- Health check: `http://127.0.0.1:8317/healthz`
- CPA-PC info: `http://127.0.0.1:8317/cpa-pc/info`

## 发布产物

Windows amd64 打包会生成发布目录和同名 zip：

```text
dist/
  cpa-pc_<version>_windows_amd64/
    cpa-pc.exe
    manage-cpa-pc.ps1
    start-cpa-pc.vbs
    config.example.yaml
    static/
      management.html
    data/
    logs/
  cpa-pc_<version>_windows_amd64.zip
```

`README.md` 和 `LICENSE` 会在存在时一并复制到发布目录。

## 项目结构

```text
cmd/cpa-pc/                 Go 主程序入口
internal/                   Go 应用包、配置、HTTP API、usage 和存储逻辑
web/                        React/Vite 管理页面源码
static/                     构建后的 management.html，打包时复制到发布目录
scripts/package-windows.ts  Windows 发布目录和 zip 打包脚本
scripts/win/                Windows 计划任务管理脚本和隐藏启动器
docs/specs/                 已确认的规格文档和参考脚本
config.example.yaml         发布包示例配置
dist/                       本地生成的发布目录和 zip
```

## 运行脚本说明

根目录 npm 脚本：

| 命令 | 说明 |
| --- | --- |
| `npm run build:web` | 构建前端并更新 `static/management.html`。 |
| `npm run package:windows -- --version dev` | 构建 Windows amd64 发布目录和同名 zip。 |
| `npm run package:windows -- --version dev --build-frontend` | 先构建前端，再构建 Windows 发布目录和 zip。 |
| `npm run package:windows -- --help` | 查看 Windows 打包参数。 |

Windows 管理脚本在发布目录内运行：

| 命令 | 说明 |
| --- | --- |
| `.\manage-cpa-pc.ps1` | 打开交互式管理菜单。 |
| `.\manage-cpa-pc.ps1 -Action status` | 查看 `CPAPCTask` 计划任务和 `cpa-pc` 进程状态。 |
| `.\manage-cpa-pc.ps1 -Action create` | 注册隐藏的当前用户登录自启动计划任务。 |
| `.\manage-cpa-pc.ps1 -Action start` | 触发计划任务立即启动 `cpa-pc.exe`。 |
| `.\manage-cpa-pc.ps1 -Action stop` | 停止当前运行的 `cpa-pc` 进程。 |
| `.\manage-cpa-pc.ps1 -Action remove` | 删除 `CPAPCTask` 计划任务。 |

`start-cpa-pc.vbs` 是计划任务使用的隐藏启动器，默认不需要手动运行。它会从自身所在目录启动同级 `cpa-pc.exe`。

开发检查命令：

```powershell
go test ./...
go build ./cmd/cpa-pc
npm --prefix web test -- --run
npm --prefix web run lint
npm --prefix web run build
```

## 本地数据和配置

默认路径来自 `config.example.yaml`，相对配置文件所在目录解析：

- `data-dir`: `./data`
- `logs-dir`: `./logs`
- `static-dir`: `./static`
- `usage.db-path`: `./data/usage.sqlite`
- `auth-dir`: `~/.cli-proxy-api`

默认 `logging-to-file: false`，开启后日志写入配置的日志目录。

## 注意事项

- 本项目代码主要由 AI 生成，尚未经过严格人工 review，请勿在生产环境使用。
- 默认 Management Key 是 `123456`，仅建议本机试用；对外开放前必须修改 `config.yaml` 中的 `remote-management.secret-key`。
- `manage-cpa-pc.ps1 -Action create` 会注册本机计划任务，脚本会在需要时请求管理员权限。
- `manage-cpa-pc.ps1 -Action remove` 会删除名为 `CPAPCTask` 的计划任务。
- `manage-cpa-pc.ps1 -Action stop` 会停止进程名为 `cpa-pc` 的本机进程。
- `dist/` 是生成产物目录，不作为源码维护入口。
