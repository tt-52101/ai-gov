# UX Redesign 批次索引

> 蜂群作战批次目录索引 (2026-08-03)

| 批次 | 主题 | 状态 | 周期 | 报告 |
|---|---|---|---|---|
| batch-001 | 蜂群作战生命周期契约建立 | ✅ 完成 | 2026-08-03 | (本仓库前序) |
| batch-009 | UX W2 基础设施落地 | ✅ 完成 | 2026-08-03 | [README](../README.md) |
| batch-010 | 预览极差修复 (shadcn Token 桥接) | ✅ 完成 | 2026-08-03 | [README](./batch-010-preview-fix/README.md) |
| **batch-011** | **样式回归 + 权限治理 UX 收口** | ✅ **完成** | **2026-08-03** | **[README](./batch-011-style-regression/README.md)** |

---

## Batch-010 速览

**触发**: 用户反馈"预览效果极差"

**根因**:
1. `globals.css` `:root` 中 30+ OKLCH 颜色压住 5 套皮肤
2. 5 套皮肤未定义 shadcn 核心 Token (--card/--primary/--background)
3. shadcn Card/Badge/Button 全部走 OKLCH 青绿色 → 与皮肤主色脱节

**修复** (6 文件, 净增 30 行):
- `globals.css`: 删除 64 行 OKLCH 默认值
- 4 个 `skin-*.css`: 各追加 18 行 shadcn 桥接
- `theme-dark.css`: 重写为完整 shadcn 兼容 (22 → 44 行)

**验收**: dev server HMR 重编译无错误, /preview-ux HTTP 200, 全部门禁通过

---

## Batch-011 速览

**触发**: 用户三轮指令
1. 生成移动端 × 4 皮肤响应式测试清单
2. 在皮肤/主题切换关键节点添加日志
3. 完成权限治理模块布局+功能优化, 拒绝反人类设计

**战果** (10 文件, 净增 ~410 行, 0 阻塞):
- **T11 切换日志**: `skin-provider.tsx` + `theme-provider.tsx` 加 4+3 节点结构化日志
- **T12 测试清单**: `mobile-skin-checklist.md` (16 手工 + 32 Playwright 自动化) + `test/mobile-skin-regression.spec.ts`
- **T7 权限治理收口**:
  - `dashboard.tsx` 新增权限治理卡片 (8 治理模块网格入口)
  - `gov-role-permissions.tsx` 默认 grouped + 重写 RoleGroupedView 为业务域 → 角色 → 动作三级折叠

**关键反人类设计收口**:
| 维度 | v1 反人类 | v2 收口 |
|---|---|---|
| 入口 | 侧栏底部 gov/* 8 项 | Dashboard 1 屏直达 |
| 默认视图 | 4 轴 × 16 动作矩阵 (16 列溢出) | 6 业务域分组 (1/2/3 列响应式) |
| 切换诊断 | 手动 DevTools 5 CSS | console 日志 3 秒定位 |

**验收**: TS 编译无新增错误, 中文注释 100% 覆盖, 0 新依赖, 4 项用户待验证

