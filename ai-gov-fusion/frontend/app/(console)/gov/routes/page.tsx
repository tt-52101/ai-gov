"use client";

import React from "react";
import { GitBranch, Plus, Sliders, Info } from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";
import { govFetch, govFetchJSON } from "@/lib/gov-api";

/** 路由档案数据 */
interface RouteProfile extends Record<string, unknown> {
  id: string;
  name: string;
  description: string;
  /** 策略组合 —— 后端可能返回 null，遍历前需兜底为空数组 */
  strategies_json?: StrategyConfig[] | null;
  delta_cap: number;
  max_attempts: number;
  allow_fallback: boolean;
  status: "active" | "archived";
  created_at: string;
}

/** 策略配置项 */
interface StrategyConfig {
  code: string;
  enabled: boolean;
  priority: number;
  config: Record<string, unknown>;
}

/** 注册策略信息 */
interface RegisteredStrategy {
  code: string;
  name: string;
  description: string;
  can_disable: boolean;
}

/** 策略名称映射 */
const strategyNames: Record<string, string> = {
  "S-COMPLIANCE": "合规网络",
  "S-PRI": "优先级分组",
  "S-HEALTH": "健康与熔断",
  "S-WEIGHT": "权重与负载",
  "S-AFFINITY": "会话亲和",
  "S-COST": "成本感知",
  "S-LATENCY": "延迟感知",
  "S-ERROR": "错误率感知",
  "S-RATE": "限流感知",
  "S-TAG": "业务标签",
  "S-CLASSIFY": "智能任务分类",
  "S-CACHE": "缓存兜底",
};

/** 全部可用策略代码 */
const allStrategyCodes = [
  "S-COMPLIANCE", "S-PRI", "S-HEALTH", "S-WEIGHT",
  "S-AFFINITY", "S-COST", "S-LATENCY", "S-ERROR",
  "S-RATE", "S-TAG", "S-CLASSIFY", "S-CACHE",
];

/**
 * 路由档案页面 —— 路由档案列表、档案编辑器、已注册策略目录。
 * 对应 PRD UI-06 需求。delta_cap 硬上限 20%。
 */
export default function RoutesPage() {
  // 列表状态
  const [profiles, setProfiles] = React.useState<RouteProfile[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 策略目录
  const [strategies, setStrategies] = React.useState<RegisteredStrategy[]>([]);
  const [showStrategyCatalog, setShowStrategyCatalog] = React.useState(false);

  // 编辑器状态
  const [showEditor, setShowEditor] = React.useState(false);
  const [editingProfile, setEditingProfile] = React.useState<RouteProfile | null>(null);
  const [editorForm, setEditorForm] = React.useState({
    name: "",
    description: "",
    delta_cap: 0,
    max_attempts: 3,
    allow_fallback: true,
  });
  const [strategyConfigs, setStrategyConfigs] = React.useState<
    { code: string; enabled: boolean; priority: number }[]
  >([]);
  const [saving, setSaving] = React.useState(false);

  // 获取列表
  const fetchProfiles = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      const json = await govFetchJSON<{ data: RouteProfile[]; total: number }>(`/route-profiles?${params}`);
      setProfiles(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取路由档案列表失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  React.useEffect(() => { fetchProfiles(); }, [fetchProfiles]);

  // 获取策略目录
  const fetchStrategies = async () => {
    try {
      const json = await govFetchJSON<{ data: RegisteredStrategy[] }>("/route-strategies");
      setStrategies(json.data ?? []);
      setShowStrategyCatalog(true);
    } catch {
      setStrategies([]);
      setShowStrategyCatalog(true);
    }
  };

  // 打开编辑器——先拉取最新列表再切编辑态（plan-002 B-04：避免数据陈旧导致 UX 死循环）
  const openEditor = (profile?: RouteProfile) => {
    // 切换编辑态前先同步后端数据，确保编辑目标为最新
    void fetchProfiles();
    if (profile) {
      setEditingProfile(profile);
      setEditorForm({
        name: profile.name,
        description: profile.description,
        delta_cap: profile.delta_cap,
        max_attempts: profile.max_attempts,
        allow_fallback: profile.allow_fallback,
      });
      setStrategyConfigs(
        (profile.strategies_json ?? []).map((s) => ({
          code: s.code,
          enabled: s.enabled,
          priority: s.priority,
        }))
      );
    } else {
      setEditingProfile(null);
      setEditorForm({ name: "", description: "", delta_cap: 0, max_attempts: 3, allow_fallback: true });

      // 默认策略配置
      const defaults: { code: string; enabled: boolean; priority: number }[] = [
        { code: "S-COMPLIANCE", enabled: true, priority: 0 },
        { code: "S-PRI", enabled: true, priority: 10 },
        { code: "S-HEALTH", enabled: true, priority: 20 },
        { code: "S-WEIGHT", enabled: false, priority: 30 },
        { code: "S-COST", enabled: false, priority: 40 },
        { code: "S-AFFINITY", enabled: false, priority: 50 },
        { code: "S-LATENCY", enabled: false, priority: 60 },
        { code: "S-ERROR", enabled: false, priority: 70 },
        { code: "S-RATE", enabled: false, priority: 80 },
        { code: "S-TAG", enabled: false, priority: 90 },
        { code: "S-CLASSIFY", enabled: false, priority: 100 },
        { code: "S-CACHE", enabled: false, priority: 110 },
      ];
      setStrategyConfigs(defaults);
    }
    setShowEditor(true);
  };

  // 切换策略启用
  const toggleStrategy = (code: string) => {
    setStrategyConfigs((prev) =>
      prev.map((s) => (s.code === code ? { ...s, enabled: !s.enabled } : s))
    );
  };

  // 更新策略权重/优先级
  const updateStrategyPriority = (code: string, priority: number) => {
    setStrategyConfigs((prev) =>
      prev.map((s) => (s.code === code ? { ...s, priority } : s))
    );
  };

  // 保存档案
  const handleSave = async () => {
    if (editorForm.delta_cap > 0.2) {
      setError("delta_cap 不得超过 20%（0.20）");
      return;
    }
    setSaving(true);
    try {
      const enabledStrategies = strategyConfigs
        .filter((s) => s.enabled)
        .sort((a, b) => a.priority - b.priority);

      const body = {
        name: editorForm.name,
        description: editorForm.description,
        strategies_json: enabledStrategies.map((s) => ({
          code: s.code,
          enabled: true,
          priority: s.priority,
          config: {},
        })),
        delta_cap: editorForm.delta_cap,
        max_attempts: editorForm.max_attempts,
        allow_fallback: editorForm.allow_fallback,
      };

      const url = editingProfile
        ? `/route-profiles/${editingProfile.id}`
        : `/route-profiles`;
      const method = editingProfile ? "PUT" : "POST";

      await govFetch(url, {
        method,
        body: JSON.stringify(body),
      });
      setShowEditor(false);
      fetchProfiles();
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存路由档案失败");
    } finally {
      setSaving(false);
    }
  };

  // 表格列
  const columns: ColumnDef<RouteProfile>[] = [
    { key: "name", header: "档案名称" },
    {
      key: "strategies",
      header: "策略组合",
      render: (p) => (
        <div className="flex flex-wrap gap-1">
          {(p.strategies_json ?? []).map((s) => (
            <span
              key={s.code}
              className="inline-flex rounded bg-blue-100 px-1.5 py-0.5 text-xs font-medium text-blue-700"
            >
              {s.code}
            </span>
          ))}
        </div>
      ),
    },
    {
      key: "delta_cap",
      header: "δ 值",
      render: (p) => (
        <span className={`font-mono font-medium ${p.delta_cap > 0.15 ? "text-red-600" : "text-gray-700"}`}>
          {(p.delta_cap * 100).toFixed(0)}%
        </span>
      ),
    },
    {
      key: "status",
      header: "状态",
      render: (p) => (
        <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
          p.status === "active" ? "bg-green-100 text-green-800" : "bg-gray-100 text-gray-600"
        }`}>
          {p.status === "active" ? "生效" : "已归档"}
        </span>
      ),
    },
    {
      key: "actions",
      header: "操作",
      render: (p) => (
        <button
          onClick={(e) => { e.stopPropagation(); openEditor(p); }}
          className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-blue-600"
          title="编辑"
        >
          <Sliders className="h-4 w-4" />
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">路由档案</h1>
          <p className="mt-1 text-sm text-gray-500">管理路由策略组合与 δ 价格上限（硬上限 20%）</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={fetchStrategies}
            className="inline-flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            <Info className="h-4 w-4" />
            策略目录
          </button>
          <button
            onClick={() => openEditor()}
            className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            <Plus className="h-4 w-4" />
            新建档案
          </button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchProfiles} dismissible />}

      {/* 档案列表 */}
      <DataTable
        data={profiles}
        columns={columns}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无路由档案"
      />

      {/* ===== 路由档案编辑器 ===== */}
      {showEditor && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowEditor(false)} />
          <div className="relative z-10 mx-4 w-full max-w-3xl rounded-lg bg-white p-6 shadow-xl max-h-[85vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-gray-900">
              {editingProfile ? "编辑路由档案" : "新建路由档案"}
            </h2>

            {/* 基本信息 */}
            <div className="mt-4 grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">名称 *</label>
                <input
                  type="text"
                  value={editorForm.name}
                  onChange={(e) => setEditorForm({ ...editorForm, name: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">最大重试次数</label>
                <input
                  type="number"
                  value={editorForm.max_attempts}
                  onChange={(e) => setEditorForm({ ...editorForm, max_attempts: parseInt(e.target.value) || 3 })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  min={1}
                  max={10}
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium text-gray-700">描述</label>
                <textarea
                  value={editorForm.description}
                  onChange={(e) => setEditorForm({ ...editorForm, description: e.target.value })}
                  rows={2}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
            </div>

            {/* delta_cap 滑杆 */}
            <div className="mt-4">
              <label className="flex items-center justify-between text-sm font-medium text-gray-700">
                <span>delta_cap（价格上限偏差，硬上限 20%）</span>
                <span className={`font-mono font-bold ${editorForm.delta_cap > 0.2 ? "text-red-600" : "text-blue-600"}`}>
                  {(editorForm.delta_cap * 100).toFixed(0)}%
                </span>
              </label>
              <input
                type="range"
                min={0}
                max={0.2}
                step={0.01}
                value={editorForm.delta_cap}
                onChange={(e) => setEditorForm({ ...editorForm, delta_cap: parseFloat(e.target.value) })}
                className="mt-2 w-full"
              />
              <div className="flex justify-between text-xs text-gray-400">
                <span>0%</span>
                <span>10%</span>
                <span>20% (max)</span>
              </div>
              {editorForm.delta_cap > 0.2 && (
                <p className="mt-1 text-xs text-red-600">delta_cap 超出硬上限 20%，无法保存</p>
              )}
            </div>

            {/* 允许回退 */}
            <div className="mt-3 flex items-center gap-2">
              <input
                type="checkbox"
                id="allow-fallback"
                checked={editorForm.allow_fallback}
                onChange={(e) => setEditorForm({ ...editorForm, allow_fallback: e.target.checked })}
                className="rounded"
              />
              <label htmlFor="allow-fallback" className="text-sm text-gray-700">
                允许回退（无合格候选时降级）
              </label>
            </div>

            {/* 12 策略开关 + 权重滑块 */}
            <div className="mt-5">
              <h3 className="text-sm font-semibold text-gray-700">
                策略配置（12 种策略，启用的按优先级排序执行）
              </h3>
              <p className="mt-1 text-xs text-gray-400">
                S-COMPLIANCE 启用时必须排在第一位。拖动滑块调整优先级。
              </p>

              <div className="mt-3 space-y-2">
                {strategyConfigs.map((sc) => (
                  <div
                    key={sc.code}
                    className={`flex items-center gap-3 rounded-lg border px-3 py-2 ${
                      sc.enabled ? "border-blue-200 bg-blue-50" : "border-gray-200 bg-white"
                    }`}
                  >
                    {/* 启用开关 */}
                    <input
                      type="checkbox"
                      checked={sc.enabled}
                      onChange={() => toggleStrategy(sc.code)}
                      className="rounded"
                    />

                    {/* 策略名称 */}
                    <div className="min-w-0 flex-1">
                      <span className="text-sm font-medium text-gray-800">
                        {strategyNames[sc.code] ?? sc.code}
                      </span>
                      <span className="ml-2 text-xs text-gray-400">{sc.code}</span>
                    </div>

                    {/* 优先级滑块（仅启用时） */}
                    {sc.enabled && (
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-gray-500">优先级:</span>
                        <input
                          type="range"
                          min={0}
                          max={120}
                          value={sc.priority}
                          onChange={(e) => updateStrategyPriority(sc.code, parseInt(e.target.value))}
                          className="w-20"
                        />
                        <span className="w-6 text-right text-xs font-mono text-gray-600">{sc.priority}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* 操作按钮 */}
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setShowEditor(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleSave}
                disabled={!editorForm.name || saving}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 策略目录展示 ===== */}
      {showStrategyCatalog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowStrategyCatalog(false)} />
          <div className="relative z-10 mx-4 w-full max-w-2xl rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">已注册策略目录</h2>
            {strategies.length === 0 ? (
              <div className="mt-4 space-y-2">
                {allStrategyCodes.map((code) => (
                  <div key={code} className="rounded-lg border border-gray-200 bg-gray-50 p-3">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-gray-800">{strategyNames[code] ?? code}</span>
                      <span className="text-xs text-gray-400">{code}</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                      {code === "S-COMPLIANCE"
                        ? "硬策略，不可对受限主体关闭"
                        : "可选策略，支持启停组合"}
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="mt-4 space-y-2">
                {strategies.map((s) => (
                  <div key={s.code} className="rounded-lg border border-gray-200 bg-gray-50 p-3">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-gray-800">{s.name}</span>
                      <span className="text-xs text-gray-400">{s.code}</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">{s.description}</p>
                    {!s.can_disable && (
                      <span className="mt-1 inline-flex rounded bg-yellow-100 px-1.5 py-0.5 text-xs text-yellow-800">
                        不可禁用
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
            <div className="mt-4 flex justify-end">
              <button
                onClick={() => setShowStrategyCatalog(false)}
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
