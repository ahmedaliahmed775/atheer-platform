// الإحصائيات — رسوم بيانية متعددة مع فلاتر
"use client";

import React, { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet } from "@/lib/api";
import {
  LineChart, Line, AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";

// ── الأنواع ──
interface VolumePoint { date: string; count: number; amount: number; }
interface VolumeRes { data: VolumePoint[]; period: string; group: string; }
interface SummaryRes {
  totalTransactions: number; successRate: number; totalVolume: number;
  failedTransactions: number; pendingTransactions: number; period: string;
}
interface LatencyRes { averageMs: number; p50Ms: number; p95Ms: number; p99Ms: number; period: string; }
interface ErrorRes { totalErrors: number; errorRate: number; byCode: { code: string; count: number }[]; byWallet: { walletId: string; count: number }[]; period: string; }

// ألوان Recharts
const COLORS = ["#3b82f6", "#8b5cf6", "#10b981", "#f59e0b", "#ef4444", "#06b6d4", "#ec4899", "#84cc16"];
const PIE_COLORS = ["#10b981", "#ef4444", "#f59e0b"];

const PERIOD_OPTIONS = [
  { value: "7d", label: "7 أيام" },
  { value: "30d", label: "30 يوم" },
  { value: "90d", label: "90 يوم" },
];

function formatNum(n: number) { return new Intl.NumberFormat("ar-YE").format(n); }

export default function AnalyticsPage() {
  const [period, setPeriod] = useState("7d");
  const [walletFilter, setWalletFilter] = useState("");

  const [volume, setVolume] = useState<VolumePoint[]>([]);
  const [summary, setSummary] = useState<SummaryRes | null>(null);
  const [latency, setLatency] = useState<LatencyRes | null>(null);
  const [errors, setErrors] = useState<ErrorRes | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = { period };
      if (walletFilter) params.walletId = walletFilter;

      const [volRes, sumRes, latRes, errRes] = await Promise.allSettled([
        apiGet<VolumeRes>("/admin/v1/analytics/volume", { ...params, group: "day" }),
        apiGet<SummaryRes>("/admin/v1/analytics/summary", params),
        apiGet<LatencyRes>("/admin/v1/analytics/latency", params),
        apiGet<ErrorRes>("/admin/v1/analytics/errors", params),
      ]);
      if (volRes.status === "fulfilled") setVolume(volRes.value.data || []);
      if (sumRes.status === "fulfilled") setSummary(sumRes.value);
      if (latRes.status === "fulfilled") setLatency(latRes.value);
      if (errRes.status === "fulfilled") setErrors(errRes.value);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [period, walletFilter]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  // بيانات PieChart
  const pieData = summary ? [
    { name: "ناجحة", value: summary.totalTransactions - summary.failedTransactions - summary.pendingTransactions },
    { name: "فاشلة", value: summary.failedTransactions },
    { name: "معلّقة", value: summary.pendingTransactions },
  ].filter(d => d.value > 0) : [];

  // بيانات الكمون
  const latencyData = latency ? [
    { name: "P50", value: latency.p50Ms },
    { name: "P95", value: latency.p95Ms },
    { name: "P99", value: latency.p99Ms },
    { name: "المتوسط", value: latency.averageMs },
  ] : [];

  // بيانات أكواد الأخطاء (أعمدة أفقية)
  const errorCodesData = errors?.byCode?.slice(0, 10) || [];

  if (loading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
          <p className="text-sm text-muted-foreground">جارٍ تحميل الإحصائيات...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* العنوان + الفلاتر */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">الإحصائيات</h1>
          <p className="text-sm text-muted-foreground">تحليل شامل لأداء سويتش Atheer</p>
        </div>
        <div className="flex gap-2">
          <input type="text" value={walletFilter}
            onChange={(e) => setWalletFilter(e.target.value)}
            placeholder="فلتر المحفظة" dir="ltr"
            className="flex h-9 w-32 rounded-lg border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
          <div className="flex rounded-lg border border-border/50 bg-card overflow-hidden">
            {PERIOD_OPTIONS.map(opt => (
              <button key={opt.value} onClick={() => setPeriod(opt.value)}
                className={`px-3 py-2 text-xs font-medium transition-colors ${
                  period === opt.value
                    ? "bg-blue-500/20 text-blue-400"
                    : "text-muted-foreground hover:bg-muted/50"
                }`}>
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* الصف الأول: حجم المعاملات + دائري */}
      <div className="grid gap-4 xl:grid-cols-3">
        {/* رسم خطي — حجم المعاملات */}
        <Card className="border-border/50 xl:col-span-2">
          <CardHeader><CardTitle className="text-base font-semibold">حجم المعاملات</CardTitle></CardHeader>
          <CardContent>
            <div className="h-72" dir="ltr">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={volume} margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                  <defs>
                    <linearGradient id="gCount" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="gAmount" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(217 32.6% 17.5%)" />
                  <XAxis dataKey="date" tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 11 }} tickLine={false} axisLine={{ stroke: "hsl(217 32.6% 17.5%)" }} tickFormatter={v => v.slice(5)} />
                  <YAxis tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 11 }} axisLine={false} tickLine={false} width={40} />
                  <Tooltip contentStyle={{ backgroundColor: "hsl(222.2 84% 4.9%)", border: "1px solid hsl(217 32.6% 17.5%)", borderRadius: "8px", fontSize: "12px" }}
                    formatter={(v: number, n: string) => [formatNum(v), n === "count" ? "العدد" : "المبلغ"]} />
                  <Area type="monotone" dataKey="count" stroke="#3b82f6" strokeWidth={2} fill="url(#gCount)" name="count" />
                  <Area type="monotone" dataKey="amount" stroke="#8b5cf6" strokeWidth={2} fill="url(#gAmount)" name="amount" />
                  <Legend formatter={(v) => v === "count" ? "عدد المعاملات" : "إجمالي المبلغ"} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* رسم دائري — نسبة النجاح/الفشل */}
        <Card className="border-border/50">
          <CardHeader><CardTitle className="text-base font-semibold">نسبة النجاح / الفشل</CardTitle></CardHeader>
          <CardContent>
            <div className="h-72" dir="ltr">
              {pieData.length === 0 ? (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">لا توجد بيانات</div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={pieData} cx="50%" cy="50%" innerRadius={55} outerRadius={90} paddingAngle={4} dataKey="value" label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`} labelLine={false}>
                      {pieData.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
                    </Pie>
                    <Tooltip contentStyle={{ backgroundColor: "hsl(222.2 84% 4.9%)", border: "1px solid hsl(217 32.6% 17.5%)", borderRadius: "8px", fontSize: "12px" }} />
                  </PieChart>
                </ResponsiveContainer>
              )}
            </div>
            {/* ملخص رقمي */}
            <div className="mt-2 grid grid-cols-3 gap-2 text-center">
              <div><p className="text-lg font-bold text-emerald-400">{formatNum(summary?.totalTransactions ? summary.totalTransactions - summary.failedTransactions - summary.pendingTransactions : 0)}</p><p className="text-[10px] text-muted-foreground">ناجحة</p></div>
              <div><p className="text-lg font-bold text-red-400">{formatNum(summary?.failedTransactions ?? 0)}</p><p className="text-[10px] text-muted-foreground">فاشلة</p></div>
              <div><p className="text-lg font-bold text-amber-400">{formatNum(summary?.pendingTransactions ?? 0)}</p><p className="text-[10px] text-muted-foreground">معلّقة</p></div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* الصف الثاني: توزيع المبالغ + Top أكواد أخطاء */}
      <div className="grid gap-4 xl:grid-cols-2">
        {/* رسم أعمدة — توزيع المبالغ اليومية */}
        <Card className="border-border/50">
          <CardHeader><CardTitle className="text-base font-semibold">توزيع المبالغ اليومية</CardTitle></CardHeader>
          <CardContent>
            <div className="h-64" dir="ltr">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={volume} margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(217 32.6% 17.5%)" />
                  <XAxis dataKey="date" tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 10 }} tickLine={false} axisLine={{ stroke: "hsl(217 32.6% 17.5%)" }} tickFormatter={v => v.slice(5)} />
                  <YAxis tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 10 }} axisLine={false} tickLine={false} width={50} />
                  <Tooltip contentStyle={{ backgroundColor: "hsl(222.2 84% 4.9%)", border: "1px solid hsl(217 32.6% 17.5%)", borderRadius: "8px", fontSize: "12px" }}
                    formatter={(v: number) => [formatNum(v), "المبلغ"]} />
                  <Bar dataKey="amount" radius={[4, 4, 0, 0]}>
                    {volume.map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} fillOpacity={0.8} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* رسم أعمدة أفقية — Top أكواد الأخطاء */}
        <Card className="border-border/50">
          <CardHeader><CardTitle className="text-base font-semibold">أعلى أكواد الأخطاء</CardTitle></CardHeader>
          <CardContent>
            <div className="h-64" dir="ltr">
              {errorCodesData.length === 0 ? (
                <div className="flex h-full flex-col items-center justify-center gap-2">
                  <div className="rounded-full bg-emerald-500/10 p-3">
                    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-emerald-400"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
                  </div>
                  <p className="text-sm text-muted-foreground">لا توجد أخطاء في هذه الفترة</p>
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={errorCodesData} layout="vertical" margin={{ top: 5, right: 10, left: 60, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(217 32.6% 17.5%)" />
                    <XAxis type="number" tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 10 }} axisLine={false} tickLine={false} />
                    <YAxis type="category" dataKey="code" tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 10 }} axisLine={false} tickLine={false} width={80} />
                    <Tooltip contentStyle={{ backgroundColor: "hsl(222.2 84% 4.9%)", border: "1px solid hsl(217 32.6% 17.5%)", borderRadius: "8px", fontSize: "12px" }} />
                    <Bar dataKey="count" fill="#ef4444" radius={[0, 4, 4, 0]} fillOpacity={0.8} />
                  </BarChart>
                </ResponsiveContainer>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* الصف الثالث: زمن الاستجابة */}
      <Card className="border-border/50">
        <CardHeader><CardTitle className="text-base font-semibold">متوسط المدة (بالملي ثانية)</CardTitle></CardHeader>
        <CardContent>
          <div className="h-56" dir="ltr">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={latencyData} margin={{ top: 5, right: 20, left: 20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(217 32.6% 17.5%)" />
                <XAxis dataKey="name" tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 12 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fill: "hsl(215 20.2% 65.1%)", fontSize: 11 }} axisLine={false} tickLine={false} width={40} />
                <Tooltip contentStyle={{ backgroundColor: "hsl(222.2 84% 4.9%)", border: "1px solid hsl(217 32.6% 17.5%)", borderRadius: "8px", fontSize: "12px" }}
                  formatter={(v: number) => [`${v} ms`, "المدة"]} />
                <Bar dataKey="value" radius={[6, 6, 0, 0]}>
                  {latencyData.map((_, i) => <Cell key={i} fill={COLORS[i]} fillOpacity={0.85} />)}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
