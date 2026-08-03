"use client";

import React from "react";
import {
  LayoutDashboard,
  TrendingUp,
  Wallet,
  ShieldAlert,
  AlertTriangle,
  BarChart3,
} from "lucide-react";
import { StatCard } from "../_components/StatCard";
import { ErrorAlert } from "../_components/ErrorAlert";
import { govFetchJSON } from "@/lib/gov-api";

/** 仪表盘 API 响应 */
interface DashboardData {
  period: { from: string; to: string };
  consumption: {
    total_sell: number;
    total_cost: number;
    markup_pct: number;
    trend: { date: string; sell: number }[];
  };
  balance: {
    total_available: number;
    total_frozen: number;
    total_budget_limit: number;
    utilization_pct: number;
  };
  budget_status: {
    accounts_at_warning: number;
    accounts_exceeded: number;
    accounts_near_limit: number;
  };
  block_rates: Record<string, number>;
  top_consumers: { party_id: string; party_name: string; sell: number; pct: number }[];
  generated_at: string;
}

/**
 * 仪表盘页面 —— 消耗趋势图、余额总览、预算消耗进度、拦截统计。
 * 对应 PRD UI-07 需求。
 */
export default function DashboardPage() {
  const [data, setData] = React.useState<DashboardData | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [period, setPeriod] = React.useState("current_month");

  // 获取仪表盘数据
  const fetchDashboard = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ period });
      const json = await govFetchJSON<DashboardData>(`/dashboard?${params}`);
      setData(json);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取仪表盘数据失败");
    } finally {
      setLoading(false);
    }
  }, [period]);

  React.useEffect(() => { fetchDashboard(); }, [fetchDashboard]);

  // 格式化金额
  const fmtMoney = (v: number) =>
    new Intl.NumberFormat("zh-CN", { style: "currency", currency: "CNY", maximumFractionDigits: 0 }).format(v);

  // 格式化百分比
  const fmtPct = (v: number) => `${v.toFixed(1)}%`;

  // 消耗趋势柱状图（简易 CSS 实现，不引入图表库）
  const renderTrendChart = () => {
    if (!data?.consumption.trend?.length) return null;
    const trends = data.consumption.trend;
    const maxSell = Math.max(...trends.map((t) => t.sell));

    return (
      <div className="mt-4">
        <h3 className="text-sm font-medium text-gray-700">消耗趋势（sell 金额）</h3>
        <div className="mt-2 flex items-end gap-1" style={{ height: 160 }}>
          {trends.map((point, i) => {
            const heightPct = maxSell > 0 ? (point.sell / maxSell) * 100 : 0;
            return (
              <div
                key={i}
                className="group relative flex flex-1 flex-col items-center justify-end"
              >
                {/* 悬停提示 */}
                <div className="absolute -top-8 left-1/2 -translate-x-1/2 whitespace-nowrap rounded bg-gray-800 px-2 py-1 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100 z-10">
                  {point.date}: {fmtMoney(point.sell)}
                </div>
                {/* 柱状条 */}
                <div
                  className="w-full max-w-[40px] rounded-t bg-blue-500 transition-colors hover:bg-blue-600"
                  style={{ height: `${Math.max(heightPct, 2)}%` }}
                />
              </div>
            );
          })}
        </div>
        {/* 横轴标签 */}
        <div className="mt-1 flex gap-1">
          {trends.map((point, i) => (
            <span
              key={i}
              className="flex-1 text-center text-xs text-gray-400"
              style={{ maxWidth: 40 }}
            >
              {point.date.slice(5)}
            </span>
          ))}
        </div>
      </div>
    );
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-7 w-20 rounded bg-gray-200" />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-24 rounded-lg bg-gray-200" />
          ))}
        </div>
        <div className="h-48 rounded-lg bg-gray-200" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">仪表盘</h1>
          <p className="mt-1 text-sm text-gray-500">
            {data ? `${data.period.from.slice(0, 10)} ~ ${data.period.to.slice(0, 10)}` : ""}
          </p>
        </div>
        <select
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-700"
        >
          <option value="current_day">今日</option>
          <option value="current_month">本月</option>
          <option value="current_quarter">本季</option>
          <option value="current_year">本年</option>
        </select>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchDashboard} dismissible />}

      {data && (
        <>
          {/* 统计卡片行 */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title="总消耗 (sell)"
              value={fmtMoney(data.consumption.total_sell)}
              description={`成本 ${fmtMoney(data.consumption.total_cost)} | 加价率 ${fmtPct(data.consumption.markup_pct)}`}
              icon={TrendingUp}
              colorClass="text-blue-600"
              trend={12.5 /* TODO: 替换为真实环比数据，需要后端提供上期基准值 */ }
            />
            <StatCard
              title="可用余额"
              value={fmtMoney(data.balance.total_available)}
              description={`冻结 ${fmtMoney(data.balance.total_frozen)}`}
              icon={Wallet}
              colorClass="text-green-600"
              trend={-3.2 /* TODO: 替换为真实环比数据 */ }
            />
            <StatCard
              title="预算利用率"
              value={fmtPct(data.balance.utilization_pct)}
              description={`总预算帽 ${fmtMoney(data.balance.total_budget_limit)}`}
              icon={BarChart3}
              colorClass={data.balance.utilization_pct > 80 ? "text-red-600" : "text-yellow-600"}
              trend={5.8 /* TODO: 替换为真实环比数据 */ }
            />
            <StatCard
              title="拦截统计"
              value={Object.values(data.block_rates).reduce((a, b) => a + b, 0)}
              description="各类拦截事件总数"
              icon={ShieldAlert}
              colorClass="text-purple-600"
              trend={-15.0 /* TODO: 替换为真实环比数据 */ }
            />
          </div>

          {/* 消耗趋势图 */}
          <section className="rounded-lg border border-gray-200 bg-white p-5">
            {renderTrendChart()}
          </section>

          {/* 预算状态 + 拦截统计 + 消耗排行（三栏） */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {/* 预算状态 */}
            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="text-sm font-semibold text-gray-700">预算状态</h2>
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600">告警账户</span>
                  <span className={`font-mono text-lg font-bold ${data.budget_status.accounts_at_warning > 0 ? "text-yellow-600" : "text-gray-400"}`}>
                    {data.budget_status.accounts_at_warning}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600">已超额账户</span>
                  <span className={`font-mono text-lg font-bold ${data.budget_status.accounts_exceeded > 0 ? "text-red-600" : "text-gray-400"}`}>
                    {data.budget_status.accounts_exceeded}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-600">接近上限账户</span>
                  <span className={`font-mono text-lg font-bold ${data.budget_status.accounts_near_limit > 0 ? "text-orange-500" : "text-gray-400"}`}>
                    {data.budget_status.accounts_near_limit}
                  </span>
                </div>
              </div>
            </section>

            {/* 拦截统计 */}
            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="text-sm font-semibold text-gray-700">拦截统计分类</h2>
              <div className="mt-4 space-y-2">
                {Object.entries(data.block_rates).map(([code, count]) => (
                  <div key={code} className="flex items-center justify-between">
                    <span className="text-xs font-mono text-gray-600">{code}</span>
                    <span className={`text-sm font-bold ${count > 0 ? "text-red-600" : "text-gray-400"}`}>
                      {count}
                    </span>
                  </div>
                ))}
                {Object.keys(data.block_rates).length === 0 && (
                  <p className="text-sm text-gray-400">本周期无拦截事件</p>
                )}
              </div>
            </section>

            {/* 消耗排行 */}
            <section className="rounded-lg border border-gray-200 bg-white p-5">
              <h2 className="text-sm font-semibold text-gray-700">消耗排行 (Top)</h2>
              <div className="mt-4 space-y-3">
                {data.top_consumers.map((consumer, i) => (
                  <div key={consumer.party_id} className="flex items-center justify-between">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`flex h-5 w-5 items-center justify-center rounded-full text-xs font-bold ${
                        i === 0 ? "bg-yellow-400 text-white" :
                        i === 1 ? "bg-gray-300 text-gray-600" :
                        i === 2 ? "bg-orange-300 text-white" :
                        "bg-gray-100 text-gray-500"
                      }`}>
                        {i + 1}
                      </span>
                      <span className="truncate text-sm text-gray-700">{consumer.party_name}</span>
                    </div>
                    <span className="text-sm font-medium text-gray-600">
                      {fmtMoney(consumer.sell)}
                      <span className="ml-1 text-xs text-gray-400">({fmtPct(consumer.pct)})</span>
                    </span>
                  </div>
                ))}
                {data.top_consumers.length === 0 && (
                  <p className="text-sm text-gray-400">暂无消费数据</p>
                )}
              </div>
            </section>
          </div>

          {/* 生成时间 */}
          <p className="text-right text-xs text-gray-400">
            数据生成时间：{new Date(data.generated_at).toLocaleString("zh-CN")}
          </p>
        </>
      )}
    </div>
  );
}
