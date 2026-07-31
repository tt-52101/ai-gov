# 两阶段提交（2PC）与分布式事务补偿

结合本产品的划拨 / 冻结 / 结算场景说明：**单库账本优先用本地 ACID 事务**；跨库、跨服务时再考虑 2PC 或补偿（Saga）。下文先讲 2PC 原理与实现，再深入补偿机制，并给出与资金域的落地建议。

---

## 1. 为什么会提到 2PC

划拨守恒要求：

\[
\Delta B_{from}+\Delta B_{to}=0
\]

在**同一 PostgreSQL 事务**里对两行 `FOR UPDATE` 即可满足。  
一旦变成：

- 账户在库 A、流水在库 B  
- 或「划拨服务」与「审计/通知服务」各有存储  
- 或跨区域多活各写本地账本  

单机事务不够，才需要**分布式事务**协议或**补偿**。

---

## 2. 两阶段提交（2PC）原理

### 2.1 角色

| 角色 | 职责 |
|------|------|
| 协调者（Coordinator） | 驱动协议、写事务日志、做最终决定 |
| 参与者（Participant） | 本地资源管理器（如各库、各服务内的本地事务） |

### 2.2 阶段一：准备（Prepare / Vote）

1. 协调者给所有参与者发 `PREPARE`。  
2. 每个参与者做本地校验与资源预留（如锁行、写 undo/redo），**不提交**。  
3. 成功则回复 `YES`（已进入 prepared，崩溃后仍能提交或回滚）；失败回复 `NO`。

### 2.3 阶段二：提交或中止（Commit / Abort）

- 若**全部 YES**：协调者持久化决定 `COMMIT`，再通知各参与者 `COMMIT`；参与者正式提交并释放资源。  
- 若**任一 NO** 或超时：协调者持久化 `ABORT`，通知 `ROLLBACK`；参与者回滚预留。

### 2.4 核心性质

- **原子性（在同步、多数存活假设下）：** 要么都提交，要么都回滚。  
- **阻塞点：** 参与者在 `YES` 之后、收到最终决定之前处于**不确定状态**；协调者宕机可能导致参与者长时间持锁。  
- **同步开销：** 至少两轮网络；延迟明显高于本地事务。

### 2.5 协议状态（参与者侧简化）

```text
初始 → 收到 PREPARE
  → 本地可做 → 写 prepare 日志 → YES → [不确定窗口] → COMMIT / ROLLBACK → 结束
  → 本地不可 → NO → 本地已回滚 → 结束
```

协调者必须把「全局决定」先落盘再广播，否则自身崩溃会丢决定。

---

## 3. 2PC 实现方案要点

### 3.1 经典形态（XA）

- 数据库支持 XA：`XA START` → 业务 SQL → `XA END` → `XA PREPARE` → 协调者根据票决定 `XA COMMIT` / `XA ROLLBACK`。  
- 协调者可用事务管理器（如旧式 Java EE TM）；Go 生态较少用完整 XA。

### 3.2 应用层“伪 2PC”（自研协调者）

适用于「多个本地事务资源」但无 XA：

```text
协调者事务日志表：txn_id, state(preparing|committed|aborted), participants[]

1. 写 txn 日志 preparing
2. 依次/并行调用各服务 Prepare API
   - 服务内：本地事务写入「预留记录」+ 状态 prepared，不改最终业务视图或只改可见性标记
3. 全部成功 → 日志 committed → 调各服务 Commit
4. 任一失败 → 日志 aborted → 调各服务 Rollback
```

**难点：** Commit 阶段部分成功要靠重试做到最终一致；这已接近补偿思维。

### 3.3 与本产品账户的映射示例（跨两库时）

| 参与者 | Prepare | Commit | Rollback |
|--------|---------|--------|----------|
| 源账户库 | 锁行，校验 available，写 `pending_out` | balance-=A，落 ledger，清 pending | 删 pending，解锁 |
| 目标账户库 | 锁行，写 `pending_in` | balance+=A，落 ledger，清 pending | 删 pending |

全局成功当且仅当两边 Commit 都完成。

### 3.4 2PC 的固有问题

| 问题 | 说明 |
|------|------|
| 同步阻塞 | 持锁时间长，吞吐差 |
| 协调者单点 | 需高可用与日志复制 |
| 启发式恢复 | 协调者长期失联时 DBA/策略强制提交或回滚，有错账风险 |
| 与长耗时操作不配 | 调用上游 LLM 不能放进 2PC 参与者 |
| 云上跨区 | 延迟与分区使 2PC 几乎不可用 |

因此：**网关数据面的「冻结→调模型→结算」绝不使用 2PC 包住上游 HTTP。**

---

## 4. 分布式事务补偿机制（深入）

当无法或不想用 2PC 时，采用**先执行、后用反向动作修齐**的思路，追求**最终一致**，而非强一致瞬时全局原子。

### 4.1 补偿 vs 回滚

| | 本地事务回滚 | 补偿 |
|--|--------------|------|
| 时机 | 提交前 | 往往已提交本地事务 |
| 手段 | 数据库 undo | 业务定义的反向操作 |
| 隔离 | ACID 隔离 | 中间态可被读到（需设计可见性） |
| 典型 | 单库划拨 | 跨服务：已扣款要退款、已预留要释放 |

### 4.2 Saga 模式

把长事务拆成多个**本地事务步骤** \(T_1, T_2, \ldots, T_n\)，每个 \(T_i\) 有补偿 \(C_i\)。

**成功路径：** \(T_1 \rightarrow T_2 \rightarrow \cdots \rightarrow T_n\)  

**在 \(T_k\) 失败：** 执行 \(C_{k-1}, C_{k-2}, \ldots, C_1\)（顺序与实现有关）。

两种编排：

| 类型 | 做法 | 特点 |
|------|------|------|
| 编排式（Orchestration） | 中心 Saga 协调者发命令、听结果 | 易审计、易控补偿顺序；协调者要可靠 |
| 事件式（Choreography） | 每步发领域事件，下游订阅执行 | 解耦；排查与循环依赖更难 |

资金域更适合**编排式**，便于与幂等键、流水对齐。

### 4.3 补偿设计原则

1. **补偿必须幂等：** 网络重试不能双重退款。  
2. **补偿尽量语义反向，而非“删历史”：** 用 `allocate_reverse` / `unfreeze` 等新流水，保留审计。  
3. **明确中间态：** 如 `transferring`、`freeze_open`，避免外部误读成终态。  
4. **超时与死信：** 补偿失败入死信队列 + 人工/自动再试。  
5. **不可补偿步骤后置：** 无法反向的外部副作用（真发邮件、真调不可逆 API）尽量放最后，或改成可对账的“意向”。

### 4.4 与 TCC 的关系

TCC = Try / Confirm / Cancel，可视为**带业务预备的补偿框架**：

| 阶段 | 含义 | 资金例子 |
|------|------|----------|
| Try | 预留资源 | 冻结额度、写 pending 划拨 |
| Confirm | 确认生效 | 解冻并实扣、pending→已入账 |
| Cancel | 释放预留 | 解冻、取消 pending |

TCC 比“先直接提交再补偿”更干净，中间态清晰；实现成本高于单 SQL 事务。  
本产品**数据面预扣**本质就是账户维度的 TCC：`Freeze=Try`，`Settle=Confirm`，`Timeout/Unfreeze=Cancel`。

---

## 5. 本产品场景下的选择

### 5.1 划拨（Allocate）

| 部署 | 推荐 |
|------|------|
| 两账户同库 | **本地事务 + FOR UPDATE**（前文守恒实现），不必 2PC |
| 账户分库 | 优先 **Saga/TCC**：Try 双边 pending → Confirm 落余额；或业务上避免分库 |
| 划拨 + 发消息通知 | 本地事务提交后 **发箱（Outbox）** 投递消息，不把 MQ 加入 2PC |

### 5.2 调用主路径（冻结 → 上游 → 结算）

```text
本地事务：Freeze（Try）
    ↓
HTTP 上游模型（不可加入 2PC）
    ↓
本地事务：Settle（Confirm）或超时任务 Cancel
```

这是**刻意拆开的 Saga**，不是分布式事务失败，而是架构选择：外部调用与账本解耦。

### 5.3 清算

状态机本身就是长流程 Saga：

`block_new → drain（等冻结 Cancel/Confirm）→ transfer（本地守恒事务）→ liquidated`  

drain 阶段靠冻结 TTL 与结算补偿，而不是 2PC 锁全库。

---

## 6. 补偿机制实现方案（可落地）

### 6.1 编排表示例

```text
SagaAllocate (跨库时):
  steps:
    - name: prepare_source
      action: POST /accounts/{from}/pending_out
      compensate: POST /accounts/{from}/pending_out/cancel
    - name: prepare_target
      action: POST /accounts/{to}/pending_in
      compensate: POST /accounts/{to}/pending_in/cancel
    - name: commit_source
      action: POST /accounts/{from}/pending_out/confirm
      compensate: POST /accounts/{from}/reverse_out   # 若 confirm 后失败需反向
    - name: commit_target
      action: POST /accounts/{to}/pending_in/confirm
      compensate: POST /accounts/{to}/reverse_in
```

更稳妥的 TCC 边界：**Confirm 只做“pending→正式”**，保证 Confirm 几乎不会失败；失败集中在 Try。

### 6.2 状态表（协调者）

```sql
CREATE TABLE saga_instances (
  id            UUID PRIMARY KEY,
  saga_type     VARCHAR(64) NOT NULL,  -- allocate_cross|settle_orphan|...
  business_key  VARCHAR(128) NOT NULL, -- 如 idempotency_key
  state         VARCHAR(32) NOT NULL,  -- running|compensating|completed|failed
  payload       JSONB NOT NULL,
  current_step  INT NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE saga_steps (
  saga_id     UUID NOT NULL REFERENCES saga_instances(id),
  step_index  INT NOT NULL,
  name        VARCHAR(64) NOT NULL,
  state       VARCHAR(32) NOT NULL, -- pending|succeeded|failed|compensated
  request     JSONB,
  response    JSONB,
  UNIQUE (saga_id, step_index)
);
```

工作线程：推进 `running`；失败改 `compensating` 并逆序调补偿；补偿也幂等。

### 6.3 冻结超时：补偿的经典例子

```text
Try:    freeze 成功（本地事务）
Confirm: 上游成功 → settle
Cancel:  上游失败或 expires_at 到期 → unfreeze + ledger(unfreeze_timeout)
```

后台扫描 `freezes WHERE status='open' AND expires_at < now()`，**幂等**更新为 `timeout_released` 并改余额。多实例用 `UPDATE ... WHERE status='open' RETURNING` 抢占，避免双释放。

### 6.4 结算孤儿补偿（调用成功但冻结丢失）

```text
检测到：上游成功日志有，freeze 已 timeout
→ 写 settle_orphan 流水尝试扣 available
→ 不足则记负向挂账/告警工单（产品策略），禁止静默丢成本
```

这是**业务补偿 + 人工闭环**，不是 2PC 能自动解决的。

### 6.5 Outbox：避免“事务 + MQ”二阶段

```text
同一本地事务：
  更新 accounts + ledgers
  INSERT outbox(event)
提交后轮询/CDC 发 MQ
消费者幂等处理通知、报表
```

不要对 MQ 做 XA。

---

## 7. 2PC 与补偿对比（决策表）

| 维度 | 2PC | 补偿 / Saga / TCC |
|------|-----|-------------------|
| 一致性 | 提交瞬间强一致 | 最终一致，有中间态 |
| 性能 | 差，锁时间长 | 较好，步间无全局锁 |
| 外部 HTTP | 不适合 | 适合（调用放步间） |
| 实现复杂度 | 依赖 XA/协调者 | 要设计补偿与幂等 |
| 运维 | 阻塞与启发式难 | 死信与对账可观察 |
| 本产品划拨同库 | **不必要** | 不必要 |
| 本产品调用链 | **禁止** | **Freeze/Settle 即 TCC** |
| 跨库划拨 | 可用但重 | **更推荐 TCC/Saga** |

---

## 8. 推荐架构结论（针对当前网关）

1. **划拨守恒：** 单库本地事务实现（有序 `FOR UPDATE` + 双边流水 + 幂等），不引入 2PC。  
2. **预扣与结算：** 按 TCC 语义实现（Freeze / Settle / Unfreeze），上游调用在 Try 与 Confirm 之间，用超时任务做 Cancel。  
3. **跨服务副作用：** Outbox + 幂等消费者；失败靠重试与死信，不靠 2PC。  
4. **真跨库账户：** 优先业务避免；若必须，用编排 Saga + 账户侧 pending TCC，补偿发行反向流水，并配对账。  
5. **禁止：** 把「调 LLM」或「发邮件」放进 2PC 参与者列表。

---

## 9. 最小实现路线图

| 步骤 | 内容 |
|------|------|
| 1 | 同库 Allocate 本地事务（已有详述） |
| 2 | Freeze/Settle/TTL Cancel 做成明确状态机 + 幂等 |
| 3 | Outbox 发“划拨完成/预算告警”事件 |
| 4 | （可选）saga_instances 表只用于跨库或复杂清算子流程 |
| 5 | 对账作业：按日校验 \(\sum out=\sum in\)、冻结悬挂、孤儿结算 |

---

## 10. 一句话对照

- **2PC：** 提交前大家举手，再一起落锤；强一致、易阻塞，不适合带外部 IO 的网关热路径。  
- **补偿/Saga/TCC：** 能做的先本地提交或预留，失败用**幂等反向业务动作**修齐；适合冻结—调用—结算，也是分布式下的务实选择。  

对当前 Token 治理底座：**守恒划拨用本地事务；分布式问题用 TCC 式冻结与 Saga 式补偿，而不是上 2PC 扛调用链。**