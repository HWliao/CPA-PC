# CPA-PC v0 配置到 v8 配置迁移计划

状态：已实施（2026-10-04）；管理 API 已迁移到 v8，配置采用双文件，上游新版页面已同步并保留 CPA-PC 扩展。

## 目标与现状

- 将 CLIProxyAPI 的旧版平铺 YAML 配置迁移到 SDK v8 的分组 YAML 格式，同时保留 CPA-PC 的目录、usage、管理鉴权和 Windows 单进程交付行为。
- 将前端所有管理 API 调用切到 SDK v8；CPA-PC 自有管理接口也发布在 `/v8/management` 下，不再由新前端调用 `/v0/management`。`/cpa-pc/info`、`/usage-service/*`、`/setup`、`/status` 是产品自身的非 v0 管理接口，单独盘点契约，不因统一前缀而误改。
- 本次一并将管理前端与本地上游源项目的最新版本对齐；先审查差异，再按用户确认的范围升级 `web/`，保留 CPA-PC 扩展的监控功能。
- 迁移前 `internal/config.Load` 先读同一份 YAML 的旧字段，`internal/app.loadCPAConfig` 再调用 SDK `LoadConfig`；后者兼容旧格式，前者不认识 v8 的 `server.*`、`management.*`、`oauth.auth-dir` 等字段。
- 迁移前 `web/src/services/api/configFile.ts` 经 `/v0/management/config.yaml` 读写原始 YAML；旧可视化编辑器也只理解旧字段。上游 `v1.25.3` 页面与 API 客户端已采用 v8 路径。
- SDK v8 读取旧格式不会主动迁移，但成功的 `/v8/management/config*` 写入会迁移文件，并把未知根字段注释掉。因此直接写入 v8 会使 `data-dir`、`logs-dir`、`static-dir`、`usage` 不再生效；随后 CPA-PC 重启还可能回退端口、认证目录和管理密钥。

依据：[上游 v8 管理 API 文档](https://github.com/router-for-me/CLIProxyAPI/blob/v8.0.13/docs/management-api-v8.md)、[SDK v8 配置迁移实现](https://github.com/router-for-me/CLIProxyAPI/blob/v8.0.13/internal/config/config_v8.go)。上游保留 `/v0/management` 路由，不要求同时升级所有管理 API。

上游前端本地源项目已同步到 `v1.25.3`（`ee79a79`）；它默认仅调用 `/v8/management`。本计划的 v8 路由变更是用户的新决定，覆盖早期图表规格中“保持 `/v0/management/usage/charts` 路径”的范围限制；旧规格作为历史记录保留。

## 实施前决策

1. **已确认：分离配置所有权。** `config.yaml` 仅存 SDK v8 配置；新增同目录 `cpa-pc.yaml` 存 `data-dir`、`logs-dir`、`static-dir`、`usage`。`-config` 仍指向 SDK 的 `config.yaml`；过渡期无旁路文件时从旧单文件读取 CPA-PC 字段。相对路径仍以 `config.yaml` 所在目录为基准。这样 SDK 的 v8 写入不会碰到 CPA-PC 字段，但需调整示例、打包和现有单文件配置规范。
2. 实施显式迁移/预览步骤；普通启动不自动创建 `cpa-pc.yaml` 或迁移 `config.yaml`，迁移前先备份、校验，再切换。

双文件决定已由用户确认，覆盖 `docs/specs/2026-05-22-merge-cpa-config-example.md` 中已批准的单文件约定；实施时同步更新相关规范及 `AGENTS.md`，不能直接删掉原示例的 CPA-PC 字段。

## 字段归属与兼容规则

| 旧字段 | v8 SDK 字段 / 归属 | 验证重点 |
| --- | --- | --- |
| `host`、`port`、`tls` | `server.*` | 实际监听地址、TLS；不可被 CPA-PC 默认值覆盖 |
| `remote-management.*` | `management.*` | SDK 与 CPA-PC 扩展路由使用同一密钥及远程访问策略；保留默认示例密钥 `123456` |
| `auth-dir` | `oauth.auth-dir` | 相对/`~` 路径和现有认证文件 |
| `api-keys`（客户端密钥列表） | `access.api-keys` | 不与 v8 根部的上游供应商 `api-keys` 映射混淆 |
| `logging-to-file` | `observability.logs.logging-to-file` | CPA-PC 与 SDK 日志开关一致 |
| `request-retry` 等重试字段 | `routing.retry.*` | 在 v8 保存后保持原值，特别是显式 `0` |
| `data-dir`、`logs-dir`、`static-dir`、`usage.*` | CPA-PC 专用，建议迁入 `cpa-pc.yaml` | 路径、SQLite 文件、启用状态及查询上限不变 |

SDK 其他旧字段以 SDK v8 配置映射为准，不在 CPA-PC 另造一套全量转换器。旧格式读取与 v8 字段同时出现时采用 SDK 的“v8 优先”规则；CPA-PC 读取所需字段必须用相同优先级，保留显式 `false`、`0` 与缺省值的区别。若新旧值冲突，应在迁移预览中报告，不能静默使用默认值。管理密钥可能已由 SDK 写成 bcrypt 哈希；迁移不得重置或明文输出密钥。

## 管理 API v8 切换范围

- SDK 原生管理接口按上游 v8 路由和请求/响应格式接入，不能只替换 `MANAGEMENT_API_PREFIX`：例如旧 `/config.yaml`、OAuth、认证文件、供应商设置在 v8 中有不同的路径或载荷。
- CPA-PC 自有接口统一放在 `/v8/management/cpa-pc/` 命名空间，避免与 SDK 现有的 `/v8/management/observability/usage/api-keys`、`/queue` 等路由冲突。初定映射如下，落地前对照 SDK 路由表做逐项冲突检查：

| 现有 CPA-PC 管理路由 | 新路由（沿用原方法和响应） |
| --- | --- |
| `/v0/management/usage`、`/usage/charts`、`/usage/export`、`/usage/import` | `/v8/management/cpa-pc/usage`、`/usage/charts`、`/usage/export`、`/usage/import` |
| `/v0/management/model-prices`、`/model-prices/sync` | `/v8/management/cpa-pc/model-prices`、`/model-prices/sync` |
| `/v0/management/api-key-aliases`、`/api-key-aliases/:apiKeyHash` | `/v8/management/cpa-pc/api-key-aliases`、`/api-key-aliases/:apiKeyHash` |

- v8 扩展路由必须沿用管理密钥与远程访问约束；`internal/app/buildinfo_headers.go` 的版本头也应覆盖 v8 管理路由。迁移期可保留旧 CPA-PC 路由作为外部客户端兼容别名，但新前端和新测试以 v8 为准；SDK 自身的 v0 路由由 SDK 管理，不在 CPA-PC 重注册。
- 建立前端管理请求清单：SDK 原生域（配置、凭证、OAuth、插件、日志、供应商）与 CPA-PC 域（usage、价格、别名）分别验证请求方法、载荷、鉴权、返回值。`usage-service` 等产品 API 不应因全局前缀改为 v8 而失联。

## 分阶段任务与验收

实施顺序调整：用户要求先同步上游新版页面，再决定并实施配置迁移。因此实际先完成下方阶段 5、6 的前端同步与适配，再落地阶段 2～4 的双文件迁移；阶段编号保留为任务依赖分组，不代表执行时间顺序。全部上游页面及 v8 管理 API 已获用户确认。

### 阶段 1：建立可复现基线

1. 在临时目录准备旧单文件配置及 v8 配置样例，覆盖非默认端口/密钥、关闭 usage、自定义数据库路径、显式 `false`/`0`、认证目录和注释；绝不对用户的 `config.yaml` 或 `config.example.yaml` 运行会哈希密钥的启动检查。
2. 补充针对当前问题的回归测试：证明 SDK 的 v8 写入会注释专用字段、当前 CPA-PC 读取 v8 字段会回退默认值；记录修改前后行为。
3. 验收：明确所有 SDK 字段映射和前端旧路径的使用点，列出 SDK 与 CPA-PC 的 v8 路由合同及冲突检查结果；单文件/旁路文件方案已获确认。

### 阶段 2：先支持读取，不自动改写

1. 调整 `internal/config` 与 `internal/app` 的配置所有权：SDK 字段从 v8/旧格式的有效值取得，CPA-PC 字段按确认的方案读取；不要让 `loadCPAConfig` 用 CPA-PC 默认值覆盖 SDK 已解析的 v8 值。
2. 让 SDK 管理路由与 `internal/httpapi` 扩展路由共享同一有效管理密钥/远程限制。定义配置在线修改后的生效策略：要么安全刷新两边，要么明确要求重启且在生效前避免两边鉴权不一致。
3. 在旧单文件尚未迁移时，拦截会使它变为 v8 布局的写入（包括 `/v8/management/config*` 和提交 v8 YAML 的 `/v0/management/config.yaml`）；检测到已迁移但缺少 CPA-PC 专用配置时明确报错，不以默认目录/密钥继续运行。新安装应随 v8 示例提供完整的配套专用配置。
4. 验收：旧单文件可启动；配套专用配置的 v8 示例能正确得到端口、认证目录、密钥、日志和 usage 路径；普通启动不主动创建或迁移配置（SDK 现有的明文管理密钥哈希写回行为除外）。

### 阶段 3：迁移 CPA-PC 管理路由

1. 在 `internal/httpapi` 增加上述 v8 路由，不与 SDK 的同方法/路径重复注册；保留业务响应结构和鉴权校验。根据实际 SDK 路由表检查 Gin 注册顺序，并补足 `X-CPA-VERSION` 等 v8 响应头。
2. 测试无密钥、错密钥、正确密钥、远程访问与 usage 禁用/空库时的新旧路由；确保兼容别名不会产生写入语义分叉。验收：所有 CPA-PC 管理能力在 v8 下可用，无路由冲突。

### 阶段 4：提供显式迁移与回滚

1. 迁移入口先读取原文件、解析双方配置、生成无密钥内容的预览（目标路径、未映射字段、SDK 设置/作用域变化），未确认时不写文件。通过 SDK 公开的 v8 配置处理入口生成候选文件，在写入前验证；不依赖上游 `internal/config` 私有包。
2. 显式执行迁移时拒绝覆盖已有旁路文件/备份，先备份原 `config.yaml`，再创建旁路文件，最后写入已验证的 SDK v8 配置。若写入或验证失败则从备份恢复，不允许半迁移状态启动并悄悄套用默认值。
3. 提供回滚说明：停止进程、恢复两个备份文件、重启；SQLite 数据文件及认证目录不移动、不转换。重复执行迁移应无额外改写，若已有目标文件且内容冲突，应提示而非覆盖。
4. 验收：从真实旧格式样例迁移后重启，监听端口、管理鉴权、认证文件、usage 数据库路径/内容、静态页面和日志行为与迁移前一致；备份可恢复旧配置。

### 阶段 5：同步上游前端并确认采纳范围

1. 先检查 `D:\develop\source-workspace\Cli-Proxy-API-Management-Center` 的本地改动，再同步到可用的最新上游版本，记录上游版本/提交及同步前的基线；不覆盖本地未提交工作。
2. 按功能对比该源项目和 `./web`：页面与路由、配置编辑、OAuth/认证、API 客户端、监控模块、样式与依赖。列出上游新增/改变、CPA-PC 自有扩展、冲突点及预计适配成本；上游 v8 API 适配是本次必做项。
3. 已向用户列出差异并确认采纳**全部上游页面**；合并时保留监控模块及 CPA-PC 专用接口。

### 阶段 6：合并确认的前端变更并适配配置编辑器

1. 合并 v8 管理 API 客户端与用户确认的上游页面和功能，保留 CPA-PC 的请求监控、监控图表、usage/API-key 别名/模型价格等扩展；不整目录覆盖 `./web`。保持 hash 路由、单文件静态资源构建和已有的 Gemini API 路径兼容。
2. 将 SDK 原生管理调用逐项改为 v8 路由/载荷，CPA-PC 监控调用指向 `/v8/management/cpa-pc/*`，不要对产品自身的 `/usage-service/*` 请求套用 SDK 管理前缀。配置编辑器必须识别 v8 布局：在可视化表单完整支持 v8 字段路径及旧格式之前，对 v8 文件禁用可视化写入，允许明确标注的原始 YAML 编辑；不能向 v8 文件重新写入旧根字段。
3. 校验原始 YAML 的 `/v8/management/config.yaml` 保存、SDK 配置分段写入、CPA-PC usage/价格/别名调用均不会影响专用配置；回归登录、配置读取和保存、OAuth、监控页及选择采纳的上游功能。新前端不应再发往 `/v0/management`。

### 阶段 7：示例、文档与交付

经确认后更新 `config.example.yaml`、新的 CPA-PC 专用配置示例、README、`AGENTS.md` 和 Windows 打包输入；区分“旧单文件可继续使用”与“新配置使用 v8 + 旁路文件”的说明。现有安装不强制迁移。重新生成 `static/management.html`，将确认的前端版本和采纳清单随交付记录。

## 最终验证与完成标准

- 自动化：`go test ./internal/config ./internal/app ./internal/httpapi`、`go test ./...`、`go build ./cmd/cpa-pc`；前端运行 `npm --prefix web test`、`npm --prefix web run lint`、`npm --prefix web run type-check`、`npm --prefix web run build`。
- 在隔离目录执行端到端测试：旧文件启动、预览、迁移、重启、v0 配置写入、v8 配置写入、再次重启、回滚；管理扩展路由与 SDK 路由对同一密钥给出一致结果。检查明文和哈希密钥均未泄露到日志或预览输出。
- 按前端管理请求清单检查网络请求：SDK 原生和 CPA-PC 自有管理请求均走 `/v8/management`，没有遗留的 `/v0/management`；新管理路由的错误码和响应头在本地与远程访问场景下通过验证。
- 若修改交付布局，运行 `npm run package:windows -- --version dev`，核对新旧配置示例与静态资源进入 Windows 包。
- 成功条件：v8 配置可用、CPA-PC 专用设置不会被 SDK v8 注释或丢失、旧配置无需迁移仍可运行、迁移可预览且可回滚；全部管理调用使用 v8，前端按用户确认的非 API 功能清单同步上游，监控扩展仍可用，`web/` 不会把 v8 文件写回旧字段。
