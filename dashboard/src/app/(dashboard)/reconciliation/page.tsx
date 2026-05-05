// التسوية — تشغيل تسوية جديدة + قائمة التقارير + تفاصيل + تصدير Excel
"use client";

import React, { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { apiGet, apiPost } from "@/lib/api";
import { hasRole } from "@/lib/auth";

// ── الأنواع ──
interface ReconReport {
  id: number; reportDate: string; walletId: string;
  totalTxCount: number; totalAmount: number; successCount: number;
  failedCount: number; disputedCount: number; status: string;
  notes: string; createdAt: string; updatedAt: string;
}
interface ReconListRes { reports: ReconReport[]; totalCount: number; page: number; pageSize: number; }
interface RunReconRes { reports: { walletId: string; reportDate: string; totalTxCount: number; status: string }[]; message: string; }

const STATUS_MAP: Record<string, { label: string; cls: string }> = {
  PENDING:  { label: "قيد المراجعة", cls: "bg-amber-500/10 text-amber-400 border-amber-500/20" },
  VERIFIED: { label: "تم التحقق", cls: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" },
  DISPUTED: { label: "متنازع", cls: "bg-red-500/10 text-red-400 border-red-500/20" },
  RESOLVED: { label: "تم الحل", cls: "bg-blue-500/10 text-blue-400 border-blue-500/20" },
  ERROR:    { label: "خطأ", cls: "bg-red-500/10 text-red-400 border-red-500/20" },
};

function formatAmount(v: number) {
  return new Intl.NumberFormat("ar-YE", { style: "currency", currency: "YER", minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(v / 100);
}
function formatDate(s: string) {
  if (!s) return "—";
  const d = new Date(s);
  return isNaN(d.getTime()) ? s : new Intl.DateTimeFormat("ar-YE", { year: "numeric", month: "2-digit", day: "2-digit" }).format(d);
}

export default function ReconciliationPage() {
  const [reports, setReports] = useState<ReconReport[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

  // حوار بدء تسوية جديدة
  const [runOpen, setRunOpen] = useState(false);
  const [runWallet, setRunWallet] = useState("");
  const [runDate, setRunDate] = useState(new Date().toISOString().slice(0, 10));
  const [runLoading, setRunLoading] = useState(false);
  const [runResult, setRunResult] = useState<string | null>(null);
  const [runError, setRunError] = useState<string | null>(null);

  // حوار تفاصيل
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailReport, setDetailReport] = useState<ReconReport | null>(null);

  const fetchReports = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiGet<ReconListRes>("/admin/v1/reconciliation/reports", { page: String(page), pageSize: "20" });
      setReports(res.reports || []);
      setTotalCount(res.totalCount || 0);
    } catch { setReports([]); }
    finally { setLoading(false); }
  }, [page]);

  useEffect(() => { fetchReports(); }, [fetchReports]);

  const handleRun = async () => {
    setRunLoading(true);
    setRunError(null);
    setRunResult(null);
    try {
      const res = await apiPost<RunReconRes>("/admin/v1/reconciliation/run", {
        reportDate: runDate, walletId: runWallet || undefined,
      });
      setRunResult(res.message || "تم بنجاح");
      fetchReports();
    } catch (err) {
      setRunError(err instanceof Error ? err.message : "حدث خطأ");
    } finally { setRunLoading(false); }
  };

  const handleExport = async (report: ReconReport) => {
    try {
      const XLSX = await import("xlsx");
      const wsData = [{
        "التاريخ": report.reportDate,
        "المحفظة": report.walletId,
        "إجمالي المعاملات": report.totalTxCount,
        "إجمالي المبلغ": report.totalAmount / 100,
        "الناجحة": report.successCount,
        "الفاشلة": report.failedCount,
        "المتنازع عليها": report.disputedCount,
        "الحالة": report.status,
        "ملاحظات": report.notes || "—",
      }];
      const ws = XLSX.utils.json_to_sheet(wsData);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, "تسوية");
      XLSX.writeFile(wb, `recon_${report.walletId}_${report.reportDate}.xlsx`);
    } catch { /* ignore */ }
  };

  const totalPages = Math.ceil(totalCount / 20);

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">التسوية</h1>
          <p className="text-sm text-muted-foreground">تسوية المعاملات بين السويتش والمحافظ</p>
        </div>
        {hasRole("ADMIN") && (
          <button onClick={() => { setRunOpen(true); setRunError(null); setRunResult(null); }}
            className="flex items-center gap-2 rounded-lg bg-gradient-to-l from-blue-600 to-violet-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-600/20 transition-all hover:shadow-blue-600/30">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg>
            بدء تسوية جديدة
          </button>
        )}
      </div>

      {/* KPI */}
      <div className="grid gap-4 sm:grid-cols-4">
        <Card className="border-border/50">
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">إجمالي التقارير</p>
            <p className="text-2xl font-bold text-foreground">{totalCount.toLocaleString("ar-YE")}</p>
          </CardContent>
        </Card>
        <Card className="border-border/50">
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">تم التحقق</p>
            <p className="text-2xl font-bold text-emerald-400">{reports.filter(r => r.status === "VERIFIED").length}</p>
          </CardContent>
        </Card>
        <Card className="border-border/50">
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">قيد المراجعة</p>
            <p className="text-2xl font-bold text-amber-400">{reports.filter(r => r.status === "PENDING").length}</p>
          </CardContent>
        </Card>
        <Card className="border-border/50">
          <CardContent className="p-4">
            <p className="text-xs text-muted-foreground">متنازع</p>
            <p className="text-2xl font-bold text-red-400">{reports.filter(r => r.status === "DISPUTED").length}</p>
          </CardContent>
        </Card>
      </div>

      {/* جدول التقارير */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50 hover:bg-transparent">
                  <TableHead className="text-xs font-semibold">التاريخ</TableHead>
                  <TableHead className="text-xs font-semibold">المحفظة</TableHead>
                  <TableHead className="text-xs font-semibold">المعاملات</TableHead>
                  <TableHead className="text-xs font-semibold">الناجحة</TableHead>
                  <TableHead className="text-xs font-semibold">الفاشلة</TableHead>
                  <TableHead className="text-xs font-semibold">إجمالي المبلغ</TableHead>
                  <TableHead className="text-xs font-semibold">الحالة</TableHead>
                  <TableHead className="text-xs font-semibold">الإجراءات</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={8} className="h-48 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
                        <span className="text-sm text-muted-foreground">جارٍ التحميل...</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : reports.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="h-48 text-center">
                      <span className="text-sm text-muted-foreground">لا توجد تقارير تسوية</span>
                    </TableCell>
                  </TableRow>
                ) : reports.map(r => {
                  const st = STATUS_MAP[r.status] || { label: r.status, cls: "bg-muted text-muted-foreground" };
                  return (
                    <TableRow key={r.id || `${r.reportDate}-${r.walletId}`} className="border-border/30 hover:bg-muted/30">
                      <TableCell><span className="text-sm font-medium">{formatDate(r.reportDate)}</span></TableCell>
                      <TableCell><Badge variant="secondary" className="text-[10px]">{r.walletId}</Badge></TableCell>
                      <TableCell><span className="font-mono text-xs">{r.totalTxCount}</span></TableCell>
                      <TableCell><span className="font-mono text-xs text-emerald-400">{r.successCount}</span></TableCell>
                      <TableCell><span className="font-mono text-xs text-red-400">{r.failedCount}</span></TableCell>
                      <TableCell><span className="text-xs font-semibold">{formatAmount(r.totalAmount)}</span></TableCell>
                      <TableCell><Badge variant="outline" className={st.cls}>{st.label}</Badge></TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          <button onClick={() => { setDetailReport(r); setDetailOpen(true); }}
                            className="rounded-md bg-blue-500/10 px-2.5 py-1.5 text-[10px] font-medium text-blue-400 hover:bg-blue-500/20">
                            تفاصيل
                          </button>
                          <button onClick={() => handleExport(r)}
                            className="rounded-md bg-emerald-500/10 px-2.5 py-1.5 text-[10px] font-medium text-emerald-400 hover:bg-emerald-500/20">
                            Excel
                          </button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-border/50 px-4 py-3">
              <span className="text-sm text-muted-foreground">صفحة {page} من {totalPages}</span>
              <div className="flex gap-2">
                <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}
                  className="rounded-md border border-border/50 bg-card px-3 py-1.5 text-xs hover:bg-muted/50 disabled:opacity-50">السابقة</button>
                <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page >= totalPages}
                  className="rounded-md border border-border/50 bg-card px-3 py-1.5 text-xs hover:bg-muted/50 disabled:opacity-50">التالية</button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ═══ حوار بدء تسوية ═══ */}
      <Dialog open={runOpen} onOpenChange={setRunOpen}>
        <DialogContent className="border-border/50">
          <DialogHeader>
            <DialogTitle>بدء تسوية جديدة</DialogTitle>
            <DialogDescription>اختر المحفظة ونطاق التاريخ لتشغيل التسوية</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">المحفظة (اتركه فارغاً لكل المحافظ)</label>
              <input type="text" value={runWallet} onChange={e => setRunWallet(e.target.value)}
                placeholder="jawali" dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
            </div>
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground">التاريخ</label>
              <input type="date" value={runDate} onChange={e => setRunDate(e.target.value)} dir="ltr"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
            </div>
            {runResult && <div className="rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-sm text-emerald-400">✓ {runResult}</div>}
            {runError && <div className="rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm text-red-400">{runError}</div>}
          </div>
          <DialogFooter className="gap-2">
            <button onClick={() => setRunOpen(false)} className="rounded-lg border border-border/50 px-4 py-2 text-sm hover:bg-muted/50">إلغاء</button>
            <button onClick={handleRun} disabled={runLoading}
              className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20 disabled:opacity-50">
              {runLoading ? "جارٍ التشغيل..." : "تشغيل التسوية"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ═══ حوار تفاصيل التقرير ═══ */}
      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <DialogContent className="border-border/50 max-w-2xl">
          <DialogHeader>
            <DialogTitle>تفاصيل التقرير</DialogTitle>
            <DialogDescription>
              تقرير تسوية {detailReport?.walletId} — {detailReport?.reportDate}
            </DialogDescription>
          </DialogHeader>
          {detailReport && (
            <div className="space-y-4 py-2">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                <div className="rounded-lg bg-muted/30 p-3 text-center">
                  <p className="text-[10px] text-muted-foreground">إجمالي المعاملات</p>
                  <p className="text-xl font-bold">{detailReport.totalTxCount}</p>
                </div>
                <div className="rounded-lg bg-emerald-500/5 p-3 text-center">
                  <p className="text-[10px] text-muted-foreground">ناجحة</p>
                  <p className="text-xl font-bold text-emerald-400">{detailReport.successCount}</p>
                </div>
                <div className="rounded-lg bg-red-500/5 p-3 text-center">
                  <p className="text-[10px] text-muted-foreground">فاشلة</p>
                  <p className="text-xl font-bold text-red-400">{detailReport.failedCount}</p>
                </div>
                <div className="rounded-lg bg-amber-500/5 p-3 text-center">
                  <p className="text-[10px] text-muted-foreground">متنازع</p>
                  <p className="text-xl font-bold text-amber-400">{detailReport.disputedCount}</p>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><p className="text-xs text-muted-foreground">إجمالي المبلغ</p><p className="text-lg font-bold">{formatAmount(detailReport.totalAmount)}</p></div>
                <div><p className="text-xs text-muted-foreground">الحالة</p><Badge variant="outline" className={STATUS_MAP[detailReport.status]?.cls || ""}>{STATUS_MAP[detailReport.status]?.label || detailReport.status}</Badge></div>
              </div>
              {detailReport.notes && (
                <div><p className="text-xs text-muted-foreground">ملاحظات</p><p className="text-sm">{detailReport.notes}</p></div>
              )}

              {/* قائمة أنماط الفروقات */}
              <div>
                <p className="text-xs font-semibold text-muted-foreground mb-2">أنماط التحقق</p>
                <div className="grid grid-cols-2 gap-2">
                  {[
                    { key: "MATCH", label: "متطابق", icon: "✅", cls: "border-emerald-500/20 bg-emerald-500/5 text-emerald-400" },
                    { key: "AMOUNT_MISMATCH", label: "اختلاف المبلغ", icon: "💰", cls: "border-amber-500/20 bg-amber-500/5 text-amber-400" },
                    { key: "STATUS_MISMATCH", label: "اختلاف الحالة", icon: "⚠️", cls: "border-orange-500/20 bg-orange-500/5 text-orange-400" },
                    { key: "MISSING", label: "مفقود", icon: "❌", cls: "border-red-500/20 bg-red-500/5 text-red-400" },
                  ].map(t => (
                    <div key={t.key} className={`rounded-lg border px-3 py-2 text-sm ${t.cls}`}>
                      <span className="ml-1">{t.icon}</span> {t.label}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
          <DialogFooter className="gap-2">
            {detailReport && (
              <button onClick={() => handleExport(detailReport)}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-emerald-600/20 hover:bg-emerald-700">
                تصدير Excel
              </button>
            )}
            <button onClick={() => setDetailOpen(false)} className="rounded-lg border border-border/50 px-4 py-2 text-sm hover:bg-muted/50">إغلاق</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
