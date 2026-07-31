# 任务批次 005：E2E全矩阵功能测试
| 项 | 值 |
|----|-----|
| 批次编号 | batch-005 |
| 任务主题 | 零Mock/真实DB/全链路端到端功能测试 |
| 执行日期 | 2026-07-31 |
| 蜂群配置 | 6 Agent(DB+Auth+API+Flow+UI+Error) |
| 验收结论 | ⚠️ 条件不通过——5致命+14高+8中 |

## 🔴 致命缺陷(5项/E2E-3)
- policies/evaluate缺{policy_id}路径段
- DELETE parties无后端handler
- amount发number→后端要string
- allocate缺channel必填字段
- liquidate缺party_id必填字段

## 🔴 高风险(关键汇总)
- 无前端路由守卫(middleware.ts不存在)
- ConsoleLayout不渲染{children}
- 路由3错误码无结构化码
- 6 FK缺失
- 27后端错误码前端无映射
- INTERNAL_ONLY出网管控命名不一致

## 单兵记录
详见 agents/ 目录下6份执行轨迹。