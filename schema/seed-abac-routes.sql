-- ============================================================================
-- seed-abac-routes.sql — 前端全量 ABAC+PABC 动态路由权限种子
-- 覆盖: gov 8模块 + 旧系统30+页面 → 统一 ABAC 治理
-- ============================================================================

-- 1. sys_action_catalogs 补全四轴动作（基础9个→完整38个）
INSERT OR IGNORE INTO sys_action_catalogs (action_code, action_name, axis, resource_type) VALUES
('fund.balance.read','查看余额','fund','account'),('fund.ledger.read','查看流水','fund','account'),('fund.allocate','资金划拨','fund','account'),('fund.liquidate','清算','fund','account'),('fund.budget.write','预算帽配置','fund','account'),
('data.usage.read','查看用量','data','request_log'),('data.report.read','查看报表','data','report'),('data.member.read','查看成员','data','party'),('data.dashboard.read','查看仪表盘','data','dashboard'),('data.overview.read','查看总览','data','overview'),('data.audit.read','查看审计日志','data','audit_event'),('data.security.report','安全报表','data','security_report'),('data.billing.read','账单查看','data','billing'),
('iam.key.create','创建Key','iam','api_key'),('iam.key.revoke','吊销Key','iam','api_key'),('iam.key.read','查看Key','iam','api_key'),('iam.user.disable','禁用用户','iam','user'),('iam.user.read','查看用户','iam','user'),('iam.user.manage','用户管理','iam','user'),('iam.member.add','添加成员','iam','party'),('iam.member.remove','移除成员','iam','party'),('iam.party.create','创建主体','iam','party'),('iam.party.delete','删除主体','iam','party'),('iam.abac.role.write','ABAC角色管理','iam','sys_role'),('iam.abac.policy.write','ABAC策略管理','iam','sys_access_policy'),('iam.abac.ui.write','UI权限管理','iam','sys_ui_menu'),
('routing.price.write','编辑价目','routing','model_price'),('routing.price.read','查看价目','routing','model_price'),('routing.route_profile.write','编辑路由档案','routing','route_profile'),('routing.route_profile.read','查看路由档案','routing','route_profile'),('routing.channel.write','编辑渠道','routing','provider'),('routing.channel.read','查看渠道','routing','provider'),('routing.upstream_secret.write','编辑上游密钥','routing','provider_resource'),('routing.model.read','查看模型','routing','model'),('routing.model_grant.write','模型授权','routing','model_grant'),('routing.provider.manage','供应商管理','routing','provider'),('routing.quota.manage','配额管理','routing','quota');

-- 2. sys_ui_menus 完整菜单树（gov 8项 + 旧系统分类）
INSERT OR IGNORE INTO sys_ui_menus (id, menu_code, parent_id, label, icon, sort_order) VALUES
(1,'dashboard',NULL,'仪表盘','LayoutDashboard',1),(2,'parties',NULL,'Party管理','Building2',2),(3,'fund',NULL,'资金操作','Wallet',3),(4,'pricing',NULL,'价目维护','Tags',4),(5,'routes',NULL,'路由档案','GitBranch',5),(6,'abac',NULL,'ABAC策略','Shield',6),(7,'ui_permissions',NULL,'UI权限','Eye',7),(8,'audit',NULL,'审计日志','ClipboardList',8),
(10,'overview',NULL,'总览','BarChart3',10),(11,'models',NULL,'模型管理','Box',11),(12,'providers',NULL,'供应商管理','Server',12),(13,'api_keys',NULL,'API密钥','Key',13),(14,'users',NULL,'用户管理','Users',14),(15,'usage',NULL,'用量报表','BarChart',15),(16,'billing',NULL,'账单管理','DollarSign',16),(17,'security',NULL,'安全策略','Lock',17),(18,'settings',NULL,'系统设置','Settings',19),(19,'reports',NULL,'报表中心','FileText',20),
(20,'playground',11,'模型调试','Terminal',1),(21,'quota_policies',11,'配额策略','Gauge',2),(22,'proxies',12,'代理配置','Network',1),(23,'monitors',12,'健康监控','Activity',2),(24,'notifications',NULL,'通知管理','Bell',18),(25,'alerts',24,'告警规则','AlertTriangle',1),(26,'alert_events',24,'告警事件','BellRing',2);

-- 3. sys_ui_routes 全量路由→ABAC action 映射
INSERT OR IGNORE INTO sys_ui_routes (route_path, menu_id, required_action_id) 
SELECT rp, m.id, a.id FROM (VALUES
('/gov/dashboard',1,'data.dashboard.read'),('/gov/parties',2,'iam.party.create'),('/gov/parties/[id]',2,'iam.party.create'),('/gov/fund',3,'fund.balance.read'),('/gov/fund/[id]',3,'fund.balance.read'),('/gov/pricing',4,'routing.price.read'),('/gov/pricing/[id]',4,'routing.price.read'),('/gov/routes',5,'routing.route_profile.read'),('/gov/routes/[id]',5,'routing.route_profile.read'),('/gov/abac',6,'iam.abac.role.write'),('/gov/abac/[id]',6,'iam.abac.role.write'),('/gov/ui-permissions',7,'iam.abac.ui.write'),('/gov/audit',8,'data.audit.read'),('/gov/audit/[id]',8,'data.audit.read'),
('/overview',10,'data.overview.read'),('/models',11,'routing.model.read'),('/playground',20,'routing.model.read'),('/quota-policies',21,'routing.quota.manage'),('/providers',12,'routing.channel.read'),('/proxies',22,'routing.channel.write'),('/monitors',23,'routing.channel.read'),('/provider-resources',12,'routing.channel.write'),('/api-keys',13,'iam.key.read'),('/users',14,'iam.user.read'),('/identity-providers',14,'iam.user.manage'),('/usage',15,'data.usage.read'),('/billing',16,'data.billing.read'),('/budgets',3,'fund.budget.write'),('/chargebacks',16,'data.billing.read'),('/cost-centers',3,'fund.balance.read'),('/invoices',16,'data.billing.read'),('/alerts',25,'data.report.read'),('/alert-events',26,'data.report.read'),('/notification-channels',24,'routing.channel.write'),('/security-policies',17,'data.security.report'),('/gateway',5,'routing.route_profile.read'),('/reports',19,'data.report.read'),('/audit-log',8,'data.audit.read'),('/approvals',2,'iam.party.create'),('/settings',18,'iam.user.manage'),('/database-status',18,'routing.channel.write'),('/projects',2,'iam.party.create'),('/project-members',2,'data.member.read')
) AS t(rp, mid, ac)
JOIN sys_ui_menus m ON m.id=t.mid
JOIN sys_action_catalogs a ON a.action_code=t.ac;

-- 4. sys_ui_action_bindings 按钮→ABAC action
INSERT OR IGNORE INTO sys_ui_action_bindings (button_code, button_label, page_route, required_action_id)
SELECT bc, bl, pr, a.id FROM (VALUES
('fund.allocate','划拨','/gov/fund','fund.allocate'),('fund.liquidate','清算','/gov/fund','fund.liquidate'),('fund.budget.edit','预算配置','/gov/fund','fund.budget.write'),
('party.create','创建Party','/gov/parties','iam.party.create'),('party.delete','删除Party','/gov/parties','iam.party.delete'),('party.member.add','添加成员','/gov/parties','iam.member.add'),('party.member.remove','移除成员','/gov/parties','iam.member.remove'),
('pricing.create','创建价目','/gov/pricing','routing.price.write'),('pricing.delete','删除价目','/gov/pricing','routing.price.write'),
('route.create','创建档案','/gov/routes','routing.route_profile.write'),('route.delete','删除档案','/gov/routes','routing.route_profile.write'),
('abac.role.create','创建角色','/gov/abac','iam.abac.role.write'),('abac.role.delete','删除角色','/gov/abac','iam.abac.role.write'),('abac.policy.create','创建策略','/gov/abac','iam.abac.policy.write'),('abac.policy.delete','删除策略','/gov/abac','iam.abac.policy.write'),
('apikey.create','创建Key','/api-keys','iam.key.create'),('apikey.revoke','吊销Key','/api-keys','iam.key.revoke'),
('provider.create','添加供应商','/providers','routing.channel.write'),('provider.delete','删除供应商','/providers','routing.channel.write'),
('model.grant','模型授权','/models','routing.model_grant.write'),
('user.disable','禁用用户','/users','iam.user.disable')
) AS t(bc, bl, pr, ac)
JOIN sys_action_catalogs a ON a.action_code=t.ac;

-- 5. 角色权限回灌
INSERT OR IGNORE INTO sys_role_permissions (role_id, action_id) SELECT (SELECT id FROM sys_roles WHERE role_code='super_admin'), id FROM sys_action_catalogs;
INSERT OR IGNORE INTO sys_role_permissions (role_id, action_id) SELECT (SELECT id FROM sys_roles WHERE role_code='finance_mgr'), id FROM sys_action_catalogs WHERE axis IN ('fund','data');
INSERT OR IGNORE INTO sys_role_permissions (role_id, action_id) SELECT (SELECT id FROM sys_roles WHERE role_code='dept_leader'), id FROM sys_action_catalogs WHERE axis IN ('data','iam') AND action_code NOT LIKE '%abac%';
INSERT OR IGNORE INTO sys_role_permissions (role_id, action_id) SELECT (SELECT id FROM sys_roles WHERE role_code='employee'), id FROM sys_action_catalogs WHERE action_code IN ('data.usage.read','iam.key.read','data.dashboard.read');
INSERT OR IGNORE INTO sys_role_permissions (role_id, action_id) SELECT (SELECT id FROM sys_roles WHERE role_code='auditor'), id FROM sys_action_catalogs WHERE axis='data';

-- 6. 内置 ABAC 策略
INSERT OR IGNORE INTO sys_access_policies (policy_code, policy_name, effect, conditions_json, priority, is_system) VALUES
('P-SOD-FUND','职责分离-财务','deny','{"axis":"routing"}',100,1),('P-SOD-ROUTING','职责分离-路由','deny','{"axis":"fund"}',100,1),('P-SOD-IAM','职责分离-身份','deny','{"axis":"routing"}',100,1),('P-AUDIT-ONLY','审计只读','allow','{"actions":["data.audit.read","data.usage.read","data.report.read"]}',50,1),('P-DEFAULT-UI','默认UI可见','allow','{"actions":["data.dashboard.read","data.overview.read"]}',10,1);
