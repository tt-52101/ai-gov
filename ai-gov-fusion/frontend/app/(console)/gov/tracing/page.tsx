"use client";

import React from "react";
import {
  Search,
  Eye,
  Copy,
  Check,
  ChevronRight,
  Activity,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";

/** 请求日志 */
interface RequestLog extends Record<string, unknown> {
  id: string;
  request_id: string;
  key_id: string;
  key_name: string;
  model_id: string;
  model_name: string;
  source_ip: string;
  status: "success" | "failure" | "pending";
  latency_ms: number;
  cost: string;
  sell: string;
  created_at: string;
}

/** 调用链详情 */
interface TraceDetail {
  request_id: string;
  key_id: string;
  key_name: string;
  model_id: string;
  model_name: string;
  source_ip: string;
  status: string;
  latency_ms: number;
  cost: string;
  sell: string;
  created_at: string;
  completed_at: string | null;
  route_profile_id: string | null;
  party_id: string | null;
  party_name: string | null;
  freeze_id: string | null;
  upstream_request_id: string | null;
  upstream_status: string | null;
  error_code: string | null;
  error_message: string | null;
  stages: TraceStage[];
  metadata: Record<string, unknown> | null;
}

/** 调用链阶段 */
interface TraceStage {
  stage_name: string;
  started_at: string;
  completed_at: string | null;
  duration_ms: number;
  status: "success" | "failure" | "skipped";
  detail: string;
}

const API_BASE = "/gov";

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

  // 获取请求日志列表
  const fetchLogs = React.useCallback(async (search?: string) => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      if (search) params.set("request_id", search);
      const res = await fetch(`${API_BASE}/tracing/logs?${params}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
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

  // 获取调用链详情
  const fetchTrace = async (requestId: string) => {
    setDetailLoading(true);
    try {
      const res = await fetch(`${API_BASE}/tracing/traces/${requestId}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setTraceDetail(json);
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

  // 状态标签
  const statusBadge = (status: string) => {
    const map: Record<string, { label: string; cls: string }> = {
      success: { label: "成功", cls: "bg-green-100 text-green-800" },
      failure: { label: "失败", cls: "bg-red-100 text-red-800" },
      pending: { label: "处理中", cls: "bg-yellow-100 text-yellow-800" },
    };
    const s = map[status] ?? { label: status, cls: "bg-gray-100 text-gray-600" };
    return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${s.cls}`}>{s.label}</span>;
  };

  // 表格列
  const columns: ColumnDef<RequestLog>[] = [
    {
      key: "request_id",
      header: "Request ID",
      render: (l) => <span className="font-mono text-xs">{l.request_id.slice(0, 12)}...</span>,
    },
    { key: "key_name", header: "密钥" },
    { key: "model_name", header: "模型" },
    { key: "source_ip", header: "来源 IP" },
    { key: "status", header: "状态", render: (l) => statusBadge(l.status) },
    {
      key: "latency_ms",
      header: "延迟",
      render: (l) => <span className="font-mono text-xs">{l.latency_ms}ms</span>,
    },
    {
      key: "sell",
      header: "费用",
      render: (l) => <span className="font-mono text-xs">{l.sell}</span>,
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

      {/* ===== 调用链详情弹窗 ===== */}
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
                    <button
                      onClick={() => handleCopy(traceDetail.request_id)}
                      className="rounded p-1 text-gray-400 hover:text-blue-600"
                      title="复制 request_id"
                    >
                      {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                    </button>
                  </div>
                  {statusBadge(traceDetail.status)}
                </div>

                {/* 基本信息 */}
                <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-gray-500">Request ID</span>
                    <p className="font-mono text-xs">{traceDetail.request_id}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">密钥</span>
                    <p className="font-medium">{traceDetail.key_name}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">模型</span>
                    <p className="font-medium">{traceDetail.model_name}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">来源 IP</span>
                    <p>{traceDetail.source_ip}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">延迟</span>
                    <p className="font-mono">{traceDetail.latency_ms}ms</p>
                  </div>
                  <div>
                    <span className="text-gray-500">费用</span>
                    <p className="font-mono text-xs">cost: {traceDetail.cost} / sell: {traceDetail.sell}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">Party</span>
                    <p>{traceDetail.party_name ?? <span className="text-gray-400">-</span>}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">路由档案</span>
                    <p className="font-mono text-xs">{traceDetail.route_profile_id?.slice(0, 12) ?? "-"}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">创建时间</span>
                    <p>{formatTime(traceDetail.created_at)}</p>
                  </div>
                  <div>
                    <span className="text-gray-500">完成时间</span>
                    <p>{formatTime(traceDetail.completed_at)}</p>
                  </div>
                </div>

                {/* 错误信息 */}
                {traceDetail.error_code && (
                  <div className="mt-4 rounded border border-red-200 bg-red-50 p-3">
                    <span className="text-sm font-medium text-red-700">错误: {traceDetail.error_code}</span>
                    {traceDetail.error_message && <p className="mt-1 text-xs text-red-600">{traceDetail.error_message}</p>}
                  </div>
                )}

                {/* 上游信息 */}
                {traceDetail.upstream_request_id && (
                  <div className="mt-4">
                    <span className="text-sm text-gray-500">上游信息</span>
                    <div className="mt-1 grid grid-cols-2 gap-2 text-sm">
                      <div>
                        <span className="text-gray-500">上游 Request ID</span>
                        <p className="font-mono text-xs">{traceDetail.upstream_request_id}</p>
                      </div>
                      <div>
                        <span className="text-gray-500">上游状态</span>
                        <p>{traceDetail.upstream_status ?? "-"}</p>
                      </div>
                    </div>
                  </div>
                )}

                {/* 调用链阶段 */}
                {traceDetail.stages && traceDetail.stages.length > 0 && (
                  <div className="mt-5">
                    <h3 className="text-sm font-medium text-gray-700">调用链阶段</h3>
                    <div className="mt-2 space-y-2">
                      {traceDetail.stages.map((stage, i) => (
                        <div key={i} className="flex items-start gap-3 rounded border border-gray-200 bg-gray-50 p-3">
                          {/* 阶段编号 */}
                          <div className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 text-xs font-medium text-blue-700">
                            {i + 1}
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center justify-between">
                              <span className="text-sm font-medium text-gray-800">{stage.stage_name}</span>
                              <div className="flex items-center gap-2">
                                <span className="text-xs text-gray-400">{stage.duration_ms}ms</span>
                                {stage.status === "success" ? (
                                  <span className="text-xs text-green-600">成功</span>
                                ) : stage.status === "failure" ? (
                                  <span className="text-xs text-red-600">失败</span>
                                ) : (
                                  <span className="text-xs text-gray-400">跳过</span>
                                )}
                              </div>
                            </div>
                            <p className="mt-0.5 text-xs text-gray-500">{stage.detail}</p>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* 元数据 */}
                {traceDetail.metadata && Object.keys(traceDetail.metadata).length > 0 && (
                  <div className="mt-4">
                    <h3 className="text-sm font-medium text-gray-700">元数据</h3>
                    <div className="mt-2">
                      <CodeBlock data={traceDetail.metadata} maxHeight="200px" />
                    </div>
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
    </div>
  );
}