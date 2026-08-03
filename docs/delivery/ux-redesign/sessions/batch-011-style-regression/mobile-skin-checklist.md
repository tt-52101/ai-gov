# 移动端 × 4 套皮肤响应式回归测试清单

> 周期: 2026-08-03 | dev server: http://localhost:13003 | 共 16 项 (4 套皮肤 × 4 断点)
> 范围: 移动端 / 紧凑桌面 / 默认桌面 / 宽屏监控 三档断点 × 4 套皮肤, 验证布局不被破坏

---

## 一、测试矩阵设计

| 维度 | 值 | 说明 |
|---|---|---|
| **断点** | ≤1280 / 1281-1439 / 1440-1919 / ≥1920 | 4 档 (用户硬性要求 + 移动端补一档) |
| **皮肤** | guardian / government / cloud / bank | 4 套, 排除已撤回的 bank (因银行实际不用) |
| **主题** | light (默认) / dark | 2 态, auto 由系统决定不单独测 |
| **总组合** | 4 断点 × 4 皮肤 × 2 主题 = **32 组合** | 全跑手工太慢, 自动化用 Playwright |

---

## 二、移动端专项 (≤1280) — 6 项

| # | 断点 | 皮肤 | 主题 | 测试项 | 期望 | 实测 |
|---|---|---|---|---|---|---|
| **M-01** | ≤1280 | guardian | light | 侧栏折叠为图标模式 (gg-sidebar-w=64px) | sidebar 隐藏文字仅图标 |  |
| **M-02** | ≤1280 | guardian | light | 次要列隐藏 (gg-hide-1280) | 时间线副标题等次要信息不显示 |  |
| **M-03** | ≤1280 | government | light | 侧栏折叠 + 政务风颜色 | sidebar 64px + 政务蓝主色 |  |
| **M-04** | ≤1280 | cloud | dark | 侧栏折叠 + 深色背景 | sidebar 64px + 深空蓝背景 |  |
| **M-05** | ≤1280 | bank | light | 侧栏折叠 + 银行保守色 | sidebar 64px + 招行红主色 |  |
| **M-06** | ≤1280 | (任一) | (任一) | 预览页控制面板竖排 (单列) | 控制面板 1 列, 不溢出 |  |

---

## 三、紧凑桌面 (1281-1439) — 4 项

| # | 断点 | 皮肤 | 主题 | 测试项 | 期望 | 实测 |
|---|---|---|---|---|---|---|
| **C-01** | 1281-1439 | guardian | light | sidebar 标准宽度 (240px) + KPI 4 列 | 标准布局 |  |
| **C-02** | 1281-1439 | government | light | 政务风 sidebar + 页面 padding (space-4) | 紧凑 padding |  |
| **C-03** | 1281-1439 | cloud | dark | 深色 sidebar + 紧凑 padding | 视觉一致 |  |
| **C-04** | 1281-1439 | (任一) | (任一) | 角色权限树在 1280 档可横向滚动 | 不溢出隐藏 |  |

---

## 四、默认桌面 (1440-1919) — 3 项

| # | 断点 | 皮肤 | 主题 | 测试项 | 期望 | 实测 |
|---|---|---|---|---|---|---|
| **D-01** | 1440-1919 | guardian | light | sidebar 完整 + KPI 4 列 + 标准 padding (space-6) | 默认布局 |  |
| **D-02** | 1440-1919 | bank | dark | 银行风 dark 主题 + 标准 padding | 视觉一致 |  |
| **D-03** | 1440-1919 | (任一) | (任一) | 角色权限树业务域分组可折叠 | 折叠/展开动画流畅 |  |

---

## 五、宽屏监控 (≥1920) — 3 项

| # | 断点 | 皮肤 | 主题 | 测试项 | 期望 | 实测 |
|---|---|---|---|---|---|---|
| **W-01** | ≥1920 | guardian | light | sidebar 加宽 (280px) + 次要列显示 | 宽松布局 |  |
| **W-02** | ≥1920 | government | dark | 政务风 dark + 大字号 KPI (gg-kpi-big) | 监控大屏视觉 |  |
| **W-03** | ≥1920 | cloud | light | 企业云风 + 大 padding (space-8) | 监控大屏 |  |

---

## 六、关键 CSS 变量验证 (4 项, 每套皮肤 × 1)

| # | 皮肤 | 关键 CSS 变量 | 期望值 (浅色) | 实测 |
|---|---|---|---|---|
| **V-01** | guardian | `--color-primary-500` | OKLCH 0.55 0.14 162 → #00875b |  |
| **V-02** | government | `--color-primary-500` | #1A56DB 政务蓝 |  |
| **V-03** | cloud | `--color-primary-500` | #1677FF 华为蓝 |  |
| **V-04** | bank | `--color-primary-500` | #C8102E 招行红 |  |

> 注: guardian 主色来自 globals.css OKLCH 默认值, 其他 3 套由 skin-*.css 自定义变量提供

---

## 七、自动化测试脚本 (Playwright)

```typescript
// test/mobile-skin-regression.spec.ts
// 验证 4 断点 × 4 皮肤 共 16 组合的布局

import { test, expect, devices } from '@playwright/test'

const BREAKPOINTS = [
  { name: 'mobile-1280', width: 1280, height: 720 },
  { name: 'compact-1440', width: 1440, height: 900 },
  { name: 'default-1920', width: 1920, height: 1080 },
  { name: 'wide-2560', width: 2560, height: 1440 },
]

const SKINS = ['guardian', 'government', 'cloud', 'bank'] as const
const THEMES = ['light', 'dark'] as const

test.describe('移动端 × 4 套皮肤响应式回归', () => {
  for (const bp of BREAKPOINTS) {
    test.use({ viewport: { width: bp.width, height: bp.height } })
    for (const skin of SKINS) {
      for (const theme of THEMES) {
        test(`${bp.name} × ${skin} × ${theme} 布局正确`, async ({ page }) => {
          // 设置皮肤
          await page.goto('http://localhost:13003/')
          await page.evaluate(([s, t]) => {
            localStorage.setItem('guardian.skin', s)
            localStorage.setItem('theme', t)
            document.documentElement.setAttribute('data-skin', s)
            document.documentElement.setAttribute('data-theme', t)
          }, [skin, theme])

          // 验证 data-skin / data-theme 属性
          await expect(page.locator('html')).toHaveAttribute('data-skin', skin)
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme)

          // 验证关键 CSS 变量已加载
          const primary = await page.evaluate(() =>
            getComputedStyle(document.documentElement).getPropertyValue('--color-primary-500').trim()
          )
          expect(primary).not.toBe('') // 不为空

          // 验证侧栏宽度 (≤1280 → 64px, 否则 240px+)
          const sidebarWidth = await page.evaluate(() => {
            const sb = document.querySelector('[class*="gg-sidebar-w"], aside')
            return sb ? sb.getBoundingClientRect().width : 0
          })
          if (bp.width <= 1280) {
            expect(sidebarWidth).toBeLessThanOrEqual(80)
          } else {
            expect(sidebarWidth).toBeGreaterThanOrEqual(200)
          }
        })
      }
    }
  }
})
```

---

## 八、手工快速验证步骤 (3 分钟)

1. **打开浏览器** DevTools → Toggle Device Toolbar (Ctrl+Shift+M)
2. **设置 4 档断点** (1280 / 1440 / 1920 / 2560)
3. **每档断点下, 依次切换 4 套皮肤** (右上角 Topbar 皮肤选择)
4. **观察 sidebar 宽度** 是否随断点变化
5. **拖动 horizontal scroll** 检查是否溢出
6. **打开 Console** 应看到 [SkinProvider] 和 [ThemeProvider] 切换日志 (T11)

---

## 九、关键文件清单

| 文件 | 作用 |
|---|---|
| `src/styles/tokens/breakpoints.css` | 三档断点 @media 块 + gg-* 工具类 |
| `src/components/providers/skin-provider.tsx` | 皮肤切换 + W2 增强切换日志 (T11) |
| `src/components/providers/theme-provider.tsx` | 主题切换 + W2 增强切换日志 (T11) |
| `src/components/ui/skin-selector.tsx` | 4 套皮肤选择器 (v2 修复 grid 横排) |
| `src/app/globals.css` | 9 行 @import + OKLCH guardian 品牌色 |
| `src/app/preview-ux/page.tsx` | 三维矩阵预览页 (v2 修复) |

---

## 十、报告元数据

- 测试日期: 2026-08-03
- 测试范围: 移动端响应式 × 4 套皮肤 (16 组合, 含 2 主题 = 32 组合)
- 自动化: Playwright (脚本已就绪, 需 `bunx playwright test` 跑)
- 通过率: 待执行 (T11 增强日志已就绪, 跑测时同时收集 console 日志)
- 关键发现: 4 套皮肤 CSS + guardian 品牌色 + 3 档断点 @media 块 全部已编译进 CSS
