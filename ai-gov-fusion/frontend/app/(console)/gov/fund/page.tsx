"use client";

import React from "react";
import { Wallet, ArrowRightLeft, ScrollText, AlertTriangle, RefreshCw } from "lucide-react";
import { StatCard } from "../_components/StatCard";
import { DataTable, type ColumnDef } from "../_components/DataTable";
import { ConfirmDialog } from "../_components/ConfirmDialog";
import { ErrorAlert } from "../_components/ErrorAlert";
import { govFetch, govFetchJSON } from "@/lib/gov-api";

/** 账户数据结构 */
interface Account extends Record<string, unknown> {
  id: string;
  party_id: string;
  party_name: string;
  available_balance: number;
  frozen_balance: number;
  status: "active" | "frozen" | "liquidating" | "closed";
  /** 预算信息 —— 后端可能不返回该嵌套对象，访问时需可选链 */
  budget?: {
    limit_amount: number | null;
    warn_ratio: number | null;
    period: string;
    consumed_amount: number;
    consumption_pct: number;
    warn_active: boolean;
  } | null;
}

/** 流水记录 */
interface LedgerEntry extends Record<string, unknown> {
  id: string;
  account_id: string;
  direction: "debit" | "credit" | "freeze" | "unfreeze" | "settle";
  amount: number;
  balance_after: number;
  reason: string;
  created_at: string;
}

/**
 * 资金操作页面 —— 账户总览、划拨操作、流水历史、清算操作。
 * 对应 PRD UI-03 需求。
 */
export default function FundPage() {
  // 账户数据
  const [accounts, setAccounts] = React.useState<Account[]>([]);
  const [accountTotal, setAccountTotal] = React.useState(0);
  const [page, setPage] = React.useState(1);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  // 选中的账户
  const [selectedAccount, setSelectedAccount] = React.useState<Account | null>(null);

  // 流水数据
  const [ledgers, setLedgers] = React.useState<LedgerEntry[]>([]);
  const [ledgerTotal, setLedgerTotal] = React.useState(0);
  const [ledgerPage, setLedgerPage] = React.useState(1);
  const [ledgerLoading, setLedgerLoading] = React.useState(false);
  const [ledgerDirection, setLedgerDirection] = React.useState("");

  // 划拨表单
  const [showAllocate, setShowAllocate] = React.useState(false);
  const [allocateForm, setAllocateForm] = React.useState({
    dst_account_id: "",
    amount: "",
    reason: "",
  });
  const [allocating, setAllocating] = React.useState(false);

  // 清算确认
  const [showLiquidateConfirm, setShowLiquidateConfirm] = React.useState(false);

  // 获取账户列表
  const fetchAccounts = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: "20" });
      const json = await govFetchJSON<{ data: Account[]; total: number }>(`/accounts?${params}`);
      setAccounts(json.data ?? []);
      setAccountTotal(json.total ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取账户列表失败");
    } finally {
      setLoading(false);
    }
  }, [page]);

  React.useEffect(() => { fetchAccounts(); }, [fetchAccounts]);

  // 获取流水列表
  const fetchLedgers = React.useCallback(async (accountId: string) => {
    setLedgerLoading(true);
    try {
      const params = new URLSearchParams({ page: String(ledgerPage), page_size: "20" });
      if (ledgerDirection) params.set("direction", ledgerDirection);
      const json = await govFetchJSON<{ data: LedgerEntry[]; total: number }>(`/accounts/${accountId}/ledgers?${params}`);
      setLedgers(json.data ?? []);
      setLedgerTotal(json.total ?? 0);
    } catch {
      setLedgers([]);
    } finally {
      setLedgerLoading(false);
    }
  }, [ledgerPage, ledgerDirection]);

  // 选择账户时加载流水
  React.useEffect(() => {
    if (selectedAccount) fetchLedgers(selectedAccount.id);
  }, [selectedAccount, fetchLedgers]);

  // 执行划拨
  const handleAllocate = async () => {
    if (!selectedAccount) return;
    // 金额必须为正数
    const amount = parseFloat(allocateForm.amount);
    if (!Number.isFinite(amount) || amount <= 0) {
      setError("划拨金额必须为大于 0 的数字");
      return;
    }
    setAllocating(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      await govFetch(
        `/accounts/${selectedAccount.id}/allocate`,
        {
          method: "POST",
          headers: {
            "Idempotency-Key": idempotencyKey,
          },
          body: JSON.stringify({
            dst_account_id: allocateForm.dst_account_id,
            amount,
            channel: "parent", // 默认使用 parent 划拨通道
            reason: allocateForm.reason || undefined,
          }),
        }
      );
      setShowAllocate(false);
      setAllocateForm({ dst_account_id: "", amount: "", reason: "" });
      fetchAccounts();
      fetchLedgers(selectedAccount.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "划拨操作失败");
    } finally {
      setAllocating(false);
    }
  };

  // 执行清算
  const handleLiquidate = async () => {
    if (!selectedAccount) return;
    try {
      const idempotencyKey = crypto.randomUUID();
      await govFetch(
        `/accounts/${selectedAccount.id}/liquidate`,
        {
          method: "POST",
          headers: {
            "Idempotency-Key": idempotencyKey,
          },
          body: JSON.stringify({
            target_account_id: allocateForm.dst_account_id || selectedAccount.id,
            party_id: selectedAccount.party_id, // 关联 Party ID，清算必填
            reason: "管理员手动清算",
          }),
        }
      );
      setShowLiquidateConfirm(false);
      fetchAccounts();
    } catch (err) {
      setError(err instanceof Error ? err.message : "清算操作失败");
    }
  };

  // 计算汇总数据
  const totalAvailable = accounts.reduce((s, a) => s + a.available_balance, 0);
  const totalFrozen = accounts.reduce((s, a) => s + a.frozen_balance, 0);
  const budgetConsumption = accounts.length > 0
    ? accounts.reduce((s, a) => s + (a.budget?.consumed_amount ?? 0), 0)
    : 0;
  const budgetLimit = accounts.reduce((s, a) => s + (a.budget?.limit_amount ?? 0), 0);

  // 格式化金额显示
  const fmtMoney = (v: number) =>
    new Intl.NumberFormat("zh-CN", { style: "currency", currency: "CNY" }).format(v);

  // 账户表格列
  const accountColumns: ColumnDef<Account>[] = [
    { key: "party_name", header: "所属 Party" },
    {
      key: "available_balance",
      header: "可用余额",
      render: (a) => (
        <span className="font-mono font-medium text-green-700">
          {fmtMoney(a.available_balance)}
        </span>
      ),
    },
    {
      key: "frozen_balance",
      header: "冻结金额",
      render: (a) => (
        <span className="font-mono text-gray-500">{fmtMoney(a.frozen_balance)}</span>
      ),
    },
    {
      key: "status",
      header: "状态",
      render: (a) => (
        <span
          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
            a.status === "active" ? "bg-green-100 text-green-800" : "bg-gray-100 text-gray-600"
          }`}
        >
          {a.status === "active" ? "活跃" : a.status}
        </span>
      ),
    },
    {
      key: "budget",
      header: "预算进度",
      render: (a) => (
        <div className="w-32">
          <div className="flex justify-between text-xs text-gray-500">
            <span>{a.budget?.consumption_pct?.toFixed(0) ?? 0}%</span>
            {a.budget?.warn_active && (
              <span className="text-yellow-600">告警</span>
            )}
          </div>
          <div className="mt-1 h-1.5 rounded-full bg-gray-200">
            <div
              className={`h-full rounded-full transition-all ${
                (a.budget?.consumption_pct ?? 0) > 90 ? "bg-red-500" :
                (a.budget?.consumption_pct ?? 0) > 70 ? "bg-yellow-500" :
                "bg-blue-500"
              }`}
              style={{ width: `${Math.min(a.budget?.consumption_pct ?? 0, 100)}%` }}
            />
          </div>
        </div>
      ),
    },
  ];

  // 流水表格列
  const ledgerColumns: ColumnDef<LedgerEntry>[] = [
    {
      key: "direction",
      header: "方向",
      render: (l) => {
        const style: Record<string, { label: string; color: string }> = {
          debit: { label: "支出", color: "text-red-600" },
          credit: { label: "收入", color: "text-green-600" },
          freeze: { label: "冻结", color: "text-blue-600" },
          unfreeze: { label: "解冻", color: "text-teal-600" },
          settle: { label: "结算", color: "text-purple-600" },
        };
        const s = style[l.direction] ?? { label: l.direction, color: "text-gray-600" };
        return <span className={`font-medium ${s.color}`}>{s.label}</span>;
      },
    },
    {
      key: "amount",
      header: "金额",
      render: (l) => <span className="font-mono">{fmtMoney(l.amount)}</span>,
    },
    {
      key: "balance_after",
      header: "操作后余额",
      render: (l) => <span className="font-mono text-gray-500">{fmtMoney(l.balance_after)}</span>,
    },
    { key: "reason", header: "说明" },
    {
      key: "created_at",
      header: "时间",
      render: (l) => new Date(l.created_at).toLocaleString("zh-CN"),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">资金操作</h1>
        <p className="mt-1 text-sm text-gray-500">管理账户余额、资金划拨、流水查询与清算</p>
      </div>

      {/* 错误提示 */}
      {error && <ErrorAlert message={error} onRetry={fetchAccounts} dismissible />}

      {/* 账户总览卡片 */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="可用余额"
          value={fmtMoney(totalAvailable)}
          icon={Wallet}
          colorClass="text-green-600"
        />
        <StatCard
          title="冻结金额"
          value={fmtMoney(totalFrozen)}
          icon={Wallet}
          colorClass="text-blue-600"
        />
        <StatCard
          title="预算消耗"
          value={`${budgetLimit > 0 ? ((budgetConsumption / budgetLimit) * 100).toFixed(1) : 0}%`}
          description={`${fmtMoney(budgetConsumption)} / ${fmtMoney(budgetLimit)}`}
          icon={ArrowRightLeft}
          colorClass="text-yellow-600"
        />
        <StatCard
          title="活跃账户"
          value={accounts.filter((a) => a.status === "active").length}
          icon={ScrollText}
          colorClass="text-purple-600"
        />
      </div>

      {/* 账户列表 */}
      <section>
        <h2 className="mb-3 text-lg font-semibold text-gray-900">账户列表</h2>
        <DataTable
          data={accounts}
          columns={accountColumns}
          page={page}
          pageSize={20}
          total={accountTotal}
          onPageChange={setPage}
          loading={loading}
          onRowClick={(a) => setSelectedAccount(a)}
          emptyText="暂无账户数据"
        />
      </section>

      {/* 选中账户的详情与操作 */}
      {selectedAccount && (
        <section className="rounded-lg border border-gray-200 bg-white p-6">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">
              当前账户：{selectedAccount.party_name}
            </h2>
            <div className="flex gap-2">
              {/* 划拨按钮 */}
              <button
                onClick={() => {
                  setShowAllocate(true);
                  setAllocateForm({ dst_account_id: "", amount: "", reason: "" });
                }}
                className="inline-flex items-center gap-1.5 rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
              >
                <ArrowRightLeft className="h-4 w-4" />
                划拨
              </button>
              {/* 清算按钮（危险操作） */}
              <button
                onClick={() => setShowLiquidateConfirm(true)}
                disabled={selectedAccount.status !== "active"}
                className="inline-flex items-center gap-1.5 rounded-md border border-red-300 px-3 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
              >
                <AlertTriangle className="h-4 w-4" />
                清算
              </button>
            </div>
          </div>

          {/* 账户详细信息 */}
          <div className="mt-4 grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
            <div>
              <span className="text-gray-500">可用余额</span>
              <p className="font-mono font-medium text-green-700">{fmtMoney(selectedAccount.available_balance)}</p>
            </div>
            <div>
              <span className="text-gray-500">冻结金额</span>
              <p className="font-mono font-medium text-blue-700">{fmtMoney(selectedAccount.frozen_balance)}</p>
            </div>
            <div>
              <span className="text-gray-500">状态</span>
              <p className="font-medium">{selectedAccount.status}</p>
            </div>
            <div>
              <span className="text-gray-500">预算使用率</span>
              <p className={`font-medium ${(selectedAccount.budget?.consumption_pct ?? 0) > 80 ? "text-red-600" : "text-gray-700"}`}>
                {selectedAccount.budget?.consumption_pct?.toFixed(1) ?? 0}%
              </p>
            </div>
          </div>

          {/* 流水记录 */}
          <div className="mt-6">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="font-medium text-gray-900">流水历史</h3>
              <select
                value={ledgerDirection}
                onChange={(e) => { setLedgerDirection(e.target.value); setLedgerPage(1); }}
                className="rounded-md border border-gray-300 px-2 py-1 text-sm"
              >
                <option value="">全部方向</option>
                <option value="debit">支出</option>
                <option value="credit">收入</option>
                <option value="freeze">冻结</option>
                <option value="unfreeze">解冻</option>
                <option value="settle">结算</option>
              </select>
            </div>
            <DataTable
              data={ledgers}
              columns={ledgerColumns}
              page={ledgerPage}
              pageSize={20}
              total={ledgerTotal}
              onPageChange={setLedgerPage}
              loading={ledgerLoading}
              emptyText="暂无流水记录"
            />
          </div>
        </section>
      )}

      {/* 划拨操作对话框 */}
      {showAllocate && selectedAccount && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="fixed inset-0 bg-black/50" onClick={() => setShowAllocate(false)} />
          <div className="relative z-10 mx-4 w-full max-w-md rounded-lg bg-white p-6 shadow-xl">
            <h2 className="text-lg font-semibold text-gray-900">资金划拨</h2>
            <p className="mt-1 text-sm text-gray-500">
              从 {selectedAccount.party_name}（余额 {fmtMoney(selectedAccount.available_balance)}）划出
            </p>
            <div className="mt-4 space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">目标账户 ID</label>
                <input
                  type="text"
                  value={allocateForm.dst_account_id}
                  onChange={(e) => setAllocateForm({ ...allocateForm, dst_account_id: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="UUID"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">金额</label>
                <input
                  type="number"
                  value={allocateForm.amount}
                  onChange={(e) => setAllocateForm({ ...allocateForm, amount: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="0.00"
                  min="0"
                  step="0.01"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">划拨说明</label>
                <input
                  type="text"
                  value={allocateForm.reason}
                  onChange={(e) => setAllocateForm({ ...allocateForm, reason: e.target.value })}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="可选"
                />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => setShowAllocate(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                取消
              </button>
              <button
                onClick={handleAllocate}
                disabled={!allocateForm.dst_account_id || !allocateForm.amount || allocating}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {allocating ? "划拨中..." : "确认划拨"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 清算确认弹窗 */}
      <ConfirmDialog
        open={showLiquidateConfirm}
        title="确认清算"
        message={`确定要对账户「${selectedAccount?.party_name}」执行清算操作？清算后将冻结账户、排空冻结金额并转移剩余余额。此操作不可撤销。`}
        danger
        confirmLabel="确认清算"
        loading={false}
        onConfirm={handleLiquidate}
        onCancel={() => setShowLiquidateConfirm(false)}
      />
    </div>
  );
}
