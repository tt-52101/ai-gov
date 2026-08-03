# 蜂群 Batch-008 · W2/W3 后续规划 (轻量版)

> ⚠️ **本规划为后续手动启动 W2/W3 提供路线图, 不立即执行**
> ⚠️ **用户硬性要求: 以下全部属于必做范围, 一项不能少**
> 4 套方案 + guardian-gateway 优化版 + 主题 (light/dark/auto) + 多语言体系 + 4 套皮肤系统 + 主题切换器

---

## 0. 战略级变更概要 (全部必做)

| 维度 | 原计划 | 用户决策 | 影响 | 状态 |
|------|-------|---------|------|------|
| 风格 | 三方融合 60/25/15 (P1 推荐) | **4 套全部保留 + guardian-gateway 优化版** | 设计 Token 必须支持多皮肤 | ✅ **必做** |
| 主题 | 单 light (默认) | **light/dark/auto 三态** | CSS Variables 双层 + next-themes | ✅ **必做** |
| i18n | YAGNI 排除 | **必做: 中英双语 + 术语库** | next-intl 完整接入 + JSON 词条 | ✅ **必做** |
| 推送 | 待定 | **本地保留, W2/W3 一起推** | 推 tt-52101/ai-gov 父级 | ✅ **必做策略** |
| 优先 | 无 | **guardian-gateway 优化版优先交付** | 当前系统深度优化 | ✅ **必做优先** |
| 4 套皮肤系统 | 无 | **4 套方案作为可动态换肤** | CSS Variables + 切换器 + localStorage | ✅ **必做** |
| 主题切换器 | YAGNI | **必做** (light/dark/auto 切换) | next-themes + UI 组件 | ✅ **必做** |

---

## 1. W2 规划: Design System + 主题 + i18n 落地

### 1.1 蜂群矩阵 (建议 5 Agent · 4-5h)

| Agent | 角色 | 任务 | 产出 | 复杂度 |
|---|---|---|---|---|
| **D1** | Design System 架构师 | 4 套皮肤 CSS Variables + light/dark/auto Token | `src/styles/tokens/*.css` (4 套) + `tailwind.config.ts` 扩展 | 高 |
| **D2** | 主题切换器工程师 | next-themes 接入 + ThemeProvider + 切换器组件 | `src/components/theme-provider.tsx` + `theme-toggle.tsx` | 中 |
| **D3** | i18n 工程师 | next-intl 完整接入 + 4 套语言 (中/英) + 术语库 | `src/messages/{zh,en}.json` + 路由 /[locale] | 中 |
| **D4** | 组件库升级 | shadcn 组件按 4 套主题扩变体 + 演示页面 | `src/components/ui/*.tsx` (49 组件) | 高 |
| **D5** | 质量门禁 | E2E 视觉回归 + a11y + 多语言截图对比 | `qa-tmp/visual-*.ts` + 4 套截图 | 中 |

### 1.2 Design Token 架构 (D1 关键产出)

```css
/* src/styles/tokens/base.css — 基础 Token (4 套共用) */
:root {
  --space-1: 4px; --space-2: 8px; /* ... */
  --font-size-xs: 12px; /* ... */
  --radius-sm: 4px; /* ... */
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05); /* ... */
}

/* src/styles/tokens/skin-government.css — 政务风 */
[data-skin="government"] {
  --color-primary-500: #1A56DB; /* 政务蓝 */
  --color-link: #0066FF;
  --color-bg-base: #F7F8FA;
}

/* src/styles/tokens/skin-cloud.css — 企业云风 */
[data-skin="cloud"] {
  --color-primary-500: #0066FF; /* 华为青 */
  --color-link: #0066FF;
  --color-bg-base: #F0F4F8;
}

/* src/styles/tokens/skin-bank.css — 银行风 */
[data-skin="bank"] {
  --color-primary-500: #1A365D; /* 银行深蓝 */
  --color-danger: #E60012; /* 招行红 */
  --color-bg-base: #F5F6F8;
}

/* src/styles/tokens/skin-guardian.css — guardian-gateway 优化版 (优先) */
[data-skin="guardian"] {
  --color-primary-500: #0066FF; /* 优化主色 */
  --color-link: #0066FF;
  --color-bg-base: #F7F8FA;
  /* 综合 4 套优点 */
}

/* src/styles/tokens/theme-dark.css — dark 主题 (4 套皮肤共用) */
[data-theme="dark"] {
  --color-bg-base: #0A1929; /* 深空蓝 */
  --color-bg-elevated: #132F4C;
  --color-text-primary: #E6F1FF;
}
```

### 1.3 主题切换器 (D2 关键)

```tsx
// src/components/theme-switcher.tsx
"use client"
import { useTheme } from "next-themes"
import { useSkin } from "@/lib/skin-context"

export function ThemeSwitcher() {
  const { theme, setTheme } = useTheme()
  const { skin, setSkin } = useSkin()
  
  return (
    <div className="flex items-center gap-2">
      <select value={theme} onChange={(e) => setTheme(e.target.value)}>
        <option value="light">浅色</option>
        <option value="dark">深色</option>
        <option value="system">跟随系统</option>
      </select>
      <select value={skin} onChange={(e) => setSkin(e.target.value)}>
        <option value="guardian">Guardian 优化版</option>
        <option value="government">政务风</option>
        <option value="cloud">企业云风</option>
        <option value="bank">银行风</option>
      </select>
    </div>
  )
}
```

### 1.4 i18n 词条 (D3 关键)

```json
// src/messages/zh.json
{
  "nav": {
    "dashboard": "仪表盘",
    "organization": "组织管理",
    "fund": "资金管理",
    "apiKey": "API Key",
    "model": "模型管理",
    "governance": "治理",
    "settings": "系统设置"
  },
  "kpi": {
    "totalCalls": "总调用量",
    "totalCost": "总成本",
    "activeKeys": "活跃 Key",
    "onlineModels": "在线模型"
  },
  "error": {
    "INSUFFICIENT_BALANCE": "可用余额不足",
    "MODEL_ACCESS_DENIED": "模型访问被拒绝",
    "BUDGET_CAP_EXCEEDED": "预算帽超限"
  }
}

// src/messages/en.json
{
  "nav": {
    "dashboard": "Dashboard",
    "organization": "Organization",
    ...
  }
}
```

---

## 2. W3 规划: 业务页适配 + 验证

### 2.1 蜂群矩阵 (建议 4 Agent · 3h)

| Agent | 角色 | 任务 | 产出 | 复杂度 |
|---|---|---|---|---|
| **B1** | 重点页适配 | Dashboard + Sidebar + Topbar 适配 4 套皮肤 + dark/light | `src/app/page.tsx` + `src/components/gg/app-sidebar.tsx` + `topbar.tsx` | 高 |
| **B2** | 治理 6 域适配 | 资金/模型/审计/ABAC/PBAC/系统设置 | 6 业务模块组件升级 | 高 |
| **X1** | 可用性测试 | Nielsen 10 启发式 + a11y 审计 | `qa-tmp/ux-audit.md` | 中 |
| **X2** | 视觉回归 | Playwright 多皮肤/多主题/多语言截图对比 | 4×3×2=24 张回归截图 | 中 |

### 2.2 业务页适配策略

| 页面 | 4 套皮肤行为 | dark/light | i18n |
|------|-------------|-----------|------|
| Dashboard | 4 套皮肤 KPI 卡 + 趋势图 | dark: 深空蓝 Hero | 中英 |
| Sidebar | 4 套皮肤左侧栏 | dark: 深色背景 | 中英 |
| 资金页 | guardian 用银行风, 其他 3 套保留 | dark: 深色余额 | 中英 |
| 模型页 | 企业云风优先, 其他 3 套 | dark: 深色卡片 | 中英 |
| 审计页 | 时间线 + 严重度色 (4 套统一) | dark: 深色时间线 | 中英 |
| ABAC | 政务风优先 + 矩阵 | dark: 深色矩阵 | 中英 |

---

## 3. 改动预算复核

| 阶段 | 预估净增 | 红线 |
|------|---------|------|
| W0 (设计规范) | +332 行 | ✅ 已完成 |
| W1 (3 套原型) | +5950 行 (仅原型 demo) | ✅ 已完成 |
| W2 (Design System + 主题 + i18n) | +1500~2000 行 | ⚠️ 接近红线 |
| W3 (业务页 + 验证) | +1200~1800 行 | ⚠️ 接近红线 |
| **W2+W3 合计** | **+2700~3800 行** | **⚠️ 需调整预算到 ≤ 5000 行** |

**调整建议**:
- 原 ≤ 2000 行红线不再适用, 调整为 **≤ 5000 行净增** (因新增 主题 + i18n + 4 套皮肤)
- 或分阶段交付: W2 (主题+i18n) 一波, W3 (业务页) 另一波

---

## 4. 验收门禁 (W2+W3)

- [ ] 4 套皮肤动态切换 (用户在 UI 可选, 立即生效)
- [ ] light/dark/auto 三态切换 (跟随系统)
- [ ] 中英双语完整 (无英文硬编码)
- [ ] guardian-gateway 优化版优先 + 4 套可选
- [ ] 业务页 (Dashboard + 治理 6 域) 全部适配
- [ ] 视觉回归 24 张截图 (4 皮肤 × 3 主题 × 2 语言)
- [ ] E1 8 步回归全绿 (业务面不动)
- [ ] UX 评分 ≥ 90/100 (Nielsen 10 启发式)
- [ ] 改动 ≤ 5000 行净增
- [ ] 0 新依赖 (next-themes + next-intl 已有)

---

## 5. 风险与依赖

| 风险 | 缓解 |
|------|------|
| 4 套皮肤视觉差异大, 维护成本高 | 用 CSS Variables 隔离, 业务代码不感知 |
| 主题切换时闪烁 | next-themes 自带 class strategy, 无闪烁 |
| i18n 词条遗漏 | 严格无硬编码 + E2E 视觉回归 + 词条覆盖率审计 |
| dark 模式 a11y 问题 | WCAG AA 对比度审计 (X1 任务) |
| 工作量超预算 | 分阶段交付, W2 必做, W3 可拆 |

---

## 6. 启动条件

W2/W3 启动需用户明确触发, 建议提供:
- 启动命令: "启动 W2" 或 "启动 W3"
- 或启动整个 W2+W3: "启动 UX 实施"
- 或分批启动: "先启动 D1+D2+D3 (主题+i18n), B1+B2 等皮肤稳定后再做"

---

## 报告元数据

- 规划版本: v1 (2026-08-03)
- 蜂群规模预估: W2 5 Agent + W3 4 Agent = 9 Agent
- 总耗时预估: 4-5h + 3h = 7-8h
- 总代码预估: +2700~3800 行
- 依赖: next-themes (已有) + next-intl (已有) + 0 新增
- 推送归属: 全部推到 `tt-52101/ai-gov` 父级 (与项目总仓一致)
