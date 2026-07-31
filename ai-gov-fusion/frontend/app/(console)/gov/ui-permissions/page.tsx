"use client";

import React from "react";
import {
  Eye,
  Plus,
  Trash2,
  GripVertical,
  EyeOff,
  Link2,
} from "lucide-react";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";
import { ConfirmDialog } from "../_components/ConfirmDialog";

/** 菜单节点 */
interface MenuItem {
  id: string;
  menu_code: string;
  parent_id: string | null;
  label: string;
  label_zh: string;
  icon: string;
  sort_order: number;
  required_action_id: string | null;
  children?: MenuItem[];
}

/** 路由权限 */
interface UiRoute {
  id: string;
  route_path: string;
  menu_id: string;
  required_action_id: string;
  label: string;
  label_zh: string;
}

/** 按钮绑定 */
interface ActionBinding {
  id: string;
  button_code: string;
  button_label: string;
  button_label_zh: string;
  page_route: string;
  required_action_id: string;
}

const API_BASE = "/gov";

/**
 * UI 权限管理页面 —— 菜单树编辑器、路由权限配置、按钮显隐配置。
 * 对应 PRD UI-13 需求。
 */
export default function UiPermissionsPage() {
  // 标签页
  const [tab, setTab] = React.useState<"menus" | "routes" | "buttons">("menus");
  const [error, setError] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);

  // 菜单数据
  const [menus, setMenus] = React.useState<MenuItem[]>([]);
  const [menusLoading, setMenusLoading] = React.useState(true);
  const [showMenuDialog, setShowMenuDialog] = React.useState(false);
  const [editingMenu, setEditingMenu] = React.useState<MenuItem | null>(null);
  const [menuForm, setMenuForm] = React.useState({
    menu_code: "",
    parent_id: "",
    label: "",
    label_zh: "",
    icon: "",
    sort_order: 0,
    required_action_id: "",
  });

  // 路由数据
  const [routes, setRoutes] = React.useState<UiRoute[]>([]);
  const [routesLoading, setRoutesLoading] = React.useState(true);
  const [showRouteDialog, setShowRouteDialog] = React.useState(false);
  const [routeForm, setRouteForm] = React.useState({
    route_path: "",
    menu_id: "",
    required_action_id: "",
    label: "",
    label_zh: "",
  });

  // 按钮绑定数据
  const [bindings, setBindings] = React.useState<ActionBinding[]>([]);
  const [bindingsLoading, setBindingsLoading] = React.useState(true);
  const [showBindingDialog, setShowBindingDialog] = React.useState(false);
  const [bindingForm, setBindingForm] = React.useState({
    button_code: "",
    button_label: "",
    button_label_zh: "",
    page_route: "",
    required_action_id: "",
  });

  // 删除确认
  const [confirmDelete, setConfirmDelete] = React.useState<{
    type: "menu" | "route" | "binding";
    id: string;
    label: string;
  } | null>(null);

  // 获取菜单
  const fetchMenus = React.useCallback(async () => {
    setMenusLoading(true);
    try {
      const res = await fetch(`${API_BASE}/ui-menus`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setMenus(json.data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取菜单列表失败");
    } finally {
      setMenusLoading(false);
    }
  }, []);

  // 获取路由
  const fetchRoutes = React.useCallback(async () => {
    setRoutesLoading(true);
    try {
      const res = await fetch(`${API_BASE}/ui-routes`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setRoutes(json.data ?? []);
    } catch {
      setRoutes([]);
    } finally {
      setRoutesLoading(false);
    }
  }, []);

  // 获取按钮绑定
  const fetchBindings = React.useCallback(async () => {
    setBindingsLoading(true);
    try {
      const res = await fetch(`${API_BASE}/ui-action-bindings`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setBindings(json.data ?? []);
    } catch {
      setBindings([]);
    } finally {
      setBindingsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    setError(null);
    if (tab === "menus") fetchMenus();
    else if (tab === "routes") fetchRoutes();
    else if (tab === "buttons") fetchBindings();
  }, [tab, fetchMenus, fetchRoutes, fetchBindings]);

  // 保存菜单
  const handleSaveMenu = async () => {
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        menu_code: menuForm.menu_code,
        label: menuForm.label,
      };
      if (menuForm.parent_id) body.parent_id = menuForm.parent_id;
      if (menuForm.label_zh) body.label_zh = menuForm.label_zh;
      if (menuForm.icon) body.icon = menuForm.icon;
      if (menuForm.sort_order) body.sort_order = menuForm.sort_order;
      if (menuForm.required_action_id) body.required_action_id = menuForm.required_action_id;

      const res = await fetch(`${API_BASE}/ui-menus`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setShowMenuDialog(false);
      fetchMenus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建菜单失败");
    } finally {
      setSaving(false);
    }
  };

  // 保存路由
  const handleSaveRoute = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/ui-routes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(routeForm),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setShowRouteDialog(false);
      fetchRoutes();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建路由失败");
    } finally {
      setSaving(false);
    }
  };

  // 保存按钮绑定
  const handleSaveBinding = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/ui-action-bindings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(bindingForm),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setShowBindingDialog(false);
      fetchBindings();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建按钮绑定失败");
    } finally {
      setSaving(false);
    }
  };

  // 执行删除
  const handleDelete = async () => {
    if (!confirmDelete) return;
    try {
      if (confirmDelete.type === "menu") {
        await fetch(`${API_BASE}/ui-menus/${confirmDelete.id}`, { method: "DELETE" });
        fetchMenus();
      } else if (confirmDelete.type === "route") {
        await fetch(`${API_BASE}/ui-routes/${confirmDelete.id}`, { method: "DELETE" });
        fetchRoutes();
      } else {
        await fetch(`${API_BASE}/ui-action-bindings/${confirmDelete.id}`, { method: "DELETE" });
        fetchBindings();
      }
      setConfirmDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    }
  };

  // 递归渲染菜单树节点
  const renderMenuNode = (node: MenuItem, depth = 0) => (
    <div key={node.id}>
      <div
        className="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 transition-colors hover:bg-gray-50"
        style={{ marginLeft: depth * 24 }}
      >
        <GripVertical className="h-4 w-4 flex-shrink-0 cursor-grab text-gray-300" />
        <span className="min-w-0 flex-1 text-sm text-gray-800">
          {node.label_zh || node.label}
          <span className="ml-2 text-xs text-gray-400">({node.menu_code})</span>
        </span>
        <span className="text-xs text-gray-400">排序: {node.sort_order}</span>
        <button
          onClick={() => {
            setEditingMenu(node);
            setMenuForm({
              menu_code: node.menu_code,
              parent_id: node.parent_id ?? "",
              label: node.label,
              label_zh: node.label_zh,
              icon: node.icon,
              sort_order: node.sort_order,
              required_action_id: node.required_action_id ?? "",
            });
            setShowMenuDialog(true);
          }}
          className="rounded p-1 text-gray-400 hover:text-blue-600"
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" /></svg>
        </button>
        <button
          onClick={() => setConfirmDelete({ type: "menu", id: node.id, label: node.label_zh || node.label })}
          className="rounded p-1 text-gray-400 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      </div>
      {/* 递归渲染子节点 */}
      {node.children?.map((child) => renderMenuNode(child, depth + 1))}
    </div>
  );

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">UI 权限管理</h1>
        <p className="mt-1 text-sm text-gray-500">管理菜单可见性、路由守卫和按钮显隐配置</p>
      </div>

      {error && <ErrorAlert message={error} dismissible />}

      {/* 标签导航 */}
      <div className="flex border-b border-gray-200">
        {([
          { key: "menus" as const, label: "菜单树", icon: Eye },
          { key: "routes" as const, label: "路由权限", icon: Link2 },
          { key: "buttons" as const, label: "按钮显隐", icon: EyeOff },
        ]).map((t) => {
          const Icon = t.icon;
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
                tab === t.key
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700"
              }`}
            >
              <Icon className="h-4 w-4" />
              {t.label}
            </button>
          );
        })}
      </div>

      {/* ===== 菜单树编辑器 ===== */}
      {tab === "menus" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setEditingMenu(null);
                setMenuForm({ menu_code: "", parent_id: "", label: "", label_zh: "", icon: "", sort_order: 0, required_action_id: "" });
                setShowMenuDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 添加菜单
            </button>
          </div>

          {menusLoading ? (
            <div className="animate-pulse space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-10 rounded-lg bg-gray-200" />
              ))}
            </div>
          ) : menus.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-10">暂无菜单配置</p>
          ) : (
            <div className="space-y-1">{menus.map((m) => renderMenuNode(m))}</div>
          )}
        </div>
      )}

      {/* ===== 路由权限 ===== */}
      {tab === "routes" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setRouteForm({ route_path: "", menu_id: "", required_action_id: "", label: "", label_zh: "" });
                setShowRouteDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 添加路由
            </button>
          </div>

          {routesLoading ? (
            <div className="animate-pulse space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-12 rounded-lg bg-gray-200" />
              ))}
            </div>
          ) : routes.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-10">暂无路由配置</p>
          ) : (
            <div className="space-y-2">
              {routes.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg border border-gray-200 bg-white px-4 py-2">
                  <div>
                    <span className="font-mono text-sm text-gray-800">{r.route_path}</span>
                    <span className="ml-3 text-sm text-gray-500">{r.label_zh || r.label}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-400">动作: {r.required_action_id.slice(0, 12)}...</span>
                    <button
                      onClick={() => setConfirmDelete({ type: "route", id: r.id, label: r.label_zh || r.label })}
                      className="rounded p-1 text-gray-400 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ===== 按钮显隐 ===== */}
      {tab === "buttons" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setBindingForm({ button_code: "", button_label: "", button_label_zh: "", page_route: "", required_action_id: "" });
                setShowBindingDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 添加按钮绑定
            </button>
          </div>

          {bindingsLoading ? (
            <div className="animate-pulse space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-12 rounded-lg bg-gray-200" />
              ))}
            </div>
          ) : bindings.length === 0 ? (
            <p className="text-center text-sm text-gray-400 py-10">暂无按钮绑定</p>
          ) : (
            <div className="space-y-2">
              {bindings.map((b) => (
                <div key={b.id} className="flex items-center justify-between rounded-lg border border-gray-200 bg-white px-4 py-2">
                  <div>
                    <span className="text-sm font-medium text-gray-800">{b.button_label_zh || b.button_label}</span>
                    <span className="ml-2 font-mono text-xs text-gray-400">{b.button_code}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-xs text-gray-400">路由: {b.page_route}</span>
                    <span className="text-xs text-gray-400">动作: {b.required_action_id.slice(0, 12)}...</span>
                    <button
                      onClick={() => setConfirmDelete({ type: "binding", id: b.id, label: b.button_label_zh || b.button_label })}
                      className="rounded p-1 text-gray-400 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ===== 菜单编辑对话框 ===== */}
      {showMenuDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowMenuDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">{editingMenu ? "编辑菜单" : "添加菜单"}</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">菜单编码 *</label>
                <input type="text" value={menuForm.menu_code} onChange={(e) => setMenuForm({ ...menuForm, menu_code: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">英文标签 *</label>
                  <input type="text" value={menuForm.label} onChange={(e) => setMenuForm({ ...menuForm, label: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">中文标签</label>
                  <input type="text" value={menuForm.label_zh} onChange={(e) => setMenuForm({ ...menuForm, label_zh: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">图标名称</label>
                <input type="text" value={menuForm.icon} onChange={(e) => setMenuForm({ ...menuForm, icon: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="如 wallet, shield" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">排序值</label>
                  <input type="number" value={menuForm.sort_order} onChange={(e) => setMenuForm({ ...menuForm, sort_order: parseInt(e.target.value) || 0 })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">父菜单 ID</label>
                  <input type="text" value={menuForm.parent_id} onChange={(e) => setMenuForm({ ...menuForm, parent_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="空 = 根节点" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Required Action ID</label>
                <input type="text" value={menuForm.required_action_id} onChange={(e) => setMenuForm({ ...menuForm, required_action_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="空 = 始终可见" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowMenuDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSaveMenu} disabled={!menuForm.menu_code || !menuForm.label || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 路由编辑对话框 ===== */}
      {showRouteDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowRouteDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">添加路由权限</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">路由路径 *</label>
                <input type="text" value={routeForm.route_path} onChange={(e) => setRouteForm({ ...routeForm, route_path: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="/console/fund/allocate" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">关联菜单 ID *</label>
                  <input type="text" value={routeForm.menu_id} onChange={(e) => setRouteForm({ ...routeForm, menu_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Required Action ID *</label>
                  <input type="text" value={routeForm.required_action_id} onChange={(e) => setRouteForm({ ...routeForm, required_action_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">英文标签 *</label>
                  <input type="text" value={routeForm.label} onChange={(e) => setRouteForm({ ...routeForm, label: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">中文标签</label>
                  <input type="text" value={routeForm.label_zh} onChange={(e) => setRouteForm({ ...routeForm, label_zh: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowRouteDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSaveRoute} disabled={!routeForm.route_path || !routeForm.menu_id || !routeForm.required_action_id || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 按钮绑定对话框 ===== */}
      {showBindingDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowBindingDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">添加按钮绑定</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">按钮编码 *</label>
                <input type="text" value={bindingForm.button_code} onChange={(e) => setBindingForm({ ...bindingForm, button_code: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">英文标签 *</label>
                  <input type="text" value={bindingForm.button_label} onChange={(e) => setBindingForm({ ...bindingForm, button_label: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">中文标签</label>
                  <input type="text" value={bindingForm.button_label_zh} onChange={(e) => setBindingForm({ ...bindingForm, button_label_zh: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">页面路由 *</label>
                <input type="text" value={bindingForm.page_route} onChange={(e) => setBindingForm({ ...bindingForm, page_route: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Required Action ID *</label>
                <input type="text" value={bindingForm.required_action_id} onChange={(e) => setBindingForm({ ...bindingForm, required_action_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowBindingDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSaveBinding} disabled={!bindingForm.button_code || !bindingForm.button_label || !bindingForm.page_route || !bindingForm.required_action_id || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 删除确认弹窗 */}
      <ConfirmDialog
        open={!!confirmDelete}
        title={`确认删除${confirmDelete?.type === "menu" ? "菜单" : confirmDelete?.type === "route" ? "路由" : "按钮绑定"}`}
        message={`确定要删除「${confirmDelete?.label}」吗？`}
        danger
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
