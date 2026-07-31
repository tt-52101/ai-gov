# UX-2 全页面体验走查报告

- **审计人**: UX-2 (UED 体验交互专家)
- **审计日期**: 2026-07-31
- **审计范围**: `ai-gov-fusion/frontend/app/(console)/gov/` 全部 8 个模块
- **审计方法**: 逐文件代码走查，基于事实，不做推测

---

## 一、模块逐项检查结果

### 1.1 dashboard（仪表盘）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| StatCard 是否包含图标 | PASS | `StatCard.tsx` 接受 `icon: LucideIcon`，dashboard 传入 `TrendingUp`/`Wallet`/`BarChart3`/`ShieldAlert` |
| StatCard 是否包含数值 | PASS | `StatCard.tsx` 接受 `value: string\|number`，渲染为 `text-2xl font-bold` |
| StatCard 是否包含趋势指示 | PARTIAL | `StatCard` 组件定义了 `trend` prop（正负百分比），但 **dashboard 中 4 个 StatCard 均未传入 `trend`**。趋势指示能力存在但未被使用 |
| 消耗图表是否存在 | PASS | `renderTrendChart()` 实现了简易 CSS 柱状图，带 hover tooltip 显示日期/金额，横轴标签 |

**问题**: StatCard 的 `trend` 能力闲置 —— 仪表盘 StatCard 未展示环比/同比趋势数值。

---

### 1.2 parties（Party 管理）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| Party 表格是否支持类型筛选 | PASS | 页面顶部 `<select>` 提供 "全部类型" / "组织(org)" / "项目(project)" |
| 创建对话框是否支持 org/project 选择 | PASS | 创建对话框中 `<select>` 包含 `org` / `project` 两个选项 |
| 边管理面板是否展示 7 种边类型 | PASS | `edgeTypeLabels` 定义了 7 种：parent(上级-下级)、sponsors(出资)、owns(主办)、participates(协作)、allocates(个人划拨)、merged_into(合并并入)、split_from(拆出) |

**问题**: 
1. Party 删除功能未实现 —— 表格操作列无删除按钮，页面底部的 `ConfirmDialog` 仅绑定 `onConfirm={() => setConfirmDelete(null)}`（空的确认回调），`confirmDelete` state 从未被设为非 null 值，该对话框为死代码。

---

### 1.3 fund（资金操作）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 账户卡片是否显示可用余额 | PASS | StatCard "可用余额" 显示 `totalAvailable`，账户详情区显示 `selectedAccount.available_balance` |
| 账户卡片是否显示冻结额 | PASS | StatCard "冻结金额" 显示 `totalFrozen`，账户详情区显示 `selectedAccount.frozen_balance` |
| 划拨对话框是否包含目标账户选择 | PASS | 划拨对话框包含 "目标账户 ID" 文本输入框 |
| 流水列表是否有方向筛选 | PASS | 流水区 `<select>` 提供 5 种方向：全部/支出(debit)/收入(credit)/冻结(freeze)/解冻(unfreeze)/结算(settle) |

**问题**: 无功能缺失。

---

### 1.4 pricing（价目维护）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 价目编辑器是否覆盖 5 种计价模式 | PASS | `pricingModes` 数组含 5 种：flat_fee, usage_per_unit, usage_tiered, usage_volume, amortization_fixed。编辑器 cost/sell 模式下拉均引用此数组 |
| 双轨 cost/sell 是否并排显示 | PASS | 预览面板 `<table>` 包含 cost 模式/费率 和 sell 模式/费率 列并排展示 |
| CodeBlock JSON 视图是否存在 | PASS | 预览面板底部 `<CodeBlock data={previewPrice.price_json} maxHeight="300px" />` |

**问题**: 无功能缺失。

---

### 1.5 routes（路由档案）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 策略开关是否 12 个全部可切换 | PASS | `allStrategyCodes` 定义 12 项（S-COMPLIANCE ~ S-CACHE），编辑器中全部用 checkbox 渲染，`toggleStrategy` 支持单独切换 |
| delta 滑杆是否硬限制 ≤20% | PASS | 双重保障：(1) range input `max={0.2}` 限制 UI 拖动范围；(2) `handleSave` 第一行 `if (editorForm.delta_cap > 0.2)` 阻止保存并显示错误 |

**问题**: 无功能缺失。

---

### 1.6 abac（ABAC 策略管理）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 4 标签页是否完整 | PASS | `tab` state 类型为 `"roles" \| "policies" \| "bindings" \| "simulator"`，4 个标签按钮均有图标+文字 |
| 策略模拟器是否显示匹配链 | PASS | 模拟结果区渲染 `simResult.evaluation_details` 列表，每个 detail 显示 policy_code / effect / matched / reason |
| is_system 角色是否不可删除 | PASS | `roleColumns` 操作列：`!r.is_system` 时渲染删除按钮，`r.is_system` 时显示 "不可操作" 文本 |

**问题**: 无功能缺失。

---

### 1.7 ui-permissions（UI 权限管理）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 3 标签页是否完整 | PASS | `tab` state 类型为 `"menus" \| "routes" \| "buttons"`，3 个标签按钮渲染完整 |
| 菜单树是否可展开 | PASS | `renderMenuNode` 递归渲染 `node.children`，通过 `depth` 参数控制 `marginLeft` 缩进（depth * 24px） |
| 路由是否可绑定 required_action | PASS | 路由对话框 `routeForm.required_action_id` 输入框，"添加路由" mutation body 包含该字段；列表展示 "动作: {required_action_id}" |

**问题**: 无功能缺失。

---

### 1.8 audit（审计日志查询）

| 检查项 | 结果 | 证据/文件 |
|--------|------|-----------|
| 事件列表是否支持筛选 | PASS | 筛选栏包含操作者(input)、动作(select 6 种 action)、日期范围(date from/to) |
| 详情是否并排显示 before/after 快照 | PASS | `grid grid-cols-2` 布局，左侧 before（红色标注），右侧 after（绿色标注），下方有 diff 汇总行 |
| 不可变标记是否可见 | PASS | 两处：页面标题栏右侧 "不可删除" 徽章（Lock 图标 + 文字）；详情面板底部黄色警告条 "审计记录不可编辑、不可删除。保留不少于 180 天。" |

**问题**: 无功能缺失。

---

## 二、状态覆盖矩阵

| 模块 | loading.tsx | error.tsx | 空数据状态 |
|------|:-----------:|:---------:|-----------|
| dashboard | PASS (骨架屏：标题+4卡片+图表) | PASS (重试按钮 + error.message) | PASS：block_rates 空时 "本周期无拦截事件"，top_consumers 空时 "暂无消费数据"。但 data 为 null 且非 loading/error 时仅显示标题（边界情况） |
| parties | PASS (骨架屏：标题+表格行) | PASS (重试按钮 + 警告图标) | PASS：DataTable emptyText="暂无 Party 数据"，边/成员面板有各自空提示 |
| fund | PASS (骨架屏：标题+4卡片+表格) | PASS (重试按钮 + error.message) | PASS：DataTable emptyText 覆盖账户列表和流水列表 |
| pricing | PASS (骨架屏：标题+表格) | PASS (重试按钮 + error.message) | PASS：DataTable emptyText="暂无价目数据"，编辑器内 "暂无定价项，点击添加项开始配置" |
| routes | PASS (骨架屏：标题+表格) | PASS (重试按钮 + error.message) | PASS：DataTable emptyText="暂无路由档案" |
| abac | PASS (骨架屏：标题+4标签+表格) | PASS (重试按钮 + error.message) | PASS：DataTable emptyText 覆盖角色/策略/绑定三标签，模拟器结果区 "尚未执行模拟评估" |
| ui-permissions | PASS (骨架屏：标题+3标签+内容) | PASS (重试按钮 + error.message) | PASS：菜单树 "暂无菜单配置"，路由/按钮列表各有 "暂无路由配置"/"暂无按钮绑定" |
| audit | PASS (骨架屏：标题+筛选栏+表格) | PASS (重试按钮 + error.message) | PASS：DataTable emptyText="暂无审计事件"，详情加载中 spinner |

**结论**: 全部 8 个模块均包含 loading.tsx、error.tsx 和空数据状态处理。

---

## 三、共享组件检查

| 组件 | 文件 | 用途 | 质量评估 |
|------|------|------|----------|
| StatCard | `_components/StatCard.tsx` | 统计卡片（图标/数值/趋势/描述） | 完整，支持 trend 但 dashboard 未使用 |
| DataTable | `_components/DataTable.tsx` | 通用表格（排序/分页/搜索/骨架/空态） | 完整，内置 loading 骨架行和 empty 行 |
| CodeBlock | `_components/CodeBlock.tsx` | JSON 代码展示（格式化/复制/折叠） | 完整 |
| ErrorAlert | `_components/ErrorAlert.tsx` | 错误提示（重试/关闭） | 完整 |
| ConfirmDialog | `_components/ConfirmDialog.tsx` | 危险操作确认弹窗 | 完整，支持 danger 模式/loading/Esc 关闭 |

---

## 四、发现的问题汇总

### 4.1 功能缺陷

| # | 模块 | 严重度 | 描述 |
|---|------|--------|------|
| P1 | parties | HIGH | Party 行删除功能未实现。表格操作列无删除按钮，页面底部 `ConfirmDialog` 的 `onConfirm` 回调为 `() => setConfirmDelete(null)`（空操作），`confirmDelete` state 从未被赋值为 Party 对象。该对话框为死代码 |
| P2 | dashboard | MEDIUM | StatCard 组件支持 `trend` 趋势指示 prop（正负百分比+颜色），但仪表盘 4 个 StatCard 均未传入 `trend`，导致趋势信息缺失 |

### 4.2 体验一致性

| # | 模块 | 严重度 | 描述 |
|---|------|--------|------|
| C1 | dashboard | LOW | 页面同时拥有 Next.js 的 `loading.tsx`（路由级骨架屏）和页面内部的 `loading` 状态（数据获取级骨架屏），两者骨架布局相似但不完全一致，可能造成加载闪烁 |
| C2 | ui-permissions | LOW | 菜单/路由/按钮三个子标签的 loading 状态使用内联 `animate-pulse` div，而非共享骨架组件；但 abac 模块同样模式，风格一致 |

---

## 五、审计结论

### 通过项（所有强制检查项均通过）

- dashboard: StatCard（图标/数值/描述）、消耗趋势图
- parties: 类型筛选、org/project 选择、7 种边类型
- fund: 可用余额/冻结额、目标账户选择、方向筛选
- pricing: 5 种计价模式、双轨 cost/sell 并排、CodeBlock JSON 视图
- routes: 12 策略开关可切换、delta 硬限制 ≤20%
- abac: 4 标签页完整、匹配链展示、is_system 不可删除
- ui-permissions: 3 标签页完整、菜单树可展开、路由绑定 required_action
- audit: 事件筛选、before/after 快照并排、不可变标记
- 状态覆盖: 全部 8 模块均具备 loading.tsx + error.tsx + 空数据状态

### 阻塞项

**P1 - parties 模块 Party 删除功能缺失**。表格中无删除操作的入口，已存在的 `ConfirmDialog` 确认回调为空操作。需补充删除按钮并正确连线删除 API。

### 建议项

1. **P2**: 为 dashboard StatCard 接入 `trend` 数据（环比/同比），当前 `trend` prop 闲置。
2. **C1**: 统一 dashboard 的加载态 —— 考虑去掉页面内部 `loading` state 的骨架屏渲染，完全依赖 `loading.tsx`，避免双层 skeleton。
