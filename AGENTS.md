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

## 第 6 章：代码质量铁律

### 6.1 函数级注释（强制执行）

**铁律：** 每个导出函数/方法必须有标准 Go Doc 注释，每个关键内部函数必须有行内注释说明意图。

```go
// ✅ 正确：完整 Doc 注释
// Allocate transfers funds from source account to destination account.
// Both accounts must belong to parties connected by an allowed fund edge
// (parent downward, sponsors direction, or explicit whitelist).
// The operation is atomic: both ledger entries are written in a single transaction.
// Idempotency is guaranteed via Idempotency-Key: duplicate keys return the
// original result without double-charging.
func (s *Service) Allocate(ctx context.Context, req AllocateRequest) (*AllocateResult, error) {
```

```go
// ❌ 禁止：无注释、无意义注释、"TODO"占位
func (s *Service) Allocate(ctx context.Context, req AllocateRequest) (*AllocateResult, error) {
// TODO: implement later
func DoStuff(x int) error { ... }
```

**注释必须覆盖：**
| 信息 | 要求 |
|------|------|
| 函数目的 | 做什么，一句话说清 |
| 参数含义 | 每个参数的业务语义，非类型名复读 |
| 返回值 | 成功/失败的业务含义 |
| 副作用 | 是否修改数据库/缓存/外部服务 |
| 并发安全 | 是否 goroutine-safe，是否需要外部加锁 |
| 幂等保证 | 资金写操作必须声明幂等机制 |

### 6.2 关键业务日志与链路追踪

**铁律：** 所有资金操作、权限判定、路由决策、安全拦截必须输出**结构化日志**，且每条日志必须携带 `request_id` 串联全链路。

**日志必须包含的字段：**

| 字段 | 出现场景 | 格式 |
|------|---------|------|
| `request_id` | **全部** | UUID v4 |
| `trace_id` | 跨服务调用 | UUID v4 |
| `account_id` | 资金操作 | int64 |
| `freeze_id` | 冻结/解冻/结算 | UUID |
| `idempotency_key` | 幂等写操作 | UUID |
| `amount` | 金额变更 | NUMERIC 字符串（禁止浮点） |
| `direction` | 资金方向 | debit/credit/freeze/unfreeze/settle |
| `balance_after` | 余额变更后 | NUMERIC 字符串 |
| `error_code` | 异常路径 | PRD §6 统一错误码 |
| `latency_ms` | 关键路径耗时 | int64 |

```go
// ✅ 正确：结构化日志 + 全链路字段
slog.InfoContext(ctx, "freeze_acquired",
    "request_id", req.RequestID,
    "account_id", acct.ID,
    "freeze_id", freeze.ID,
    "amount", freeze.Amount.String(),
    "balance_after", acct.AvailableBalance.Sub(freeze.Amount).String(),
    "expires_at", freeze.ExpiresAt,
)

// ❌ 禁止：无结构化字段、无 request_id
log.Println("freeze done")
fmt.Printf("froze %v\n", amount)
```

**日志级别规范：**

| 级别 | 使用场景 |
|------|---------|
| `ERROR` | 余额不足、冻结失败、上游调用失败、数据不一致检测、资金守恒校验失败 |
| `WARN` | 预算告警比例触发、冻结即将过期、流式续期累计上限临近、熔断器半开 |
| `INFO` | 划拨成功、冻结成功、结算成功、路由决策、安全拦截、模型授权判定 |
| `DEBUG` | 策略评分明细、候选过滤明细、用量规范化中间值 |

### 6.3 代码结构与模块化

**铁律：** 代码遵循标准 Go 项目布局，每个包职责单一、接口清晰、依赖方向从外层到内层。

```
✅ 正确的包结构：
fund/
  service.go      # Service struct + 公开方法（Allocate/Freeze/Settle/...）
  model.go         # 数据模型（Account/Ledger/Freeze）
  store.go         # Store 接口定义
  sqlstore/        # SQL 实现
    pg.go          # PostgreSQL
    sqlite.go      # SQLite
  errors.go        # 包级错误定义

❌ 禁止的结构：
fund/
  everything.go    # 所有逻辑堆一个文件
  utils.go         # 万能工具函数
  helper.go        # 无明确职责的杂项
  temp.go          # 临时文件未清理
```

**模块化规则：**

| 规则 | 说明 |
|------|------|
| **单一职责** | 一个包只做一件事；一个文件只定义一个核心概念 |
| **接口隔离** | 包对外暴露接口，隐藏实现细节；消费者只依赖接口 |
| **依赖倒置** | 高层包定义接口，低层包实现接口（如 `fund.Store` 接口由 `sqlstore` 实现） |
| **禁止循环依赖** | 严格遵循 §11.2 的四层依赖图，上层可依赖下层，反之禁止 |
| **文件行数** | 单文件不超过 500 行；超过则拆分 |
| **函数行数** | 单函数不超过 80 行；超过则提取子函数 |
| **参数数量** | 单函数参数不超过 5 个；超过则封装为 Request struct |

### 6.4 解决简单问题复杂化

**铁律：** 优先使用标准库和已有能力，禁止为简单问题引入过度抽象。

```
❌ 禁止的过度设计：
- 为单一实现定义 3 层接口抽象
- 用一个接口只有一个实现的"策略模式"
- 用反射替代编译期类型检查
- 为 2 个字段建一张新表
- 用一个 goroutine + channel 替代一个简单的 if 判断
- 用消息队列替代同步函数调用（除非有明确的异步/削峰需求）
- 引入第三方框架替代标准库能解决的问题

✅ 鼓励的简单设计：
- 优先用 net/http 标准库，不引入 gin/echo/chi（与 TokenHub 一致）
- 优先用 database/sql + GORM（与 TokenHub 一致）
- 优先用 encoding/json 标准库
- 优先用 context.Context 传递请求级数据
- 错误处理用 fmt.Errorf("%w", err) 包装，不用第三方错误库
```

**自检清单（每次提交前）：**
1. 这个抽象是否只有一个实现？→ 如果是，移除抽象层
2. 这个 goroutine 是否可以用同步调用替代？→ 如果是，改为同步
3. 这个配置项是否有超过 1 个有效值？→ 如果不是，改为常量
4. 这个接口是否只有 1 个方法被外部调用？→ 如果是，缩小接口

### 6.5 测试纪律

| 规则 | 说明 |
|------|------|
| 单元测试 | 每个导出函数必须有对应 `_test.go`，覆盖正常路径 + 至少 1 个异常路径 |
| 资金测试 | 必须包含：划拨守恒验证、冻结超时验证、幂等重复调用验证 |
| 测试数据 | 禁止依赖生产数据；使用 `testing.T` 临时目录或内存数据库 |
| 测试隔离 | 每个 `TestXxx` 独立运行，不依赖执行顺序 |
| Mock | 仅 Mock 外部边界（HTTP/Redis）；内部逻辑用真实实现 |

---

## 附录：本项目仓库信息

| 项 | 值 |
|----|-----|
| 仓库地址 | `https://github.com/tt-52101/ai-gov` |
| 默认分支 | `main` |
| 当前基线标签 | `v3.2.0-baseline` |
| 项目代号 | `ai-gov` |
| 本地路径 | `D:\ai-work\grok\ai-gov` |
