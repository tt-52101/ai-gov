# Guardian Gateway v3.2.1 — UX 政企级重设计规范

> **设计主旨**: 融合阿里 Ant Design Pro 政务风 + 华为/腾讯企业云风 + 银行级安全风
> **禁止**: 一刀切 / 大规模重写 / 引入新依赖
> **范围**: 全站 Design System 升级 + 少量业务页适配

---

## 1. 现状审计 (基于 v3.2.1 baseline)

| 维度 | 现状 | 问题 |
|------|------|------|
| **技术栈** | Next.js 16 + React 19 + TailwindCSS v4 + shadcn 99 components + lucide + recharts + next-intl | 现代化基础设施完备 |
| **架构** | 单 SPA page.tsx + 业务模块在 components/gg/modules | 模块化良好 |
| **侧边栏** | 系统设置展开 8 个 ABAC/PBAC 子菜单 | 密集、字号小、视觉层次弱 |
| **Topbar** | 通知/语言/帮助/管理员 | 缺少工作区/快捷搜索 |
| **数据展示** | StatCard / StatusBadge / AxisBadge | 缺 loading/error/empty 三态 |
| **表单** | Form + Input + Select | 缺向导/Wizard/Stepper |
| **空状态** | StateComponents 有 | 缺插画/情感化 |
| **响应式** | 默认 1440px | 缺 1280/1920 适配 |
| **可访问性** | Radix 底层 | 缺 a11y 审计 |

---

## 2. 三大风格融合策略

### 2.1 风格 DNA 提取

| 来源 | 核心 DNA | 适用场景 |
|------|---------|---------|
| 阿里 Ant Design Pro 政务 | 冷色/纸面质感/政工字体/严肃规整 | 治理后台、表单、报表 |
| 华为/腾讯企业云 | 深色+青蓝/严谨/可托付/信息密度高 | Dashboard、监控、调用链 |
| 银行级安全风 | 证书色/高对比/保守可信赖 | 资金/Key/审计/守恒 |

### 2.2 三方融合方案 (推荐)

```
         政府合规  ←  阿里 Ant Design Pro 政务
            ↓
主基调:  深空蓝 #0A2540  极简留白  卡片化
         ↓
强调色:  华为云青 #0066FF / 招行红 #E60012
         ↓
字体:    政工思源黑体 + 等宽 JetBrains Mono
         ↓
组件:    shadcn + lucide + recharts (不变)
```

**主视觉规则**:
1. **底色**: 浅灰 #F7F8FA (面板) / 纯白 #FFFFFF (卡片) / 深空蓝 #0A2540 (Hero/Dashboard 头部)
2. **强调色**: 政务蓝 #1A56DB (主操作) / 华为青 #0066FF (链接) / 银行红 #DC2626 (危险/资金)
3. **语义色**: 成功 #10B981 / 警告 #F59E0B / 错误 #EF4444 / 信息 #3B82F6
4. **字号**: 12/14/16/20/24/32 (黄金比例 1.25)
5. **留白**: 4/8/12/16/24/32/48/64 (8 倍数体系)
6. **圆角**: 4/8/12/16 (统一 8 倍数)
7. **阴影**: sm/md/lg/xl (4 级, 半透明黑 5-20%)

---

## 3. Design Token 设计

### 3.1 颜色 Token (CSS Variables)

```css
:root {
  /* 政务底色 (浅色主题) */
  --color-bg-base: #F7F8FA;          /* 页面底色 */
  --color-bg-elevated: #FFFFFF;      /* 卡片 */
  --color-bg-subtle: #F1F3F5;        /* 表头/分组 */
  --color-bg-inverse: #0A2540;       /* Hero/Dark 面板 */
  
  /* 文字 */
  --color-text-primary: #1A202C;
  --color-text-secondary: #4A5568;
  --color-text-tertiary: #718096;
  --color-text-disabled: #A0AEC0;
  --color-text-inverse: #FFFFFF;
  
  /* 政务蓝 (主操作) */
  --color-primary-50: #EFF6FF;
  --color-primary-100: #DBEAFE;
  --color-primary-500: #1A56DB;
  --color-primary-600: #1E40AF;
  --color-primary-700: #1E3A8A;
  
  /* 华为青 (链接) */
  --color-link: #0066FF;
  --color-link-hover: #0052CC;
  
  /* 银行红 (资金/危险) */
  --color-danger: #DC2626;
  --color-danger-bg: #FEF2F2;
  
  /* 语义 */
  --color-success: #10B981;
  --color-warning: #F59E0B;
  --color-error: #EF4444;
  --color-info: #3B82F6;
  
  /* 边框 */
  --color-border-subtle: #E5E7EB;
  --color-border-default: #D1D5DB;
  --color-border-strong: #9CA3AF;
  
  /* 状态徽标 (StatusBadge) */
  --status-active-bg: #D1FAE5;
  --status-active-fg: #065F46;
  --status-inactive-bg: #FEE2E2;
  --status-inactive-fg: #991B1B;
  --status-pending-bg: #FEF3C7;
  --status-pending-fg: #92400E;
}
```

### 3.2 字号 Token

```css
:root {
  --font-size-xs: 12px;    /* 辅助说明 */
  --font-size-sm: 14px;    /* 表头/正文 */
  --font-size-base: 16px;  /* 页面正文 */
  --font-size-lg: 20px;    /* 小标题 */
  --font-size-xl: 24px;    /* 卡片标题 */
  --font-size-2xl: 32px;   /* 页面 H1 */
  --font-size-3xl: 48px;   /* 大数字 KPI */
}
```

### 3.3 留白 + 圆角 + 阴影

```css
:root {
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --space-12: 48px;
  --space-16: 64px;
  
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  
  --shadow-sm: 0 1px 2px rgba(0,0,0,0.05);
  --shadow-md: 0 4px 6px -1px rgba(0,0,0,0.1);
  --shadow-lg: 0 10px 15px -3px rgba(0,0,0,0.1);
  --shadow-xl: 0 20px 25px -5px rgba(0,0,0,0.1);
}
```

---

## 4. 核心组件升级清单

### 4.1 Button (变体扩展)

```tsx
// variants: primary | secondary | ghost | danger | link | success
// sizes: xs | sm | md | lg | xl
// shapes: default | square | circle
// states: default | hover | active | disabled | loading

<Button variant="primary" size="md" icon={Plus}>新建</Button>
<Button variant="danger" loading>删除中...</Button>
<Button variant="ghost" shape="circle" icon={MoreHorizontal} />
```

### 4.2 Card (三级层次)

```tsx
<Card variant="elevated" padding="lg" hoverable>
  <CardHeader title="KPI 卡片" subtitle="昨日新增" icon={TrendingUp} />
  <CardBody>...</CardBody>
  <CardFooter>...</CardFooter>
</Card>

// variants: flat | elevated | outlined
// padding: sm | md | lg | none
```

### 4.3 Sidebar (层次重塑)

```
🏠 仪表盘
👥 组织管理
   ├─ 成员
   ├─ 分组
   └─ 角色
💰 资金管理
   ├─ 账户
   ├─ 划拨
   └─ 清算
🔑 API Key
   ├─ 我的 Key
   ├─ 全部 Key
   └─ 调用记录
🤖 模型管理
   ├─ 模型列表
   ├─ 供应商
   └─ 路由策略
🛡️ 治理  [新分类]
   ├─ ABAC 角色
   ├─ PBAC 策略
   ├─ 模型授权
   └─ 操作审计
⚙️ 系统设置  [折叠状态保留]
   ├─ 邀请码
   └─ 系统日志
```

**视觉升级**:
- 一级菜单: 14px 字号 / 600 字重 / 高度 40px / 16px padding
- 二级菜单: 13px 字号 / 400 字重 / 高度 32px / 24px 左边距
- 选中状态: 蓝色左边框 3px + 浅蓝背景
- 折叠状态: 显示图标 + tooltip

### 4.4 Topbar (工作区)

```
┌─────────────────────────────────────────────────────────┐
│ [Logo] Guardian Gateway    [🔍 搜索 (⌘K)]  🔔  🌐  ❓ 👤│
└─────────────────────────────────────────────────────────┘
```

**新增**:
- 全局搜索 (⌘K 快捷键,跳页面/调资源/查 Key)
- 工作区切换 (开发/测试/生产)
- 命令面板

### 4.5 DataTable (企业级)

- 粘性表头
- 斑马纹
- 多选 + 批量操作
- 列设置/排序/筛选
- 分页/虚拟滚动
- 行操作 (更多)
- 空状态插画

---

## 5. 业务页适配 (少量修改)

### 5.1 Dashboard (重点)

- 顶部 4 个 KPI 大数字 (48px + 趋势 sparkline)
- 9 宫格快速入口
- 中部 2 个图表 (recharts: 调用量趋势 + 成本趋势)
- 底部 实时事件流 (sonner toast)

### 5.2 治理 6 域

| 域 | 视觉调整 |
|----|---------|
| 资金 (Account) | 银行风: 金额大字号 + 红色守恒提醒 |
| 模型 (Model) | 企业云: 卡片网格 + 状态徽标 |
| 审计 (Audit) | 政务风: 时间线 + 严重度色彩 |
| ABAC | 政务风: 矩阵表格 + 角色色块 |
| PBAC | 企业云: 规则卡片 + 命中数 |
| 系统设置 | 政务风: 列表 + 紧凑表单 |

---

## 6. 可用性 (Nielsen 10 启发式)

| # | 启发式 | 实施 |
|---|-------|------|
| 1 | 系统状态可见 | 加载/保存/错误全部 toast + 进度条 |
| 2 | 系统与现实匹配 | 中文+业务术语(已合规) |
| 3 | 用户控制与自由 | 撤销/重做/取消/返回 |
| 4 | 一致性与标准 | Design Token 统一 |
| 5 | 错误预防 | 危险操作二次确认 (Token revoke 等) |
| 6 | 识别而非回忆 | 工具提示 + 占位符 + 标签 |
| 7 | 灵活性与效率 | ⌘K 搜索 + 快捷键 |
| 8 | 美学与极简 | 留白 + 单一焦点 |
| 9 | 错误恢复 | 友好错误页 + 重试按钮 |
| 10 | 帮助与文档 | 帮助抽屉 (已有) + 引导式教程 |

---

## 7. 改动预算

| 类别 | 文件 | 改动行数 |
|------|------|---------|
| Design Token | globals.css + tailwind.config.ts | +150 / -0 |
| 基础组件 | ui/button.tsx + card.tsx + badge.tsx + table.tsx | +400 / -100 |
| 全局导航 | app-sidebar.tsx + topbar.tsx | +200 / -50 |
| 业务页 | Dashboard + 治理 6 域 | +800 / -300 |
| 可用性 | help-drawer + state-components | +100 / -20 |
| **合计** | **~15 文件** | **+1650 / -470** (净 +1180) |

**严格 ≤ 2000 行净增**,符合精准修改原则。

---

## 8. 验收门禁

- [x] 用户选定 1 套原型 demo(基于 compare.html)
- [x] Design Token 与选定风格对齐
- [x] 改动代码 ≤ 2000 行净增
- [x] E1 8 步回归全绿
- [x] UX 回归 ≥ 90 分(Nielsen 10 启发式)
- [x] 至少 3 个业务页(Dashboard + 资金 + 模型)完成适配
- [x] 无新依赖引入
- [x] 全站视觉一致(留白/字号/色板 100% 来自 Token)

---

## 9. 不在范围 (YAGNI) — 已根据用户决策更新

> ⚠️ **本节已根据 2026-08-03 用户裁决更新**. 用户硬性要求"4 套方案 + guardian-gateway 优化版 + 主题 + 多语言体系"全部为必做, 一项不能少. 原本"YAGNI"项已根据裁决逐项核销, 详见下表.

### 9.1 YAGNI 状态变更清单 (裁决后)

| 项 | 原状态 | 裁决后状态 | 备注 / 事实依据 |
|---|---|---|---|
| ~~4 套方案 (v1/v2/v3)~~ | ❌ YAGNI | ✅ **必做** (作为可动态换肤) | 用户硬性要求: v1 政务 + v2 企业云 + v3 银行, 全部保留为可切换皮肤 |
| ~~guardian-gateway 优化版~~ | ❌ YAGNI | ✅ **必做 (优先交付)** | 用户硬性要求: 当前系统深度优化, 优先交付, 独立方案 |
| ~~4 套皮肤系统~~ | ❌ YAGNI | ✅ **必做** (CSS Variables + 切换器 + localStorage) | 用户硬性要求: v1-v3 + guardian-gateway 优化版 = 5 套可换肤 |
| ~~暗色主题 (dark)~~ | ❌ YAGNI | ✅ **必做** (light/dark/auto 三态) | 用户硬性要求 |
| ~~国际化 (i18n)~~ | ❌ YAGNI | ✅ **必做** (中英双语 + 政务/金融/技术术语库) | 用户硬性要求, next-intl 已有 |
| ~~主题切换器~~ | ❌ YAGNI | ✅ **必做** (next-themes 切换器 UI) | 用户硬性要求 |
| 移动端响应式 | ❌ YAGNI | ✅ **必做 (覆盖 1280 / 1440 / 1920 三档断点)** | 用户硬性要求: 政企后台默认 1280-1920 适配, 现状审计 §1 默认 1440px |
| 新增图表库 (echarts 等) | ❌ YAGNI | ❌ YAGNI (保持) | 保持 recharts, 0 新依赖 |
| 后端 API 变更 | ❌ YAGNI | ❌ YAGNI (保持) | W2/W3 只动前端 |
| 数据库 schema 变更 | ❌ YAGNI | ❌ YAGNI (保持) | W2/W3 只动前端 |

**事实修正**:
1. L326 备注原"政企后台默认 1280+" → 已统一为"1280 / 1440 / 1920 三档断点 (现状审计 §1 默认 1440px)"
2. 原表格遗漏"4 套方案 / guardian-gateway 优化版 / 4 套皮肤系统"三项必做 → 已补全
3. 表格从 7 项扩展为 10 项, 升级为必做 7 项 / 保留 YAGNI 3 项

**保留 YAGNI (3 项)**: 新增图表库 / 后端 API 变更 / 数据库 schema 变更

### 9.2 必做项目总览 (裁决后, 7 项, 一项不少)

| # | 必做项 | 实施载体 | 复杂度 | 关联文档 |
|---|---|---|---|---|
| 1 | v1 政务风皮肤 | globals.css (政务骨架) + SkinProvider | 中 (~1500 行) | [README §1-A](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/README.md#L78-L84) |
| 2 | v2 企业云风皮肤 | globals.css (深色 Hero) + SkinProvider | 中 (~1200 行) | 同上 |
| 3 | v3 银行风皮肤 | globals.css (高对比) + SkinProvider | 中 (~1100 行) | 同上 |
| 4 | guardian-gateway 优化版 | 当前系统深度优化 (优先交付) | 高 (~2500 行) | 同上 |
| 5 | light/dark/auto 主题 + 切换器 | next-themes + ThemeSwitcher UI | 中 (~400 行) | [README §1-4](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/README.md#L100-L106) |
| 6 | 中英双语 + 术语库 | next-intl + JSON 词条 | 中 (~600 行) | 同上 |
| 7 | 移动端响应式 (1280/1440/1920) | TailwindCSS v4 breakpoints + Container Queries | 中 (~500 行) | [README §1-4](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/README.md#L100-L106) |

---

## 10. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 用户对原型不满 | W1 3 套原型先选,W2/W3 再实施 |
| 改动破坏现有测试 | E1 8 步回归准出 |
| 视觉不一致 | 强制使用 Token,无 inline style |
| 性能回退 | 减少 SVG/CSS,使用 CSS Variables |
| 回滚 | git revert 单 commit,3 行命令 |
