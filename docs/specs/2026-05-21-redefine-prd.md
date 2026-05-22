# CPA-PC PRD

## 背景

CLIProxyAPI 提供 CPA 核心代理能力和可嵌入 Go SDK。CPA-Manager 提供管理页面和 Usage Service，用 SQLite 持久化请求统计。CPA-PC 要做的是仿照 CPA-Manager 的交付形态，但不是单独启动一个外部 Usage Service，而是做一个新的单体 Go 应用：在同一个进程内集成 CPA SDK、扩展 usage service、迁移 CPA-Manager 页面能力，最终面向个人 PC 打包分发。

## 相关源码位置

- CPA 项目：`../CLIProxyAPI`
- CPA-Manager 项目：`../CPA-Manager`
- CPA-PC 项目：当前仓库

这些路径表示本地开发环境中的兄弟源码目录。CPA-PC 初始阶段可以参考 CLIProxyAPI 和 CPA-Manager 的实现，但不把两个项目源码目录直接加入本仓库，也不把 CPA-Manager 作为长期依赖。

## 产品目标

面向个人 PC 用户提供一个开箱即用的 CPA 本地服务程序。用户运行 `cpa-pc.exe` 后，同时获得：

- OpenAI/Gemini/Claude/Codex 兼容代理 API。
- 本地管理页面。
- 请求统计持久化和可视化。
- 本地配置文件、认证文件、日志和 SQLite 数据目录。

## 非目标

- 不做外部脚本启动器。
- 不把 CLIProxyAPI 和 CPA-Manager 作为子目录或 submodule 加入 CPA-PC。
- 不长期依赖 CPA-Manager 的源码目录、构建产物或发布包。
- 不在第一阶段重新设计一套全新的管理页面，初始页面能力从 CPA-Manager 迁移而来。
- 第一阶段不做托盘程序或原生 GUI。
- 第一阶段不支持生产部署场景。

## 技术方向

CPA-PC 是一个 Go module，主程序结构参考 CPA-Manager Usage Service，但核心服务由 CLIProxyAPI SDK 内嵌启动。

建议依赖：

- `github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy`
- `github.com/router-for-me/CLIProxyAPI/v7/sdk/config`

CPA SDK 的最小嵌入方式：

```go
cfg, err := config.LoadConfig("config.yaml")
if err != nil {
    return err
}

svc, err := cliproxy.NewBuilder().
    WithConfig(cfg).
    WithConfigPath("config.yaml").
    Build()
if err != nil {
    return err
}

if err := svc.Run(ctx); err != nil {
    return err
}
```

CPA SDK 已提供 `RegisterUsagePlugin` 能力，CPA-PC 应优先通过 usage plugin 订阅请求统计，而不是再通过外部 HTTP/RESP 队列消费自身数据。

## 运行拓扑

```text
cpa-pc.exe
  -> embedded CLIProxyAPI SDK
      -> /v1/*, /v1beta/*, /backend-api/codex/*
      -> /v0/management/*
      -> emits usage records

  -> CPA-PC usage service extension
      -> receives SDK usage plugin records
      -> writes SQLite
      -> exposes /v0/management/usage* and usage-service APIs

  -> local management panel
      -> serves /management.html
      -> built from CPA-PC-owned frontend source
```

第一阶段建议只监听一个本地端口，默认沿用 CPA 的 `8317`。管理页面入口为：

```text
http://127.0.0.1:8317/management.html
```

## 前端迁移策略

CPA-Manager 前端已经通过 Vite 构建成单文件 `dist/index.html`，并在 Usage Service 中嵌入为 `internal/httpapi/web/management.html`。CPA-PC 初始阶段直接迁移 CPA-Manager 前端源码到本仓库，之后作为 CPA-PC 自己的前端独立迭代。

迁移原则：

- 初始阶段以一次性代码导入的方式整目录拷贝 CPA-Manager 的前端源码作为起点，不保留 CPA-Manager 的提交历史。
- 拷贝后立即把代码所有权切换到 CPA-PC，后续不再依赖 `../CPA-Manager` 构建。
- 第一阶段尽量保持页面调用的 API shape 不变，降低后端适配风险。
- 与 CPA-PC 无关的 setup wizard、Docker 文案、远程 CPA 连接假设，可以在确认页面跑通后逐步裁剪。
- 页面构建产物仍输出单文件 `management.html`，外置到发布包中的 `static/management.html`。

为了贴近 CPA 的交付形态，发布包中应包含外置的 `static/management.html`，第一阶段不 embed 到 `.exe` 中。

## Usage Service 改造策略

CPA-Manager 的 Usage Service 代码目前在 `usage-service/internal/...` 下。由于 Go `internal` 包限制，CPA-PC 不能作为独立 module 直接 import 这些包。即使未来可以抽公共包，CPA-PC 也不把 CPA-Manager 作为长期依赖。

决策：CPA-PC 在本仓库内建立自己的 usage service Go module/package，初始实现可以参考或拷贝 CPA-Manager Usage Service 的 store、query、HTTP API 结构，但必须按 CPA-PC 的目标改造。

核心改造点：

- 数据来源从 CPA-Manager 的 HTTP/RESP usage queue collector 改为 CLIProxyAPI SDK usage plugin。
- CPA 连接配置从“外部 CPA 地址 + Management Key”改为“本进程内嵌 CPA SDK + 本地 management 配置”。
- SQLite schema 和查询结果优先保持与页面需求一致，而不是追求与 CPA-Manager 内部实现逐行一致。
- HTTP API 只对页面和本地使用场景负责，不保留 Docker/远程 CPA 等非目标场景的复杂度。
- 初始可以拷贝代码降低风险，但拷贝后应重命名、裁剪并纳入 CPA-PC 的测试体系。

建议包边界：

```text
cmd/cpa-pc/
internal/app/
internal/config/
internal/httpapi/
internal/store/
internal/usage/
web/
static/
```

## 配置与数据目录

发布包建议结构参考 CPA：

```text
cpa-pc_<version>_windows_amd64/
  cpa-pc.exe
  config.example.yaml
  static/
    management.html
  data/
  logs/
```

运行时默认读取程序同目录下的 `config.yaml`。没有 `config.yaml` 时，用户可以复制 `config.example.yaml` 后修改。

推荐默认配置：

```yaml
host: ""
port: 8317

remote-management:
  allow-remote: false
  secret-key: "123456"
  disable-control-panel: false

auth-dir: "~/.cli-proxy-api"
api-keys:
  - "your-api-key-1"

usage:
  enabled: true
  db-path: "./data/usage.sqlite"
  query-limit: 50000

logging-to-file: false
```

说明：除默认 Management Key 使用 CPA-PC 自有默认值 `123456` 外，CPA 原生配置默认值保持和 CLIProxyAPI 一致。
说明：`remote-management.disable-control-panel: false` 让 CPA SDK 继续托管 `/management.html`；CPA-PC 发布包提供外置 `static/management.html`，避免重复注册 Gin 路由。

## HTTP 接口边界

CPA-PC 需要保持页面兼容所需接口：

- `GET /management.html`：返回 CPA-Manager 页面。
- `GET /usage-service/info`：返回内嵌服务信息。
- `GET /usage-service/config`：返回当前 usage 配置或连接状态。
- `PUT /usage-service/config`：保存 usage 配置，第一阶段可只支持必要字段。
- `GET /status` 或 `GET /health`：返回 CPA-PC 服务状态。
- `GET /v0/management/usage*`：从 SQLite 返回请求统计。
- `GET/PUT /v0/management/model-prices*`：如页面需要模型价格，按 CPA-Manager 兼容接口实现或降级。

其他 CPA Management API 由内嵌 CLIProxyAPI SDK 继续提供。

## 第一阶段 MVP

- 建立 CPA-PC Go module 和 `cmd/cpa-pc` 入口。
- 读取 `config.yaml` 并启动内嵌 CLIProxyAPI SDK。
- 注册 usage plugin，把 SDK usage records 写入 SQLite。
- 迁移 CPA-Manager 前端源码到 CPA-PC，并构建出 `static/management.html`。
- 实现页面启动所需的最小 usage-service API。
- 提供 `config.example.yaml`。
- 提供 Windows amd64 构建产物结构：`.exe + static/management.html + config.example.yaml`。
- 第一阶段只支持 Windows amd64。

## 第一阶段成功标准

- 执行 `cpa-pc.exe -config config.yaml` 可以启动单一进程。
- 打开 `http://127.0.0.1:8317/management.html` 可以进入复用的 CPA-Manager 页面。
- 页面可以调用同源的 CPA Management API。
- 代理请求产生的 usage record 可以写入 `data/usage.sqlite`。
- 页面能展示基础请求统计。
- 前端源码和 usage service 后端代码已归入 CPA-PC 仓库，不依赖 `../CPA-Manager` 参与运行或发布。
- 发布包包含 `.exe`、`static/management.html`、`config.example.yaml`。
- `management.html` 以外置文件形式存在于 `static/` 目录，不要求 embed 到 `.exe`。

## 主要风险

- CPA-Manager 页面迁移后可能仍依赖完整 Usage Service API，第一阶段需要先识别页面启动和监控页真正依赖的接口。
- CPA SDK 的 management routes 和 CPA-PC 自定义 usage routes 可能存在路由顺序或路径覆盖问题。
- CPA-Manager Usage Service 现有代码位于 `internal`，不能直接 import；CPA-PC 初始拷贝/参考后需要承担后续独立维护成本。
- SQLite 写入模型要和页面查询模型保持兼容，否则页面可以加载但统计为空或字段不匹配。
- 单端口托管需要确认 CPA SDK 是否允许 CPA-PC 在默认路由之后覆盖 `/management.html` 和 usage 相关接口。

## 已确认决策

- CPA-Manager 前端源码采用一次性代码导入，不保留原提交历史。
- `management.html` 外置在发布包 `static/` 目录中，不 embed 到 `.exe`。
- 默认 Management Key 保留 CPA 原始逻辑；如果 CPA 原始逻辑没有默认值，则使用 `123456`。
- 第一阶段只考虑 Windows amd64。
- 第一阶段保留 CPA-Manager 原有页面流程，包括首次 setup wizard；CPA-PC 通过兼容接口提供本机默认值。

## 待确认问题

- 暂无。
