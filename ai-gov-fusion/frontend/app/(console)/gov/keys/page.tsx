"use client";

import React from "react";
import {
  Key,
  Plus,
  RotateCw,
  Ban,
  Copy,
  Check,
  Eye,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";
import { govFetch, govFetchJSON } from "@/lib/gov-api";

/** 网关密钥数据 */
interface GatewayKey extends Record<string, unknown> {
  id: string;
  name: string;
  key_prefix: string;
  status: "active" | "revoked" | "expired";
  account_id: string;
  created_at: string;
  expires_at: string | null;
  last_used_at: string | null;
}

/** 密钥详情 */
interface GatewayKeyDetail extends GatewayKey {
  key_full: string;
  description: string | null;
  tags: string[];
  rotated_at: string | null;
  rotated_from_key_id: string | null;
}

/**
 * 网关密钥管理页面 —— 密钥列表、详情查看、吊销、轮换操作。
 * 对应 PRD UI-05 需求。
 */
export default function KeysPage() {
  // 列表状态
  const [keys, setKeys] = React.useState<GatewayKey[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 详情弹窗
  const [selectedKey, setSelectedKey] = React.useState<GatewayKeyDetail | null>(null);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [showDetail, setShowDetail] = React.useState(false);

  // 创建对话框
  const [showCreate, setShowCreate] = React.useState(false);
  const [createForm, setCreateForm] = React.useState({ name: "", description: "", expires_in_days: "" });
  const [createdKey, setCreatedKey] = React.useState<string | null>(null);
  const [saving, setSaving] = React.useState(false);

  // 操作确认
  const [confirmAction, setConfirmAction] = React.useState<{
    action: "revoke" | "rotate";
    keyId: string;
    keyName: string;
  } | null>(null);
  const [actionLoading, setActionLoading] = React.useState(false);

  // 复制成功标记
  const [copied, setCopied] = React.useState(false);

  // 获取密钥列表
  const fetchKeys = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      const json = await govFetchJSON<{ data: GatewayKey[]; total: number }>(`/keys?${params}`);
      setKeys(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取密钥列表失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  React.useEffect(() => { fetchKeys(); }, [fetchKeys]);

  // 获取密钥详情
  const fetchDetail = async (keyId: string) => {
    setDetailLoading(true);
    try {
      const json = await govFetchJSON<GatewayKeyDetail>(`/keys/${keyId}`);
      setSelectedKey(json);
      setShowDetail(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取密钥详情失败");
    } finally {
      setDetailLoading(false);
    }
  };

  // 创建密钥
  const handleCreate = async () => {
    setSaving(true);
    setCreatedKey(null);
    try {
      const body: Record<string, unknown> = { name: createForm.name };
      if (createForm.description) body.description = createForm.description;
      if (createForm.expires_in_days) body.expires_in_days = parseInt(createForm.expires_in_days);

      const json = await govFetchJSON<{ key_full?: string; id: string }>("/keys", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setCreatedKey(json.key_full ?? json.id);
      fetchKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建密钥失败");
    } finally {
      setSaving(false);
    }
  };

  // 吊销密钥
  const handleRevoke = async () => {
    if (!confirmAction) return;
    setActionLoading(true);
    try {
      await govFetch(`/keys/${confirmAction.keyId}`, {
        method: "DELETE",
      });
      setConfirmAction(null);
      setShowDetail(false);
      fetchKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : "吊销密钥失败");
    } finally {
      setActionLoading(false);
    }
  };

  // 轮换密钥
  const handleRotate = async () => {
    if (!confirmAction) return;
    setActionLoading(true);
    try {
      await govFetch(`/keys/${confirmAction.keyId}/rotate`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      setConfirmAction(null);
      setShowDetail(false);
      fetchKeys();
    } catch (err) {
      setError(err instanceof Error ? err.message : "轮换密钥失败");
    } finally {
      setActionLoading(false);
    }
  };

  // 复制密钥到剪贴板
  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch { /* 静默失败 */ }
  };

  // 格式化时间
  const formatTime = (iso: string | null) => iso ? new Date(iso).toLocaleString("zh-CN") : "-";

  // 状态标签渲染
  const statusBadge = (status: string) => {
    const map: Record<string, { label: string; cls: string }> = {
      active: { label: "正常", cls: "bg-green-100 text-green-800" },
      revoked: { label: "已吊销", cls: "bg-red-100 text-red-800" },
      expired: { label: "已过期", cls: "bg-gray-100 text-gray-600" },
    };
    const s = map[status] ?? { label: status, cls: "bg-gray-100 text-gray-600" };
    return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${s.cls}`}>{s.label}</span>;
  };

  // 表格列定义
  const columns: ColumnDef<GatewayKey>[] = [
    { key: "name", header: "名称" },
    {
      key: "key_prefix",
      header: "密钥前缀",
      render: (k) => <span className="font-mono text-xs">{k.key_prefix}...</span>,
    },
    {
      key: "status",
      header: "状态",
      render: (k) => statusBadge(k.status),
    },
    {
      key: "created_at",
      header: "创建时间",
      render: (k) => formatTime(k.created_at),
    },
    {
      key: "expires_at",
      header: "过期时间",
      render: (k) => formatTime(k.expires_at),
    },
    {
      key: "actions",
      header: "操作",
      render: (k) => k.status === "active" ? (
        <div className="flex items-center gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setConfirmAction({ action: "rotate", keyId: k.id, keyName: k.name }); }}
            className="rounded p-1 text-gray-400 hover:text-blue-600"
            title="轮换密钥"
          >
            <RotateCw className="h-4 w-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); setConfirmAction({ action: "revoke", keyId: k.id, keyName: k.name }); }}
            className="rounded p-1 text-gray-400 hover:text-red-600"
            title="吊销密钥"
          >
            <Ban className="h-4 w-4" />
          </button>
        </div>
      ) : <span className="text-xs text-gray-400">-</span>,
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Key 管理</h1>
          <p className="mt-1 text-sm text-gray-500">管理网关密钥，支持创建、吊销和轮换操作</p>
        </div>
        <button
          onClick={() => { setCreatedKey(null); setCreateForm({ name: "", description: "", expires_in_days: "" }); setShowCreate(true); }}
          className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" /> 创建密钥
        </button>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchKeys} dismissible />}

      {/* 密钥列表 */}
      <DataTable
        data={keys}
        columns={columns}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无密钥数据"
        onRowClick={(k) => fetchDetail(k.id)}
      />

      {/* ===== 密钥详情弹窗 ===== */}
      {showDetail && selectedKey && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowDetail(false)} />
          <div className="relative z-10 mx-4 w-full max-w-2xl rounded-lg bg-white p-6 shadow-xl max-h-[85vh] overflow-y-auto">
            {detailLoading ? (
              <div className="flex items-center justify-center py-10">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-semibold text-gray-900">密钥详情</h2>
                  {statusBadge(selectedKey.status)}
                </div>

                {/* 信息网格 */}
                <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-gray-500">名称</span>
                    <p className="font-medium">{selectedKey.name}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">密钥前缀</span>
                    <p className="font-mono text-xs">{selectedKey.key_prefix}...</p>
                  </div>
                  <div>
                    <span className="text-gray-500">所属账户</span>
                    <p className="font-mono text-xs">{selectedKey.account_id}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">创建时间</span>
                    <p>{formatTime(selectedKey.created_at)}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">过期时间</span>
                    <p>{formatTime(selectedKey.expires_at)}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">最后使用</span>
                    <p>{formatTime(selectedKey.last_used_at)}</p>
                  </div>
                  {selectedKey.rotated_at && (
                    <div>
                      <span className="text-gray-500">上次轮换</span>
                      <p>{formatTime(selectedKey.rotated_at)}</p>
                    </div>
                  )}
                </div>

                {/* 完整密钥（仅创建后可查看） */}
                {selectedKey.key_full && (
                  <div className="mt-4">
                    <span className="text-sm text-gray-500">完整密钥</span>
                    <div className="mt-1 flex items-center gap-2 rounded border border-gray-200 bg-gray-50 px-3 py-2">
                      <code className="flex-1 break-all text-sm font-mono">{selectedKey.key_full}</code>
                      <button
                        onClick={() => handleCopy(selectedKey.key_full!)}
                        className="flex-shrink-0 rounded p-1 text-gray-400 hover:text-blue-600"
                      >
                        {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </button>
                    </div>
                    <p className="mt-1 text-xs text-yellow-700">请立即复制密钥，关闭后将无法再次查看完整密钥。</p>
                  </div>
                )}

                {/* 描述 */}
                {selectedKey.description && (
                  <div className="mt-4">
                    <span className="text-sm text-gray-500">描述</span>
                    <p className="mt-1 text-sm">{selectedKey.description}</p>
                  </div>
                )}

                {/* 操作按钮 */}
                {selectedKey.status === "active" && (
                  <div className="mt-6 flex gap-3">
                    <button
                      onClick={() => setConfirmAction({ action: "rotate", keyId: selectedKey.id, keyName: selectedKey.name })}
                      className="inline-flex items-center gap-1.5 rounded-md border border-blue-300 px-4 py-2 text-sm font-medium text-blue-700 hover:bg-blue-50"
                    >
                      <RotateCw className="h-4 w-4" /> 轮换密钥
                    </button>
                    <button
                      onClick={() => setConfirmAction({ action: "revoke", keyId: selectedKey.id, keyName: selectedKey.name })}
                      className="inline-flex items-center gap-1.5 rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
                    >
                      <Ban className="h-4 w-4" /> 吊销密钥
                    </button>
                  </div>
                )}
              </>
            )}

            <div className="mt-4 flex justify-end">
              <button
                onClick={() => setShowDetail(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 创建密钥对话框 ===== */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowCreate(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            {createdKey ? (
              <>
                <h2 className="text-lg font-semibold text-gray-900">密钥创建成功</h2>
                <div className="mt-4">
                  <span className="text-sm text-gray-500">完整密钥（仅此一次可见）</span>
                  <div className="mt-1 flex items-center gap-2 rounded border border-gray-200 bg-yellow-50 px-3 py-2">
                    <code className="flex-1 break-all text-sm font-mono">{createdKey}</code>
                    <button
                      onClick={() => handleCopy(createdKey)}
                      className="flex-shrink-0 rounded p-1 text-gray-400 hover:text-blue-600"
                    >
                      {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                    </button>
                  </div>
                  <p className="mt-2 text-xs text-yellow-700">请立即复制并安全保存。关闭对话框后将无法再次查看。</p>
                </div>
                <div className="mt-6 flex justify-end">
                  <button
                    onClick={() => { setShowCreate(false); setCreatedKey(null); }}
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                  >
                    关闭
                  </button>
                </div>
              </>
            ) : (
              <>
                <h2 className="text-lg font-semibold text-gray-900">创建网关密钥</h2>
                <div className="mt-4 space-y-3">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">密钥名称 *</label>
                    <input type="text" value={createForm.name} onChange={(e) => setCreateForm({ ...createForm, name: e.target.value })}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" placeholder="如 production-gateway-key" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700">描述</label>
                    <textarea value={createForm.description} onChange={(e) => setCreateForm({ ...createForm, description: e.target.value })}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" rows={2} />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700">过期天数（留空不过期）</label>
                    <input type="number" value={createForm.expires_in_days} onChange={(e) => setCreateForm({ ...createForm, expires_in_days: e.target.value })}
                      className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm" min="1" />
                  </div>
                </div>
                <div className="mt-6 flex justify-end gap-3">
                  <button onClick={() => setShowCreate(false)} className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50">取消</button>
                  <button onClick={handleCreate} disabled={!createForm.name || saving} className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                    {saving ? "创建中..." : "创建"}
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* 吊销/轮换确认 */}
      <ConfirmDialog
        open={!!confirmAction}
        title={confirmAction?.action === "revoke" ? "确认吊销密钥" : "确认轮换密钥"}
        message={
          confirmAction?.action === "revoke"
            ? `确定要吊销密钥「${confirmAction?.keyName}」吗？吊销后使用该密钥的请求将被拒绝。`
            : `确定要轮换密钥「${confirmAction?.keyName}」吗？轮换后将生成新密钥，旧密钥仍保持有效。`
        }
        danger={confirmAction?.action === "revoke"}
        confirmLabel={confirmAction?.action === "revoke" ? "吊销" : "轮换"}
        loading={actionLoading}
        onConfirm={confirmAction?.action === "revoke" ? handleRevoke : handleRotate}
        onCancel={() => setConfirmAction(null)}
      />
    </div>
  );
}