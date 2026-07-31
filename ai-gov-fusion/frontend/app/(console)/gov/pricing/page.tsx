"use client";

import React from "react";
import { Tags, Eye, Edit3, Plus } from "lucide-react";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { CodeBlock } from "../_components/CodeBlock";
import { ErrorAlert } from "../_components/ErrorAlert";
import { extractErrorMessage } from "@/lib/error-codes";

/** 价目数据结构 */
interface ModelPrice {
  id: string;
  model_id: string;
  channel_id: string | null;
  reference_id: string;
  price_json: Record<string, unknown>;
  status: "active" | "archived";
  effective_start_at: string | null;
  created_at: string;
}

/** 计价模式选项 */
const pricingModes = [
  { value: "flat_fee", label: "按次固定 (flat_fee)" },
  { value: "usage_per_unit", label: "按单位用量 (usage_per_unit)" },
  { value: "usage_tiered", label: "阶梯价格 (usage_tiered)" },
  { value: "usage_volume", label: "总量落档 (usage_volume)" },
  { value: "amortization_fixed", label: "固定摊销 (amortization_fixed)" },
];

/** itemCode 选项 */
const itemCodes = [
  "prompt_tokens",
  "completion_tokens",
  "prompt_cached_tokens",
  "prompt_write_cached_tokens",
  "completion_reasoning_tokens",
  "prompt_audio_tokens",
  "completion_audio_tokens",
  "image_count",
  "image_resolution_tier",
  "video_duration_seconds",
];

/** 单个定价项 */
interface PriceItem {
  itemCode: string;
  cost: { mode: string; rate: number; monthly_rate?: number };
  sell: { mode: string; rate: number; monthly_rate?: number; cache_discount_ratio?: number };
}

const API_BASE = "/gov";

/**
 * 价目维护页面 —— 渠道x模型价目列表、价目编辑器、双轨预览。
 * 对应 PRD UI-04 需求。
 */
export default function PricingPage() {
  // 列表状态
  const [prices, setPrices] = React.useState<ModelPrice[]>([]);
  const [total, setTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [modelFilter, setModelFilter] = React.useState("");

  // 编辑器状态
  const [editingPrice, setEditingPrice] = React.useState<ModelPrice | null>(null);
  const [showEditor, setShowEditor] = React.useState(false);
  const [editorForm, setEditorForm] = React.useState({
    model_id: "",
    channel_id: "",
    reference_id: "",
    effective_start_at: "",
  });
  const [priceItems, setPriceItems] = React.useState<PriceItem[]>([]);
  const [saving, setSaving] = React.useState(false);

  // 预览状态
  const [previewPrice, setPreviewPrice] = React.useState<ModelPrice | null>(null);

  // 获取价目列表
  const fetchPrices = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      if (modelFilter) params.set("model_id", modelFilter);
      const res = await fetch(`${API_BASE}/model-prices?${params}`);
      if (!res.ok) throw new Error(await extractErrorMessage(res));
      const json = await res.json();
      setPrices(json.data ?? []);
      setTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取价目列表失败");
    } finally {
      setLoading(false);
    }
  }, [page, modelFilter]);

  React.useEffect(() => { fetchPrices(); }, [fetchPrices]);

  // 新增空定价项
  const addPriceItem = () => {
    setPriceItems([
      ...priceItems,
      {
        itemCode: "prompt_tokens",
        cost: { mode: "usage_per_unit", rate: 0 },
        sell: { mode: "usage_per_unit", rate: 0 },
      },
    ]);
  };

  // 更新指定定价项
  const updatePriceItem = (index: number, field: string, value: unknown) => {
    const items = [...priceItems];
    const keys = field.split(".");

    if (keys.length === 3) {
      // 如 cost.mode, sell.rate
      const [track, subKey] = [keys[0], keys[1]] as [string, string];
      if (track === "cost" || track === "sell") {
        (items[index] as Record<string, unknown>)[track] = {
          ...((items[index] as Record<string, Record<string, unknown>>)[track]),
          [keys[2] as string]: value,
        };
      }
    } else if (keys[0] === "itemCode") {
      items[index].itemCode = value as string;
    }
    setPriceItems(items);
  };

  // 移除定价项
  const removePriceItem = (index: number) => {
    setPriceItems(priceItems.filter((_, i) => i !== index));
  };

  // 构建 price_json
  const buildPriceJson = (): Record<string, unknown> => ({
    items: priceItems,
    schedule: { timezone: "Asia/Shanghai", overrides: [] },
  });

  // 保存价目
  const handleSave = async () => {
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        model_id: editorForm.model_id,
        reference_id: editorForm.reference_id || crypto.randomUUID(),
        price_json: buildPriceJson(),
      };
      if (editorForm.channel_id) body.channel_id = editorForm.channel_id;
      if (editorForm.effective_start_at) body.effective_start_at = editorForm.effective_start_at;
      if (editingPrice) body.id = editingPrice.id;

      const res = await fetch(`${API_BASE}/model-prices`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        throw new Error(await extractErrorMessage(res));
      }
      setShowEditor(false);
      setEditingPrice(null);
      setPriceItems([]);
      fetchPrices();
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存价目失败");
    } finally {
      setSaving(false);
    }
  };

  // 打开编辑器（编辑已有价目）
  const openEditor = (price?: ModelPrice) => {
    if (price) {
      setEditingPrice(price);
      setEditorForm({
        model_id: price.model_id,
        channel_id: price.channel_id ?? "",
        reference_id: price.reference_id,
        effective_start_at: price.effective_start_at ?? "",
      });
      // 从 price_json.items 恢复定价项
      const items = (price.price_json as { items?: PriceItem[] })?.items ?? [];
      setPriceItems(items);
    } else {
      setEditingPrice(null);
      setEditorForm({ model_id: "", channel_id: "", reference_id: "", effective_start_at: "" });
      setPriceItems([]);
    }
    setShowEditor(true);
  };

  // 表格列
  const columns: ColumnDef<ModelPrice>[] = [
    { key: "model_id", header: "模型" },
    {
      key: "channel_id",
      header: "渠道",
      render: (p) => p.channel_id ?? <span className="text-gray-400">默认</span>,
    },
    { key: "reference_id", header: "版本标识" },
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
        <div className="flex gap-1">
          <button
            onClick={(e) => { e.stopPropagation(); setPreviewPrice(p); }}
            className="rounded p-1 text-gray-400 hover:bg-gray-100"
            title="预览双轨"
          >
            <Eye className="h-4 w-4" />
          </button>
          <button
            onClick={(e) => { e.stopPropagation(); openEditor(p); }}
            className="rounded p-1 text-gray-400 hover:bg-gray-100"
            title="编辑"
          >
            <Edit3 className="h-4 w-4" />
          </button>
        </div>
      ),
    },
  ];

  /** 计算双轨预览的行数据 */
  const buildDualTrackRows = (price: ModelPrice) => {
    const items = (price.price_json as { items?: PriceItem[] })?.items ?? [];
    return items.map((item) => ({
      itemCode: item.itemCode,
      costMode: item.cost.mode,
      costRate: item.cost.rate ?? item.cost.monthly_rate ?? 0,
      sellMode: item.sell.mode,
      sellRate: item.sell.rate ?? item.sell.monthly_rate ?? 0,
      cacheDiscount: item.sell.cache_discount_ratio ?? null,
    }));
  };

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">价目维护</h1>
          <p className="mt-1 text-sm text-gray-500">管理渠道x模型双轨价目（cost/sell）</p>
        </div>
        <button
          onClick={() => openEditor()}
          className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          新增价目
        </button>
      </div>

      {error && <ErrorAlert message={error} onRetry={fetchPrices} dismissible />}

      {/* 模型筛选 */}
      <div className="flex items-center gap-3">
        <label className="text-sm text-gray-600">按模型筛选：</label>
        <input
          type="text"
          value={modelFilter}
          onChange={(e) => { setModelFilter(e.target.value); setPage(1); }}
          placeholder="如 gpt-4o"
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm"
        />
      </div>

      {/* 价目列表 */}
      <DataTable
        data={prices}
        columns={columns}
        page={page}
        pageSize={20}
        total={total}
        onPageChange={setPage}
        loading={loading}
        emptyText="暂无价目数据"
      />

      {/* ===== 价目编辑器 ===== */}
      {showEditor && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowEditor(false)} />
          <div className="relative z-10 mx-4 w-full max-w-3xl rounded-lg bg-white p-6 shadow-xl max-h-[85vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-gray-900">
              {editingPrice ? "编辑价目" : "新增价目"}
            </h2>

            {/* 基本信息 */}
            <div className="mt-4 grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">模型 ID *</label>
                <input
                  type="text"
                  value={editorForm.model_id}
                  onChange={(e) => setEditorForm({ ...editorForm, model_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="如 gpt-4o"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">渠道 ID</label>
                <input
                  type="text"
                  value={editorForm.channel_id}
                  onChange={(e) => setEditorForm({ ...editorForm, channel_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="null = 默认价"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">版本标识 *</label>
                <input
                  type="text"
                  value={editorForm.reference_id}
                  onChange={(e) => setEditorForm({ ...editorForm, reference_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">生效时间</label>
                <input
                  type="datetime-local"
                  value={editorForm.effective_start_at}
                  onChange={(e) => setEditorForm({ ...editorForm, effective_start_at: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
              </div>
            </div>

            {/* 定价项列表 */}
            <div className="mt-6">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-700">定价项 (items)</h3>
                <button
                  onClick={addPriceItem}
                  className="inline-flex items-center gap-1 rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50"
                >
                  <Plus className="h-3 w-3" /> 添加项
                </button>
              </div>

              {priceItems.length === 0 && (
                <p className="mt-3 text-sm text-gray-400">暂无定价项，点击"添加项"开始配置</p>
              )}

              {priceItems.map((item, idx) => (
                <div key={idx} className="mt-3 rounded-lg border border-gray-200 bg-gray-50 p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-gray-500">定价项 #{idx + 1}</span>
                    <button
                      onClick={() => removePriceItem(idx)}
                      className="text-xs text-red-500 hover:text-red-700"
                    >
                      移除
                    </button>
                  </div>

                  <div className="mt-2 grid grid-cols-3 gap-3">
                    {/* itemCode */}
                    <div>
                      <label className="block text-xs text-gray-500">itemCode</label>
                      <select
                        value={item.itemCode}
                        onChange={(e) => updatePriceItem(idx, "itemCode", e.target.value)}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                      >
                        {itemCodes.map((c) => (
                          <option key={c} value={c}>{c}</option>
                        ))}
                      </select>
                    </div>

                    {/* cost 模式 */}
                    <div>
                      <label className="block text-xs text-gray-500">cost 模式</label>
                      <select
                        value={item.cost.mode}
                        onChange={(e) => updatePriceItem(idx, "cost.mode", e.target.value)}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                      >
                        {pricingModes.map((m) => (
                          <option key={m.value} value={m.value}>{m.value}</option>
                        ))}
                      </select>
                    </div>

                    {/* cost 费率 */}
                    <div>
                      <label className="block text-xs text-gray-500">cost 费率</label>
                      <input
                        type="number"
                        value={item.cost.rate ?? item.cost.monthly_rate ?? 0}
                        onChange={(e) => updatePriceItem(idx, "cost.rate", parseFloat(e.target.value))}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                        step="0.0001"
                      />
                    </div>

                    {/* sell 模式 */}
                    <div>
                      <label className="block text-xs text-gray-500">sell 模式</label>
                      <select
                        value={item.sell.mode}
                        onChange={(e) => updatePriceItem(idx, "sell.mode", e.target.value)}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                      >
                        {pricingModes.map((m) => (
                          <option key={m.value} value={m.value}>{m.value}</option>
                        ))}
                      </select>
                    </div>

                    {/* sell 费率 */}
                    <div>
                      <label className="block text-xs text-gray-500">sell 费率</label>
                      <input
                        type="number"
                        value={item.sell.rate ?? item.sell.monthly_rate ?? 0}
                        onChange={(e) => updatePriceItem(idx, "sell.rate", parseFloat(e.target.value))}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                        step="0.0001"
                      />
                    </div>

                    {/* 缓存折扣比率 */}
                    <div>
                      <label className="block text-xs text-gray-500">
                        缓存折扣比率 (0-1)
                      </label>
                      <input
                        type="number"
                        value={item.sell.cache_discount_ratio ?? ""}
                        onChange={(e) => updatePriceItem(idx, "sell.cache_discount_ratio", e.target.value ? parseFloat(e.target.value) : undefined)}
                        className="mt-1 block w-full rounded-md border border-gray-300 px-2 py-1 text-xs"
                        min="0"
                        max="1"
                        step="0.1"
                        placeholder="可选"
                      />
                    </div>
                  </div>
                </div>
              ))}
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
                disabled={!editorForm.model_id || saving}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {saving ? "保存中..." : "保存"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ===== 双轨预览面板 ===== */}
      {previewPrice && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setPreviewPrice(null)} />
          <div className="relative z-10 mx-4 w-full max-w-3xl rounded-lg bg-white p-6 shadow-xl max-h-[85vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-gray-900">
              双轨预览 — {previewPrice.model_id}
            </h2>
            <p className="mt-1 text-sm text-gray-500">
              渠道：{previewPrice.channel_id ?? "默认"} | 版本：{previewPrice.reference_id}
            </p>

            {/* cost/sell 对比表格 */}
            <table className="mt-4 w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left">
                  <th className="py-2 font-medium text-gray-600">itemCode</th>
                  <th className="py-2 font-medium text-gray-600">cost 模式</th>
                  <th className="py-2 font-medium text-blue-700">cost 费率</th>
                  <th className="py-2 font-medium text-gray-600">sell 模式</th>
                  <th className="py-2 font-medium text-purple-700">sell 费率</th>
                  <th className="py-2 font-medium text-gray-600">缓存折扣</th>
                </tr>
              </thead>
              <tbody>
                {buildDualTrackRows(previewPrice).map((row, i) => (
                  <tr key={i} className="border-b border-gray-100">
                    <td className="py-2 font-mono text-xs">{row.itemCode}</td>
                    <td className="py-2 text-xs text-gray-500">{row.costMode}</td>
                    <td className="py-2 font-mono text-blue-700">{row.costRate}</td>
                    <td className="py-2 text-xs text-gray-500">{row.sellMode}</td>
                    <td className="py-2 font-mono text-purple-700">{row.sellRate}</td>
                    <td className="py-2 text-xs">
                      {row.cacheDiscount !== null ? `${(row.cacheDiscount * 100).toFixed(0)}%` : "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* 原始 JSON 展示 */}
            <div className="mt-4">
              <h3 className="text-sm font-medium text-gray-700">原始价目 JSON</h3>
              <div className="mt-2">
                <CodeBlock data={previewPrice.price_json} maxHeight="300px" />
              </div>
            </div>

            <div className="mt-4 flex justify-end">
              <button
                onClick={() => setPreviewPrice(null)}
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
