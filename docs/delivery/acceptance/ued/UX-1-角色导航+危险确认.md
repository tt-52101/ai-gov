# UX-1 UED 体验交互走查报告：角色化导航 + 危险确认

**走查人：** UX-1（UED 体验交互专家）  
**走查日期：** 2026-07-31  
**走查范围：** PRD §9.9 UI-01~14 中的 UI-01、UI-03、错误文案对齐（PRD §6）、DataTable 组件  
**审查文件：**
- `ai-gov-fusion/frontend/app/(console)/gov/layout.tsx`
- `ai-gov-fusion/frontend/app/(console)/gov/fund/page.tsx`
- `ai-gov-fusion/frontend/app/(console)/gov/_components/ConfirmDialog.tsx`
- `ai-gov-fusion/frontend/app/(console)/gov/_components/DataTable.tsx`
- `ai-gov-fusion/frontend/app/(console)/gov/_components/ErrorAlert.tsx`
- `ai-gov-fusion/frontend/features/admin/core/navigation.tsx`
- `ai-gov-fusion/backend/internal/server/fund/errors.go`
- `ai-gov-fusion/backend/internal/server/modelgrant/checker.go`
- `docs/prd/AI-GOV-PRD-v3.0.0-融合架构完整方案.md`

---

## 1. UI-01: 角色化导航

### 1.1 导航项数量检查

**结论：通过。**

`layout.tsx` 第 32-41 行的 `navItems` 数组包含 8 个导航项，与 PRD §9.10 UI-01 要求一致：

| # | 路径 | 标签 | 图标 |
|---|------|------|------|
| 1 | `/gov/dashboard` | 仪表盘 | LayoutDashboard |
| 2 | `/gov/parties` | Party 管理 | Building2 |
| 3 | `/gov/fund` | 资金操作 | Wallet |
| 4 | `/gov/pricing` | 价目维护 | Tags |
| 5 | `/gov/routes` | 路由档案 | GitBranch |
| 6 | `/gov/abac` | ABAC 策略 | Shield |
| 7 | `/gov/ui-permissions` | UI 权限 | Eye |
| 8 | `/gov/audit` | 审计日志 | ClipboardList |

### 1.2 ABAC 权限隐藏

**结论：不通过。**

`layout.tsx` 无条件渲染全部 8 个导航项，未实现按 ABAC 权限隐藏无权限菜单。

- `layout.tsx` 第 69-97 行：`navItems.map()` 直接渲染所有菜单项，没有任何权限检查逻辑。
- 代码中未导入或调用任何 ABAC 检查函数（如 `canAccessView`、`usePermission`、projector 调用）。
- 项目中的权限过滤逻辑（`canAccessView`、`filterNavItemByAccess`）位于 `features/admin/core/navigation.tsx`，仅被 TokenHub 管理台壳（`admin-console.tsx`）使用，gov 布局未复用。

**风险：** 无权限用户可能看到无法操作的菜单项，点击后进入页面虽然 API 层可能拒绝请求，但前端未做菜单级隐藏，不符合"按授权显示"的 PRD 要求。

### 1.3 版本徽章

**结论：通过。**

`layout.tsx` 第 103 行：`<p className="text-xs text-gray-400">v3.2.0</p>`  
版本号 `v3.2.0` 显示在侧边栏底部，格式正确。

---

## 2. UI-03: 危险操作二次确认

### 2.1 ConfirmDialog 存在性

**结论：通过。**

`fund/page.tsx` 第 490-499 行使用 `<ConfirmDialog>` 组件包裹清算操作，`danger` 属性设为 `true`（红色警告样式），确认后调用 `handleLiquidate`。

### 2.2 ESC 键与遮罩关闭

**结论：通过。**

- **ESC 键关闭：** `ConfirmDialog.tsx` 第 46-54 行注册 `keydown` 事件监听，`Escape` 键触发 `onCancel`。`loading` 期间不响应 ESC（防止误关闭）。
- **遮罩关闭：** `ConfirmDialog.tsx` 第 71 行：半透明遮罩层 `onClick={loading ? undefined : onCancel}`，点击遮罩外区域可关闭对话框。`loading` 期间同样禁用。

### 2.3 确认按钮文案

**结论：通过。**

`fund/page.tsx` 第 495 行：`confirmLabel="确认清算"`，非通用"确认"，符合 PRD 要求。  
`ConfirmDialog.tsx` 第 36 行默认值为 `"确认"`，但调用方显式传入具体文案。

### 2.4 清算操作额外检查

**结论：通过（有改进空间）。**

- 清算按钮在非 `active` 状态账户时禁用（第 365 行：`disabled={selectedAccount.status !== "active"}`）。
- 清算请求发送 `Idempotency-Key` 头（第 159 行），符合 PRD §11.4 幂等要求。
- **问题：** `handleLiquidate`（第 174 行）仅检查 `res.ok`，未解析后端返回的错误码（如 `LIQUIDATION_STAGE_INVALID`），错误直接显示 `HTTP ${res.status}`，用户无法理解具体失败原因。

---

## 3. 错误文案对齐（PRD §6）

### 3.1 错误码定义（后端）

**结论：通过（后端定义完整）。**

后端已正确定义 PRD §6 中的三个关键错误码：

| 错误码 | 后端定义位置 | HTTP 状态码 | PRD 含义 |
|--------|-------------|-------------|----------|
| `BUDGET_CAP_EXCEEDED` | `fund/errors.go:100` | 402 | 命中预算上限（余额可能>0） |
| `INSUFFICIENT_BALANCE` | `fund/errors.go:82` | 402 | 可用余额不足以完成本次冻结 |
| `MODEL_ACCESS_DENIED` | `modelgrant/checker.go:16` | 403 | ModelGrant 不允许该模型 |

### 3.2 前端错误码映射

**结论：不通过。**

前端**完全缺失错误码到中文用户提示的映射层**。具体问题：

1. **fund/page.tsx 第 140-143 行（划拨）：** 尝试从响应 JSON 解析 `err.error?.message`，失败时回退到 `HTTP ${res.status}`。未对 `BUDGET_CAP_EXCEEDED`、`INSUFFICIENT_BALANCE` 等错误码做中文翻译。

2. **fund/page.tsx 第 174 行（清算）：** 仅检查 `res.ok`，错误统一显示 `HTTP ${res.status}`，未解析响应体中的错误码。

3. **所有 gov 页面统一模式：** 错误处理均为 `setError(err instanceof Error ? err.message : "XXX失败")`，后端返回的英文技术描述或 HTTP 状态码直接暴露给最终用户。

4. **不存在共享的错误映射模块：** 搜索 `errorMap`、`translateError`、`ERROR_MAP` 均无结果。各页面独立处理错误，无统一的 `code -> 中文提示` 映射。

| PRD 错误码 | 期望中文提示 | 实际前端行为 |
|------------|-------------|-------------|
| `BUDGET_CAP_EXCEEDED` | 预算已达上限 | 显示英文后端消息或 `HTTP 402` |
| `INSUFFICIENT_BALANCE` | 可用余额不足 | 显示英文后端消息或 `HTTP 402` |
| `MODEL_ACCESS_DENIED` | 无权访问该模型 | 未在 gov 模块中使用，无映射 |

---

## 4. DataTable 组件

### 4.1 排序

**结论：通过。**

- `ColumnDef` 接口含 `sortable?: boolean`（第 15 行），设为 `true` 的列标题可点击切换排序。
- `DataTableProps` 含 `sortKey`、`sortDirection`、`onSort` 回调（第 38-42 行）。
- 排序图标：升序显示 `ChevronUp`，降序显示 `ChevronDown`（第 95-102 行）。

**注意：** fund/page.tsx 中所有列定义均未设置 `sortable: true`，排序功能虽然组件支持但未启用。

### 4.2 分页

**结论：通过。**

- 组件支持 `page`、`pageSize`、`total`、`onPageChange` 全套分页参数（第 31-37 行）。
- 分页控件：上一页/下一页按钮 + "共 N 条记录，第 X/Y 页" 文本（第 194-219 行）。
- 边界处理：首页时"上一页"禁用，末页时"下一页"禁用。

### 4.3 搜索

**结论：通过（组件层面）。**

- `searchPlaceholder` 属性控制搜索框显隐（第 113 行），提供时渲染搜索输入框。
- 搜索通过表单提交触发 `onSearch` 回调（第 81-84 行）。

**注意：** fund/page.tsx 使用 DataTable 时未传 `searchPlaceholder`，因此搜索框不显示。组件能力就绪，但调用方未启用。

### 4.4 空状态

**结论：通过。**

- 数据为空时显示居中文本（第 161-170 行）。
- 默认文本 `"暂无数据"`，调用方可自定义 `emptyText` 属性。
- fund/page.tsx 传入 `emptyText="暂无账户数据"` 和 `emptyText="暂无流水记录"`。

### 4.5 加载骨架屏

**结论：通过。**

- `loading` 为 `true` 时渲染 3 行动画骨架（第 150-160 行）。
- 骨架行使用 `animate-pulse` + 灰色占位 `div`（`h-4 w-3/4 rounded bg-gray-200`）。
- 仅在非空数据加载时显示骨架；数据已存在时不会闪烁。

---

## 5. 汇总评估

| 检查项 | 结果 | 严重级别 |
|--------|------|----------|
| UI-01 导航项数量（8 项） | 通过 | -- |
| UI-01 ABAC 权限隐藏菜单 | **不通过** | 高 |
| UI-01 版本徽章 v3.2.0 | 通过 | -- |
| UI-03 ConfirmDialog 存在 | 通过 | -- |
| UI-03 ESC 关闭 | 通过 | -- |
| UI-03 遮罩关闭 | 通过 | -- |
| UI-03 确认按钮文案"确认清算" | 通过 | -- |
| 错误码后端定义 | 通过 | -- |
| 错误码前端中文映射 | **不通过** | 高 |
| DataTable 排序 | 通过（组件层面） | -- |
| DataTable 分页 | 通过 | -- |
| DataTable 搜索 | 通过（组件层面） | -- |
| DataTable 空状态 | 通过 | -- |
| DataTable 加载骨架屏 | 通过 | -- |

### 不通过项详情

**P0-1: ABAC 权限隐藏（UI-01）**

`layout.tsx` 无条件渲染全部 8 个导航项，未调用任何 ABAC 权限检查接口。PRD §9.10 明确要求"角色化导航（按授权显示）"。建议：
- 在 `layout.tsx` 中引入权限检查 hook 或 projector 调用
- 对 `navItems` 进行 `filter`，仅渲染用户有权访问的菜单项
- 可复用 `features/admin/core/navigation.tsx` 中的 `canAccessView` 逻辑或建立 gov 模块独立的权限判断

**P0-2: 错误码前端中文映射（PRD §6）**

前端普遍将后端英文错误信息或 HTTP 状态码直接展示给用户。PRD §11.6 UED 验收门禁明确要求"错误文案"走查通过。建议：
- 建立 `gov/_components/error-messages.ts` 统一错误码映射表
- 映射关系至少覆盖：`BUDGET_CAP_EXCEEDED` -> "预算已达上限"、`INSUFFICIENT_BALANCE` -> "可用余额不足"、`MODEL_ACCESS_DENIED` -> "无权访问该模型"
- 所有 fetch 调用的 catch 分支应解析响应中的 `code` 字段并查表翻译
- ErrorAlert 组件显示的 `message` 应为中文用户提示而非技术错误

---

*走查完成。两个 P0 不通过项需在 UED 复验前修复。*
