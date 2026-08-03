# UX Redesign 批次索引

> 蜂群作战批次目录索引 (2026-08-03)

| 批次 | 主题 | 状态 | 周期 | 报告 |
|---|---|---|---|---|
| batch-001 | 蜂群作战生命周期契约建立 | ✅ 完成 | 2026-08-03 | (本仓库前序) |
| batch-009 | UX W2 基础设施落地 | ✅ 完成 | 2026-08-03 | [README](../README.md) |
| **batch-010** | **预览极差修复 (shadcn Token 桥接)** | ✅ **完成** | **2026-08-03** | [README](./batch-010-preview-fix/README.md) |

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
