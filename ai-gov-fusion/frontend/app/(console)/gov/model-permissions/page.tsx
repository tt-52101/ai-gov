"use client";

import React from "react";
import {
  Brain,
  Plus,
  Trash2,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";

/** 模型授权规则 */
interface ModelGrantRule extends Record<string, unknown> {
  id: string;
  model_id: string;
  model_name: string;
  model_provider: string;
  party_id: string | null;
  party_name: string | null;
  effect: "allow" | "deny";
  priority: number;
  created_at: string;
}

/** 模型列表 */
interface ModelItem extends Record<string, unknown> {
  id: string;
  model_name: string;
  provider: string;
  grant_count: number;
  access_count: number;
  latest_decision: "allow" | "deny" | null;
}

const API_BASE = "/gov";

/**
 * 模型权限管理页面 —— ModelGrant 规则列表、模型访问总览。
 * 对应 PRD UI-09 需求。
 */
export default function ModelPermissionsPage() {
  // 标签页
  const [tab, setTab] = React.useState<"grants" | "models">("grants");

  // 授权规则状态
  const [rules, setRules] = React.useState<ModelGrantRule[]>([]);
  const [rulesTotal, setRulesTotal] = React.useState(0);
  const [rulesPage, setRulesPage] = React.useState(1);
  const [rulesLoading, setRulesLoading] = React.useState(true);

  // 模型列表状态
  const [models, setModels] = React.useState<ModelItem[]>([]);
  const [modelsTotal, setModelsTotal] = React.useState(0);
  const [modelsPage, setModelsPage] = React.useState(1);
  const [modelsLoading, setModelsLoading] = React.useState(true);

  // 创建对话框
  const [showCreate, setShowCreate] = React.useState(false);
  const [createForm, setCreateForm] = React.useState({
    model_id: "",
    party_id: "",
    effect: "deny" as "allow" | "deny",
    priority: 0,
  });
  const [saving, setSaving] = React.useState(false);

  // 删除确认
  const [confirmDelete, setConfirmDelete] = React.useState<{ id: string; label: string } | null>(null);

  const [error, setError] = React.useState<string | null>(null);

  // 获取授权规则
  const fetchRules = React.useCallback(async () => {
    setRulesLoading(true);
    try {
      const params = new URLSearchParams({ page: String(rulesPage), page_size: "20" });
      const res = await fetch(`${API_BASE}/model-grants?${params}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setRules(json.data ?? []);
      setRulesTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取授权规则失败");
    } finally {
      setRulesLoading(false);
    }
  }, [rulesPage]);

  // 获取模型列表
  const fetchModels = React.useCallback(async () => {
    setModelsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(modelsPage), page_size: "20" });
      const res = await fetch(`${API_BASE}/models?${params}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setModels(json.data ?? []);
      setModelsTotal(json.total ?? 0);
    } catch {
      setModels([]);
    } finally {
      setModelsLoading(false);
    }
  }, [modelsPage]);

  React.useEffect(() => {
    setError(null);
    if (tab === "grants") fetchRules();
    else fetchModels();
  }, [tab, fetchRules, fetchModels]);

  // 创建授权规则
  const handleCreate = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/model-grants`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(createForm),
      });
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      setShowCreate(false);
      fetchRules();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建授权规则失败");
    } finally {
      setSaving(false);
    }
  };

  // 删除规则
  const handleDelete = async () => {
    if (!confirmDelete) return;
    try {
      await fetch(`${API_BASE}/model-grants/${confirmDelete.id}`, { method: "DELETE" });
      setConfirmDelete(null);
      fetchRules();
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除授权规则失败");
    }
  };

  const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

  // 效果标签
  const effectBadge = (effect: string) => (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
      effect === "deny" ? "bg-red-100 text-red-800" : "bg-green-100 text-green-800"
    }`}>
      {effect === "deny" ? "拒绝" : "允许"}
    </span>
  );

  // 授权规则表格列
  const ruleColumns: ColumnDef<ModelGrantRule>[] = [
    { key: "model_name", header: "模型" },
    { key: "model_provider", header: "提供商" },
    { key: "party_name", header: "Party", render: (r) => r.party_name ?? <span className="text-gray-400">全局</span> },
    { key: "effect", header: "效果", render: (r) => effectBadge(r.effect) },
    { key: "priority", header: "优先级" },
    { key: "created_at", header: "创建时间", render: (r) => formatTime(r.created_at) },
    {
      key: "actions",
      header: "操作",
      render: (r) => (
        <button
          onClick={(e) => { e.stopPropagation(); setConfirmDelete({ id: r.id, label: `${r.model_name} - ${r.effect}` }); }}
          className="rounded p-1 text-gray-400 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      ),
    },
  ];

  // 模型列表表格列
  const modelColumns: ColumnDef<ModelItem>[] = [
    { key: "model_name", header: "模型名称" },
    { key: "provider", header: "提供商" },
    { key: "grant_count", header: "授权规则数" },
    { key: "access_count", header: "访问次数" },
    {
      key: "latest_decision",
      header: "最近决策",
      render: (m) => m.latest_decision ? effectBadge(m.latest_decision) : <span className="text-gray-400">-</span>,
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">模型权限管理</h1>
        <p className="mt-1 text-sm text-gray-500">管理 ModelGrant 授权规则，控制模型访问权限</p>
      </div>

      {error && <ErrorAlert message={error} dismissible />}

      {/* 标签导航 */}
      <div className="flex border-b border-gray-200">
        {([
          { key: "grants" as const, label: "授权规则", icon: ToggleRight },
          { key: "models" as const, label: "模型列表", icon: Brain },
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

      {/* ===== 授权规则 ===== */}
      {tab === "grants" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <button
              onClick={() => {
                setCreateForm({ model_id: "", party_id: "", effect: "deny", priority: 0 });
                setShowCreate(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              <Plus className="h-4 w-4" /> 创建规则
            </button>
          </div>
          <DataTable
            data={rules}
            columns={ruleColumns}
            page={rulesPage}
            pageSize={20}
            total={rulesTotal}
            onPageChange={setRulesPage}
            loading={rulesLoading}
            emptyText="暂无授权规则"
          />
        </div>
      )}

      {/* ===== 模型列表 ===== */}
      {tab === "models" && (
        <DataTable
          data={models}
          columns={modelColumns}
          page={modelsPage}
          pageSize={20}
          total={modelsTotal}
          onPageChange={setModelsPage}
          loading={modelsLoading}
          emptyText="暂无模型数据"
        />
      )}

      {/* ===== 创建规则对话框 ===== */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowCreate(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">创建授权规则</h2>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">模型 ID *</label>
                <input type="text" value={createForm.model_id} onChange={(e) => setCreateForm({ ...createForm, model_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="模型 UUID" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Party ID（留空为全局规则）</label>
                <input type="text" value={createForm.party_id} onChange={(e) => setCreateForm({ ...createForm, party_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="空 = 全局" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700">效果</label>
                  <select value={createForm.effect} onChange={(e) => setCreateForm({ ...createForm, effect: e.target.value as "allow" | "deny" })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm">
                    <option value="deny">拒绝 (deny)</option>
                    <option value="allow">允许 (allow)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">优先级</label>
                  <input type="number" value={createForm.priority} onChange={(e) => setCreateForm({ ...createForm, priority: parseInt(e.target.value) || 0 })}
                    className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" />
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button onClick={() => setShowCreate(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
              <button onClick={handleCreate} disabled={!createForm.model_id || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {saving ? "保存中..." : "创建"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 删除确认 */}
      <ConfirmDialog
        open={!!confirmDelete}
        title="确认删除授权规则"
        message={`确定要删除规则「${confirmDelete?.label}」吗？`}
        danger
        confirmLabel="删除"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}