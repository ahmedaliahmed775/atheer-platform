// تفاصيل المعاملة — كل الحقول + Timeline + رمز الخطأ
"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { apiGet } from "@/lib/api";

// ── الأنواع ──

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
  createdAt: string;
}

interface TimelineEvent {
  timestamp: number;
  event: string;
  detail: string;
}

interface TransactionDetailResponse {
  transaction: Transaction;
  timeline: TimelineEvent[];
}

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

/** تنسيق التاريخ الكامل */
function formatDateTime(dateStr: string): string {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  return new Intl.DateTimeFormat("ar-YE", {
    year: "numeric",
    month: "long",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(d);
}

/** تنسيق الطابع الزمني */
function formatTimestamp(ts: number): string {
  if (!ts) return "—";
  const d = new Date(ts * 1000);
  return new Intl.DateTimeFormat("ar-YE", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  }).format(d);
}

/** شارة الحالة */
function StatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; className: string }> = {
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
    <Badge variant="outline" className={`${c.className} text-sm px-3 py-1`}>
      {c.label}
    </Badge>
  );
}

/** ترجمة أحداث الجدول الزمني */
const EVENT_LABELS: Record<string, { label: string; icon: string; color: string }> = {
  CREATED: {
    label: "إنشاء المعاملة",
    icon: "📝",
    color: "border-blue-500 bg-blue-500",
  },
  GATE_PASSED: {
    label: "اجتياز البوابة",
    icon: "🚪",
    color: "border-cyan-500 bg-cyan-500",
  },
  VERIFY_PASSED: {
    label: "اجتياز التحقق",
    icon: "🔐",
    color: "border-violet-500 bg-violet-500",
  },
  DEBIT_COMPLETED: {
    label: "تم الخصم",
    icon: "💳",
    color: "border-amber-500 bg-amber-500",
  },
  CREDIT_COMPLETED: {
    label: "تم الإيداع",
    icon: "✅",
    color: "border-emerald-500 bg-emerald-500",
  },
  FAILED: {
    label: "فشل",
    icon: "❌",
    color: "border-red-500 bg-red-500",
  },
  REVERSED: {
    label: "تم العكس",
    icon: "↩️",
    color: "border-orange-500 bg-orange-500",
  },
};

/** حقل معلومات */
function InfoField({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className={`text-sm text-foreground ${mono ? "font-mono" : ""}`} dir={mono ? "ltr" : undefined}>
        {value || "—"}
      </p>
    </div>
  );
}

export default function TransactionDetailPage() {
  const params = useParams();
  const router = useRouter();
  const txId = params.id as string;

  const [detail, setDetail] = useState<TransactionDetailResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDetail = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiGet<TransactionDetailResponse>(
        `/admin/v1/transactions/${txId}`
      );
      setDetail(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "خطأ في جلب تفاصيل المعاملة");
    } finally {
      setLoading(false);
    }
  }, [txId]);

  useEffect(() => {
    if (txId) fetchDetail();
  }, [txId, fetchDetail]);

  if (loading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="h-10 w-10 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
          <p className="text-sm text-muted-foreground">جارٍ تحميل التفاصيل...</p>
        </div>
      </div>
    );
  }

  if (error || !detail) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <Card className="w-full max-w-md border-red-500/20">
          <CardContent className="flex flex-col items-center gap-4 p-8">
            <div className="rounded-full bg-red-500/10 p-4">
              <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-red-400"><circle cx="12" cy="12" r="10"/><line x1="15" x2="9" y1="9" y2="15"/><line x1="9" x2="15" y1="9" y2="15"/></svg>
            </div>
            <p className="text-center text-sm text-red-400">{error || "المعاملة غير موجودة"}</p>
            <button
              onClick={() => router.back()}
              className="rounded-lg bg-muted px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted/80"
            >
              العودة
            </button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const tx = detail.transaction;
  const timeline = detail.timeline || [];

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.back()}
          className="flex h-9 w-9 items-center justify-center rounded-lg border border-border/50 text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m9 18 6-6-6-6"/></svg>
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-foreground">تفاصيل المعاملة</h1>
          <p className="font-mono text-xs text-muted-foreground" dir="ltr">
            {tx.transactionId}
          </p>
        </div>
        <StatusBadge status={tx.status} />
      </div>

      {/* المحتوى الرئيسي */}
      <div className="grid gap-6 xl:grid-cols-3">
        {/* تفاصيل المعاملة */}
        <Card className="border-border/50 xl:col-span-2">
          <CardHeader>
            <CardTitle className="text-base font-semibold">
              بيانات المعاملة
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* الصف الأول — المعلومات الأساسية */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <InfoField label="معرّف المعاملة" value={tx.transactionId} mono />
              <InfoField label="التاريخ" value={formatDateTime(tx.createdAt)} />
              <InfoField label="المدة" value={`${tx.durationMs} ms`} />
            </div>

            <Separator className="bg-border/30" />

            {/* الصف الثاني — أطراف المعاملة */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <InfoField label="معرّف الدافع" value={tx.payerPublicId} mono />
              <InfoField label="معرّف التاجر" value={tx.merchantId} mono />
              <InfoField label="العداد" value={String(tx.counter)} />
            </div>

            <Separator className="bg-border/30" />

            {/* الصف الثالث — المبلغ والمحافظ */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground">المبلغ</p>
                <p className="text-2xl font-bold text-foreground">
                  {formatAmount(tx.amount, tx.currency)}
                </p>
              </div>
              <InfoField label="محفظة الدافع" value={tx.payerWalletId} />
              <InfoField label="محفظة التاجر" value={tx.merchantWalletId} />
            </div>

            <Separator className="bg-border/30" />

            {/* الصف الرابع — المراجع */}
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <InfoField label="مرجع الخصم" value={tx.debitRef} mono />
              <InfoField label="مرجع الإيداع" value={tx.creditRef} mono />
              <InfoField label="العملة" value={tx.currency} />
            </div>

            {/* رمز الخطأ */}
            {tx.status === "FAILED" && tx.errorCode && (
              <>
                <Separator className="bg-border/30" />
                <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4">
                  <div className="flex items-center gap-3">
                    <div className="rounded-full bg-red-500/10 p-2">
                      <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-red-400"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>
                    </div>
                    <div>
                      <p className="text-sm font-semibold text-red-400">سبب الفشل</p>
                      <p className="mt-0.5 font-mono text-xs text-red-400/80" dir="ltr">
                        {tx.errorCode}
                      </p>
                    </div>
                  </div>
                </div>
              </>
            )}
          </CardContent>
        </Card>

        {/* الجدول الزمني */}
        <Card className="border-border/50">
          <CardHeader>
            <CardTitle className="text-base font-semibold">
              الجدول الزمني
            </CardTitle>
          </CardHeader>
          <CardContent>
            {timeline.length === 0 ? (
              <p className="text-center text-sm text-muted-foreground py-8">
                لا توجد أحداث مسجّلة
              </p>
            ) : (
              <div className="relative space-y-0">
                {/* الخط العمودي */}
                <div className="absolute right-[18px] top-2 bottom-2 w-px bg-border/50" />

                {timeline.map((event, idx) => {
                  const config = EVENT_LABELS[event.event] || {
                    label: event.event,
                    icon: "📌",
                    color: "border-muted bg-muted",
                  };

                  return (
                    <div key={idx} className="relative flex gap-4 pb-6 last:pb-0">
                      {/* النقطة */}
                      <div className="relative z-10 flex h-9 w-9 flex-shrink-0 items-center justify-center">
                        <span
                          className={`flex h-4 w-4 items-center justify-center rounded-full ${config.color} shadow-lg`}
                        />
                      </div>

                      {/* المحتوى */}
                      <div className="flex-1 rounded-lg bg-muted/20 px-4 py-3 transition-colors hover:bg-muted/30">
                        <div className="flex items-center justify-between">
                          <p className="text-sm font-medium text-foreground">
                            {config.icon} {config.label}
                          </p>
                          <span className="font-mono text-[10px] text-muted-foreground" dir="ltr">
                            {formatTimestamp(event.timestamp)}
                          </span>
                        </div>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {event.detail}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* ملخّص المراحل */}
            <Separator className="my-4 bg-border/30" />
            <div className="space-y-2">
              <p className="text-xs font-semibold text-muted-foreground">
                مراحل المعالجة
              </p>
              <div className="flex items-center justify-center gap-1">
                {/* GATE */}
                <div className="flex flex-col items-center">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-cyan-500/10 text-cyan-400">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 8V6a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h8"/><path d="M10 19v-6.8a2 2 0 0 1 .8-1.6l5.2-3.9a2 2 0 0 1 2.4 0l5.2 3.9a2 2 0 0 1 .8 1.6V19a2 2 0 0 1-2 2h-4"/><path d="M10 19a2 2 0 0 1-2-2v-1.5"/></svg>
                  </div>
                  <span className="mt-1 text-[10px] text-muted-foreground">GATE</span>
                </div>

                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground/30 rotate-180"><path d="m9 18 6-6-6-6"/></svg>

                {/* VERIFY */}
                <div className="flex flex-col items-center">
                  <div className="flex h-10 w-10 items-center justify-center rounded-full bg-violet-500/10 text-violet-400">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                  </div>
                  <span className="mt-1 text-[10px] text-muted-foreground">VERIFY</span>
                </div>

                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground/30 rotate-180"><path d="m9 18 6-6-6-6"/></svg>

                {/* EXECUTE */}
                <div className="flex flex-col items-center">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-full ${
                    tx.status === "SUCCESS"
                      ? "bg-emerald-500/10 text-emerald-400"
                      : tx.status === "FAILED"
                      ? "bg-red-500/10 text-red-400"
                      : "bg-amber-500/10 text-amber-400"
                  }`}>
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
                  </div>
                  <span className="mt-1 text-[10px] text-muted-foreground">EXECUTE</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
