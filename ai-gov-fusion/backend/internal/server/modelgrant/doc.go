// Package modelgrant 实现模型访问治理。
//
// 本包提供两套能力：
//   - 模型授权规则 CRUD：对 model_grants 表的增删改查
//   - 模型访问检查器（Checker）：数据面 Pipeline 第 4 步，判断主体是否有权访问指定模型
//
// 核心规则：
//   - ALLOW/DENY 规则：DENY 优先于 ALLOW（A-CON-04）
//   - 级联顺序：Key > Person > Party > 全局默认
//   - 模型级配额（quota_limit）：双层预算第二层，与 Account 预算帽取交集
//   - 禁止仅因 Leader 头衔自动拥有全平台模型权（A-CON-05）
//   - 无匹配规则默认拒绝（A-CON-02：最小权限默认）
//
// Checker 的三个方法对应数据面三层校验：
//   - CheckAccess：访问权判定（是否有权调用该模型）
//   - CheckQuotaLimit：配额判定（模型级预算是否超限）
//   - ConsumeQuota：消耗记录（请求成功后累加消耗）
package modelgrant
