# 单兵轨迹 · QA-CODE-UI

| 项 | 值 |
|----|-----|
| Agent ID | QA-CODE-UI |
| 所属蜂群 | batch-008 代码事实核查 |
| 作战指令 | 基于事实代码核查前端UI域（8模块+middleware+错误码+ABAC投影） |
| 执行耗时 | 约 4 分钟 |
| 验收判定 | ⚠️ 条件不通过——5 项 UI 条款未实现 + 4 项 P1 |

## 情报收集（逐文件）

### 前端结构
- package.json：next 16.2.9 + react 19.2.7 ✅
- middleware.ts：82行 ✅ 存在（batch-005 修复确认）；但仅校验 cookie 存在性，不校验 token 有效性 ⚠️
- next.config.ts：standalone 输出 ✅

### ConsoleLayout
- app/(console)/layout.tsx：13行 ✅ 渲染 {children}（batch-005 修复确认）
- app/(console)/console-router.tsx：149行 ✅ FAIL-CLOSED（batch-007 R6-08 修复确认）
- app/(console)/gov/layout.tsx：198行 ✅ 渲染 {children}；8项导航硬编码 + ABAC 投影过滤

### 8 模块核查
| 模块 | 文件 | 行数 | 状态 |
|------|------|------|------|
| dashboard | page.tsx | 289 | ✅ 真实实现，但 trend 字段硬编码 mock（4处 TODO） |
| parties | page.tsx | 702 | ✅ 真实实现 |
| fund | page.tsx | 504 | ✅ 真实实现，含 Idempotency-Key |
| pricing | page.tsx | 547 | ✅ 真实实现，5种计价模式 |
| routes | page.tsx | 536 | ✅ 真实实现，δ≤20% 校验 |
| abac | page.tsx | 812 | ✅ 真实实现，含策略模拟器 |
| ui-permissions | page.tsx | 593 | ✅ 真实实现 |
| audit | page.tsx | 357 | ✅ 真实实现，含快照对比 |

### 错误码映射
- lib/error-codes.ts：92行，25 个错误码映射 ✅
- getErrorMessage + extractErrorMessage 函数 ✅

### ABAC 投影
- gov/layout.tsx L69-117：调用 /v1/gov/ui-permissions/snapshot 获取 menus[].visible ✅
- 失败回退显示全部 8 项 ⚠️ FAIL-OPEN（P2）

### 国际化
- features/admin/i18n/runtime.tsx：218行，支持 zh-CN/en/ja ✅
- 但 gov/ 8 模块硬编码中文，未走 i18n ❌

## 战果产出

| 文件 | 行数 | 关键发现 |
|------|------|---------|
| middleware.ts | 82 | 存在但仅校验 cookie 存在性（P2 FAIL-OPEN） |
| gov/layout.tsx | 198 | 8项导航硬编码 + ABAC 投影过滤；失败回退 FAIL-OPEN |
| dashboard/page.tsx | 289 | trend 字段 4处硬编码 mock（P1-3） |
| lib/error-codes.ts | 92 | 25 个错误码映射（P1-2 数量差距） |
| _components/ConfirmDialog.tsx | 115 | useEffect 违反 Hooks 规则（P2） |

## 发现结论

### P0 阻塞
无

### P1 重要（4 项）
- P1-9: 5 个 UI 条款未实现（UI-05 Key/UI-08 密钥仓库/UI-09 模型权限/UI-10 安全报表/UI-11 调用追踪）
- P1-2(隐含): 错误码映射 25 个，与后端 27+ 存在差距
- D-03: dashboard trend 硬编码 mock
- D-06: 治理控制台未走 i18n

### P2 一般（4 项）
- D-04: middleware 仅校验 cookie 存在性
- D-05: ConfirmDialog useEffect 违反 Hooks 规则
- D-07: Next.js 16 废弃 middleware 约定
- D-08: ABAC 投影失败回退 FAIL-OPEN

## 编译/测试结果
- npx tsc --noEmit ✅ 通过
- npx next build ✅ 通过（43 路由）
