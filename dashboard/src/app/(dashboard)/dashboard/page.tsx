// لوحة القيادة الرئيسية — بطاقات KPI، حالة المحوّلات، رسم بياني، تنبيهات
// تحديث تلقائي كل 30 ثانية
"use client";

import React, { useEffect, useState, useCallback, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { apiGet } from "@/lib/api";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from "recharts";

// ── الأنواع ──

/** استجابة ملخص الأداء */
interface SummaryData {
  totalTransactions: number;
  successRate: number;
  totalVolume: number;
  averageAmount: number;
  failedTransactions: number;
  pendingTransactions: number;
  period: string;
}

/** استجابة زمن الاستجابة */
interface LatencyData {
  averageMs: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  period: string;
}

/** حالة محوّل المحفظة */
interface AdapterHealth {
  walletId: string;
  status: string;
  circuitState: string;
  lastCheckedAt: number;
  responseTimeMs: number;
}

/** استجابة حالة المحوّلات */
interface AdaptersResponse {
  adapters: AdapterHealth[];
  overall: string;
}

/** نقطة بيانات حجم المعاملات */
interface VolumePoint {
  date: string;
  count: number;
  amount: number;
}

/** استجابة الحجم */
interface VolumeResponse {
  data: VolumePoint[];
  period: string;
  group: string;
}

/** تنبيه نشط */
interface Alert {
  id: string;
  level: "info" | "warning" | "error";
  message: string;
  timestamp: number;
}

/** فترة التحديث التلقائي بالملي ثانية */
const REFRESH_INTERVAL = 30_000;

/** تنسيق الأرقام بالعربية */
function formatNumber(n: number): string {
  return new Intl.NumberFormat("ar-YE").format(n);
}

/** تنسيق المبلغ بالريال اليمني */
function formatAmount(amount: number): string {
  // المبلغ بالوحدة الصغرى — نقسم على 100
  const val = amount / 100;
  return new Intl.NumberFormat("ar-YE", {
    style: "currency",
    currency: "YER",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(val);
}

/** تنسيق النسبة المئوية */
function formatPercent(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

export default function DashboardPage() {
  const [summary, setSummary] = useState<SummaryData | null>(null);
  const [latency, setLatency] = useState<LatencyData | null>(null);
  const [adapters, setAdapters] = useState<AdaptersResponse | null>(null);
  const [volume, setVolume] = useState<VolumePoint[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());
  const intervalRef = useRef<NodeJS.Timeout>();

  /** جلب جميع البيانات */
  const fetchAll = useCallback(async () => {
    try {
      const [summaryRes, latencyRes, adaptersRes, volumeRes] =
        await Promise.allSettled([
          apiGet<SummaryData>("/admin/v1/analytics/summary", { period: "24h" }),
          apiGet<LatencyData>("/admin/v1/analytics/latency", { period: "24h" }),
          apiGet<AdaptersResponse>("/admin/v1/health/adapters"),
          apiGet<VolumeResponse>("/admin/v1/analytics/volume", {
            period: "24h",
            group: "hour",
          }),
        ]);

      if (summaryRes.status === "fulfilled") setSummary(summaryRes.value);
      if (latencyRes.status === "fulfilled") setLatency(latencyRes.value);
      if (adaptersRes.status === "fulfilled") setAdapters(adaptersRes.value);
      if (volumeRes.status === "fulfilled") setVolume(volumeRes.value.data || []);

      // بناء تنبيهات من حالة المحوّلات
      if (adaptersRes.status === "fulfilled") {
        const newAlerts: Alert[] = [];
        adaptersRes.value.adapters?.forEach((a) => {
          if (a.status === "DOWN") {
            newAlerts.push({
              id: `adapter-${a.walletId}`,
              level: "error",
              message: `محوّل ${a.walletId} غير متاح`,
              timestamp: a.lastCheckedAt,
            });
          } else if (a.circuitState === "OPEN") {
            newAlerts.push({
              id: `circuit-${a.walletId}`,
              level: "warning",
              message: `قاطع الدائرة مفتوح — ${a.walletId}`,
              timestamp: a.lastCheckedAt,
            });
          }
        });
        setAlerts(newAlerts);
      }

      setLastRefresh(new Date());
    } catch {
      // أخطاء الشبكة لا تُوقف اللوحة
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAll();
    intervalRef.current = setInterval(fetchAll, REFRESH_INTERVAL);
    return () => clearInterval(intervalRef.current);
  }, [fetchAll]);

  /** لون حالة المحوّل */
  const statusColor = (status: string) => {
    switch (status) {
      case "UP":
        return "bg-emerald-500 shadow-emerald-500/30";
      case "DEGRADED":
        return "bg-amber-500 shadow-amber-500/30";
      case "DOWN":
        return "bg-red-500 shadow-red-500/30";
      default:
        return "bg-slate-500";
    }
  };

  /** لون حالة قاطع الدائرة */
  const circuitLabel = (state: string) => {
    switch (state) {
      case "CLOSED":
        return { text: "مغلق", variant: "default" as const };
      case "OPEN":
        return { text: "مفتوح", variant: "destructive" as const };
      case "HALF_OPEN":
        return { text: "نصف مفتوح", variant: "secondary" as const };
      default:
        return { text: state, variant: "outline" as const };
    }
  };

  if (loading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
          <p className="text-sm text-muted-foreground">جارٍ تحميل لوحة القيادة...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">لوحة القيادة</h1>
          <p className="text-sm text-muted-foreground">
            نظرة عامة على أداء سويتش Atheer
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" />
          آخر تحديث: {lastRefresh.toLocaleTimeString("ar-YE")}
        </div>
      </div>

      {/* بطاقات KPI */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {/* معاملات اليوم */}
        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <div className="absolute -left-4 -top-4 h-24 w-24 rounded-full bg-blue-500/5" />
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              معاملات اليوم
            </CardTitle>
            <div className="rounded-lg bg-blue-500/10 p-2">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-blue-400"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">
              {formatNumber(summary?.totalTransactions ?? 0)}
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              فاشلة: {formatNumber(summary?.failedTransactions ?? 0)} • معلّقة: {formatNumber(summary?.pendingTransactions ?? 0)}
            </p>
          </CardContent>
        </Card>

        {/* نسبة النجاح */}
        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <div className="absolute -left-4 -top-4 h-24 w-24 rounded-full bg-emerald-500/5" />
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              نسبة النجاح
            </CardTitle>
            <div className="rounded-lg bg-emerald-500/10 p-2">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-emerald-400"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">
              {formatPercent(summary?.successRate ?? 0)}
            </div>
            <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-gradient-to-l from-emerald-400 to-emerald-600 transition-all duration-500"
                style={{ width: `${(summary?.successRate ?? 0) * 100}%` }}
              />
            </div>
          </CardContent>
        </Card>

        {/* إجمالي المبالغ */}
        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <div className="absolute -left-4 -top-4 h-24 w-24 rounded-full bg-violet-500/5" />
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              إجمالي المبالغ
            </CardTitle>
            <div className="rounded-lg bg-violet-500/10 p-2">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-violet-400"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">
              {formatAmount(summary?.totalVolume ?? 0)}
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              متوسط المعاملة: {formatAmount(summary?.averageAmount ?? 0)}
            </p>
          </CardContent>
        </Card>

        {/* متوسط المدة */}
        <Card className="relative overflow-hidden border-border/50 bg-gradient-to-br from-card to-card/80">
          <div className="absolute -left-4 -top-4 h-24 w-24 rounded-full bg-amber-500/5" />
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              متوسط المدة
            </CardTitle>
            <div className="rounded-lg bg-amber-500/10 p-2">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-amber-400"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold text-foreground">
              {formatNumber(latency?.averageMs ?? 0)}
              <span className="mr-1 text-base font-normal text-muted-foreground">ms</span>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              P50: {latency?.p50Ms ?? 0}ms • P95: {latency?.p95Ms ?? 0}ms • P99: {latency?.p99Ms ?? 0}ms
            </p>
          </CardContent>
        </Card>
      </div>

      {/* الصف الثاني — الرسم البياني + المحوّلات + التنبيهات */}
      <div className="grid gap-4 xl:grid-cols-3">
        {/* رسم بياني — حجم المعاملات */}
        <Card className="border-border/50 xl:col-span-2">
          <CardHeader>
            <CardTitle className="text-base font-semibold">
              معاملات آخر 24 ساعة
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="h-72" dir="ltr">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={volume} margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                  <defs>
                    <linearGradient id="colorCount" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="colorAmount" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(217 32.6% 17.5%)" />
                  <XAxis
                    dataKey="date"
                    tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 11 }}
                    axisLine={{ stroke: "hsl(217 32.6% 17.5%)" }}
                    tickLine={false}
                    tickFormatter={(v) => {
                      // اعرض الساعة فقط إذا كان التنسيق يحتوي على T
                      if (v.includes("T")) return v.split("T")[1]?.slice(0, 5) || v;
                      // اعرض آخر 5 أحرف (الشهر-اليوم)
                      return v.slice(5);
                    }}
                  />
                  <YAxis
                    tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 11 }}
                    axisLine={false}
                    tickLine={false}
                    width={40}
                  />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "hsl(222.2 84% 4.9%)",
                      border: "1px solid hsl(217 32.6% 17.5%)",
                      borderRadius: "8px",
                      fontSize: "12px",
                      direction: "rtl",
                    }}
                    labelStyle={{ color: "hsl(215 20.2% 65.1%)" }}
                    formatter={(value: number, name: string) => [
                      formatNumber(value),
                      name === "count" ? "عدد المعاملات" : "المبلغ",
                    ]}
                  />
                  <Area
                    type="monotone"
                    dataKey="count"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    fill="url(#colorCount)"
                  />
                  <Area
                    type="monotone"
                    dataKey="amount"
                    stroke="#8b5cf6"
                    strokeWidth={2}
                    fill="url(#colorAmount)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* حالة المحوّلات + التنبيهات */}
        <div className="flex flex-col gap-4">
          {/* حالة المحوّلات */}
          <Card className="border-border/50">
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-base font-semibold">
                  حالة المحوّلات
                </CardTitle>
                <Badge
                  variant={
                    adapters?.overall === "UP"
                      ? "default"
                      : adapters?.overall === "DEGRADED"
                      ? "secondary"
                      : "destructive"
                  }
                  className="text-[10px]"
                >
                  {adapters?.overall === "UP"
                    ? "يعمل"
                    : adapters?.overall === "DEGRADED"
                    ? "متدهور"
                    : "متوقف"}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {(!adapters?.adapters || adapters.adapters.length === 0) && (
                <p className="text-center text-sm text-muted-foreground py-4">
                  لا توجد محوّلات مسجّلة
                </p>
              )}
              {adapters?.adapters?.map((a) => {
                const circuit = circuitLabel(a.circuitState);
                return (
                  <div
                    key={a.walletId}
                    className="flex items-center justify-between rounded-lg bg-muted/30 px-4 py-3 transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center gap-3">
                      <span
                        className={`h-3 w-3 rounded-full shadow-lg ${statusColor(a.status)}`}
                      />
                      <div>
                        <p className="text-sm font-medium text-foreground">
                          {a.walletId}
                        </p>
                        <p className="text-[10px] text-muted-foreground">
                          {a.responseTimeMs > 0 ? `${a.responseTimeMs}ms` : "—"}
                        </p>
                      </div>
                    </div>
                    <Badge variant={circuit.variant} className="text-[10px]">
                      {circuit.text}
                    </Badge>
                  </div>
                );
              })}
            </CardContent>
          </Card>

          {/* التنبيهات النشطة */}
          <Card className="border-border/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-base font-semibold">
                التنبيهات النشطة
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {alerts.length === 0 && (
                <div className="flex flex-col items-center gap-2 py-6 text-center">
                  <div className="rounded-full bg-emerald-500/10 p-3">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-emerald-400"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    لا توجد تنبيهات — كل شيء يعمل بشكل طبيعي
                  </p>
                </div>
              )}
              {alerts.map((alert) => (
                <div
                  key={alert.id}
                  className={`flex items-start gap-3 rounded-lg border px-3 py-2.5 text-sm ${
                    alert.level === "error"
                      ? "border-red-500/20 bg-red-500/5 text-red-400"
                      : alert.level === "warning"
                      ? "border-amber-500/20 bg-amber-500/5 text-amber-400"
                      : "border-blue-500/20 bg-blue-500/5 text-blue-400"
                  }`}
                >
                  <span className="mt-0.5">
                    {alert.level === "error" ? "⛔" : alert.level === "warning" ? "⚠️" : "ℹ️"}
                  </span>
                  <span>{alert.message}</span>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
