# CPA-PC v0 配置到 v8 配置迁移计划

状态：草案，待确认配置方案及上游前端功能采纳范围；本文不执行迁移或前端同步。

## 目标与现状

- 将 CLIProxyAPI 的旧版平铺 YAML 配置迁移到 SDK v8 的分组 YAML 格式，同时保留 CPA-PC 的目录、usage、管理鉴权和 Windows 单进程交付行为。
- 本次一并将管理前端与本地上游源项目的最新版本对齐；先审查差异，再按用户确认的范围升级 `web/`，保留 CPA-PC 扩展的监控功能。
- 当前 `internal/config.Load` 先读同一份 YAML 的旧字段，`internal/app.loadCPAConfig` 再调用 SDK `LoadConfig`；后者兼容旧格式，前者不认识 v8 的 `server.*`、`management.*`、`oauth.auth-dir` 等字段。
- 当前 `web/src/services/api/configFile.ts` 经 `/v0/management/config.yaml` 读写原始 YAML；`web/src/hooks/useVisualConfig.ts` 的可视化编辑器也只理解旧字段。
- SDK v8 读取旧格式不会主动迁移，但成功的 `/v8/management/config*` 写入会迁移文件，并把未知根字段注释掉。因此直接写入 v8 会使 `data-dir`、`logs-dir`、`static-dir`、`usage` 不再生效；随后 CPA-PC 重启还可能回退端口、认证目录和管理密钥。

依据：[上游 v8 管理 API 文档](https://github.com/router-for-me/CLIProxyAPI/blob/v8.0.13/docs/management-api-v8.md)、[SDK v8 配置迁移实现](https://github.com/router-for-me/CLIProxyAPI/blob/v8.0.13/internal/config/config_v8.go)。上游保留 `/v0/management` 路由，不要求同时升级所有管理 API。

## 实施前决策

1. **推荐：分离配置所有权。** `config.yaml` 仅存 SDK v8 配置；新增同目录 `cpa-pc.yaml` 存 `data-dir`、`logs-dir`、`static-dir`、`usage`。`-config` 仍指向 SDK 的 `config.yaml`；过渡期无旁路文件时从旧单文件读取 CPA-PC 字段。相对路径仍以 `config.yaml` 所在目录为基准。这样 SDK 的 v8 写入不会碰到 CPA-PC 字段，但需调整示例、打包和现有单文件配置规范。
2. 如果必须保持**单文件**，需先确定 SDK 可维护的扩展字段保留机制（优先上游支持）；不能仅依赖前端隐藏 v8 按钮，因为直接调用 v8 API 仍可触发字段注释。该路线在机制和回归测试明确前不落地。
3. 确认是否允许迁移时自动创建 `cpa-pc.yaml`：建议先提供显式迁移/预览步骤，不在普通启动时改写配置；先备份、校验，再切换。

上述决定涉及 `docs/specs/2026-05-22-merge-cpa-config-example.md` 中已批准的单文件约定；实施前由用户确认并同步更新相关规范及 `AGENTS.md`，不能直接删掉原示例的 CPA-PC 字段。

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

## 分阶段任务与验收

### 阶段 1：建立可复现基线

1. 在临时目录准备旧单文件配置及 v8 配置样例，覆盖非默认端口/密钥、关闭 usage、自定义数据库路径、显式 `false`/`0`、认证目录和注释；绝不对用户的 `config.yaml` 或 `config.example.yaml` 运行会哈希密钥的启动检查。
2. 补充针对当前问题的回归测试：证明 SDK 的 v8 写入会注释专用字段、当前 CPA-PC 读取 v8 字段会回退默认值；记录修改前后行为。
3. 验收：明确所有 SDK 字段映射和前端旧路径的使用点，明确单文件/旁路文件方案已获确认。

### 阶段 2：先支持读取，不自动改写

1. 调整 `internal/config` 与 `internal/app` 的配置所有权：SDK 字段从 v8/旧格式的有效值取得，CPA-PC 字段按确认的方案读取；不要让 `loadCPAConfig` 用 CPA-PC 默认值覆盖 SDK 已解析的 v8 值。
2. 让 SDK 管理路由与 `internal/httpapi` 扩展路由共享同一有效管理密钥/远程限制。定义配置在线修改后的生效策略：要么安全刷新两边，要么明确要求重启且在生效前避免两边鉴权不一致。
3. 在旧单文件尚未迁移时，拦截会使它变为 v8 布局的写入（包括 `/v8/management/config*` 和提交 v8 YAML 的 `/v0/management/config.yaml`）；检测到已迁移但缺少 CPA-PC 专用配置时明确报错，不以默认目录/密钥继续运行。新安装应随 v8 示例提供完整的配套专用配置。
4. 验收：旧单文件可启动；配套专用配置的 v8 示例能正确得到端口、认证目录、密钥、日志和 usage 路径；普通启动不主动创建或迁移配置（SDK 现有的明文管理密钥哈希写回行为除外）。

### 阶段 3：提供显式迁移与回滚

1. 迁移入口先读取原文件、解析双方配置、生成预览（字段映射、冲突、未知字段、目标路径），未确认时不写文件。若采用推荐方案，先将 CPA-PC 专用字段提取到同目录旁路文件，再利用 SDK 的 v8 配置接口在隔离副本上生成/校验 v8 SDK 文件；不依赖上游 `internal/config` 私有包，也不对生产文件直接执行 v8 写入。
2. 保存时备份原 `config.yaml` 和已有旁路文件，保护文件权限；验证生成文件的 SDK 可读性、CPA-PC 可读性、路径和管理密钥后再切换。失败时保留原文件供恢复，不允许半迁移状态启动并悄悄套用默认值。
3. 提供回滚说明：停止进程、恢复两个备份文件、重启；SQLite 数据文件及认证目录不移动、不转换。重复执行迁移应无额外改写，若已有目标文件且内容冲突，应提示而非覆盖。
4. 验收：从真实旧格式样例迁移后重启，监听端口、管理鉴权、认证文件、usage 数据库路径/内容、静态页面和日志行为与迁移前一致；备份可恢复旧配置。

### 阶段 4：同步上游前端并确认采纳范围

1. 先检查 `D:\develop\source-workspace\Cli-Proxy-API-Management-Center` 的本地改动，再同步到可用的最新上游版本，记录上游版本/提交及同步前的基线；不覆盖本地未提交工作。
2. 按功能对比该源项目和 `./web`：页面与路由、配置编辑、OAuth/认证、API 客户端、监控模块、样式与依赖。列出上游新增/改变、CPA-PC 自有扩展、冲突点及预计适配成本；重点检查上游是否已改用 v8 管理接口。
3. 将差异清单交用户逐项确认“采纳/跳过/按 CPA-PC 方式适配”；在确认前不向 `./web` 合并上游变更。验收：采纳范围与跳过原因有记录，监控模块及 CPA-PC 专用接口的保留范围明确。

### 阶段 5：合并确认的前端变更并适配配置编辑器

1. 仅合并用户确认的上游页面和功能，保留 CPA-PC 的请求监控、监控图表、usage/API-key 别名/模型价格等扩展；不整目录覆盖 `./web`。保持 hash 路由、单文件静态资源构建和已有的 Gemini API 路径兼容。
2. 根据新前端的实际 API 使用情况，对接 SDK v8 与 CPA-PC 扩展路由；对已有 `/v0/management` 调用维持兼容。配置编辑器必须识别 v8 布局：在可视化表单完整支持 v8 字段路径及旧格式之前，对 v8 文件禁用可视化写入，允许明确标注的原始 YAML 编辑；不能向 v8 文件重新写入旧根字段。
3. 校验原始 YAML 的 `/v0/management/config.yaml` 写入 v8 文档及直接 `/v8/management/config*` 写入均不会影响 CPA-PC 专用配置；回归登录、配置读取和保存、OAuth、监控页及选择采纳的上游功能。

### 阶段 6：示例、文档与交付

经确认后更新 `config.example.yaml`、新的 CPA-PC 专用配置示例、README、`AGENTS.md` 和 Windows 打包输入；区分“旧单文件可继续使用”与“新配置使用 v8 + 旁路文件”的说明。现有安装不强制迁移。重新生成 `static/management.html`，将确认的前端版本和采纳清单随交付记录。

## 最终验证与完成标准

- 自动化：`go test ./internal/config ./internal/app ./internal/httpapi`、`go test ./...`、`go build ./cmd/cpa-pc`；前端运行 `npm --prefix web test`、`npm --prefix web run lint`、`npm --prefix web run type-check`、`npm --prefix web run build`。
- 在隔离目录执行端到端测试：旧文件启动、预览、迁移、重启、v0 配置写入、v8 配置写入、再次重启、回滚；管理扩展路由与 SDK 路由对同一密钥给出一致结果。检查明文和哈希密钥均未泄露到日志或预览输出。
- 若修改交付布局，运行 `npm run package:windows -- --version dev`，核对新旧配置示例与静态资源进入 Windows 包。
- 成功条件：v8 配置可用、CPA-PC 专用设置不会被 SDK v8 注释或丢失、旧配置无需迁移仍可运行、迁移可预览且可回滚；前端已按用户确认的清单同步上游功能，监控扩展仍可用，`web/` 不会把 v8 文件写回旧字段。
