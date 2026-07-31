// Package ui_permission 实现 UI 权限投影——ABAC 引擎在展示层的投影。
// 控制菜单可见性、路由守卫、页面按钮显隐，是 ABAC 权限的前端映射，不是独立的权限体系。
//
// 安全原则（PRD §7.4.3）：前端隐藏按钮减少误操作，真正的安全在后端 ABAC 引擎。
// 即使前端绕过 UI 限制直接调用 API，ABAC 引擎仍会拒绝。
package ui_permission
