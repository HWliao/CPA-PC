---
name: imile-git-comment
description: "Use this skill only when the user's goal is to produce or judge the text of a Git commit message/comment itself. Trigger for requests to generate one commit message from a branch name, requirement ID, or change summary; to review whether an existing commit message follows the iMile commit format; to rewrite, normalize, improve, or explain a proposed commit message; or to choose the correct commit action such as feat, fix, chore, or docs for a specific commit title. Do not use this skill for nearby but different writing tasks: PR titles or descriptions, reviewer-facing wording, branch names, release notes, changelog entries, sprint summaries, commitlint/config debugging, Git command explanations, or actually running git commands. If the user says they do not want a commit message, or the requested output is for a pull request/release/changelog rather than a commit, stay out of the way."
---

# Git Commit Comment

用于生成、评审、改写 git commit comment。
只负责提交文本，不默认执行 git 命令。

## When to Use

- 用户要写 commit message 或 commit comment。
- 用户要 review 某条提交信息是否合规。
- 用户要改写、规范化现有提交信息。
- 用户给了分支名或需求 ID，希望直接产出提交信息。
- 任何需要 git commit comment 的场景，都优先使用本 skill。

## Rules

- 默认格式：`[需求ID] action: 修改内容`
- 固定使用 `action: 修改内容`，不要写 `feat(scope): ...`、`fix(scope): ...` 这类 scope 格式
- 如果没有稳定的 `需求ID`，直接输出：`action: 修改内容`
- `需求ID` 优先从分支名中提取 `日期_需求标识` 后的部分
- 如果用户直接提供了明确的 `需求ID` 且没有可用分支名，使用用户提供的 `需求ID`
- 示例：`dev_20260410_ops-rd_1022` -> `ops-rd_1022`
- 如果分支名无法稳定提取需求 ID，不要臆造，不要保留空的 `[]`
- `action` 只使用：`feat`、`fix`、`chore`、`docs`
- `修改内容` 只写本次最核心的改动目标，保持简短、具体
- 不要写“优化代码”“调整逻辑”这类空泛描述
- 如果一次改动包含多个主题，优先围绕最核心的交付目的写标题
- 如果只是顺手补了注释或文档，不要因此覆盖主要改动的 `action`

## Action Guide

- `feat`：新增能力、扩展功能、新流程
- `fix`：修复缺陷、错误行为、异常场景
- `chore`：工程性调整、配置整理、构建维护
- `docs`：文档、注释、说明文本更新

## Output

- 生成时，直接给 1 条推荐提交信息
- 评审时，先判断是否合规，再指出问题，最后给出修正版
- 用户要求解释时，再补充 1-2 句理由

## Conflict

- 规范冲突优先级：用户临时决策 > `AGENTS.md` > `imile-git-comment`
- 有冲突时必须显式说明：`<规范描述> 存在冲突! 将应用<规范来源>`
