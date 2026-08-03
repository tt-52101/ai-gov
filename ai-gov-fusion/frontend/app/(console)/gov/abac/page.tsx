"use client";

import React from "react";
import {
  Shield,
  Plus,
  Link2,
  Play,
  Lock,
  UserCheck,
  Trash2,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";
import { govFetch, govFetchJSON } from "@/lib/gov-api";

/** 角色数据 */
interface Role extends Record<string, unknown> {
  id: string;
  role_code: string;
  role_name: string;
  description: string;
  is_system: boolean;
  /** 后端可能返回 null（角色未配置权限项） */
  permissions: string[] | null;
  created_at: string;
}

/** ABAC 策略数据 */
interface Policy extends Record<string, unknown> {
  id: string;
  policy_code: string;
  policy_name: string;
  description: string;
  effect: "allow" | "deny";
  priority: number;
  is_system: boolean;
  conditions_json: Record<string, unknown>;
  created_at: string;
}

/** 主体角色绑定 */
interface RoleBinding extends Record<string, unknown> {
  id: string;
  subject_type: "user" | "party";
  subject_id: string;
  role_id: string;
  role_code: string;
  scope_party_id: string | null;
  valid_from: string | null;
  valid_until: string | null;
  created_at: string;
}

/** 策略模拟结果 */
interface PolicyEvalResult {
  result: "allow" | "deny";
  matched_policy_ids: string[];
  evaluation_details: {
    policy_id: string;
    policy_code: string;
    effect: string;
    matched: boolean;
    reason: string;
  }[];
  evaluated_at: string;
}

/**
 * ABAC 策略管理页面 —— 角色 CRUD、策略管理、主体角色绑定、策略模拟评估。
 * 对应 PRD UI-12 需求。is_system 角色/策略不可编辑删除。
 */
export default function AbacPage() {
  // 活动标签页
  const [tab, setTab] = React.useState<"roles" | "policies" | "bindings" | "simulator">("roles");

  // 角色状态
  const [roles, setRoles] = React.useState<Role[]>([]);
  const [rolesLoading, setRolesLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 策略状态
  const [policies, setPolicies] = React.useState<Policy[]>([]);
  const [policiesTotal, setPoliciesTotal] = React.useState(0);
  const [policiesPage, setPoliciesPage] = React.useState(1);
  const [policiesLoading, setPoliciesLoading] = React.useState(true);

  // 绑定状态
  const [bindings, setBindings] = React.useState<RoleBinding[]>([]);
  const [bindingsTotal, setBindingsTotal] = React.useState(0);
  const [bindingsPage, setBindingsPage] = React.useState(1);
  const [bindingsLoading, setBindingsLoading] = React.useState(true);

  // 对话框状态
  const [showRoleDialog, setShowRoleDialog] = React.useState(false);
  const [showPolicyDialog, setShowPolicyDialog] = React.useState(false);
  const [showBindingDialog, setShowBindingDialog] = React.useState(false);
  const [editingItem, setEditingItem] = React.useState<unknown>(null);

  // 角色表单
  const [roleForm, setRoleForm] = React.useState({
    role_code: "",
    role_name: "",
    description: "",
    is_system: false,
    permissions: [] as string[],
  });
  const [permissionInput, setPermissionInput] = React.useState("");

  // 策略表单
  const [policyForm, setPolicyForm] = React.useState({
    policy_code: "",
    policy_name: "",
    description: "",
    effect: "deny" as "allow" | "deny",
    priority: 0,
    is_system: false,
    conditions_json: "{}",
  });

  // 绑定表单
  const [bindingForm, setBindingForm] = React.useState({
    subject_type: "user" as "user" | "party",
    subject_id: "",
    role_id: "",
    scope_party_id: "",
    valid_from: "",
    valid_until: "",
  });

  // 模拟器状态
  const [simForm, setSimForm] = React.useState({
    subject_user_id: "",
    resource_type: "account",
    resource_id: "",
    action: "fund.allocate",
  });
  const [simResult, setSimResult] = React.useState<PolicyEvalResult | null>(null);
  const [simulating, setSimulating] = React.useState(false);

  // 删除确认
  const [confirmDelete, setConfirmDelete] = React.useState<{
    type: "role" | "policy" | "binding";
    item: Role | Policy | RoleBinding;
  } | null>(null);

  // 保存中
  const [saving, setSaving] = React.useState(false);

  // 获取角色列表
  const fetchRoles = React.useCallback(async () => {
    setRolesLoading(true);
    try {
      const json = await govFetchJSON<{ data: Role[] }>("/roles");
      setRoles(json.data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取角色列表失败");
    } finally {
      setRolesLoading(false);
    }
  }, []);

  // 获取策略列表
  const fetchPolicies = React.useCallback(async () => {
    setPoliciesLoading(true);
    try {
      const params = new URLSearchParams({ page: String(policiesPage), page_size: "20" });
      const json = await govFetchJSON<{ data: Policy[]; total: number }>(`/policies?${params}`);
      setPolicies(json.data ?? []);
      setPoliciesTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取策略列表失败");
    } finally {
      setPoliciesLoading(false);
    }
  }, [policiesPage]);

  // 获取绑定列表
  const fetchBindings = React.useCallback(async () => {
    setBindingsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(bindingsPage), page_size: "20" });
      const json = await govFetchJSON<{ data: RoleBinding[]; total: number }>(`/subject-role-bindings?${params}`);
      setBindings(json.data ?? []);
      setBindingsTotal(json.total ?? 0);
    } catch {
      setBindings([]);
    } finally {
      setBindingsLoading(false);
    }
  }, [bindingsPage]);

  // 标签切换时加载数据
  React.useEffect(() => {
    setError(null);
    if (tab === "roles") fetchRoles();
    else if (tab === "policies") fetchPolicies();
    else if (tab === "bindings") fetchBindings();
  }, [tab, fetchRoles, fetchPolicies, fetchBindings]);

  // 保存角色
  const handleSaveRole = async () => {
    setSaving(true);
    try {
      await govFetchJSON("/roles", {
        method: "POST",
        body: JSON.stringify(roleForm),
      });
      setShowRoleDialog(false);
      fetchRoles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建角色失败");
    } finally {
      setSaving(false);
    }
  };

  // 保存策略
  const handleSavePolicy = async () => {
    setSaving(true);
    try {
      const body = { ...policyForm, conditions_json: JSON.parse(policyForm.conditions_json) };
      await govFetchJSON("/policies", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setShowPolicyDialog(false);
      fetchPolicies();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建策略失败");
    } finally {
      setSaving(false);
    }
  };

  // 保存绑定
  const handleSaveBinding = async () => {
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        subject_type: bindingForm.subject_type,
        subject_id: bindingForm.subject_id,
        role_id: bindingForm.role_id,
      };
      if (bindingForm.scope_party_id) body.scope_party_id = bindingForm.scope_party_id;
      if (bindingForm.valid_from) body.valid_from = bindingForm.valid_from;
      if (bindingForm.valid_until) body.valid_until = bindingForm.valid_until;

      await govFetchJSON("/subject-role-bindings", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setShowBindingDialog(false);
      fetchBindings();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建绑定失败");
    } finally {
      setSaving(false);
    }
  };

  // 执行删除
  const handleDelete = async () => {
    if (!confirmDelete) return;
    const { type, item } = confirmDelete;
    try {
      if (type === "role") {
        await govFetch(`/roles/${item.id}`, { method: "DELETE" });
        fetchRoles();
      } else if (type === "policy") {
        await govFetch(`/policies/${item.id}`, { method: "DELETE" });
        fetchPolicies();
      } else if (type === "binding") {
        await govFetch(`/subject-role-bindings/${item.id}`, { method: "DELETE" });
        fetchBindings();
      }
      setConfirmDelete(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    }
  };

  // 模拟评估
  const handleSimulate = async () => {
    setSimulating(true);
    try {
      const json = await govFetchJSON<PolicyEvalResult>("/policies/evaluate", {
        method: "POST",
        body: JSON.stringify(simForm),
      });
      setSimResult(json);
    } catch (err) {
      setError(err instanceof Error ? err.message : "模拟评估失败");
    } finally {
      setSimulating(false);
    }
  };

  // 添加权限到角色
  const addPermission = () => {
    const trimmed = permissionInput.trim();
    if (trimmed && !roleForm.permissions.includes(trimmed)) {
      setRoleForm({ ...roleForm, permissions: [...roleForm.permissions, trimmed] });
      setPermissionInput("");
    }
  };

  // 角色表格列
  const roleColumns: ColumnDef<Role>[] = [
    { key: "role_code", header: "角色编码" },
    { key: "role_name", header: "角色名称" },
    {
      key: "is_system",
      header: "类型",
      render: (r) => r.is_system ? (
        <span className="inline-flex rounded bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800">系统</span>
      ) : (
        <span className="inline-flex rounded bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800">自定义</span>
      ),
    },
    {
      key: "permissions",
      header: "权限数",
      render: (r) => <span>{r.permissions?.length ?? 0} 项</span>,
    },
    {
      key: "actions",
      header: "操作",
      render: (r) => !r.is_system ? (
        <button
          onClick={(e) => { e.stopPropagation(); setConfirmDelete({ type: "role", item: r }); }}
          className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      ) : <span className="text-xs text-gray-400">不可操作</span>,
    },
  ];

  // 策略表格列
  const policyColumns: ColumnDef<Policy>[] = [
    { key: "policy_code", header: "策略编码" },
    { key: "policy_name", header: "策略名称" },
    {
      key: "effect",
      header: "效果",
      render: (p) => (
        <span className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${
          p.effect === "deny" ? "bg-red-100 text-red-800" : "bg-green-100 text-green-800"
        }`}>
          {p.effect === "deny" ? "拒绝" : "允许"}
        </span>
      ),
    },
    { key: "priority", header: "优先级" },
    {
      key: "actions",
      header: "操作",
      render: (p) => !p.is_system ? (
        <button
          onClick={(e) => { e.stopPropagation(); setConfirmDelete({ type: "policy", item: p }); }}
          className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      ) : <span className="text-xs text-gray-400">不可操作</span>,
    },
  ];

  // 绑定表格列
  const bindingColumns: ColumnDef<RoleBinding>[] = [
    {
      key: "subject_type",
      header: "主体类型",
      render: (b) => b.subject_type === "user" ? "用户" : "Party",
    },
    {
      key: "subject_id",
      header: "主体 ID",
      render: (b) => <span className="font-mono text-xs">{b.subject_id.slice(0, 8)}...</span>,
    },
    { key: "role_code", header: "角色" },
    {
      key: "scope_party_id",
      header: "作用域",
      render: (b) => b.scope_party_id ? (
        <span className="font-mono text-xs">{b.scope_party_id.slice(0, 8)}...</span>
      ) : (
        <span className="text-gray-400">全局</span>
      ),
    },
    {
      key: "actions",
      header: "操作",
      render: (b) => (
        <button
          onClick={(e) => { e.stopPropagation(); setConfirmDelete({ type: "binding", item: b }); }}
          className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">ABAC 策略管理</h1>
        <p className="mt-1 text-sm text-gray-500">管理角色、策略、主体角色绑定和策略模拟评估</p>
      </div>

      {error && <ErrorAlert message={error} dismissible />}

      {/* 标签导航 */}
      <div className="flex border-b border-gray-200">
        {([
          { key: "roles", label: "角色", icon: UserCheck },
          { key: "policies", label: "策略", icon: Shield },
          { key: "bindings", label: "角色绑定", icon: Link2 },
          { key: "simulator", label: "策略模拟", icon: Play },
        ] as const).map((t) => {
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

      {/* ===== 角色管理 ===== */}
      {tab === "roles" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setRoleForm({ role_code: "", role_name: "", description: "", is_system: false, permissions: [] });
                setPermissionInput("");
                setShowRoleDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 创建角色
            </button>
          </div>
          <DataTable
            data={roles}
            columns={roleColumns}
            loading={rolesLoading}
            emptyText="暂无角色"
          />
        </div>
      )}

      {/* ===== 策略管理 ===== */}
      {tab === "policies" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setPolicyForm({ policy_code: "", policy_name: "", description: "", effect: "deny", priority: 0, is_system: false, conditions_json: "{}" });
                setShowPolicyDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 创建策略
            </button>
          </div>
          <DataTable
            data={policies}
            columns={policyColumns}
            page={policiesPage}
            pageSize={20}
            total={policiesTotal}
            onPageChange={setPoliciesPage}
            loading={policiesLoading}
            emptyText="暂无策略"
          />
        </div>
      )}

      {/* ===== 角色绑定管理 ===== */}
      {tab === "bindings" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setBindingForm({ subject_type: "user", subject_id: "", role_id: "", scope_party_id: "", valid_from: "", valid_until: "" });
                setShowBindingDialog(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 创建绑定
            </button>
          </div>
          <DataTable
            data={bindings}
            columns={bindingColumns}
            page={bindingsPage}
            pageSize={20}
            total={bindingsTotal}
            onPageChange={setBindingsPage}
            loading={bindingsLoading}
            emptyText="暂无绑定"
          />
        </div>
      )}

      {/* ===== 策略模拟器 ===== */}
      {tab === "simulator" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* 输入表单 */}
          <div className="rounded-lg border border-gray-200 bg-white p-5">
            <h2 className="font-medium text-gray-900">模拟条件</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm text-gray-600">主体用户 ID</label>
                <input
                  type="text"
                  value={simForm.subject_user_id}
                  onChange={(e) => setSimForm({ ...simForm, subject_user_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="UUID"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-600">资源类型</label>
                <select
                  value={simForm.resource_type}
                  onChange={(e) => setSimForm({ ...simForm, resource_type: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                >
                  <option value="account">account (账户)</option>
                  <option value="party">party (主体)</option>
                  <option value="model_price">model_price (价目)</option>
                  <option value="route_profile">route_profile (路由档案)</option>
                  <option value="api_key">api_key (密钥)</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-600">资源 ID</label>
                <input
                  type="text"
                  value={simForm.resource_id}
                  onChange={(e) => setSimForm({ ...simForm, resource_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="UUID"
                />
              </div>
              <div>
                <label className="block text-sm text-gray-600">动作</label>
                <select
                  value={simForm.action}
                  onChange={(e) => setSimForm({ ...simForm, action: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                >
                  <option value="fund.allocate">fund.allocate (划拨)</option>
                  <option value="fund.balance.read">fund.balance.read (读余额)</option>
                  <option value="fund.liquidate">fund.liquidate (清算)</option>
                  <option value="routing.price.write">routing.price.write (写价目)</option>
                  <option value="routing.route_profile.write">routing.route_profile.write (写路由)</option>
                  <option value="iam.policy.write">iam.policy.write (写策略)</option>
                  <option value="data.usage.read">data.usage.read (读用量)</option>
                </select>
              </div>
              <button
                onClick={handleSimulate}
                disabled={!simForm.subject_user_id || simulating}
                className="inline-flex items-center gap-1.5 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                <Play className="h-4 w-4" />
                {simulating ? "评估中..." : "执行模拟"}
              </button>
            </div>
          </div>

          {/* 评估结果 */}
          <div className="rounded-lg border border-gray-200 bg-white p-5">
            <h2 className="font-medium text-gray-900">评估结果</h2>
            {simResult ? (
              <div className="mt-4 space-y-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-gray-600">结果：</span>
                  <span
                    className={`inline-flex rounded-full px-3 py-1 text-sm font-bold ${
                      simResult.result === "deny"
                        ? "bg-red-100 text-red-700"
                        : "bg-green-100 text-green-700"
                    }`}
                  >
                    {simResult.result === "deny" ? "拒绝" : "允许"}
                  </span>
                </div>

                {/* 匹配链 */}
                <div>
                  <h3 className="text-sm font-medium text-gray-600">匹配策略链</h3>
                  <div className="mt-2 space-y-2">
                    {simResult.evaluation_details.map((detail, i) => (
                      <div
                        key={i}
                        className={`rounded border p-2 text-xs ${
                          detail.matched
                            ? detail.effect === "deny"
                              ? "border-red-200 bg-red-50"
                              : "border-green-200 bg-green-50"
                            : "border-gray-200 bg-gray-50"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-mono font-medium">{detail.policy_code}</span>
                          <span className={`${detail.effect === "deny" ? "text-red-600" : "text-green-600"}`}>
                            {detail.effect}
                          </span>
                        </div>
                        <p className="mt-1 text-gray-500">{detail.reason}</p>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ) : (
              <p className="mt-4 text-sm text-gray-400">尚未执行模拟评估。请在左侧填写条件后点击执行。</p>
            )}
          </div>
        </div>
      )}

      {/* ===== 创建角色对话框 ===== */}
      {showRoleDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowRoleDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">创建角色</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">角色编码 *</label>
                <input type="text" value={roleForm.role_code} onChange={(e) => setRoleForm({ ...roleForm, role_code: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">角色名称 *</label>
                <input type="text" value={roleForm.role_name} onChange={(e) => setRoleForm({ ...roleForm, role_name: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input type="checkbox" checked={roleForm.is_system} onChange={(e) => setRoleForm({ ...roleForm, is_system: e.target.checked })} className="rounded" />
                  系统角色（不可删除）
                </label>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">权限列表</label>
                <div className="mt-1 flex gap-1">
                  <input type="text" value={permissionInput} onChange={(e) => setPermissionInput(e.target.value)} placeholder="如 fund.allocate" className="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm" onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); addPermission(); } }} />
                  <button onClick={addPermission} className="rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50">添加</button>
                </div>
                {roleForm.permissions.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {roleForm.permissions.map((p) => (
                      <span key={p} className="inline-flex items-center gap-1 rounded bg-blue-100 px-2 py-0.5 text-xs text-blue-700">
                        {p}
                        <button onClick={() => setRoleForm({ ...roleForm, permissions: roleForm.permissions.filter((x) => x !== p) })} className="text-blue-500 hover:text-red-500">&times;</button>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowRoleDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSaveRole} disabled={!roleForm.role_code || !roleForm.role_name || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 创建策略对话框 ===== */}
      {showPolicyDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowPolicyDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-lg rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">创建 ABAC 策略</h2>
            <div className="mt-4 space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">策略编码 *</label>
                  <input type="text" value={policyForm.policy_code} onChange={(e) => setPolicyForm({ ...policyForm, policy_code: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">策略名称 *</label>
                  <input type="text" value={policyForm.policy_name} onChange={(e) => setPolicyForm({ ...policyForm, policy_name: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">效果</label>
                  <select value={policyForm.effect} onChange={(e) => setPolicyForm({ ...policyForm, effect: e.target.value as "allow" | "deny" })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                    <option value="deny">拒绝 (deny)</option>
                    <option value="allow">允许 (allow)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">优先级</label>
                  <input type="number" value={policyForm.priority} onChange={(e) => setPolicyForm({ ...policyForm, priority: parseInt(e.target.value) || 0 })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
              <div>
                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input type="checkbox" checked={policyForm.is_system} onChange={(e) => setPolicyForm({ ...policyForm, is_system: e.target.checked })} className="rounded" />
                  系统策略（不可删除）
                </label>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">条件 (conditions_json)</label>
                <textarea value={policyForm.conditions_json} onChange={(e) => setPolicyForm({ ...policyForm, conditions_json: e.target.value })} rows={6} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm font-mono" placeholder='{"operator": "AND", "rules": [...]}' />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowPolicyDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSavePolicy} disabled={!policyForm.policy_code || !policyForm.policy_name || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 创建绑定对话框 ===== */}
      {showBindingDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowBindingDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">创建主体角色绑定</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">主体类型</label>
                <select value={bindingForm.subject_type} onChange={(e) => setBindingForm({ ...bindingForm, subject_type: e.target.value as "user" | "party" })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="user">用户</option>
                  <option value="party">Party</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">主体 ID *</label>
                <input type="text" value={bindingForm.subject_id} onChange={(e) => setBindingForm({ ...bindingForm, subject_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="UUID" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">角色 ID *</label>
                <select value={bindingForm.role_id} onChange={(e) => setBindingForm({ ...bindingForm, role_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                  <option value="">请选择</option>
                  {roles.map((r) => (
                    <option key={r.id} value={r.id}>{r.role_code} — {r.role_name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">作用域 Party ID</label>
                <input type="text" value={bindingForm.scope_party_id} onChange={(e) => setBindingForm({ ...bindingForm, scope_party_id: e.target.value })} className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="空 = 全局" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowBindingDialog(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleSaveBinding} disabled={!bindingForm.subject_id || !bindingForm.role_id || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 删除确认 */}
      <ConfirmDialog
        open={!!confirmDelete}
        title={`确认删除${confirmDelete?.type === "role" ? "角色" : confirmDelete?.type === "policy" ? "策略" : "绑定"}`}
        message={`确定要删除此${confirmDelete?.type === "role" ? "角色" : confirmDelete?.type === "policy" ? "策略" : "绑定"}吗？此操作不可撤销。`}
        danger
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
