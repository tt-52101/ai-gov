"use client";

import React from "react";
import {
  ClipboardList,
  Lock,
  Search,
  Eye,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";

/** 审计事件数据 */
interface AuditEvent {
  id: string;
  actor_user_id: string;
  actor_name: string;
  action: string;
  resource_type: string;
  resource_id: string;
  status: "success" | "failure";
  message: string | null;
  has_snapshot: boolean;
  ip: string | null;
  created_at: string;
}

/** 审计事件详情（含快照对比） */
interface AuditEventDetail extends AuditEvent {
  before_snapshot: Record<string, unknown> | null;
  after_snapshot: Record<string, unknown> | null;
  diff: Record<string, { from: unknown; to: unknown }> | null;
  user_agent: string | null;
}

const API_BASE = "/gov";

/**
 * 审计日志查询页面 —— 审计事件列表、事件详情面板、before/after 快照对比。
 * 对应 PRD UI-14 需求。审计记录不可删除。
 */
export default function AuditPage() {
  // 列表状态
  const [events, setEvents] = React.useState<AuditEvent[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 筛选条件
  const [actorFilter, setActorFilter] = React.useState("");
  const [actionFilter, setActionFilter] = React.useState("");
  const [dateFrom, setDateFrom] = React.useState("");
  const [dateTo, setDateTo] = React.useState("");

  // 详情面板
  const [selectedEvent, setSelectedEvent] = React.useState<AuditEventDetail | null>(null);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [showDetail, setShowDetail] = React.useState(false);

  // 获取审计事件列表
  const fetchEvents = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      if (actorFilter) params.set("actor_name", actorFilter);
      if (actionFilter) params.set("action", actionFilter);
      if (dateFrom) params.set("from", new Date(dateFrom).toISOString());
      if (dateTo) params.set("to", new Date(dateTo).toISOString());

      const res = await fetch(`${API_BASE}/audit-events?${params}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      setEvents(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取审计日志失败");
    } finally {
      setLoading(false);
    }
  }, [page, actorFilter, actionFilter, dateFrom, dateTo]);

  React.useEffect(() => { fetchEvents(); }, [fetchEvents]);

  // 获取事件详情
  const fetchDetail = async (eventId: string) => {
    setDetailLoading(true);
    try {
      const res = await fetch(`${API_BASE}/audit-events/${eventId}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      setSelectedEvent(json);
      setShowDetail(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取事件详情失败");
    } finally {
      setDetailLoading(false);
    }
  };

  // 格式化时间
  const formatTime = (iso: string) => new Date(iso).toLocaleString("zh-CN");

  // 表格列
  const columns: ColumnDef<AuditEvent>[] = [
    {
      key: "actor_name",
      header: "操作者",
      render: (e) => e.actor_name ?? (
        <span className="text-gray-400">{e.actor_user_id?.slice(0, 8)}...</span>
      ),
    },
    { key: "action", header: "动作" },
    { key: "resource_type", header: "资源类型" },
    {
      key: "status",
      header: "状态",
      render: (e) => (
        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
          e.status === "success" ? "bg-green-100 text-green-800" : "bg-red-100 text-red-800"
        }`}>
          {e.status === "success" ? "成功" : "失败"}
        </span>
      ),
    },
    {
      key: "has_snapshot",
      header: "快照",
      render: (e) => e.has_snapshot ? (
        <span className="text-xs text-blue-600">有</span>
      ) : (
        <span className="text-xs text-gray-400">-</span>
      ),
    },
    {
      key: "created_at",
      header: "时间",
      render: (e) => formatTime(e.created_at),
    },
    {
      key: "actions",
      header: "详情",
      render: (e) => (
        <button
          onClick={(ev) => { ev.stopPropagation(); fetchDetail(e.id); }}
          className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-blue-600"
        >
          <Eye className="h-4 w-4" />
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">审计日志查询</h1>
          <p className="mt-1 text-sm text-gray-500">
            查询配置变更与操作审计记录。审计日志不可删除、不可修改。
          </p>
        </div>
        <div className="flex items-center gap-1.5 rounded-md border border-gray-300 bg-gray-50 px-3 py-1.5 text-xs text-gray-500">
          <Lock className="h-3 w-3" />
          不可删除
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchEvents} dismissible />}

      {/* 筛选条件 */}
      <div className="flex flex-wrap items-center gap-3 rounded-lg border border-gray-200 bg-white p-4">
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-500">操作者</label>
          <input
            type="text"
            value={actorFilter}
            onChange={(e) => { setActorFilter(e.target.value); setPage(1); }}
            placeholder="姓名"
            className="rounded border border-gray-300 px-2 py-1 text-sm w-28"
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-500">动作</label>
          <select
            value={actionFilter}
            onChange={(e) => { setActionFilter(e.target.value); setPage(1); }}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          >
            <option value="">全部</option>
            <option value="fund.allocate">fund.allocate</option>
            <option value="fund.liquidate">fund.liquidate</option>
            <option value="fund.budget.write">fund.budget.write</option>
            <option value="routing.price.write">routing.price.write</option>
            <option value="routing.route_profile.write">routing.route_profile.write</option>
            <option value="iam.policy.write">iam.policy.write</option>
          </select>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-500">从</label>
          <input
            type="date"
            value={dateFrom}
            onChange={(e) => { setDateFrom(e.target.value); setPage(1); }}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-500">至</label>
          <input
            type="date"
            value={dateTo}
            onChange={(e) => { setDateTo(e.target.value); setPage(1); }}
            className="rounded border border-gray-300 px-2 py-1 text-sm"
          />
        </div>
      </div>

      {/* 事件列表 */}
      <DataTable
        data={events}
        columns={columns}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无审计事件"
        onRowClick={(e) => fetchDetail(e.id)}
      />

      {/* ===== 事件详情面板 ===== */}
      {showDetail && selectedEvent && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowDetail(false)} />
          <div className="relative z-10 mx-4 w-full max-w-3xl rounded-lg bg-white p-6 shadow-xl max-h-[85vh] overflow-y-auto">
            {detailLoading ? (
              <div className="flex items-center justify-center py-10">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
              </div>
            ) : (
              <>
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-semibold text-gray-900">事件详情</h2>
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    selectedEvent.status === "success" ? "bg-green-100 text-green-800" : "bg-red-100 text-red-800"
                  }`}>
                    {selectedEvent.status === "success" ? "成功" : "失败"}
                  </span>
                </div>

                {/* 基本信息网格 */}
                <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-gray-500">操作者</span>
                    <p className="font-medium">{selectedEvent.actor_name || selectedEvent.actor_user_id}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">动作</span>
                    <p className="font-mono font-medium">{selectedEvent.action}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">资源类型</span>
                    <p className="font-medium">{selectedEvent.resource_type}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">资源 ID</span>
                    <p className="font-mono text-xs">{selectedEvent.resource_id}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">IP 地址</span>
                    <p>{selectedEvent.ip || "-"}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">操作时间</span>
                    <p>{formatTime(selectedEvent.created_at)}</p>
                  </div>
                </div>

                {/* 操作信息 */}
                {selectedEvent.message && (
                  <div className="mt-4">
                    <span className="text-sm text-gray-500">操作说明</span>
                    <p className="mt-1 text-sm">{selectedEvent.message}</p>
                  </div>
                )}

                {/* before/after 快照并排对比 */}
                {(selectedEvent.before_snapshot || selectedEvent.after_snapshot) && (
                  <div className="mt-5">
                    <h3 className="text-sm font-medium text-gray-700">配置变更快照对比</h3>
                    <div className="mt-2 grid grid-cols-2 gap-4">
                      {/* before */}
                      <div>
                        <span className="mb-1 block text-xs font-medium text-red-600">变更前 (before)</span>
                        {selectedEvent.before_snapshot ? (
                          <CodeBlock data={selectedEvent.before_snapshot} maxHeight="300px" title="before_snapshot" />
                        ) : (
                          <p className="rounded border border-gray-200 bg-gray-50 p-4 text-sm text-gray-400">无（新建操作）</p>
                        )}
                      </div>
                      {/* after */}
                      <div>
                        <span className="mb-1 block text-xs font-medium text-green-600">变更后 (after)</span>
                        {selectedEvent.after_snapshot ? (
                          <CodeBlock data={selectedEvent.after_snapshot} maxHeight="300px" title="after_snapshot" />
                        ) : (
                          <p className="rounded border border-gray-200 bg-gray-50 p-4 text-sm text-gray-400">无（删除操作）</p>
                        )}
                      </div>
                    </div>

                    {/* diff 汇总 */}
                    {selectedEvent.diff && Object.keys(selectedEvent.diff).length > 0 && (
                      <div className="mt-3">
                        <span className="text-xs font-medium text-gray-600">差异汇总 (diff)</span>
                        <div className="mt-1 rounded border border-gray-200 bg-gray-50 p-3">
                          {Object.entries(selectedEvent.diff).map(([key, change]) => (
                            <div key={key} className="flex items-center gap-2 py-1 text-sm">
                              <span className="font-mono text-xs text-gray-500">{key}:</span>
                              <span className="text-red-600">{String(change.from)}</span>
                              <span className="text-gray-400">→</span>
                              <span className="text-green-600">{String(change.to)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* 不可删除标注 */}
                <div className="mt-5 flex items-center gap-1.5 rounded-md border border-yellow-300 bg-yellow-50 px-3 py-2 text-xs text-yellow-800">
                  <Lock className="h-3 w-3" />
                  审计记录不可编辑、不可删除。保留不少于 180 天。
                </div>
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
    </div>
  );
}
