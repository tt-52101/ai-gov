"use client";

import React from "react";
import {
  Shield,
  Lock,
  Unlock,
  CheckCircle,
  XCircle,
  RefreshCw,
} from "lucide-react";
import { StatCard } from "../_components/StatCard";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";
import { govFetchJSON } from "@/lib/gov-api";

/** 密钥仓库健康状态 —— 字段与后端 /v1/gov/key-vault/health 返回保持一致 */
interface VaultHealth {
  status: "healthy" | "degraded" | "unhealthy";
  keys_count: number;
  encrypted_at_rest: boolean;
  provider: string;
  last_rotation_at: string | null;
  checked_at: string;
}

/** 最近密钥轮换记录 */
interface RotationRecord {
  id: string;
  key_name: string;
  rotated_at: string;
  reason: string;
  status: "success" | "failed";
}

/**
 * 密钥仓库页面 —— 安全存储状态概览、密钥轮换记录。
 * 对应 PRD UI-08 需求。
 */
export default function KeyVaultPage() {
  const [health, setHealth] = React.useState<VaultHealth | null>(null);
  const [rotations, setRotations] = React.useState<RotationRecord[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    // 并行拉取健康状态与轮换记录，任一失败不阻塞另一个，降级为占位/空列表。
    try {
      const [healthJson, rotJson] = await Promise.all([
        govFetchJSON<VaultHealth>("/key-vault/health").catch(() => null),
        govFetchJSON<{ data: RotationRecord[] }>("/key-vault/rotations?page_size=5").catch(() => ({ data: [] })),
      ]);
      setHealth(healthJson);
      setRotations(rotJson.data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取密钥仓库数据失败");
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => { fetchData(); }, [fetchData]);

  const formatTime = (iso: string | null) => iso ? new Date(iso).toLocaleString("zh-CN") : "-";

  // 健康状态标签
  const healthBadge = (status: string) => {
    const map: Record<string, { label: string; cls: string }> = {
      healthy: { label: "健康", cls: "bg-green-100 text-green-800" },
      degraded: { label: "降级", cls: "bg-yellow-100 text-yellow-800" },
      unhealthy: { label: "异常", cls: "bg-red-100 text-red-800" },
    };
    const s = map[status] ?? { label: status, cls: "bg-gray-100 text-gray-600" };
    return <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${s.cls}`}>{s.label}</span>;
  };

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">密钥仓库</h1>
          <p className="mt-1 text-sm text-gray-500">密钥安全存储状态概览与轮换审计</p>
        </div>
        <button
          onClick={fetchData}
          disabled={loading}
          className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} /> 刷新
        </button>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchData} dismissible />}

      {loading ? (
        <div className="animate-pulse space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-28 rounded-lg bg-gray-200" />
            ))}
          </div>
          <div className="h-40 rounded-lg bg-gray-200" />
        </div>
      ) : !health && !error ? (
        // 健康接口请求失败时的降级占位（接口本身已由后端实现）。
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-center">
          <Lock className="mx-auto h-10 w-10 text-amber-500" />
          <h3 className="mt-3 text-base font-medium text-amber-800">暂时无法获取密钥仓库健康状态</h3>
          <p className="mt-1 text-sm text-amber-700">
            请求 <code className="rounded bg-amber-100 px-1.5 py-0.5 font-mono text-xs">/v1/gov/key-vault/health</code> 失败，请点击右上角「刷新」重试。
          </p>
        </div>
      ) : health ? (
        <>
          {/* 健康状态卡片 */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-gray-500">仓库状态</span>
                {health.status === "healthy" ? <CheckCircle className="h-5 w-5 text-green-600" /> : <XCircle className="h-5 w-5 text-red-600" />}
              </div>
              <div className="mt-2">{healthBadge(health.status)}</div>
            </div>
            <StatCard
              title="密钥总数"
              value={health.keys_count ?? 0}
              icon={Lock}
            />
            <StatCard
              title="静态加密"
              value={health.encrypted_at_rest ? "已启用" : "未启用"}
              icon={health.encrypted_at_rest ? Lock : Unlock}
              colorClass={health.encrypted_at_rest ? "text-green-600" : "text-gray-400"}
            />
            <StatCard
              title="上次轮换"
              value={formatTime(health.last_rotation_at)}
              icon={Shield}
            />
          </div>

          {/* 安全配置详情 */}
          <div className="rounded-lg border border-gray-200 bg-white p-5">
            <h2 className="font-medium text-gray-900">安全配置</h2>
            <div className="mt-4 grid grid-cols-2 gap-4 text-sm lg:grid-cols-3">
              <div>
                <span className="text-gray-500">存储后端</span>
                <p className="font-mono text-xs font-medium">{health.provider ?? "-"}</p>
              </div>
              <div>
                <span className="text-gray-500">静态加密</span>
                <p>{health.encrypted_at_rest ? "已启用" : "未启用"}</p>
              </div>
              <div>
                <span className="text-gray-500">上次轮换</span>
                <p>{formatTime(health.last_rotation_at)}</p>
              </div>
              <div>
                <span className="text-gray-500">上次健康检查</span>
                <p>{formatTime(health.checked_at)}</p>
              </div>
            </div>
          </div>

          {/* 最近轮换记录 */}
          <div className="rounded-lg border border-gray-200 bg-white p-5">
            <h2 className="font-medium text-gray-900">最近密钥轮换记录</h2>
            {rotations.length === 0 ? (
              <p className="mt-4 text-sm text-gray-400">暂无轮换记录</p>
            ) : (
              <div className="mt-4 space-y-2">
                {rotations.map((r) => (
                  <div key={r.id} className="flex items-center justify-between rounded border border-gray-100 px-3 py-2">
                    <div className="flex items-center gap-3">
                      <span className="text-sm font-medium text-gray-800">{r.key_name}</span>
                      <span className="text-xs text-gray-400">{r.reason}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400">{formatTime(r.rotated_at)}</span>
                      {r.status === "success" ? (
                        <CheckCircle className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-500" />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      ) : (
        <p className="text-center text-sm text-gray-400 py-10">暂无密钥仓库数据</p>
      )}
    </div>
  );
}