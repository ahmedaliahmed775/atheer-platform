// العمولات — إحصائيات عمولات شركة الاتصالات لكل محفظة مع فلاتر تاريخ
"use client";

import React, { useCallback, useEffect, useState } from "react";
import { apiGet } from "@/lib/api";
import { hasRole } from "@/lib/auth";
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

/** إحصائيات عمولات محفظة واحدة */
interface WalletCommission {
  walletId: string;
  totalTxCount: number;
  totalAmount: number;
  successCount: number;
  failedCount: number;
  commissionRate: number;
  commissionDue: number;
}

/** استجابة API العمولات */
interface CommissionResponse {
  wallets: WalletCommission[];
  totalTxCount: number;
  totalAmount: number;
  totalDue: number;
  rate: number;
}

/** تنسيق المبلغ بالوحدة الصغرى إلى شكل مقروء */
function formatAmount(amount: number): string {
  return (amount / 100).toLocaleString("ar-YE", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

/** تنسيق النسبة بالألف إلى نسبة مئوية */
function formatRate(rate: number): string {
  return `${(rate / 10).toFixed(1)}%`;
}

export default function CommissionsPage() {
  const [data, setData] = useState<CommissionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");

  const isAdmin = hasRole("ADMIN");

  const fetchCommissions = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (fromDate) params.fromDate = fromDate;
      if (toDate) params.toDate = toDate;

      const res = await apiGet<CommissionResponse>(
        "/admin/v1/commission/stats",
        params
      );
      setData(res);
    } catch {
      // خطأ في جلب البيانات — يتم تجاهله
    } finally {
      setLoading(false);
    }
  }, [fromDate, toDate]);

  useEffect(() => {
    if (isAdmin) {
      fetchCommissions();
    }
  }, [isAdmin, fetchCommissions]);

  if (!isAdmin) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-muted-foreground">ليس لديك صلاحية لعرض هذه الصفحة</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div>
        <h1 className="text-2xl font-bold tracking-tight">عمولات شركة الاتصالات</h1>
        <p className="text-muted-foreground mt-1">
          إحصائيات المعاملات عبر شبكة الاتصالات والعمولات المستحقة لكل محفظة
        </p>
      </div>

      {/* فلاتر التاريخ */}
      <Card className="border-border/50">
        <CardContent className="flex flex-wrap items-end gap-4 p-4">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-muted-foreground">من تاريخ</label>
            <input
              type="date"
              value={fromDate}
              onChange={(e) => setFromDate(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-muted-foreground">إلى تاريخ</label>
            <input
              type="date"
              value={toDate}
              onChange={(e) => setToDate(e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>
          <button
            onClick={() => {
              setFromDate("");
              setToDate("");
            }}
            className="h-9 px-4 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            مسح الفلاتر
          </button>
        </CardContent>
      </Card>

      {/* بطاقات الملخص */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">إجمالي المعاملات</CardTitle>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.totalTxCount ?? 0}</div>
            <p className="text-xs text-muted-foreground mt-1">عبر شبكة الاتصالات</p>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">إجمالي المبلغ</CardTitle>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data ? formatAmount(data.totalAmount) : "0"}</div>
            <p className="text-xs text-muted-foreground mt-1">ريال يمني</p>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">إجمالي العمولات</CardTitle>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground"><circle cx="12" cy="12" r="10"/><path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8"/><path d="M12 18V6"/></svg>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-emerald-500">{data ? formatAmount(data.totalDue) : "0"}</div>
            <p className="text-xs text-muted-foreground mt-1">ريال يمني مستحق</p>
          </CardContent>
        </Card>

        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">نسبة العمولة</CardTitle>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data ? formatRate(data.rate) : "0%"}</div>
            <p className="text-xs text-muted-foreground mt-1">من إجمالي المبلغ الناجح</p>
          </CardContent>
        </Card>
      </div>

      {/* جدول تفاصيل العمولات */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-lg">تفاصيل العمولات حسب المحفظة</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50 hover:bg-transparent">
                  <TableHead className="text-right">المحفظة</TableHead>
                  <TableHead className="text-right">عدد المعاملات</TableHead>
                  <TableHead className="text-right">المبلغ الإجمالي</TableHead>
                  <TableHead className="text-right">ناجحة</TableHead>
                  <TableHead className="text-right">فاشلة</TableHead>
                  <TableHead className="text-right">نسبة العمولة</TableHead>
                  <TableHead className="text-right">العمولة المستحقة</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-48 text-center">
                      <div className="flex items-center justify-center gap-2 text-muted-foreground">
                        <svg className="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                        </svg>
                        جارٍ التحميل...
                      </div>
                    </TableCell>
                  </TableRow>
                ) : !data?.wallets?.length ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-48 text-center text-muted-foreground">
                      لا توجد معاملات عبر شبكة الاتصالات في الفترة المحددة
                    </TableCell>
                  </TableRow>
                ) : (
                  data.wallets.map((w) => (
                    <TableRow key={w.walletId} className="border-border/30 transition-colors hover:bg-muted/30">
                      <TableCell className="font-medium">
                        <Badge variant="outline" className="font-mono">
                          {w.walletId}
                        </Badge>
                      </TableCell>
                      <TableCell>{w.totalTxCount.toLocaleString("ar-YE")}</TableCell>
                      <TableCell>{formatAmount(w.totalAmount)}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-emerald-500 border-emerald-500/30">
                          {w.successCount}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-red-500 border-red-500/30">
                          {w.failedCount}
                        </Badge>
                      </TableCell>
                      <TableCell>{formatRate(w.commissionRate)}</TableCell>
                      <TableCell className="font-semibold text-emerald-500">
                        {formatAmount(w.commissionDue)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
