"use client";

import React from "react";
import {
  Search,
  Eye,
  Copy,
  Check,
  Activity,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";
import { govFetchJSON } from "@/lib/gov-api";

// ──────────────────────────────────────────────────────────────────────
// 数据模型——与后端 /v1/gov/request-logs 真实结构对齐（plan-002 B-08）
// ──────────────────────────────────────────────────────────────────────

/** 请求日志——后端 RequestLog 序列化结构 */
interface RequestLog extends Record<string, unknown> {
  id: string;
  request_id: string;
  project_id: string;
  api_key_id: string;
  model: string;
  provider_id?: string;
  upstream_request_id?: string;
  status_code: number;
  error_code?: string;
  latency_ms: number;
  client_ip?: string;
  user_agent?: string;
  created_at: string;
}

/** 用量记录 */
interface UsageRecord extends Record<string, unknown> {
  id: string;
  request_id: string;
  item_code: string;
  quantity: number;
  cost_amount?: string;
  sell_amount?: string;
  created_at: string;
}

/** 路由尝试记录 */
interface RouteAttemptLog extends Record<string, unknown> {
  id: string;
  request_id: string;
  attempt_index: number;
  provider_id?: string;
  model?: string;
  status_code?: number;
  error_code?: string;
  latency_ms?: number;
  started_at?: string;
  completed_at?: string;
}

/** 调用链详情——后端实际返回 { request_log, usage_records, attempt_logs } */
interface TraceDetail {
  request_log?: RequestLog;
  usage_records?: UsageRecord[];
  attempt_logs?: RouteAttemptLog[];
}

/**
 * 调用追踪页面 —— 请求日志列表、request_id 搜索、调用链详情展示。
 * 对应 PRD UI-11 需求。
 */
export default function TracingPage() {
  // 搜索
  const [searchId, setSearchId] = React.useState("");

  // 列表状态
  const [logs, setLogs] = React.useState<RequestLog[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 详情
  const [traceDetail, setTraceDetail] = React.useState<TraceDetail | null>(null);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [showDetail, setShowDetail] = React.useState(false);

  // 复制
  const [copied, setCopied] = React.useState(false);

  // 获取请求日志列表——对齐后端 /v1/gov/request-logs（plan-002 B-08 路径纠偏）
  const fetchLogs = React.useCallback(async (search?: string) => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      if (search) params.set("request_id", search);
      const json = await govFetchJSON<{ data: RequestLog[]; total: number }>(`/request-logs?${params}`);
      setLogs(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取请求日志失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  React.useEffect(() => { fetchLogs(); }, [fetchLogs]);

  // 搜索提交
  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchLogs(searchId.trim() || undefined);
  };

  // 获取调用链详情——对齐后端 /v1/gov/request-logs/{id}（plan-002 B-08 路径纠偏）
  const fetchTrace = async (requestId: string) => {
    setDetailLoading(true);
    try {
      const trace = await govFetchJSON<TraceDetail>(`/request-logs/${encodeURIComponent(requestId)}`);
      setTraceDetail(trace);
      setShowDetail(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取调用链详情失败");
    } finally {
      setDetailLoading(false);
    }
  };

  const handleCopy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch { /* 静默 */ }
  };

  const formatTime = (iso: string | null) => iso ? new Date(iso).toLocaleString("zh-CN") : "-";

  // 状态标签——根据 status_code 推断（plan-002 B-08 字段对齐）
  const statusBadge = (statusCode: number, errorCode?: string) => {
    if (statusCode >= 200 && statusCode < 300) {
      return <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-green-100 text-green-800">成功</span>;
    }
    if (statusCode === 0 || statusCode === 403 || statusCode === 401 || errorCode) {
      return <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-red-100 text-red-800">失败</span>;
    }
    return <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-yellow-100 text-yellow-800">处理中</span>;
  };

  // 表格列——字段映射对齐后端 RequestLog（plan-002 B-08）
  const columns: ColumnDef<RequestLog>[] = [
    {
      key: "request_id",
      header: "Request ID",
      render: (l) => <span className="font-mono text-xs">{l.request_id.slice(0, 12)}...</span>,
    },
    { key: "api_key_id", header: "密钥 ID", render: (l) => <span className="font-mono text-xs">{l.api_key_id}</span> },
    { key: "model", header: "模型" },
    { key: "client_ip", header: "来源 IP", render: (l) => l.client_ip ?? "-" },
    { key: "status_code", header: "状态", render: (l) => statusBadge(l.status_code, l.error_code) },
    {
      key: "latency_ms",
      header: "延迟",
      render: (l) => <span className="font-mono text-xs">{l.latency_ms}ms</span>,
    },
    {
      key: "error_code",
      header: "错误码",
      render: (l) => l.error_code ? <span className="font-mono text-xs text-red-600">{l.error_code}</span> : <span className="text-gray-400">-</span>,
    },
    {
      key: "created_at",
      header: "时间",
      render: (l) => formatTime(l.created_at),
    },
    {
      key: "actions",
      header: "详情",
      render: (l) => (
        <button
          onClick={(e) => { e.stopPropagation(); fetchTrace(l.request_id); }}
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
      <div>
        <h1 className="text-2xl font-bold text-gray-900">调用追踪</h1>
        <p className="mt-1 text-sm text-gray-500">查询请求日志和调用链详情，支持 request_id 精确搜索</p>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => fetchLogs()} dismissible />}

      {/* 搜索栏 */}
      <div className="rounded-lg border border-gray-200 bg-white p-4">
        <form onSubmit={handleSearch} className="flex gap-3">
          <div className="relative flex-1 max-w-md">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={searchId}
              onChange={(e) => setSearchId(e.target.value)}
              placeholder="输入 request_id 精确搜索..."
              className="w-full rounded-md border border-gray-300 py-2 pl-9 pr-4 text-sm text-gray-700 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <button
            type="submit"
            className="inline-flex items-center gap-1.5 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            <Search className="h-4 w-4" /> 搜索
          </button>
          {searchId && (
            <button
              type="button"
              onClick={() => { setSearchId(""); setPage(1); fetchLogs(); }}
              className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              清除
            </button>
          )}
        </form>
      </div>

      {/* 请求日志列表 */}
      <DataTable
        data={logs}
        columns={columns}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无请求日志"
        onRowClick={(l) => fetchTrace(l.request_id)}
      />

      {/* ===== 调用链详情弹窗——对齐后端 /request-logs/{id} 真实结构 ===== */}
      {showDetail && traceDetail && (
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
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-semibold text-gray-900">调用链详情</h2>
                    {traceDetail.request_log && (
                      <button
                        onClick={() => handleCopy(traceDetail.request_log?.request_id ?? "")}
                        className="rounded p-1 text-gray-400 hover:text-blue-600"
                        title="复制 request_id"
                      >
                        {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                      </button>
                    )}
                  </div>
                  {traceDetail.request_log && statusBadge(traceDetail.request_log.status_code, traceDetail.request_log.error_code)}
                </div>

                {traceDetail.request_log ? (
                  <>
                    {/* 基本信息 */}
                    <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                      <div>
                        <span className="text-gray-500">Request ID</span>
                        <p className="font-mono text-xs">{traceDetail.request_log.request_id}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">Project ID</span>
                        <p className="font-mono text-xs">{traceDetail.request_log.project_id}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">密钥 ID</span>
                        <p className="font-mono text-xs">{traceDetail.request_log.api_key_id}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">模型</span>
                        <p className="font-medium">{traceDetail.request_log.model}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">来源 IP</span>
                        <p>{traceDetail.request_log.client_ip ?? "-"}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">延迟</span>
                        <p className="font-mono">{traceDetail.request_log.latency_ms}ms</p>
                      </div>
                      <div>
                        <span className="text-gray-500">上游 Request ID</span>
                        <p className="font-mono text-xs">{traceDetail.request_log.upstream_request_id ?? "-"}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">状态码</span>
                        <p className="font-mono">{traceDetail.request_log.status_code}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">User Agent</span>
                        <p className="text-xs text-gray-600">{traceDetail.request_log.user_agent ?? "-"}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">创建时间</span>
                        <p>{formatTime(traceDetail.request_log.created_at)}</p>
                      </div>
                    </div>

                    {/* 错误信息 */}
                    {traceDetail.request_log.error_code && (
                      <div className="mt-4 rounded border border-red-200 bg-red-50 p-3">
                        <span className="text-sm font-medium text-red-700">错误: {traceDetail.request_log.error_code}</span>
                      </div>
                    )}

                    {/* 用量记录 */}
                    {traceDetail.usage_records && traceDetail.usage_records.length > 0 && (
                      <div className="mt-5">
                        <h3 className="text-sm font-medium text-gray-700">用量记录（{traceDetail.usage_records.length}）</h3>
                        <div className="mt-2 space-y-1">
                          {traceDetail.usage_records.map((u) => (
                            <div key={u.id} className="flex items-center justify-between rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs">
                              <span className="font-mono">{u.item_code}</span>
                              <span className="font-mono text-gray-600">×{u.quantity}</span>
                              {u.cost_amount && <span className="font-mono text-gray-500">cost: {u.cost_amount}</span>}
                              {u.sell_amount && <span className="font-mono text-blue-600">sell: {u.sell_amount}</span>}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* 路由尝试 */}
                    {traceDetail.attempt_logs && traceDetail.attempt_logs.length > 0 && (
                      <div className="mt-5">
                        <h3 className="text-sm font-medium text-gray-700">路由尝试（{traceDetail.attempt_logs.length}）</h3>
                        <div className="mt-2 space-y-1">
                          {traceDetail.attempt_logs.map((a) => (
                            <div key={a.id} className="flex items-center justify-between rounded border border-gray-200 bg-gray-50 px-3 py-2 text-xs">
                              <span className="font-mono">#{a.attempt_index}</span>
                              <span className="text-gray-600">{a.provider_id ?? "-"} / {a.model ?? "-"}</span>
                              {a.status_code !== undefined && <span className="font-mono">{a.status_code}</span>}
                              {a.error_code && <span className="font-mono text-red-600">{a.error_code}</span>}
                              {a.latency_ms !== undefined && <span className="font-mono text-gray-500">{a.latency_ms}ms</span>}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* 元数据（无则不显示） */}
                    {traceDetail.attempt_logs && (
                      <div className="mt-4">
                        <h3 className="text-sm font-medium text-gray-700">请求原始信息</h3>
                        <div className="mt-2">
                          <CodeBlock
                            data={{
                              request_id: traceDetail.request_log.id,
                              upstream_request_id: traceDetail.request_log.upstream_request_id,
                              created_at: traceDetail.request_log.created_at,
                              status_code: traceDetail.request_log.status_code,
                              error_code: traceDetail.request_log.error_code,
                            }}
                            maxHeight="200px"
                          />
                        </div>
                      </div>
                    )}
                  </>
                ) : (
                  <div className="mt-4 text-sm text-gray-500">未返回调用链详情</div>
                )}

                <div className="mt-4 flex justify-end">
                  <button
                    onClick={() => setShowDetail(false)}
                    className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
                  >
                    关闭
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}