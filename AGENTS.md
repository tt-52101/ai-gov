# AGENTS.md — AI Agent 行为铁律

> 本文档是 AI 编码 Agent 在本项目中必须遵守的**不可配置宪法级约束**。违反任一条铁律即为严重事故。

---

## 第 1 章：Git 与仓库操作铁律

### 1.1 远程仓库归属校验（最高优先级）

**铁律：** 任何 `git push`、`git remote add`、仓库创建/删除操作之前，**必须先校验远程 URL 的仓库所有者是否与用户指定的完全一致**。

```
❌ 禁止：git remote add origin <url> 之后直接 git push
✅ 必须：git remote -v 输出逐字比对用户指定的 owner/repo，不一致则阻断并报告
```

**检查清单（push 前必须逐项通过）：**

| # | 检查项 | 验证命令 |
|---|--------|---------|
| 1 | remote URL 中的 `owner` 与用户指定一致 | `git remote get-url origin` |
| 2 | remote URL 中的 `repo` 与用户指定一致 | 同上 |
| 3 | 凭证用户名（`user:token@` 中的 `user`）与 owner 匹配 | 同上 |
| 4 | 远端仓库确实存在且当前 Token 有写权限 | `curl -s -o /dev/null -w "%{http_code}" https://api.github.com/repos/<owner>/<repo>` |

**发现凭证不匹配时：**
1. 立即停止所有远程操作
2. 明确报告：「当前 Git 全局凭证属于 `X`，但目标仓库 `Y/Z` 不匹配。请提供正确的 Token 或清除 `~/.git-credentials`」
3. **不得自行使用全局凭证创建/删除任何仓库**

### 1.2 禁止使用全局凭证操作他人仓库

**铁律：** 全局 git credential helper 注入的凭证可能属于其他账号。在未确认凭证归属前，**严禁**使用该凭证执行任何 GitHub API 写操作（创建仓库、删除仓库、修改设置）。

```
❌ 禁止：curl -H "Authorization: token <从remote URL中提取的token>" POST /user/repos
✅ 必须：先打印凭证用户名，向用户确认后再操作
```

### 1.3 GitHub API 操作审批

以下操作**必须经用户明确确认后才能执行**：

| 操作 | API | 风险 |
|------|-----|------|
| 创建仓库 | `POST /user/repos` 或 `POST /orgs/{org}/repos` | 可能创建在错误账号下 |
| **删除仓库** | `DELETE /repos/{owner}/{repo}` | **不可逆，代码永久丢失** |
| 修改仓库设置 | `PATCH /repos/{owner}/{repo}` | 可能暴露私仓 |
| 删除 Tag/Release | `DELETE /repos/{owner}/{repo}/git/refs/tags/{tag}` | 基线丢失 |

### 1.4 删除仓库前必须备份

**铁律：** 删除任何远程仓库之前，必须先确认本地有完整备份：
```bash
git log --oneline -5    # 确认提交历史完整
git tag -l              # 确认标签完整
git branch -a           # 确认所有分支已同步
```

---

## 第 2 章：Token 与凭证安全

### 2.1 Token 永远不可落盘为明文

- `AGENTS.md`、`.trae/`、`docs/` 中**禁止**出现任何 GitHub Token
- Token 仅在 `git remote` URL 中或临时环境变量中存在
- 会话结束时 Token 自动随环境销毁

### 2.2 Token 权限最小化

提供给 Agent 使用的 Token 建议仅包含以下 scopes：
- `repo`（读写仓库代码）
- **不要** `delete_repo`（防止误删仓库）
- **不要** `admin:org`（防止组织级误操作）

---

## 第 3 章：版本管理铁律

### 3.1 基线版本不可变

- `vX.Y.Z-baseline` 标签一旦推送，**不得删除或移动**
- 修复只能通过新增提交 + 新标签完成

### 3.2 提交信息规范

```
feat: 功能描述
fix: 修复描述
docs: 文档变更
chore: 工程配置变更
```

---

## 第 4 章：会话行为铁律

### 4.1 远程操作前必须自查

Agent 在执行任何推送/创建/删除远程资源的操作之前，必须自问：

1. **归属**：目标 owner/repo 与用户指定是否完全一致？
2. **凭证**：当前使用的 Token 属于哪个账号？是否有权限？
3. **后果**：此操作如果执行在错误仓库上，会造成什么损害？
4. **可逆**：此操作是否可逆？如果不可逆，是否已备份？

### 4.2 不确定时，停止并确认

任何导致 Agent 产生"可能有问题"直觉的操作，**必须停下来向用户确认**。不允许"先做了再说"。

---

## 第 5 章：本次事故复盘（2026-07-31）

### 事故描述

用户指定目标仓库 `tt-52101/ai-gov`，但 Agent 使用了全局 git credential 中注入的 `aethir-paas` Token 创建并推送到了错误仓库 `aethir-paas/ai-gov`。随后又使用同一 Token 删除了该错误仓库。

### 违规项

| # | 违规铁律 | 说明 |
|---|---------|------|
| 1 | 远程仓库归属校验 | `git remote -v` 显示 `aethir-paas` 时未停止确认 |
| 2 | 禁止用全局凭证操作他人仓库 | 直接使用 `aethir-paas` Token 调 GitHub API 创建/删除仓库 |
| 3 | 删除仓库前必须备份 | 删除前未做任何备份确认 |
| 4 | 不确定时停止确认 | 发现凭证不匹配后继续操作而非暂停 |

### 教训

**Agent 永远不得假定全局 git credential 属于用户指定的目标账号。** 凭证归属校验是推送前的第一道门槛，跳过去就是事故。

---

## 附录：本项目仓库信息

| 项 | 值 |
|----|-----|
| 仓库地址 | `https://github.com/tt-52101/ai-gov` |
| 默认分支 | `main` |
| 当前基线标签 | `v3.2.0-baseline` |
| 项目代号 | `ai-gov` |
| 本地路径 | `D:\ai-work\grok\ai-gov` |
