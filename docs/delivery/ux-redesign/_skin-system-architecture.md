# 前端皮肤系统 · 组件结构树 + 目录规划

> **基线扫描结果 (2026-08-03)**:
> - `next-themes` 在 `src/components/ui/sonner.tsx` 中局部使用 1 次 (sonner toast 主题感知), **无全局 ThemeProvider**
> - `next-intl` **完全未接入** (0 文件匹配)
> - `data-theme` / `data-skin` 标记 **不存在**
> - shadcn 已就绪 dark 主题 (`.dark *` 变体在 globals.css), **但无切换入口**
> - 没有任何皮肤/主题/语言 Provider

**决策**: 不改造现有 (shadcn 默认机制仅支持单套 + 暗色), **重新创建基础框架** (5 套皮肤 × 3 主题 × 2 语言 三维矩阵).

---

## 1. 整体架构 (三维矩阵)

```
                  ┌─────────────────────────────────────┐
                  │  RootLayout (app/layout.tsx)        │
                  │  ┌──────────────────────────────┐  │
                  │  │  Providers 树 (嵌套)          │  │
                  │  │  ┌────────────────────────┐  │  │
                  │  │  │ NextIntlClientProvider │  │  │  ← i18n 最外
                  │  │  │  (locale=zh|en)         │  │  │
                  │  │  ├────────────────────────┤  │  │
                  │  │  │ SkinProvider           │  │  │  ← 5 套皮肤
                  │  │  │  (skin=guardian|...)    │  │  │
                  │  │  ├────────────────────────┤  │  │
                  │  │  │ ThemeProvider          │  │  │  ← 3 态主题
                  │  │  │  (theme=light|dark|     │  │  │
                  │  │  │   system via            │  │  │
                  │  │  │   next-themes)           │  │  │
                  │  │  └────────────────────────┘  │  │
                  │  │                                │  │
                  │  │  <html data-skin="guardian"   │  │
                  │  │   data-theme="dark"            │  │
                  │  │   lang="zh-CN">                │  │
                  │  └──────────────────────────────┘  │
                  └─────────────────────────────────────┘
```

**叠加顺序 (外→内)**: i18n → Skin → Theme → 业务代码  
**标记顺序 (html 上)**: `data-skin` (外层) → `data-theme` (内层) → CSS Variable 覆盖

---

## 2. 组件结构树

```
src/
├── app/
│   ├── layout.tsx                          [ROOT] 包裹 Providers
│   ├── globals.css                         [ROOT] Tailwind 入口
│   ├── [locale]/                            [i18n 路由前缀] (zh/en)
│   │   └── (authenticated)/
│   │       ├── dashboard/
│   │       ├── finance/
│   │       ├── model/
│   │       ├── audit/
│   │       └── ...
│   └── api/                                 (保留现有, 不变)
│
├── components/
│   ├── providers/                          [新增] Providers 目录
│   │   ├── index.tsx                        Providers 组合导出
│   │   ├── intl-provider.tsx                next-intl 客户端包装
│   │   ├── skin-provider.tsx                5 套皮肤 Context
│   │   ├── theme-provider.tsx               next-themes 包装
│   │   └── locale-switcher.tsx              中英切换器
│   │
│   ├── ui/                                  [扩展] shadcn 组件库
│   │   ├── button.tsx                       (扩 5 套 variant)
│   │   ├── card.tsx                         (扩 5 套 variant)
│   │   ├── badge.tsx                        (扩 5 套 variant)
│   │   ├── input.tsx                        (扩 5 套 variant)
│   │   ├── select.tsx                       (扩 5 套 variant)
│   │   ├── table.tsx                        (扩 5 套 variant)
│   │   ├── ...
│   │   ├── sonner.tsx                       (已有, 接 SkinProvider)
│   │   └── theme-switcher.tsx               [新增] 3 态切换器
│   │
│   └── gg/                                  [保留] 业务组件
│       ├── app-sidebar.tsx                  (适配 SkinProvider)
│       ├── topbar.tsx                       (新增 ThemeSwitcher + LocaleSwitcher)
│       └── modules/                         (业务模块, 适配 5 套皮肤)
│
├── styles/                                 [新增] 皮肤/主题/Token
│   ├── tokens/
│   │   ├── base.css                        [共用] 基础 Token
│   │   ├── skin-government.css             [v1 政务风]
│   │   ├── skin-cloud.css                  [v2 企业云风]
│   │   ├── skin-bank.css                   [v3 银行风]
│   │   ├── skin-guardian.css               [guardian 优化版, 优先]
│   │   ├── theme-light.css                 [light 主题, 默认]
│   │   ├── theme-dark.css                  [dark 主题]
│   │   └── breakpoints.css                 [1280/1440/1920 三档]
│   │
│   └── i18n/                                [新增] 词条与术语
│       ├── messages/
│       │   ├── zh.json                      [中文词条 ~200 条]
│       │   └── en.json                      [英文词条 ~200 条]
│       ├── glossary/
│       │   ├── governance.json              [政务术语 ~30 条]
│       │   ├── finance.json                 [金融术语 ~50 条]
│       │   └── technical.json               [技术术语 ~120 条]
│       └── config.ts                        [next-intl 配置]
│
├── lib/
│   ├── i18n/                                [新增] i18n 工具
│   │   ├── request.ts                       next-intl server config
│   │   └── utils.ts                         词条解析
│   │
│   └── skin/                                [新增] 皮肤工具
│       ├── types.ts                         Skin 枚举与类型
│       ├── storage.ts                       localStorage 持久化
│       └── resolver.ts                      URL/默认值解析
│
└── middleware.ts                            [新增] i18n 路由中间件
```

---

## 3. 关键组件 API 契约

### 3.1 SkinProvider

```tsx
// src/components/providers/skin-provider.tsx
"use client"
type Skin = 'guardian' | 'government' | 'cloud' | 'bank'
interface SkinContextValue {
  skin: Skin
  setSkin: (s: Skin) => void
  availableSkins: { id: Skin; name: string; description: string }[]
}
```

### 3.2 ThemeSwitcher

```tsx
// src/components/ui/theme-switcher.tsx
"use client"
import { useTheme } from "next-themes"
import { useSkin } from "@/components/providers/skin-provider"
// 双层叠加: theme 在内, skin 在外
```

### 3.3 LocaleSwitcher

```tsx
// src/components/providers/locale-switcher.tsx
"use client"
import { useLocale, useTranslations } from "next-intl"
import { useRouter, usePathname } from "next/navigation"
```

### 3.4 Providers 组合

```tsx
// src/components/providers/index.tsx
"use client"
import { NextIntlClientProvider } from "next-intl"
import { ThemeProvider as NextThemesProvider } from "next-themes"
import { SkinProvider } from "./skin-provider"

export function Providers({ children, locale, messages }) {
  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <SkinProvider defaultSkin="guardian">
        <NextThemesProvider
          attribute="data-theme"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          {children}
        </NextThemesProvider>
      </SkinProvider>
    </NextIntlClientProvider>
  )
}
```

---

## 4. 5 套皮肤 CSS 模板

### 4.1 目录约定

```
src/styles/tokens/
├── base.css              ← 共用 Token (留白/字号/圆角/阴影/字体)
├── skin-guardian.css     ← P0 优先, 当前系统深度优化
├── skin-government.css   ← v1 政务风
├── skin-cloud.css        ← v2 企业云风
├── skin-bank.css         ← v3 银行风
├── theme-light.css       ← light 主题 (默认)
├── theme-dark.css        ← dark 主题
└── breakpoints.css       ← 三档断点
```

### 4.2 加载顺序 (在 globals.css 顶部)

```css
@import "tailwindcss";
@import "./styles/tokens/base.css";
@import "./styles/tokens/skin-guardian.css";
@import "./styles/tokens/skin-government.css";
@import "./styles/tokens/skin-cloud.css";
@import "./styles/tokens/skin-bank.css";
@import "./styles/tokens/theme-light.css";
@import "./styles/tokens/theme-dark.css";
@import "./styles/tokens/breakpoints.css";
```

### 4.3 选择器叠加 (双层)

```css
/* 外层: 5 套皮肤 */
[data-skin="guardian"] { --color-primary-500: #0066FF; ... }
[data-skin="government"] { --color-primary-500: #1A56DB; ... }
[data-skin="cloud"] { --color-primary-500: #0066FF; ... }
[data-skin="bank"] { --color-primary-500: #1A365D; ... }

/* 内层: light/dark 主题 */
[data-theme="light"] { --color-bg-base: #F7F8FA; --color-text-primary: #1A202C; }
[data-theme="dark"]  { --color-bg-base: #0A1929; --color-text-primary: #E6F1FF; }
```

---

## 5. 三档断点 (移动端响应式)

```css
/* src/styles/tokens/breakpoints.css */
@theme {
  --breakpoint-sm: 1280px;   /* 政企办公主流 */
  --breakpoint-md: 1440px;   /* 默认 */
  --breakpoint-lg: 1920px;   /* 宽屏 */
  --breakpoint-xl: 2560px;   /* 监控大屏 (可选) */
}
```

**TailwindCSS v4 用法**:
```tsx
<div className="
  grid                            /* 1280: 1 列 */
  md:grid-cols-2                  /* 1440: 2 列 */
  lg:grid-cols-4                  /* 1920: 4 列 */
  gap-4 md:gap-6
">
  {kpis.map(k => <KpiCard {...k} />)}
</div>
```

---

## 6. 持久化策略

| 维度 | 存储 key | 默认值 | 说明 |
|------|---------|--------|------|
| skin | `guardian.skin` | `guardian` | 5 选 1, localStorage |
| theme | `guardian.theme` | `system` | 3 选 1, next-themes 自管 |
| locale | `NEXT_LOCALE` cookie | `zh` | 2 选 1, next-intl 中间件 |

**初始化顺序** (防闪烁):
1. middleware 检测 URL → 写入 locale cookie
2. RootLayout 读 cookie → 渲染 `<html lang="zh-CN" data-skin="guardian" data-theme="dark">`
3. ThemeProvider 客户端 hydrate → 接管 theme 切换
4. SkinProvider 客户端 hydrate → 接管 skin 切换
5. **关键**: 在 `<head>` 内嵌 inline script, 在 hydration 前同步 `data-skin` / `data-theme` (避免 FOUC)

---

## 7. 与现有代码的对接

### 7.1 不变更清单 (W2/W3 严禁触碰)

- `src/app/api/**` (后端 API)
- `prisma/**` (数据库)
- 现有 99 个 shadcn 组件的 props 接口 (向后兼容)
- 业务模块的 props 接口

### 7.2 仅扩展清单 (W2/W3 唯一改动)

| 文件 | 改动 |
|------|------|
| `src/app/layout.tsx` | 包裹 `<Providers>`, 注入 `data-skin` / `data-theme` |
| `src/app/globals.css` | 顶部 8 行 import, 内部 token 迁移到 `src/styles/tokens/*.css` |
| `src/components/topbar.tsx` | 新增 ThemeSwitcher + LocaleSwitcher |
| 99 shadcn 组件 | 仅替换硬编码颜色为 `var(--color-*)` |
| 业务模块 | 仅替换硬编码颜色为 `var(--color-*)` |

### 7.3 新增清单 (W2 全部新增)

- `src/components/providers/**` (4 文件)
- `src/components/ui/theme-switcher.tsx` (1 文件)
- `src/styles/tokens/**` (8 文件)
- `src/styles/i18n/**` (5 文件)
- `src/lib/skin/**` (3 文件)
- `src/lib/i18n/**` (2 文件)
- `src/middleware.ts` (1 文件)

**W2 新增文件总计**: 24 个文件  
**W2 修改文件总计**: 102 个 (layout/globals.css + 99 组件 + 1 topbar)  
**W2 净增行数**: 约 +5000 行 (与 [§3 预算红线](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/_W2-W3-plan.md#L184-L188) 对齐)

---

## 8. 验收门禁

- [ ] 5 套皮肤切换无视觉错位
- [ ] 3 态主题切换无闪烁
- [ ] 2 语言切换无遗漏 (无硬编码)
- [ ] 3 档断点 (1280/1440/1920) 自适应
- [ ] 0 新依赖 (next-themes/next-intl/tailwind 已有)
- [ ] 业务面 0 变更 (E1 8 步回归全绿)
- [ ] a11y WCAG AA 通过
- [ ] 总计 ≤ 5000 行净增

---

## 报告元数据

- 规划版本: v1 (2026-08-03)
- 维度: 5 皮肤 × 3 主题 × 2 语言 × 3 断点 = 90 组合
- 新增文件: 24 个
- 修改文件: 102 个
- 净增行数: ~5000 行
- 启动条件: 用户明确指令 "启动 W2" / "启动 UX 实施"
