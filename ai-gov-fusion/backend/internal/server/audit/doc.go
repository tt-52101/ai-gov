// Package audit 实现不可篡改审计追踪——记录每次变更的操作者、时间戳、操作类型
// 以及变更前后的完整快照。审计事件表应用层仅允许 INSERT，禁止 UPDATE 与 DELETE，
// 并配合定期哈希链锚定（audit_chain_anchors）提供防篡改证明。
//
// 本包覆盖 PRD §7.6 审计不可篡改要求：
//   - AU-CON-01 全链路可追溯
//   - AU-CON-02 资金操作全审计（before/after 快照）
//   - AU-CON-03 配置变更全快照
//   - D-CON-04 审计不可篡改（≥180 天保留，定期哈希锚定）
package audit
