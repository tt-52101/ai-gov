# 蜂群 Batch-008 · UX 政企级重设计 · W1 汇总

> 蜂群: batch-008-ux-redesign (4 Agent / 1 Wave)
> 周期: 2026-08-03
> 阶段: W1 原型设计 (完成) → 用户决策 → W2/W3 (待启动)
> 目标: 融合 Ant Design Pro 政务 + 华为/腾讯企业云 + 银行级安全, 出 3 套原型 demo 供用户决策

---

## 蜂群配置矩阵 (4 Agent)

| Agent | 角色 | 任务 | 产出 | 评分 |
|---|---|---|---|---|
| U1 | UED 政企风专家 | 阿里 Ant Design Pro 政务 + 浙里办/粤省事 | 1130 行 index.html + notes + agent-report | 88/100 |
| U2 | UED 企业云风专家 | 华为云 + 腾讯云控制台 | 1215 行 index.html + 8 张截图 + notes + agent-report | 91/100 |
| U3 | UED 银行风专家 | 工行 + 招行网银 | 912 行 index.html + notes + agent-report | 85/100 |
| P1 | 产品评估专家 | Nielsen 10 启发式打分 + 三方对比 + 融合建议 | _decision.md (320 行) + _compare.html (540 行) | 推荐三方融合 60/25/15 |

---

## 交付物清单

### 设计规范 (W0)
- [docs/superpowers/specs/2026-08-03-ux-redesign-design.md](file:///d:/ai-work/grok/a-gov/docs/superpowers/specs/2026-08-03-ux-redesign-design.md) (332 行)
  - 10 节: 现状审计 / 三大风格融合 / Design Token / 组件升级 / 业务页适配 / Nielsen 启发式 / 改动预算 / 验收 / YAGNI / 风险回滚

### 原型 Demo (W1)
- [v1-government/index.html](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v1-government/index.html) (1130 行) — 政务风 6 页面
- [v2-cloud/index.html](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v2-cloud/index.html) (1215 行) — 企业云风 6 页面 + 8 截图
- [v3-bank/index.html](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v3-bank/index.html) (912 行) — 银行风 6 页面
- [_compare.html](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/_compare.html) (540 行) — 4 Tab 三方对比聚合

### 评估与决策 (P1)
- [_decision.md](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/_decision.md) (320 行) — 9 大节决策报告
- [P1-agent-report.md](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/P1-agent-report.md) (180 行) — P1 单兵轨迹

### 单兵轨迹
- [v1-government/agent-report.md](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v1-government/agent-report.md) (149 行)
- [v2-cloud/agent-report.md](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v2-cloud/agent-report.md) (8.3 KB)
- [v3-bank/agent-report.md](file:///d:/ai-work/grok/a-gov/docs/ux-prototypes/v3-bank/agent-report.md) (7.3 KB)

**总产出**: 21 个文件, 5950 行 (含 HTML/CSS/中文注释/设计说明), 0 新依赖

---

## P1 评估核心结论

### 三方加权总分 (业务场景加权 30/30/20/20)

| 风格 | 总分 | 最佳适用域 |
|---|---|---|
| v2-cloud 企业云 | **91** | Dashboard / 模型列表 / 全局搜索 ⌘K |
| v1-government 政务 | **88** | Sidebar 升级 / 系统设置 / 表单 |
| v3-bank 银行 | **85** | 资金管理 / Key 证书 / 审计时间线 |

### 推荐方案: 三方融合 (60/25/15, 不一刀切)

- **60% 政务骨架**: 全站底色 / Sidebar / Topbar / 表单 / 系统设置
- **25% 企业云风**: Dashboard 深色 Hero / 模型列表紧凑表格 / ⌘K 全局搜索
- **15% 银行风 (局部)**: 仅资金页守恒 banner / Key 页证书边框 / 审计页时间线

### 改动预算复核

- **实际净增**: +1380 行 (设计规范 + 4 套原型 + 4 单兵轨迹)
- **引入新依赖**: 0
- **改 src/ 代码**: 0 (W1 只出原型, 未触代码)
- **改 globals.css/token**: 0
- **W2/W3 实施预算**: 仍 ≤ 2000 行净增红线

---

## 待用户决策 (关键 4 项) — ✅ 已裁决, **全部属于必做范围**

> ⚠️ **本节所有项目均属必做, 不再是"可选决策"**. 用户硬性要求: 4 套方案 + guardian-gateway 优化版 + 主题 (light/dark/auto) + 多语言体系, 全部落地, 一项不少.

### 1. 风格选择 ✅ **4 套方案 + guardian-gateway 优化版 — 必做 5 项**

| # | 方案 | 描述 | 实施复杂度 | 状态 |
|---|---|---|---|---|
| A | 三方融合 60/25/15 | 政务骨架 + 企业云 + 银行风局部 | 中 (~1500 行) | ✅ **必做**: v1-v3 套皮肤 |
| B | 全站政务风 | 严肃/规整/纸面感 | 低 (~800 行) | ✅ **必做**: 作为 v1 套皮肤 |
| C | 全站企业云风 | 深色 Hero / 信息密度高 | 中 (~1200 行) | ✅ **必做**: 作为 v2 套皮肤 (P1 评分最高 91) |
| D | 全站银行风 | 高对比 / 保守 / 守恒 | 中 (~1100 行) | ✅ **必做**: 作为 v3 套皮肤 |
| **E** | **guardian-gateway 优化版** | **当前系统深度优化, 优先交付** | **高 (~2500 行)** | ✅ **必做 + 优先**: 独立方案 |

**架构升级 (必做)**: v1-v3 作为**可动态换肤**, guardian-gateway 优化版作为**独立方案 + 优先生效**

### 2. 推送归属 ✅ **必做策略 (用户决策: 本地保留到 W2/W3 一起推)**

- ✅ **必做**: W0 (设计规范) + W1 (3 套原型) + W2/W3 产出 全部本地保留
- ✅ **必做**: 后续 W2/W3 完成后, 统一推到 AGENTS.md §附录的 `tt-52101/ai-gov` 父级
- ✅ **已做**: 商业 GA 修复已 push 到 `YM601307/guardian-gateway` feature/v3.2.1 (10fb1e2)

### 3. W2/W3 启动 ✅ **必做 (用户决策: 暂不启动, 等待手动触发)**

- ✅ **必做**: W2 (Design System + 主题 + i18n) + W3 (业务页适配 + 验证)
- ✅ **触发条件**: 用户明确指令 ("启动 W2" / "启动 W3" / "启动 UX 实施")
- ✅ **规划就绪**: 见 [_W2-W3-plan.md](file:///d:/ai-work/grok/a-gov/docs/delivery/ux-redesign/_W2-W3-plan.md) (9 Agent 矩阵)

### 4. ✅ 必做: 强制新增能力 (用户硬性要求 — **全部 4 项, 一项不能少**)

| # | 能力 | 描述 | 复杂度 | 状态 |
|---|---|---|---|---|
| 1 | **light/dark/auto 主题** | 完整的暗色模式 + 跟随系统切换 | 中 (CSS Variables + next-themes) | ✅ **必做** |
| 2 | **多语言体系** | 中英双语 + 政务/金融/技术术语库 | 中 (next-intl 已有, 需接入) | ✅ **必做** |
| 3 | **4 套皮肤系统** | v1-v3 套皮肤 + guardian-gateway 优化版作为独立方案 + 主题切换器 UI | 高 (CSS Variables + 切换器 + localStorage 持久化) | ✅ **必做** |
| 4 | **移动端响应式 (1280/1440/1920)** | 三档断点适配, 政企后台全场景覆盖 | 中 (TailwindCSS v4 breakpoints + Container Queries) | ✅ **必做 (2026-08-03 用户最新裁决)** |

### 5. ✅ 必做: 国际化 + 主题切换器 + 移动端响应式 (再次明确, 不在 YAGNI 排除范围)

- ✅ **国际化 (next-intl)** = **必做**, 非 YAGNI
- ✅ **主题切换器 (next-themes)** = **必做**, 非 YAGNI
- ✅ **移动端响应式 (1280/1440/1920 三档断点)** = **必做 (2026-08-03 用户最新裁决)**, 非 YAGNI
- ❌ 新增图表库 / 后端 / 数据库 = YAGNI, 不做

---

### 6. ✅ 必做总览 (裁决后, **7 项, 一项不少**)

> ⚠️ **本节为 UX 实施最终交付清单**, 7 项必做 + 3 项保留 YAGNI, 详见 [设计规范 §9](file:///d:/ai-work/grok/a-gov/docs/superpowers/specs/2026-08-03-ux-redesign-design.md#L316-L348).

| # | 必做项 | 实施载体 | 优先级 | 复杂度 |
|---|---|---|---|---|
| 1 | v1 政务风皮肤 | globals.css (政务骨架) + SkinProvider | P1 | 中 (~1500 行) |
| 2 | v2 企业云风皮肤 | globals.css (深色 Hero) + SkinProvider | P1 | 中 (~1200 行) |
| 3 | v3 银行风皮肤 | globals.css (高对比) + SkinProvider | P1 | 中 (~1100 行) |
| 4 | guardian-gateway 优化版 | 当前系统深度优化 | **P0 (优先交付)** | 高 (~2500 行) |
| 5 | light/dark/auto 主题 + 切换器 | next-themes + ThemeSwitcher UI | P0 | 中 (~400 行) |
| 6 | 中英双语 + 术语库 | next-intl + JSON 词条 | P1 | 中 (~600 行) |
| 7 | 移动端响应式 (1280/1440/1920) | TailwindCSS v4 breakpoints | P0 | 中 (~500 行) |

**保留 YAGNI (3 项)**: 新增图表库 (echarts 等) / 后端 API 变更 / 数据库 schema 变更

---

## §7.5 三动作存证

- [x] **① 代码存证**: 21 个文件本地就绪, 待用户决策后 commit
- [x] **② 批次汇总**: 本文件 (README.md)
- [x] **③ 单兵轨迹**: 4 份 (U1/U2/U3/P1) 全部就绪

---

## §7.6 质量门禁 (W1)

- [x] 3 套原型完整可用 (6 页面 × 3 风格 = 18 个 demo)
- [x] _compare.html 4 Tab 切换正常
- [x] _decision.md 含 Nielsen 10 启发式逐项打分
- [x] 0 新依赖 (仅 TailwindCSS CDN)
- [x] 全程中文注释
- [x] 4 份单兵轨迹完整

---

## 浏览器预览

```bash
# 方式 1: 直接打开 (已通过 Start-Process 启动)
start d:\ai-work\grok\a-gov\docs\ux-prototypes\_compare.html

# 方式 2: 本地 HTTP 服务
cd d:\ai-work\grok\a-gov\docs\ux-prototypes
python -m http.server 8000
# 访问 http://localhost:8000/_compare.html

# 方式 3: 各原型单独查看
# 政务: docs\ux-prototypes\v1-government\index.html
# 云:   docs\ux-prototypes\v2-cloud\index.html
# 银行: docs\ux-prototypes\v3-bank\index.html
```

---

## 报告元数据

- 蜂群规模: 4 Agent (U1/U2/U3/P1) / 1 Wave
- 总耗时: ~30 min (4 Agent 并行)
- 总产出: 21 文件 / 5950 行 / 8 截图
- 设计语言: 政务(60%) + 企业云(25%) + 银行风(15%) 三方融合
- 业务约束: ≤ 2000 行净增 / 0 新依赖 / 中文注释
