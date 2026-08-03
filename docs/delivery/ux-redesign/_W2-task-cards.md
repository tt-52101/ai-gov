# W2 开发任务卡片 · 7 项必做 + 1 项脚手架 = 8 张

> **本文件为 UX 实施可执行任务清单**, 8 张任务卡片, 每张可独立分配 Agent 执行.
> 用户硬性要求 (2026-08-03 最新裁决): 7 项必做 (含移动端响应式), 一项不能少.
> 蜂群: batch-009-ux-implementation | 启动条件: 用户明确指令 "启动 W2" / "启动 UX 实施".

---

## 📋 任务卡片索引

| 卡片 | ID | 任务 | 优先级 | 复杂度 | 依赖 |
|---|---|---|---|---|---|
| [#1](#1-skin--css-variables--5-套) | S1 | Skin: CSS Variables (5 套) | P0 | 高 (~1500 行) | 无 |
| [#2](#2-skinprovider--localstorage) | S2 | SkinProvider + localStorage 持久化 | P0 | 中 (~200 行) | S1 |
| [#3](#3-themeprovider--next-themes--切换器-ui) | T1 | ThemeProvider + next-themes + 切换器 UI | P0 | 中 (~400 行) | 无 |
| [#4](#4-guardian-gateway-优化版优先交付) | G1 | guardian-gateway 优化版 (优先) | **P0** | 高 (~2500 行) | S1, S2, T1 |
| [#5](#5-shadcn-组件库按 5 套皮肤扩变体) | C1 | shadcn 组件库 5 套扩变体 | P1 | 高 (~800 行) | S1, S2 |
| [#6](#6-i18n--next-intl--双语--术语库) | I1 | i18n: next-intl + 双语 + 术语库 | P1 | 中 (~600 行) | 无 |
| [#7](#7-移动端响应式-128014401920-三档) | R1 | 移动端响应式 (1280/1440/1920 三档) | P0 | 中 (~500 行) | S1 |
| [#8](#8-视觉回归--a11y-审计) | Q1 | 视觉回归 + a11y 审计 | P1 | 中 (~300 行) | S1-S7 |

**总预估**: 8 Agent · 6-8h · +5000 行净增 (与 [§3 预算](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/_W2-W3-plan.md#L182-L188) 对齐)

---

## #1 Skin · CSS Variables · 5 套

**Agent ID**: S1 · **优先级**: P0 · **复杂度**: 高 (~1500 行)

### 任务
基于 [设计规范 §2-3](file:///d:/ai-work/grok/a-gov/docs/superpowers/specs/2026-08-03-ux-redesign-design.md), 创建 5 套皮肤的 CSS Variables 定义:
- `skin-government.css` (政务风 v1)
- `skin-cloud.css` (企业云风 v2)
- `skin-bank.css` (银行风 v3)
- `skin-guardian.css` (guardian-gateway 优化版, **优先**)
- `skin-default.css` (基础 Token, 共用)

### 必做字段
颜色 (primary/link/danger/bg/text/semantic/border) + 字号 (12/14/16/20/24/32/48) + 留白 (4/8/12/16/24/32/48/64) + 圆角 (4/8/12/16) + 阴影 (4 级)

### 验收
- [ ] 5 套皮肤切换无视觉错位
- [ ] 所有颜色/字号/留白 100% 来自 Token
- [ ] 与 [README §6 必做总览 #1-#4](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/README.md#L122-L130) 一致

---

## #2 SkinProvider + localStorage

**Agent ID**: S2 · **优先级**: P0 · **复杂度**: 中 (~200 行)

### 任务
创建 React Context (`SkinProvider`) 管理当前皮肤选择, 持久化到 `localStorage`.

### 关键 API
```tsx
const { skin, setSkin, availableSkins } = useSkin()
// skin: 'guardian' | 'government' | 'cloud' | 'bank'
// 切换时更新 <html data-skin="..."> + localStorage
```

### 验收
- [ ] 切换皮肤立即生效 (无刷新)
- [ ] 刷新后保留上次选择
- [ ] 默认值: `guardian` (优化版优先)

---

## #3 ThemeProvider + next-themes + 切换器 UI

**Agent ID**: T1 · **优先级**: P0 · **复杂度**: 中 (~400 行)

### 任务
- 接入 `next-themes` (已在 `package.json`)
- 创建 `ThemeProvider` 包裹根布局
- 创建 `ThemeSwitcher` UI 组件 (light/dark/auto 三态)
- 与 `SkinProvider` 组合 (双层叠加: skin 在外, theme 在内)

### 关键代码
```tsx
"use client"
import { useTheme } from "next-themes"
export function ThemeSwitcher() {
  const { theme, setTheme } = useTheme()
  return <ToggleGroup value={theme} onValueChange={setTheme}>
    <ToggleGroupItem value="light">浅色</ToggleGroupItem>
    <ToggleGroupItem value="dark">深色</ToggleGroupItem>
    <ToggleGroupItem value="system">跟随系统</ToggleGroupItem>
  </ToggleGroup>
}
```

### 验收
- [ ] 浅色/深色/跟随系统 三态切换无闪烁
- [ ] 4 套皮肤 × 3 主题 = 12 种组合全部生效
- [ ] 切换器放置在 Topbar 右侧

---

## #4 guardian-gateway 优化版 (优先交付)

**Agent ID**: G1 · **优先级**: **P0 优先** · **复杂度**: 高 (~2500 行)

### 任务
对当前 guardian-gateway 系统进行深度优化, 作为独立方案优先生效:
- Dashboard 增强 (KPI 大数字 + 趋势 sparkline)
- 治理 6 域视觉升级 (资金/模型/审计/ABAC/PBAC/系统设置)
- DataTable 企业级 (粘性表头 + 多选 + 列设置)
- 加载/错误/空状态三态完善
- ⌘K 全局搜索

### 验收
- [ ] Nielsen 10 启发式 ≥ 90/100
- [ ] 业务面 0 变更 (E1 8 步回归全绿)
- [ ] 当前生产用户无感知 (兼容现有数据/路由)

---

## #5 shadcn 组件库按 5 套皮肤扩变体

**Agent ID**: C1 · **优先级**: P1 · **复杂度**: 高 (~800 行)

### 任务
现有 shadcn 99 组件, 按 5 套皮肤扩 variant, 重点:
- Button (primary/secondary/ghost/danger/link/success)
- Card (elevated/outlined/flat)
- Badge (default/success/warning/danger/info)
- Input/Select/Textarea (4 状态: default/focus/error/disabled)
- Table (粘性表头 + 斑马纹 + 空状态)
- Dialog/Sheet (政务严谨 / 云端信息密度 / 银行保守)

### 验收
- [ ] 99 组件全部支持 5 套皮肤变量
- [ ] 视觉回归 5×99 = 495 张组件截图 (抽样 50 张)
- [ ] 与 [§3.1 颜色 Token](file:///d:/ai-work/grok/a-gov/docs/superpowers/specs/2026-08-03-ux-redesign-design.md) 100% 对齐

---

## #6 i18n · next-intl · 双语 + 术语库

**Agent ID**: I1 · **优先级**: P1 · **复杂度**: 中 (~600 行)

### 任务
- 接入 `next-intl` (已在 `package.json`)
- 创建 `src/messages/zh.json` + `en.json`
- 路由前缀 `/[locale]` (zh / en)
- 术语库: 政务/金融/技术 三类, 共 ~200 条
- 切换器放置在 Topbar 右侧 (主题切换器旁)

### 必做词条
nav (7 项) / kpi (4 项) / error (10 项) / 表单 (15 项) / 治理 6 域 (60 项) / 通用按钮 (8 项) ≈ 200 条

### 验收
- [ ] 中英切换无遗漏 (无硬编码英文)
- [ ] 政务术语: 资金/划拨/清算/对账 准确
- [ ] 金融术语: Token/Key/账户/预算帽 准确
- [ ] 词条覆盖率审计 ≥ 95%

---

## #7 移动端响应式 (1280/1440/1920 三档)

**Agent ID**: R1 · **优先级**: P0 · **复杂度**: 中 (~500 行)

### 任务
使用 TailwindCSS v4 breakpoints + Container Queries 实现三档适配:
- **1280px** (政企办公主流)
- **1440px** (默认, 现状审计)
- **1920px** (宽屏, 监控大屏)

### 关键调整
- Sidebar: 1280 折叠图标, 1440 完整, 1920 完整+展开
- Topbar: 1280 隐藏次要入口, 1920 全部展开
- DataTable: 1280 隐藏次要列, 1920 全部列
- Dashboard KPI: 1280 2 列, 1440 4 列, 1920 4 列+大数字
- 治理 6 域: 矩阵表格自适应列宽

### 验收
- [ ] 三档断点切换无错位
- [ ] 无横向滚动条
- [ ] 关键信息优先 (1280 不丢失功能)
- [ ] 与 [README §6 必做总览 #7](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/README.md#L130) 一致

---

## #8 视觉回归 + a11y 审计

**Agent ID**: Q1 · **优先级**: P1 · **复杂度**: 中 (~300 行)

### 任务
- Playwright 截图: 5 皮肤 × 3 主题 × 2 语言 = 30 张
- 覆盖页面: Dashboard / Sidebar / 资金 / 模型 / 审计 / ABAC = 6 页
- 总计: 30 × 6 = 180 张截图
- a11y 审计: axe-core, WCAG 2.1 AA

### 验收
- [ ] 180 张截图无视觉错位
- [ ] WCAG AA: 对比度 ≥ 4.5:1 (正文) / ≥ 3:1 (大字)
- [ ] Nielsen 10 启发式 ≥ 90/100
- [ ] 0 严重 a11y 问题

---

## §7.6 质量门禁 (W2)

- [x] 8 张任务卡片 (本文件)
- [ ] 8 Agent 全部完成
- [ ] 编译 `bun run build` 成功
- [ ] TS 类型检查 0 错
- [ ] E1 8 步回归全绿 (业务面不动)
- [ ] 视觉回归 180 张截图
- [ ] a11y WCAG AA 通过
- [ ] 改动 ≤ 5000 行净增 (红线)

---

## 报告元数据

- 任务卡片版本: v1 (2026-08-03)
- 总任务数: 8 张 (7 必做 + 1 质量门禁)
- 启动模式: 用户明确指令 "启动 W2" / "启动 UX 实施"
- 蜂群模式: 8 Agent 并行 (无依赖任务并行, 有依赖任务串行)
- 推送归属: W2 产出本地保留, W3 完成后统一推 `tt-52101/ai-gov`
