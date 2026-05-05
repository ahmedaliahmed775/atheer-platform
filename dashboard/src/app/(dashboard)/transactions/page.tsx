// قائمة المعاملات — TanStack Table مع فلاتر وصفحات وتصدير Excel
// WALLET_ADMIN يرى محفظته فقط (scope من JWT)
"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import Link from "next/link";
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { apiGet } from "@/lib/api";
import { getUserScope } from "@/lib/auth";

// ── الأنواع ──

/** بيانات المعاملة */
interface Transaction {
  id: number;
  transactionId: string;
  payerPublicId: string;
  merchantId: string;
  payerWalletId: string;
  merchantWalletId: string;
  amount: number;
  currency: string;
  counter: number;
  status: string;
  errorCode: string;
  durationMs: number;
  debitRef: string;
  creditRef: string;
  connectionSource: string;
  createdAt: string;
}

/** استجابة قائمة المعاملات من السويتش */
interface TransactionsResponse {
  transactions: Transaction[];
  totalCount: number;
  page: number;
  pageSize: number;
}

/** حالات المعاملات */
const STATUS_OPTIONS = [
  { value: "", label: "الكل" },
  { value: "SUCCESS", label: "ناجحة" },
  { value: "FAILED", label: "فاشلة" },
  { value: "PENDING", label: "معلّقة" },
  { value: "REVERSED", label: "معكوسة" },
];

/** تنسيق المبلغ */
function formatAmount(amount: number, currency: string): string {
  const val = amount / 100;
  return new Intl.NumberFormat("ar-YE", {
    style: "currency",
    currency: currency || "YER",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(val);
}

/** تنسيق التاريخ */
function formatDate(dateStr: string): string {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  return new Intl.DateTimeFormat("ar-YE", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** شارة الحالة */
function StatusBadge({ status }: { status: string }) {
  const config: Record<
    string,
    { label: string; className: string }
  > = {
    SUCCESS: {
      label: "ناجحة",
      className: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
    },
    FAILED: {
      label: "فاشلة",
      className: "bg-red-500/10 text-red-400 border-red-500/20",
    },
    PENDING: {
      label: "معلّقة",
      className: "bg-amber-500/10 text-amber-400 border-amber-500/20",
    },
    REVERSED: {
      label: "معكوسة",
      className: "bg-violet-500/10 text-violet-400 border-violet-500/20",
    },
  };

  const c = config[status] || {
    label: status,
    className: "bg-muted text-muted-foreground",
  };

  return (
    <Badge variant="outline" className={c.className}>
      {c.label}
    </Badge>
  );
}

/** حجم الصفحة الافتراضي */
const PAGE_SIZE = 20;

export default function TransactionsPage() {
  // ── حالة الفلاتر ──
  const [status, setStatus] = useState("");
  const [walletId, setWalletId] = useState("");
  const [payerPublicId, setPayerPublicId] = useState("");
  const [merchantId, setMerchantId] = useState("");
  const [connectionSource, setConnectionSource] = useState("");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");
  const [filtersOpen, setFiltersOpen] = useState(false);

  // ── حالة البيانات ──
  const [data, setData] = useState<Transaction[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // ── Pagination ──
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: PAGE_SIZE,
  });

  // نطاق WALLET_ADMIN
  const scope = getUserScope();
  const effectiveWalletId = scope && scope !== "global" ? scope : walletId;

  /** جلب المعاملات */
  const fetchTransactions = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {
        page: String(pagination.pageIndex + 1),
        pageSize: String(pagination.pageSize),
      };
      if (status) params.status = status;
      if (effectiveWalletId) params.walletId = effectiveWalletId;
      if (payerPublicId) params.payerPublicId = payerPublicId;
      if (merchantId) params.merchantId = merchantId;
      if (connectionSource) params.connectionSource = connectionSource;
      if (fromDate) params.fromDate = fromDate;
      if (toDate) params.toDate = toDate;

      const res = await apiGet<TransactionsResponse>(
        "/admin/v1/transactions",
        params
      );
      setData(res.transactions || []);
      setTotalCount(res.totalCount || 0);
    } catch {
      setData([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [pagination, status, effectiveWalletId, payerPublicId, merchantId, connectionSource, fromDate, toDate]);

  useEffect(() => {
    fetchTransactions();
  }, [fetchTransactions]);

  /** تصدير Excel */
  const handleExport = useCallback(async () => {
    try {
      // جلب كل المعاملات المصفّاة (بدون صفحات)
      const params: Record<string, string> = {
        page: "1",
        pageSize: "10000",
      };
      if (status) params.status = status;
      if (effectiveWalletId) params.walletId = effectiveWalletId;
      if (payerPublicId) params.payerPublicId = payerPublicId;
      if (merchantId) params.merchantId = merchantId;
      if (connectionSource) params.connectionSource = connectionSource;
      if (fromDate) params.fromDate = fromDate;
      if (toDate) params.toDate = toDate;

      const res = await apiGet<TransactionsResponse>(
        "/admin/v1/transactions",
        params
      );

      // استيراد xlsx ديناميكياً
      const XLSX = await import("xlsx");
      const wsData = (res.transactions || []).map((tx) => ({
        "معرّف المعاملة": tx.transactionId,
        "التاريخ": formatDate(tx.createdAt),
        "الدافع": tx.payerPublicId,
        "التاجر": tx.merchantId,
        "المحفظة": tx.payerWalletId,
        "المبلغ": tx.amount / 100,
        "العملة": tx.currency,
        "الحالة": tx.status,
        "رمز الخطأ": tx.errorCode || "—",
        "المدة (ms)": tx.durationMs,
        "مصدر الاتصال": tx.connectionSource === "carrier" ? "اتصالات" : "إنترنت",
      }));

      const ws = XLSX.utils.json_to_sheet(wsData);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, "المعاملات");
      XLSX.writeFile(wb, `atheer_transactions_${Date.now()}.xlsx`);
    } catch {
      // تجاهل أخطاء التصدير
    }
  }, [status, effectiveWalletId, payerPublicId, merchantId, fromDate, toDate]);

  // ── تعريف الأعمدة ──
  const columns = useMemo<ColumnDef<Transaction>[]>(
    () => [
      {
        accessorKey: "transactionId",
        header: "المعرّف",
        cell: ({ row }) => {
          const id = row.original.transactionId;
          return (
            <Link
              href={`/transactions/${id}`}
              className="font-mono text-xs text-blue-400 hover:underline"
              dir="ltr"
            >
              {id?.slice(0, 8)}…
            </Link>
          );
        },
      },
      {
        accessorKey: "createdAt",
        header: "التاريخ",
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDate(row.original.createdAt)}
          </span>
        ),
      },
      {
        accessorKey: "payerPublicId",
        header: "الزبون",
        cell: ({ row }) => (
          <span className="font-mono text-xs" dir="ltr">
            {row.original.payerPublicId?.slice(0, 12)}
          </span>
        ),
      },
      {
        accessorKey: "merchantId",
        header: "التاجر",
        cell: ({ row }) => (
          <span className="font-mono text-xs" dir="ltr">
            {row.original.merchantId?.slice(0, 12)}
          </span>
        ),
      },
      {
        accessorKey: "payerWalletId",
        header: "المحفظة",
        cell: ({ row }) => (
          <Badge variant="secondary" className="text-[10px]">
            {row.original.payerWalletId}
          </Badge>
        ),
      },
      {
        accessorKey: "amount",
        header: "المبلغ",
        cell: ({ row }) => (
          <span className="font-semibold">
            {formatAmount(row.original.amount, row.original.currency)}
          </span>
        ),
      },
      {
        accessorKey: "status",
        header: "الحالة",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
      {
        accessorKey: "connectionSource",
        header: "المصدر",
        cell: ({ row }) => {
          const src = row.original.connectionSource || "internet";
          return (
            <Badge
              variant="outline"
              className={
                src === "carrier"
                  ? "text-amber-500 border-amber-500/30"
                  : "text-blue-400 border-blue-400/30"
              }
            >
              {src === "carrier" ? "اتصالات" : "إنترنت"}
            </Badge>
          );
        },
      },
      {
        accessorKey: "durationMs",
        header: "المدة",
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground" dir="ltr">
            {row.original.durationMs}ms
          </span>
        ),
      },
    ],
    []
  );

  // ── إعداد الجدول ──
  const table = useReactTable({
    data,
    columns,
    pageCount: Math.ceil(totalCount / pagination.pageSize),
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  });

  const totalPages = Math.ceil(totalCount / pagination.pageSize);

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">المعاملات</h1>
          <p className="text-sm text-muted-foreground">
            عرض وتصفية جميع المعاملات المالية
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setFiltersOpen(!filtersOpen)}
            className="flex items-center gap-2 rounded-lg border border-border/50 bg-card px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted/50"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>
            الفلاتر
          </button>
          <button
            onClick={handleExport}
            className="flex items-center gap-2 rounded-lg bg-gradient-to-l from-emerald-600 to-emerald-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-emerald-600/20 transition-all hover:shadow-emerald-600/30"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>
            تصدير Excel
          </button>
        </div>
      </div>

      {/* الفلاتر */}
      {filtersOpen && (
        <Card className="border-border/50 animate-in fade-in slide-in-from-top-2 duration-200">
          <CardContent className="grid gap-4 p-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
            {/* الحالة */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                الحالة
              </label>
              <select
                value={status}
                onChange={(e) => {
                  setStatus(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
            </div>

            {/* المحفظة — مخفية لـ WALLET_ADMIN */}
            {(!scope || scope === "global") && (
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-muted-foreground">
                  المحفظة
                </label>
                <input
                  type="text"
                  value={walletId}
                  onChange={(e) => {
                    setWalletId(e.target.value);
                    setPagination((p) => ({ ...p, pageIndex: 0 }));
                  }}
                  placeholder="jawali"
                  dir="ltr"
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
              </div>
            )}

            {/* معرّف الدافع */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                معرّف الدافع
              </label>
              <input
                type="text"
                value={payerPublicId}
                onChange={(e) => {
                  setPayerPublicId(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                placeholder="usr_..."
                dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>

            {/* معرّف التاجر */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                معرّف التاجر
              </label>
              <input
                type="text"
                value={merchantId}
                onChange={(e) => {
                  setMerchantId(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                placeholder="merchant_..."
                dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>

            {/* مصدر الاتصال */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                مصدر الاتصال
              </label>
              <select
                value={connectionSource}
                onChange={(e) => {
                  setConnectionSource(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <option value="">الكل</option>
                <option value="internet">إنترنت</option>
                <option value="carrier">اتصالات</option>
              </select>
            </div>

            {/* من تاريخ */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                من تاريخ
              </label>
              <input
                type="date"
                value={fromDate}
                onChange={(e) => {
                  setFromDate(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>

            {/* إلى تاريخ */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">
                إلى تاريخ
              </label>
              <input
                type="date"
                value={toDate}
                onChange={(e) => {
                  setToDate(e.target.value);
                  setPagination((p) => ({ ...p, pageIndex: 0 }));
                }}
                dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
            </div>
          </CardContent>
        </Card>
      )}

      {/* الجدول */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="border-border/50 hover:bg-transparent">
                    {headerGroup.headers.map((header) => (
                      <TableHead
                        key={header.id}
                        className="text-xs font-semibold text-muted-foreground"
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext()
                            )}
                      </TableHead>
                    ))}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="h-48 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
                        <span className="text-sm text-muted-foreground">
                          جارٍ التحميل...
                        </span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : table.getRowModel().rows.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="h-48 text-center">
                      <div className="flex flex-col items-center gap-2">
                        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground/30"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                        <span className="text-sm text-muted-foreground">
                          لا توجد معاملات
                        </span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      className="border-border/30 transition-colors hover:bg-muted/30"
                    >
                      {row.getVisibleCells().map((cell) => (
                        <TableCell key={cell.id}>
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext()
                          )}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* شريط الصفحات */}
          <div className="flex items-center justify-between border-t border-border/50 px-4 py-3">
            <p className="text-sm text-muted-foreground">
              إجمالي {totalCount.toLocaleString("ar-YE")} معاملة
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => table.previousPage()}
                disabled={!table.getCanPreviousPage()}
                className="flex h-8 items-center rounded-md border border-border/50 bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/50 disabled:pointer-events-none disabled:opacity-50"
              >
                السابقة
              </button>
              <span className="text-sm text-muted-foreground">
                {pagination.pageIndex + 1} / {totalPages || 1}
              </span>
              <button
                onClick={() => table.nextPage()}
                disabled={!table.getCanNextPage()}
                className="flex h-8 items-center rounded-md border border-border/50 bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/50 disabled:pointer-events-none disabled:opacity-50"
              >
                التالية
              </button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
