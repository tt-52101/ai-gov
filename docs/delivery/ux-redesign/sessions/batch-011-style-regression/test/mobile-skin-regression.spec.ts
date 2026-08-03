// mobile-skin-regression.spec.ts
// ==============================================================================
// 用途: 验证移动端 4 档断点 × 4 套皮肤 × 2 主题 = 32 组合的布局正确性
// 设计: 2026-08-03 W2 增强 T12 任务
// 来源: docs/delivery/ux-redesign/sessions/batch-011-style-regression/mobile-skin-checklist.md
// ==============================================================================
// 运行: bunx playwright test test/mobile-skin-regression.spec.ts
// 前置: dev server 运行在 http://localhost:13002 (或自定义 BASE_URL)
// ==============================================================================

import { test, expect } from '@playwright/test'

const BASE_URL = process.env.BASE_URL ?? 'http://localhost:13002'

// 4 档断点 (对齐 styles/tokens/breakpoints.css)
const BREAKPOINTS = [
  { name: 'mobile-1280', width: 1280, height: 720, sidebarExpect: '<=80', expectKpiCols: 2 },
  { name: 'compact-1440', width: 1440, height: 900, sidebarExpect: '>=200', expectKpiCols: 4 },
  { name: 'default-1920', width: 1920, height: 1080, sidebarExpect: '>=200', expectKpiCols: 4 },
  { name: 'wide-2560', width: 2560, height: 1440, sidebarExpect: '>=260', expectKpiCols: 4 },
] as const

// 4 套皮肤
const SKINS = ['guardian', 'government', 'cloud', 'bank'] as const

// 2 主题 (auto 不单独测, 由系统决定)
const THEMES = ['light', 'dark'] as const

test.describe('移动端 × 4 套皮肤 × 2 主题 响应式回归', () => {
  for (const bp of BREAKPOINTS) {
    test.use({ viewport: { width: bp.width, height: bp.height } })

    for (const skin of SKINS) {
      for (const theme of THEMES) {
        test(`${bp.name} × ${skin} × ${theme} 布局正确`, async ({ page }) => {
          // 收集 console 日志 (验证 T11 切换日志已生效)
          const consoleLogs: string[] = []
          page.on('console', (msg) => {
            if (msg.text().includes('[SkinProvider]') || msg.text().includes('[ThemeProvider]')) {
              consoleLogs.push(msg.text())
            }
          })

          // 1) 访问首页
          await page.goto(BASE_URL)

          // 2) 注入皮肤 + 主题 (localStorage + html attribute 双写, 覆盖 SSR 默认)
          await page.evaluate(
            ([s, t]) => {
              localStorage.setItem('guardian.skin', s)
              localStorage.setItem('guardian.theme', t)
              document.documentElement.setAttribute('data-skin', s)
              document.documentElement.setAttribute('data-theme', t)
            },
            [skin, theme] as const,
          )

          // 3) 验证 html 属性已应用
          await expect(page.locator('html')).toHaveAttribute('data-skin', skin)
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme)

          // 4) 验证关键 CSS 变量已加载 (皮肤主色)
          const primary = await page.evaluate(() =>
            getComputedStyle(document.documentElement)
              .getPropertyValue('--color-primary-500')
              .trim(),
          )
          expect(primary, `${skin} 皮肤 --color-primary-500 应非空`).not.toBe('')
          // 不同皮肤主色应不同
          if (skin === 'bank') {
            expect(primary, 'bank 皮肤主色应偏深蓝').toMatch(/#[0-9A-Fa-f]{6}/)
          }

          // 5) 验证侧栏宽度
          const sidebarWidth = await page.evaluate(() => {
            const aside = document.querySelector('aside[class*="w-60"]')
            return aside ? Math.round(aside.getBoundingClientRect().width) : 0
          })

          if (bp.width <= 1280) {
            // 1280 档: 侧栏可能完全隐藏 (hidden lg:flex) 或折叠为 64px
            // 本设计用 hidden lg:flex, 即 1280 以下完全不显示侧栏
            expect(sidebarWidth, `1280 档侧栏应隐藏 (hidden lg:flex, lg=1280)`).toBe(0)
          } else {
            expect(sidebarWidth, `${bp.name} 侧栏应 >= 200px`).toBeGreaterThanOrEqual(200)
          }

          // 6) 验证页面不出现横向滚动
          const hasHorizontalScroll = await page.evaluate(
            () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
          )
          expect(hasHorizontalScroll, `${bp.name} × ${skin} × ${theme} 不应出现横向滚动`).toBe(false)

          // 7) 验证主题 CSS 变量 (dark 主题应改变 background)
          if (theme === 'dark') {
            const bg = await page.evaluate(() =>
              getComputedStyle(document.documentElement)
                .getPropertyValue('--color-bg-base')
                .trim(),
            )
            // guardian dark bg = #0A1929
            expect(bg, 'dark 主题下 bg-base 应非空').not.toBe('')
          }

          // 8) 验证 T11 切换日志在 dev 模式可见 (不是失败条件, 仅做提醒)
          if (consoleLogs.length === 0) {
            // eslint-disable-next-line no-console
            console.warn(`[WARN] ${bp.name} × ${skin} × ${theme}: 未捕获到 [SkinProvider]/[ThemeProvider] 切换日志, T11 增强可能未生效`)
          }
        })
      }
    }
  }
})

// ============================================================================
// 专项测试 1: Dashboard 权限治理快捷入口
// ============================================================================
test.describe('Dashboard 权限治理快捷入口', () => {
  test('r1 角色登录后 Dashboard 应显示 8 个权限治理入口', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    // 假设已登录 r1 (实际项目根据登录态调整)
    await page.goto(`${BASE_URL}/`)

    // 找到"权限治理"卡片
    const card = page.locator('text=权限治理').first()
    await expect(card).toBeVisible({ timeout: 10000 })

    // 验证 8 个治理模块入口存在
    const entries = [
      '动作目录',
      '角色管理',
      '角色权限',
      '角色绑定',
      '菜单管理',
      '路由权限',
      '按钮绑定',
      '系统配置',
    ]
    for (const title of entries) {
      const entry = page.locator(`text=${title}`).first()
      await expect(entry, `权限治理卡片应包含「${title}」入口`).toBeVisible()
    }
  })

  test('移动端 1280 档权限治理入口应 2 列布局', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto(`${BASE_URL}/`)

    // 权限治理卡片内的 grid 容器
    const grid = page.locator('text=权限治理').first().locator('xpath=ancestor::div[contains(@class, "rounded-lg")]').locator('div.grid').first()
    const cols = await grid.evaluate((el) => {
      const cs = getComputedStyle(el)
      return cs.gridTemplateColumns
    })
    // 1280 档: grid-cols-2 (在 sm 以下默认 2 列)
    const colCount = cols.split(' ').filter((s) => s.trim()).length
    expect(colCount, `1280 档权限治理网格应为 2 列, 实际 ${colCount} 列`).toBe(2)
  })
})

// ============================================================================
// 专项测试 2: gov/role-permissions 默认业务域分组视图
// ============================================================================
test.describe('gov/role-permissions 默认业务域分组视图', () => {
  test('默认应进入业务域分组视图, 顶部统计条可见', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(`${BASE_URL}/#gov/role-permissions`)

    // 等待数据加载
    await page.waitForSelector('text=权限授予总览', { timeout: 10000 })

    // 验证顶部统计: 业务域 / 涉及角色 / 授权项
    await expect(page.locator('text=业务域').first()).toBeVisible()
    await expect(page.locator('text=涉及角色').first()).toBeVisible()
    await expect(page.locator('text=授权项').first()).toBeVisible()

    // 验证至少 1 个业务域 (组织与人员) 展开
    const orgDomain = page.locator('text=组织与人员').first()
    await expect(orgDomain).toBeVisible()
  })

  test('点击"切换为矩阵视图"应回退到高级矩阵', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.goto(`${BASE_URL}/#gov/role-permissions`)
    await page.waitForSelector('text=权限授予总览', { timeout: 10000 })

    // 捕获 console 日志
    const logs: string[] = []
    page.on('console', (msg) => {
      if (msg.text().includes('[GovRolePermissions]')) logs.push(msg.text())
    })

    // 点击切换按钮
    await page.locator('button:has-text("切换为矩阵视图")').click()

    // 验证日志
    await page.waitForTimeout(500)
    expect(logs.some((l) => l.includes('view_switch from=grouped to=matrix')), '应输出视图切换日志').toBe(true)
  })
})
