"use client";

import React from "react";
import { ChevronLeft, ChevronRight, ChevronUp, ChevronDown, Search } from "lucide-react";

/** 表格列定义 */
export interface ColumnDef<T> {
  /** 数据字段键名 */
  key: string;
  /** 列标题 */
  header: string;
  /** 自定义渲染函数 */
  render?: (item: T) => React.ReactNode;
  /** 是否可排序 */
  sortable?: boolean;
  /** 列宽度（Tailwind 类名，如 "w-24"） */
  width?: string;
}

/** DataTable 属性 */
export interface DataTableProps<T> {
  /** 表格数据 */
  data: T[];
  /** 列定义 */
  columns: ColumnDef<T>[];
  /** 搜索占位文本（提供时显示搜索框） */
  searchPlaceholder?: string;
  /** 搜索过滤函数 */
  onSearch?: (query: string) => void;
  /** 当前页码（从 1 开始） */
  page?: number;
  /** 每页条数 */
  pageSize?: number;
  /** 总记录数 */
  total?: number;
  /** 页码变更回调 */
  onPageChange?: (page: number) => void;
  /** 排序字段变更回调 */
  onSort?: (key: string, direction: "asc" | "desc") => void;
  /** 当前排序字段 */
  sortKey?: string;
  /** 当前排序方向 */
  sortDirection?: "asc" | "desc";
  /** 行点击回调 */
  onRowClick?: (item: T) => void;
  /** 空状态提示文本 */
  emptyText?: string;
  /** 是否加载中 */
  loading?: boolean;
  /** 行唯一键字段名（用于 React key） */
  rowKey?: string;
}

/**
 * 通用数据表格组件 —— 支持排序、分页、搜索。
 * 用于列表类页面（Party 列表、流水记录、审计日志等）。
 * 组件不超过 300 行。
 */
export function DataTable<T extends Record<string, unknown>>({
  data,
  columns,
  searchPlaceholder,
  onSearch,
  page = 1,
  pageSize = 20,
  total,
  onPageChange,
  onSort,
  sortKey,
  sortDirection,
  onRowClick,
  emptyText = "暂无数据",
  loading = false,
  rowKey = "id",
}: DataTableProps<T>) {
  // 搜索输入状态
  const [searchQuery, setSearchQuery] = React.useState("");
  const totalPages = total !== undefined ? Math.ceil(total / pageSize) : 1;

  // 处理搜索提交
  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    onSearch?.(searchQuery);
  };

  // 处理排序切换
  const handleSortToggle = (key: string) => {
    if (!onSort) return;
    const newDirection =
      sortKey === key && sortDirection === "asc" ? "desc" : "asc";
    onSort(key, newDirection);
  };

  // 排序图标
  const sortIcon = (key: string) => {
    if (sortKey !== key) return null;
    return sortDirection === "asc" ? (
      <ChevronUp className="ml-1 h-3.5 w-3.5" />
    ) : (
      <ChevronDown className="ml-1 h-3.5 w-3.5" />
    );
  };

  // 获取单元格值
  const getCellValue = (item: T, col: ColumnDef<T>) => {
    if (col.render) return col.render(item);
    return String(item[col.key] ?? "");
  };

  return (
    <div className="rounded-lg border border-gray-200 bg-white">
      {/* 搜索栏 */}
      {searchPlaceholder && (
        <div className="border-b border-gray-200 px-4 py-3">
          <form onSubmit={handleSearch} className="relative max-w-xs">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={searchPlaceholder}
              className="w-full rounded-md border border-gray-300 py-2 pl-9 pr-4 text-sm text-gray-700 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </form>
        </div>
      )}

      {/* 表格 */}
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50 text-left">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-3 text-xs font-semibold uppercase tracking-wider text-gray-600 ${
                    col.sortable ? "cursor-pointer select-none hover:bg-gray-100" : ""
                  } ${col.width ?? ""}`}
                  onClick={() => col.sortable && handleSortToggle(col.key)}
                >
                  <span className="inline-flex items-center">
                    {col.header}
                    {col.sortable && sortIcon(col.key)}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              // 加载状态骨架
              Array.from({ length: 3 }).map((_, i) => (
                <tr key={`skel-${i}`} className="animate-pulse border-b border-gray-100">
                  {columns.map((col) => (
                    <td key={col.key} className="px-4 py-3">
                      <div className="h-4 w-3/4 rounded bg-gray-200" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data.length === 0 ? (
              // 空状态
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-4 py-12 text-center text-sm text-gray-400"
                >
                  {emptyText}
                </td>
              </tr>
            ) : (
              // 数据行
              data.map((item) => (
                <tr
                  key={String(item[rowKey] ?? Math.random())}
                  className={`border-b border-gray-100 transition-colors hover:bg-gray-50 ${
                    onRowClick ? "cursor-pointer" : ""
                  }`}
                  onClick={() => onRowClick?.(item)}
                >
                  {columns.map((col) => (
                    <td key={col.key} className="px-4 py-3 text-sm text-gray-700">
                      {getCellValue(item, col)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* 分页控件 */}
      {total !== undefined && total > 0 && (
        <div className="flex items-center justify-between border-t border-gray-200 px-4 py-3">
          <span className="text-sm text-gray-500">
            共 {total} 条记录，第 {page}/{totalPages} 页
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => onPageChange?.(page - 1)}
              disabled={page <= 1}
              className="inline-flex items-center rounded-md border border-gray-300 px-2 py-1 text-sm text-gray-600 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-sm text-gray-600">
              {page} / {totalPages}
            </span>
            <button
              onClick={() => onPageChange?.(page + 1)}
              disabled={page >= totalPages}
              className="inline-flex items-center rounded-md border border-gray-300 px-2 py-1 text-sm text-gray-600 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
