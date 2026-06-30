# use-agent-skills

`use-agent-skills` 是从 `../agent-skills` 整合而来的单一 OpenCode 技能源码。它把原仓库中的多个工程流程技能收敛到一个技能入口中，由用户主动触发后，再在技能内部选择合适的模块或命令模板。

## 来源

源仓库目录: `../agent-skills`

本次迁移范围:

- `skills/`
- `.claude/commands/`
- `agents/`
- `references/`

未迁移内容:

- `hooks/`
- `.gemini/commands/`
- `.claude-plugin/`
- 其他运行时、IDE、依赖和 Git 元数据目录

## 使用方式

该技能只应由用户主动触发。普通编码、测试、评审、发布请求不应自动加载该技能。

可触发示例:

- “使用 use-agent-skills 帮我做这个功能”
- “用 agent skills 的流程先拆一下任务”
- “进入内部 `/spec` 命令”
- “用内部 review 命令检查这次改动”

技能激活后，`SKILL.md` 会作为总入口，根据用户指定的内部命令或任务内容选择对应模块。

进入流程时需要向用户展示:

```text
Entered.
```

任务完成或用户要求退出流程时，需要向用户展示:

```text
Exited.
```

退出后，内部模块名和内部命令不再作为有效指令响应，除非用户重新主动进入 `use-agent-skills` 或 `agent-skills` 流程。

## 目录结构

```text
use-agent-skills/
├── SKILL.md
├── README.md
├── commands/
│   ├── build.md
│   ├── code-simplify.md
│   ├── plan.md
│   ├── review.md
│   ├── ship.md
│   ├── spec.md
│   └── test.md
├── modules/
│   └── <module-name>/index.md
├── agents/
│   ├── code-reviewer.md
│   ├── security-auditor.md
│   └── test-engineer.md
├── references/
│   ├── accessibility-checklist.md
│   ├── orchestration-patterns.md
│   ├── performance-checklist.md
│   ├── security-checklist.md
│   └── testing-patterns.md
└── evals/
    └── evals.json
```

注意: `<project_root>/docs/` 是 agent 执行任务时所在项目的运行时产物目录，不是本技能目录的一部分。

## 内部命令

这些命令是技能内部命令模板，不是 OpenCode slash command。

| 命令 | 文件 | 主要模块 |
| --- | --- | --- |
| `/spec` | `commands/spec.md` | `spec-driven-development` |
| `/plan` | `commands/plan.md` | `planning-and-task-breakdown` |
| `/build` | `commands/build.md` | `incremental-implementation`, `test-driven-development` |
| `/test` | `commands/test.md` | `test-driven-development` |
| `/review` | `commands/review.md` | `code-review-and-quality` |
| `/code-simplify` | `commands/code-simplify.md` | `code-simplification` |
| `/ship` | `commands/ship.md` | `shipping-and-launch` |

## 内部模块

原 `skills/` 下除 `using-agent-skills` 外的技能都迁移到 `modules/`，入口文件从 `SKILL.md` 改名为 `index.md`。

`using-agent-skills` 未作为独立模块迁移，它的路由和通用行为融入了本技能的 `SKILL.md`。

模块列表:

- `api-and-interface-design`
- `browser-testing-with-devtools`
- `ci-cd-and-automation`
- `code-review-and-quality`
- `code-simplification`
- `context-engineering`
- `debugging-and-error-recovery`
- `deprecation-and-migration`
- `documentation-and-adrs`
- `doubt-driven-development`
- `frontend-ui-engineering`
- `git-workflow-and-versioning`
- `idea-refine`
- `incremental-implementation`
- `interview-me`
- `performance-optimization`
- `planning-and-task-breakdown`
- `security-and-hardening`
- `shipping-and-launch`
- `source-driven-development`
- `spec-driven-development`
- `test-driven-development`

## Agent Prompt 模板

`agents/` 目录中的文件仅作为技能内部派发子 agent 时使用的 prompt 模板，不是用户可选择的角色依赖，也不需要预先注册为 OpenCode agent。

- `agents/code-reviewer.md`: 代码质量评审模板。
- `agents/security-auditor.md`: 安全审计模板。
- `agents/test-engineer.md`: 测试策略和覆盖分析模板。

## 参考清单

`references/` 目录从源仓库迁移而来，供内部模块按需读取。

- `references/accessibility-checklist.md`: 无障碍检查清单。
- `references/orchestration-patterns.md`: 子 agent 编排模式。
- `references/performance-checklist.md`: 性能检查清单。
- `references/security-checklist.md`: 安全检查清单。
- `references/testing-patterns.md`: 测试模式和反模式。

## 运行时文档产出规则

原技能或命令中会产出的项目文档统一写入当前项目根目录下的 `<project_root>/docs/`。

示例:

- 规格文档: `<project_root>/docs/SPEC.md`
- 计划文档: `<project_root>/docs/plan.md`
- 任务列表: `<project_root>/docs/todo.md`
- ADR: `<project_root>/docs/decisions/`
- Review 报告: `<project_root>/docs/reviews/`
- Launch 记录: `<project_root>/docs/launch/`

如果其他模块或命令依赖这些产物，也应从 `<project_root>/docs/` 下查找。

## 与源码的改动点

为方便后续升级，本迁移尽量保持源内容一致。必要改动如下:

- `skills/*/SKILL.md` 迁移为 `modules/*/index.md`。
- `skills/using-agent-skills/SKILL.md` 融入 `SKILL.md`，不再作为独立模块。
- `.claude/commands/*.md` 迁移为 `commands/*.md`，作为内部命令模板。
- `agents/*.md` 迁移到 `agents/*.md`，仅作为子 agent prompt 模板。
- 源仓库 `references` 目录下的 Markdown 文件迁移到目标 `references` 目录，供模块按需引用。
- 移除了 Claude/Gemini/plugin 安装和运行时专属表述。
- 将 `agent-skills:<skill-name>` 调用改为读取 `modules/<module-name>/index.md`。
- 将原本写入项目根目录或 `tasks/` 的产出文档改为写入 `<project_root>/docs/`。
- 将源仓库中的角色相关描述改为 child-agent prompt template 语义。

## 后续升级方式

从 `../agent-skills` 升级时，建议按以下顺序处理:

1. 对比源仓库的 `skills/`、`.claude/commands/`、`agents/`、`references/` 变化。
2. 将 `skills/*/SKILL.md` 的变更同步到 `modules/*/index.md`。
3. 将 `.claude/commands/*.md` 的变更同步到 `commands/*.md`。
4. 将 `agents/*.md` 的变更同步到 `agents/*.md`。
5. 将源仓库 `references` 目录下 Markdown 文件的变更同步到目标 `references` 目录。
6. 重新应用“与源码的改动点”中的本地规则。
7. 检查是否重新引入了 `.gemini`、`.claude` 或角色依赖相关表述。
8. 运行 eval，确认内部子模块自动路由仍然合理且稳定。

## Eval 说明

本技能不做外层触发描述 eval。外层触发规则是用户主动触发。

Eval 只评估技能激活后，`SKILL.md` 自动选择内部子模块是否合理且稳定，不评估各模块完整执行效果。
