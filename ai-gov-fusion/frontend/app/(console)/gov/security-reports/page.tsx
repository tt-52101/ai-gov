"use client";

import React from "react";
import {
  AlertTriangle,
  Shield,
  Ban,
  RotateCw,
  Search,
  Eye,
} from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { StatCard } from "../_components/StatCard";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";
import { govFetchJSON } from "@/lib/gov-api";

/** 安全事件 */
interface SecurityEvent extends Record<string, unknown> {
  id: string;
  event_type: string;
  severity: "critical" | "high" | "medium" | "low";
  source_ip: string;
  key_id: string | null;
  key_name: string | null;
  description: string;
  detected_at: string;
  resolved: boolean;
  resolved_at: string | null;
}

/** 安全统计摘要——对齐后端 /v1/gov/security-reports/summary 实际返回结构 */
interface SecuritySummary {
  total_events: number;
  blocked_requests: number;
  abnormal_access_count: number;
  key_rotation_count: number;
  by_severity: Record<string, number> | null;
  by_type: Record<string, number> | null;
  period_from: string;
  period_to: string;
}

/** 异常访问 */
interface AbnormalAccess extends Record<string, unknown> {
  id: string;
  key_id: string;
  key_name: string;
  source_ip: string;
  request_count: number;
  first_seen: string;
  last_seen: string;
  reason: string;
}

/** 密钥轮换记录 */
interface KeyRotationRecord extends Record<string, unknown> {
  id: string;
  key_id: string;
  key_name: string;
  rotated_at: string;
  reason: string;
  operator: string;
}

/**
 * 安全报表页面 —— 安全事件汇总、异常访问统计、密钥轮换记录。
 * 对应 PRD UI-10 需求。
 */
export default function SecurityReportsPage() {
  const [tab, setTab] = React.useState<"events" | "abnormal" | "rotations">("events");
  const [error, setError] = React.useState<string | null>(null);

  // 统计摘要
  const [summary, setSummary] = React.useState<SecuritySummary | null>(null);

  // 安全事件
  const [events, setEvents] = React.useState<SecurityEvent[]>([]);
  const [eventsTotal, setEventsTotal] = React.useState(0);
  const [eventsPage, setEventsPage] = React.useState(1);
  const [eventsLoading, setEventsLoading] = React.useState(true);

  // 异常访问
  const [abnormal, setAbnormal] = React.useState<AbnormalAccess[]>([]);
  const [abnormalTotal, setAbnormalTotal] = React.useState(0);
  const [abnormalPage, setAbnormalPage] = React.useState(1);
  const [abnormalLoading, setAbnormalLoading] = React.useState(true);

  // 轮换记录
  const [rotations, setRotations] = React.useState<KeyRotationRecord[]>([]);
  const [rotationsTotal, setRotationsTotal] = React.useState(0);
  const [rotationsPage, setRotationsPage] = React.useState(1);
  const [rotationsLoading, setRotationsLoading] = React.useState(true);

  // 获取摘要
  const fetchSummary = React.useCallback(async () => {
    try {
      const json = await govFetchJSON<SecuritySummary>("/security-reports/summary");
      setSummary(json);
    } catch { /* 静默 */ }
  }, []);

  // 获取安全事件
  const fetchEvents = React.useCallback(async () => {
    setEventsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(eventsPage), page_size: "20" });
      const json = await govFetchJSON<{ data: SecurityEvent[]; total: number }>(`/security-reports/events?${params}`);
      setEvents(json.data ?? []);
      setEventsTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取安全事件失败");
    } finally {
      setEventsLoading(false);
    }
  }, [eventsPage]);

  // 获取异常访问
  const fetchAbnormal = React.useCallback(async () => {
    setAbnormalLoading(true);
    try {
      const params = new URLSearchParams({ page: String(abnormalPage), page_size: "20" });
      const json = await govFetchJSON<{ data: AbnormalAccess[]; total: number }>(`/security-reports/abnormal-access?${params}`);
      setAbnormal(json.data ?? []);
      setAbnormalTotal(json.total ?? 0);
    } catch {
      setAbnormal([]);
    } finally {
      setAbnormalLoading(false);
    }
  }, [abnormalPage]);

  // 获取轮换记录
  const fetchRotations = React.useCallback(async () => {
    setRotationsLoading(true);
    try {
      const params = new URLSearchParams({ page: String(rotationsPage), page_size: "20" });
      const json = await govFetchJSON<{ data: KeyRotationRecord[]; total: number }>(`/security-reports/key-rotations?${params}`);
      setRotations(json.data ?? []);
      setRotationsTotal(json.total ?? 0);
    } catch {
      setRotations([]);
    } finally {
      setRotationsLoading(false);
    }
  }, [rotationsPage]);

  React.useEffect(() => {
    setError(null);
    // 切换 tab 时重置对应分页为 1（plan-002 B-09：避免从其他 tab 残留分页）
    if (tab === "events") {
      setEventsPage(1);
      fetchEvents();
    } else if (tab === "abnormal") {
      setAbnormalPage(1);
      fetchAbnormal();
    } else if (tab === "rotations") {
      setRotationsPage(1);
      fetchRotations();
    }
  }, [tab]); // eslint-disable-line react-hooks/exhaustive-deps

  // plan-002 B-09 增强：探测后端 /v1/gov/security-reports 子路由（summary/events/abnormal-access/key-rotations）是否就绪
  // 后端当前仅提供占位端点 /v1/gov/security-reports，子路径 404 时使用 hasData 状态显示占位 UI
  const hasAnyData = !!summary || events.length > 0 || abnormal.length > 0 || rotations.length > 0;

  const formatTime = (iso: string | null) => iso ? new Date(iso).toLocaleString("zh-CN") : "-";

  // 统计周期文案——后端 summary 返回 period_from / period_to
  const periodLabel = summary?.period_from && summary?.period_to
    ? `${new Date(summary.period_from).toLocaleDateString("zh-CN")} ~ ${new Date(summary.period_to).toLocaleDateString("zh-CN")}`
    : undefined;

  // 严重级别标签
  const severityBadge = (s: string) => {
    const map: Record<string, { label: string; cls: string }> = {
      critical: { label: "严重", cls: "bg-red-100 text-red-800" },
      high: { label: "高", cls: "bg-orange-100 text-orange-800" },
      medium: { label: "中", cls: "bg-yellow-100 text-yellow-800" },
      low: { label: "低", cls: "bg-blue-100 text-blue-800" },
    };
    const m = map[s] ?? { label: s, cls: "bg-gray-100 text-gray-600" };
    return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${m.cls}`}>{m.label}</span>;
  };

  // 安全事件表格列
  const eventColumns: ColumnDef<SecurityEvent>[] = [
    { key: "event_type", header: "事件类型" },
    { key: "severity", header: "级别", render: (e) => severityBadge(e.severity) },
    { key: "source_ip", header: "来源 IP" },
    { key: "key_name", header: "关联密钥", render: (e) => e.key_name ?? <span className="text-gray-400">-</span> },
    { key: "detected_at", header: "检测时间", render: (e) => formatTime(e.detected_at) },
    {
      key: "resolved",
      header: "状态",
      render: (e) => e.resolved
        ? <span className="text-xs text-green-600">已处理</span>
        : <span className="text-xs text-red-600">未处理</span>,
    },
  ];

  // 异常访问表格列
  const abnormalColumns: ColumnDef<AbnormalAccess>[] = [
    { key: "key_name", header: "密钥" },
    { key: "source_ip", header: "来源 IP" },
    { key: "request_count", header: "请求次数" },
    { key: "reason", header: "原因" },
    { key: "first_seen", header: "首次发现", render: (a) => formatTime(a.first_seen) },
    { key: "last_seen", header: "最后发现", render: (a) => formatTime(a.last_seen) },
  ];

  // 轮换记录表格列
  const rotationColumns: ColumnDef<KeyRotationRecord>[] = [
    { key: "key_name", header: "密钥名称" },
    { key: "reason", header: "原因" },
    { key: "operator", header: "操作者" },
    { key: "rotated_at", header: "轮换时间", render: (r) => formatTime(r.rotated_at) },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">安全报表</h1>
        <p className="mt-1 text-sm text-gray-500">安全事件汇总、异常访问统计与密钥轮换审计</p>
      </div>

      {error && <ErrorAlert message={error} dismissible />}

      {/* plan-002 B-09：后端 /v1/gov/security-reports 子路由（events/abnormal-access/key-rotations/summary）尚未实现。
          首次加载完成且无任何数据时，提示运维知晓，避免误判为产品缺陷。 */}
      {!eventsLoading && !abnormalLoading && !rotationsLoading && !hasAnyData && !error && (
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-4">
          <div className="flex items-start gap-3">
            <Shield className="h-5 w-5 flex-shrink-0 text-amber-600" />
            <div>
              <h3 className="text-sm font-medium text-amber-800">安全报表子端点待后端实现</h3>
              <p className="mt-1 text-xs text-amber-700">
                后端 <code className="rounded bg-amber-100 px-1 py-0.5 font-mono">/v1/gov/security-reports/summary</code>、
                <code className="rounded bg-amber-100 px-1 py-0.5 font-mono">/events</code>、
                <code className="rounded bg-amber-100 px-1 py-0.5 font-mono">/abnormal-access</code>、
                <code className="rounded bg-amber-100 px-1 py-0.5 font-mono">/key-rotations</code> 子路由尚未实现，页面已就绪等待联调。
              </p>
            </div>
          </div>
        </div>
      )}

      {/* 统计卡片 */}
      {summary && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard title="安全事件总数" value={summary.total_events ?? 0} icon={AlertTriangle}
            description={periodLabel}
            colorClass={(summary.total_events ?? 0) > 0 ? "text-red-600" : "text-green-600"} />
          <StatCard title="已拦截请求" value={summary.blocked_requests ?? 0} icon={Shield}
            description={periodLabel}
            colorClass={(summary.blocked_requests ?? 0) > 0 ? "text-red-600" : "text-gray-400"} />
          <StatCard title="异常访问" value={summary.abnormal_access_count ?? 0} icon={Ban}
            colorClass={(summary.abnormal_access_count ?? 0) > 0 ? "text-orange-600" : "text-gray-400"} />
          <StatCard title="密钥轮换" value={summary.key_rotation_count ?? 0} icon={RotateCw} />
        </div>
      )}

      {/* 标签导航 */}
      <div className="flex border-b border-gray-200">
        {([
          { key: "events" as const, label: "安全事件", icon: AlertTriangle },
          { key: "abnormal" as const, label: "异常访问", icon: Ban },
          { key: "rotations" as const, label: "密钥轮换", icon: RotateCw },
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

      {/* 安全事件 */}
      {tab === "events" && (
        <DataTable
          data={events}
          columns={eventColumns}
          page={eventsPage}
          pageSize={20}
          total={eventsTotal}
          onPageChange={setEventsPage}
          loading={eventsLoading}
          emptyText="暂无安全事件"
        />
      )}

      {/* 异常访问 */}
      {tab === "abnormal" && (
        <DataTable
          data={abnormal}
          columns={abnormalColumns}
          page={abnormalPage}
          pageSize={20}
          total={abnormalTotal}
          onPageChange={setAbnormalPage}
          loading={abnormalLoading}
          emptyText="暂无异常访问记录"
        />
      )}

      {/* 密钥轮换 */}
      {tab === "rotations" && (
        <DataTable
          data={rotations}
          columns={rotationColumns}
          page={rotationsPage}
          pageSize={20}
          total={rotationsTotal}
          onPageChange={setRotationsPage}
          loading={rotationsLoading}
          emptyText="暂无密钥轮换记录"
        />
      )}
    </div>
  );
}