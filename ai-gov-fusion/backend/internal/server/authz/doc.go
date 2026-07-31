// Package authz 实现四轴正交授权。
//
// 本包提供两套授权机制：
//   - grants 表直接授权：一次性直接指派（"张三对 AI 研发部账本有 balance.read 权限"）
//   - HTTP 鉴权中间件：从 context 提取主体信息，按 URL 路径匹配轴与操作，调用 grants 表评估
//
// grants 是 ABAC 策略引擎的补充——ABAC 处理策略化规则，grants 处理临时或特例授权。
// 评估顺序：ABAC deny > ABAC allow > grant deny > grant allow（DB 层 evaluate_access 函数）。
//
// 四轴（data/fund/iam/routing）独立判定，禁止一轴推导另一轴（A-CON-01）。
// 未显式授予即拒绝（A-CON-02：最小权限默认）。
package authz
