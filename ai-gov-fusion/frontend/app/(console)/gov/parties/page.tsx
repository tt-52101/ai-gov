"use client";

import React from "react";
import { Plus, Trash2, Users, Link2, Search } from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";

/** Party 数据结构 */
interface Party {
  id: string;
  type: "org" | "project";
  name: string;
  parent_party_id: string | null;
  leader_user_id: string | null;
  status: "active" | "archived" | "liquidating";
  created_at: string;
}

/** Party 关系边数据 */
interface PartyEdge {
  id: string;
  src_party_id: string;
  dst_party_id: string;
  edge_type: string;
  allows_fund: boolean;
  created_at: string;
}

/** Party 成员数据 */
interface PartyMember {
  id: string;
  party_id: string;
  user_id: string;
  user_name: string;
  role: "leader" | "member" | "observer";
  is_primary: boolean;
  joined_at: string;
}

/** 边类型中文标签映射 */
const edgeTypeLabels: Record<string, string> = {
  parent: "上级-下级",
  sponsors: "出资",
  owns: "主办",
  participates: "协作",
  allocates: "个人划拨",
  merged_into: "合并并入",
  split_from: "拆出",
};

/** API 基础路径 */
const API_BASE = "/gov";

/**
 * Party 管理页面 —— 组织/项目列表、创建、关系边管理、成员管理。
 * 对应 PRD UI-02 需求。
 */
export default function PartiesPage() {
  // 数据状态
  const [parties, setParties] = React.useState<Party[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [typeFilter, setTypeFilter] = React.useState<string>("");
  const [searchQuery, setSearchQuery] = React.useState("");

  // 模态状态
  const [showCreateDialog, setShowCreateDialog] = React.useState(false);
  const [showEdgePanel, setShowEdgePanel] = React.useState(false);
  const [showMemberPanel, setShowMemberPanel] = React.useState(false);
  const [selectedParty, setSelectedParty] = React.useState<Party | null>(null);

  // 边管理状态
  const [edges, setEdges] = React.useState<PartyEdge[]>([]);
  const [edgesLoading, setEdgesLoading] = React.useState(false);

  // 成员管理状态
  const [members, setMembers] = React.useState<PartyMember[]>([]);
  const [membersLoading, setMembersLoading] = React.useState(false);

  // 创建表单状态
  const [createForm, setCreateForm] = React.useState({
    type: "org" as "org" | "project",
    name: "",
    description: "",
    parent_party_id: "",
    leader_user_id: "",
    cost_center: "",
  });
  const [creating, setCreating] = React.useState(false);

  // 删除确认
  const [confirmDelete, setConfirmDelete] = React.useState<Party | null>(null);
  const [deleting, setDeleting] = React.useState(false);

  // 边创建表单
  const [edgeForm, setEdgeForm] = React.useState({
    dst_party_id: "",
    edge_type: "parent",
    allows_fund: true,
  });
  const [creatingEdge, setCreatingEdge] = React.useState(false);

  // 成员添加表单
  const [memberForm, setMemberForm] = React.useState({
    user_id: "",
    role: "member" as "leader" | "member" | "observer",
  });
  const [addingMember, setAddingMember] = React.useState(false);

  // 获取 Party 列表
  const fetchParties = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      if (typeFilter) params.set("type", typeFilter);
      if (searchQuery) params.set("search", searchQuery);
      const res = await fetch(`${API_BASE}/parties?${params}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setParties(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取 Party 列表失败");
    } finally {
      setLoading(false);
    }
  }, [page, typeFilter, searchQuery]);

  // 初始加载
  React.useEffect(() => {
    fetchParties();
  }, [fetchParties]);

  // 获取边列表
  const fetchEdges = React.useCallback(async (partyId: string) => {
    setEdgesLoading(true);
    try {
      const res = await fetch(`${API_BASE}/party-edges?party_id=${partyId}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setEdges(json.data ?? []);
    } catch {
      setEdges([]);
    } finally {
      setEdgesLoading(false);
    }
  }, []);

  // 获取成员列表
  const fetchMembers = React.useCallback(async (partyId: string) => {
    setMembersLoading(true);
    try {
      const res = await fetch(`${API_BASE}/party-members?party_id=${partyId}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setMembers(json.data ?? []);
    } catch {
      setMembers([]);
    } finally {
      setMembersLoading(false);
    }
  }, []);

  // 创建 Party
  const handleCreate = async () => {
    setCreating(true);
    try {
      const body: Record<string, unknown> = {
        type: createForm.type,
        name: createForm.name,
      };
      if (createForm.description) body.description = createForm.description;
      if (createForm.parent_party_id) body.parent_party_id = createForm.parent_party_id;
      if (createForm.leader_user_id) body.leader_user_id = createForm.leader_user_id;
      if (createForm.cost_center) body.cost_center = createForm.cost_center;

      const res = await fetch(`${API_BASE}/parties`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        throw new Error(await extractErrorMessage(res));
      }
      setShowCreateDialog(false);
      setCreateForm({ type: "org", name: "", description: "", parent_party_id: "", leader_user_id: "", cost_center: "" });
      fetchParties();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建 Party 失败");
    } finally {
      setCreating(false);
    }
  };

  // 创建关系边
  const handleCreateEdge = async () => {
    if (!selectedParty) return;
    setCreatingEdge(true);
    try {
      const res = await fetch(`${API_BASE}/party-edges`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          src_party_id: selectedParty.id,
          dst_party_id: edgeForm.dst_party_id,
          edge_type: edgeForm.edge_type,
          allows_fund: edgeForm.allows_fund,
        }),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setEdgeForm({ dst_party_id: "", edge_type: "parent", allows_fund: true });
      fetchEdges(selectedParty.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建关系边失败");
    } finally {
      setCreatingEdge(false);
    }
  };

  // 删除关系边
  const handleDeleteEdge = async (edgeId: string) => {
    try {
      const res = await fetch(`${API_BASE}/party-edges/${edgeId}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      if (selectedParty) fetchEdges(selectedParty.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除关系边失败");
    }
  };

  // 添加成员
  const handleAddMember = async () => {
    if (!selectedParty) return;
    setAddingMember(true);
    try {
      const res = await fetch(`${API_BASE}/party-members`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          party_id: selectedParty.id,
          user_id: memberForm.user_id,
          role: memberForm.role,
        }),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setMemberForm({ user_id: "", role: "member" });
      fetchMembers(selectedParty.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "添加成员失败");
    } finally {
      setAddingMember(false);
    }
  };

  // 移除成员
  const handleRemoveMember = async (memberId: string) => {
    try {
      const res = await fetch(`${API_BASE}/party-members/${memberId}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      if (selectedParty) fetchMembers(selectedParty.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "移除成员失败");
    }
  };

  // 删除 Party
  const handleDeleteParty = async () => {
    if (!confirmDelete) return;
    setDeleting(true);
    try {
      const res = await fetch(`${API_BASE}/parties/${confirmDelete.id}`, { method: "DELETE" });
      if (!res.ok) {
        const errBody = await res.json().catch(() => null);
        throw new Error(errBody?.error?.message ?? `HTTP ${res.status}: 删除失败`);
      }
      setConfirmDelete(null);
      fetchParties();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除 Party 失败");
      setConfirmDelete(null); // 关闭对话框让用户看到错误
    } finally {
      setDeleting(false);
    }
  };

  // 打开边管理面板
  const openEdgePanel = (party: Party) => {
    setSelectedParty(party);
    setShowEdgePanel(true);
    fetchEdges(party.id);
  };

  // 打开成员管理面板
  const openMemberPanel = (party: Party) => {
    setSelectedParty(party);
    setShowMemberPanel(true);
    fetchMembers(party.id);
  };

  // 表格列定义
  const columns: ColumnDef<Party>[] = [
    { key: "name", header: "名称" },
    {
      key: "type",
      header: "类型",
      render: (p) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
            p.type === "org"
              ? "bg-blue-100 text-blue-800"
              : "bg-purple-100 text-purple-800"
          }`}
        >
          {p.type === "org" ? "组织" : "项目"}
        </span>
      ),
    },
    {
      key: "status",
      header: "状态",
      render: (p) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
            p.status === "active"
              ? "bg-green-100 text-green-800"
              : "bg-gray-100 text-gray-600"
          }`}
        >
          {p.status === "active" ? "活跃" : p.status}
        </span>
      ),
    },
    {
      key: "actions",
      header: "操作",
      render: (p) => (
        <div className="flex gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); openEdgePanel(p); }}
            className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
            title="管理关系边"
          >
            <Link2 className="h-4 w-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); openMemberPanel(p); }}
            className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
            title="管理成员"
          >
            <Users className="h-4 w-4" />
          </button>
          {/* 删除按钮：仅非 active 状态的 party 允许删除，active 状态弹出提示 */}
          <button
            onClick={(e) => { e.stopPropagation(); setConfirmDelete(p); }}
            className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
            title="删除 Party"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题栏 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Party 管理</h1>
          <p className="mt-1 text-sm text-gray-500">管理组织、项目及其关系和成员</p>
        </div>
        <div className="flex items-center gap-3">
          {/* 类型筛选 */}
          <select
            value={typeFilter}
            onChange={(e) => { setTypeFilter(e.target.value); setPage(1); }}
            className="rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700"
          >
            <option value="">全部类型</option>
            <option value="org">组织</option>
            <option value="project">项目</option>
          </select>
          {/* 创建按钮 */}
          <button
            onClick={() => setShowCreateDialog(true)}
            className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          >
            <Plus className="h-4 w-4" />
            创建 Party
          </button>
        </div>
      </div>

      {/* 错误提示 */}
      {error && (
        <ErrorAlert message={error} onRetry={fetchParties} dismissible />
      )}

      {/* Party 列表表格 */}
      <DataTable
        data={parties}
        columns={columns}
        searchPlaceholder="搜索名称或描述..."
        onSearch={(q) => { setSearchQuery(q); setPage(1); }}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无 Party 数据"
      />

      {/* ===== 创建 Party 对话框 ===== */}
      {showCreateDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowCreateDialog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-lg rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">创建 Party</h2>
            <div className="mt-4 space-y-4">
              {/* 类型选择 */}
              <div>
                <label className="block text-sm font-medium text-gray-700">类型</label>
                <select
                  value={createForm.type}
                  onChange={(e) => setCreateForm({ ...createForm, type: e.target.value as "org" | "project" })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                >
                  <option value="org">组织 (org)</option>
                  <option value="project">项目 (project)</option>
                </select>
              </div>
              {/* 名称 */}
              <div>
                <label className="block text-sm font-medium text-gray-700">名称 *</label>
                <input
                  type="text"
                  value={createForm.name}
                  onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                  placeholder="输入 Party 名称"
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              {/* 描述 */}
              <div>
                <label className="block text-sm font-medium text-gray-700">描述</label>
                <textarea
                  value={createForm.description}
                  onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                  rows={2}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              {/* 父级 Party */}
              <div>
                <label className="block text-sm font-medium text-gray-700">父级 Party ID</label>
                <input
                  type="text"
                  value={createForm.parent_party_id}
                  onChange={(e) => setCreateForm({ ...createForm, parent_party_id: e.target.value })}
                  placeholder="可选，UUID"
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              {/* 负责人 */}
              <div>
                <label className="block text-sm font-medium text-gray-700">负责人 ID</label>
                <input
                  type="text"
                  value={createForm.leader_user_id}
                  onChange={(e) => setCreateForm({ ...createForm, leader_user_id: e.target.value })}
                  placeholder="可选，UUID"
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              {/* 成本中心 */}
              <div>
                <label className="block text-sm font-medium text-gray-700">成本中心</label>
                <input
                  type="text"
                  value={createForm.cost_center}
                  onChange={(e) => setCreateForm({ ...createForm, cost_center: e.target.value })}
                  placeholder="如 CC-AI-001"
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setShowCreateDialog(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                disabled={!createForm.name || creating}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {creating ? "创建中..." : "确认创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 关系边管理面板 ===== */}
      {showEdgePanel && selectedParty && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowEdgePanel(false)} />
          <div className="relative z-10 mx-4 w-full max-w-2xl rounded-lg bg-white p-6 shadow-xl max-h-[80vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-gray-900">
              关系边管理 — {selectedParty.name}
            </h2>

            {/* 添加边表单 */}
            <div className="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-4">
              <h3 className="text-sm font-medium text-gray-700">添加关系边</h3>
              <div className="mt-3 grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-gray-500">目标 Party ID</label>
                  <input
                    type="text"
                    value={edgeForm.dst_party_id}
                    onChange={(e) => setEdgeForm({ ...edgeForm, dst_party_id: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
                    placeholder="UUID"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500">边类型</label>
                  <select
                    value={edgeForm.edge_type}
                    onChange={(e) => setEdgeForm({ ...edgeForm, edge_type: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
                  >
                    {Object.entries(edgeTypeLabels).map(([k, v]) => (
                      <option key={k} value={k}>{v} ({k})</option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="mt-3 flex items-center gap-2">
                <input
                  type="checkbox"
                  id="allows-fund"
                  checked={edgeForm.allows_fund}
                  onChange={(e) => setEdgeForm({ ...edgeForm, allows_fund: e.target.checked })}
                  className="rounded"
                />
                <label htmlFor="allows-fund" className="text-sm text-gray-600">自动开通资金通道</label>
              </div>
              <button
                onClick={handleCreateEdge}
                disabled={!edgeForm.dst_party_id || creatingEdge}
                className="mt-3 rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {creatingEdge ? "创建中..." : "添加边"}
              </button>
            </div>

            {/* 已有关系列表 */}
            <div className="mt-4">
              <h3 className="text-sm font-medium text-gray-700">已有关条边</h3>
              {edgesLoading ? (
                <p className="mt-2 text-sm text-gray-400">加载中...</p>
              ) : edges.length === 0 ? (
                <p className="mt-2 text-sm text-gray-400">暂无关系边</p>
              ) : (
                <ul className="mt-2 divide-y divide-gray-100">
                  {edges.map((edge) => (
                    <li key={edge.id} className="flex items-center justify-between py-2">
                      <div className="text-sm">
                        <span className="font-medium">{edgeTypeLabels[edge.edge_type] ?? edge.edge_type}</span>
                        <span className="ml-2 text-gray-500">
                          → {edge.dst_party_id.slice(0, 8)}...
                        </span>
                        {edge.allows_fund && (
                          <span className="ml-2 inline-flex rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-700">
                            资金通道
                          </span>
                        )}
                      </div>
                      <button
                        onClick={() => handleDeleteEdge(edge.id)}
                        className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ===== 成员管理面板 ===== */}
      {showMemberPanel && selectedParty && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowMemberPanel(false)} />
          <div className="relative z-10 mx-4 w-full max-w-lg rounded-lg bg-white p-6 shadow-xl max-h-[80vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-gray-900">
              成员管理 — {selectedParty.name}
            </h2>

            {/* 添加成员表单 */}
            <div className="mt-4 rounded-lg border border-gray-200 bg-gray-50 p-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-gray-500">用户 ID</label>
                  <input
                    type="text"
                    value={memberForm.user_id}
                    onChange={(e) => setMemberForm({ ...memberForm, user_id: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
                    placeholder="UUID"
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500">角色</label>
                  <select
                    value={memberForm.role}
                    onChange={(e) => setMemberForm({ ...memberForm, role: e.target.value as "leader" | "member" | "observer" })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm"
                  >
                    <option value="member">成员</option>
                    <option value="leader">负责人</option>
                    <option value="observer">观察者</option>
                  </select>
                </div>
              </div>
              <button
                onClick={handleAddMember}
                disabled={!memberForm.user_id || addingMember}
                className="mt-3 rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {addingMember ? "添加中..." : "添加成员"}
              </button>
            </div>

            {/* 成员列表 */}
            <div className="mt-4">
              <h3 className="text-sm font-medium text-gray-700">当前成员</h3>
              {membersLoading ? (
                <p className="mt-2 text-sm text-gray-400">加载中...</p>
              ) : members.length === 0 ? (
                <p className="mt-2 text-sm text-gray-400">暂无成员</p>
              ) : (
                <ul className="mt-2 divide-y divide-gray-100">
                  {members.map((m) => (
                    <li key={m.id} className="flex items-center justify-between py-2">
                      <div className="text-sm">
                        <span className="font-medium">{m.user_name}</span>
                        <span className="ml-2 text-gray-400">({m.user_id.slice(0, 8)}...)</span>
                        <span className={`ml-2 inline-flex rounded px-1.5 py-0.5 text-xs font-medium ${
                          m.role === "leader" ? "bg-yellow-100 text-yellow-800" :
                          m.role === "observer" ? "bg-gray-100 text-gray-600" :
                          "bg-blue-100 text-blue-800"
                        }`}>
                          {m.role === "leader" ? "负责人" : m.role === "observer" ? "观察者" : "成员"}
                        </span>
                      </div>
                      <button
                        onClick={() => handleRemoveMember(m.id)}
                        className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-600"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      )}

      {/* 删除确认弹窗 */}
      <ConfirmDialog
        open={!!confirmDelete}
        title="确认删除"
        message={
          confirmDelete?.status === "active"
            ? `Party "${confirmDelete?.name}" 当前为"活跃"状态，建议先归档后再删除。确定要强制删除吗？`
            : `确定要删除 Party "${confirmDelete?.name}" 吗？此操作不可撤销，关联数据将被级联清理。`
        }
        danger
        confirmLabel="删除"
        loading={deleting}
        onConfirm={handleDeleteParty}
        onCancel={() => { if (!deleting) setConfirmDelete(null); }}
      />
    </div>
  );
}
