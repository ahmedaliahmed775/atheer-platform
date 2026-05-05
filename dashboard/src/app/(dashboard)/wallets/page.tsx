// إدارة المحافظ — جدول CRUD مع حوارات إضافة/تعديل وتفعيل/تعطيل واختبار الاتصال
"use client";

import React, { useEffect, useState, useCallback } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import { apiGet, apiPost, apiPut, apiPatch } from "@/lib/api";
import { hasRole } from "@/lib/auth";

// ── الأنواع ──
interface WalletInfo {
  id: number;
  walletId: string;
  baseUrl: string;
  apiKey?: string;
  secret?: string;
  maxPayerLimit: number;
  timeoutMs: number;
  maxRetries: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface WalletForm {
  walletId: string;
  baseUrl: string;
  apiKey: string;
  secret: string;
  maxPayerLimit: string;
  timeoutMs: string;
  maxRetries: string;
}

const emptyForm: WalletForm = {
  walletId: "", baseUrl: "", apiKey: "", secret: "",
  maxPayerLimit: "50000", timeoutMs: "10000", maxRetries: "2",
};

function formatAmount(v: number): string {
  return new Intl.NumberFormat("ar-YE", {
    style: "currency", currency: "YER",
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).format(v / 100);
}

export default function WalletsPage() {
  const [wallets, setWallets] = useState<WalletInfo[]>([]);
  const [loading, setLoading] = useState(true);

  // حوارات
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<"add" | "edit">("add");
  const [editingWallet, setEditingWallet] = useState<string | null>(null);
  const [form, setForm] = useState<WalletForm>(emptyForm);
  const [formLoading, setFormLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // اختبار الاتصال
  const [testingId, setTestingId] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<Record<string, string>>({});

  // كشف البيانات الحساسة
  const [revealedKeys, setRevealedKeys] = useState<Record<string, boolean>>({});
  const [revealedSecrets, setRevealedSecrets] = useState<Record<string, boolean>>({});

  const isSuperAdmin = hasRole("SUPER_ADMIN");
  const isAdmin = hasRole("ADMIN");

  const fetchWallets = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiGet<{ wallets: WalletInfo[] }>("/admin/v1/wallets");
      setWallets(res.wallets || []);
    } catch { setWallets([]); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchWallets(); }, [fetchWallets]);

  const openAdd = () => {
    setForm(emptyForm);
    setDialogMode("add");
    setFormError(null);
    setDialogOpen(true);
  };

  const openEdit = (w: WalletInfo) => {
    setForm({
      walletId: w.walletId, baseUrl: w.baseUrl, apiKey: "", secret: "",
      maxPayerLimit: String(w.maxPayerLimit), timeoutMs: String(w.timeoutMs),
      maxRetries: String(w.maxRetries),
    });
    setEditingWallet(w.walletId);
    setDialogMode("edit");
    setFormError(null);
    setDialogOpen(true);
  };

  const handleSubmit = async () => {
    if (!form.walletId || !form.baseUrl) {
      setFormError("معرّف المحفظة وعنوان API مطلوبان");
      return;
    }
    setFormLoading(true);
    setFormError(null);
    try {
      const body = {
        walletId: form.walletId, baseUrl: form.baseUrl,
        apiKey: form.apiKey || undefined, secret: form.secret || undefined,
        maxPayerLimit: parseInt(form.maxPayerLimit) || 50000,
        timeoutMs: parseInt(form.timeoutMs) || 10000,
        maxRetries: parseInt(form.maxRetries) || 2,
      };
      if (dialogMode === "add") {
        await apiPost("/admin/v1/wallets", body);
      } else {
        await apiPut(`/admin/v1/wallets/${editingWallet}`, body);
      }
      setDialogOpen(false);
      fetchWallets();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "حدث خطأ");
    } finally { setFormLoading(false); }
  };

  const toggleActive = async (w: WalletInfo) => {
    try {
      await apiPatch(`/admin/v1/wallets/${w.walletId}`, { isActive: !w.isActive });
      setWallets(prev => prev.map(x => x.walletId === w.walletId ? { ...x, isActive: !x.isActive } : x));
    } catch { /* تجاهل */ }
  };

  const testConnection = async (walletId: string) => {
    setTestingId(walletId);
    setTestResult(prev => ({ ...prev, [walletId]: "" }));
    try {
      await apiPost(`/admin/v1/wallets/${walletId}/test`);
      setTestResult(prev => ({ ...prev, [walletId]: "success" }));
    } catch {
      setTestResult(prev => ({ ...prev, [walletId]: "error" }));
    } finally {
      setTestingId(null);
      setTimeout(() => setTestResult(prev => { const n = { ...prev }; delete n[walletId]; return n; }), 4000);
    }
  };

  const setField = (key: keyof WalletForm, val: string) =>
    setForm(prev => ({ ...prev, [key]: val }));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">المحافظ</h1>
          <p className="text-sm text-muted-foreground">إدارة محافظ الدفع المتصلة بالسويتش</p>
        </div>
        {isSuperAdmin && (
          <button onClick={openAdd}
            className="flex items-center gap-2 rounded-lg bg-gradient-to-l from-blue-600 to-violet-600 px-4 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-600/20 transition-all hover:shadow-blue-600/30">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
            إضافة محفظة
          </button>
        )}
      </div>

      {/* الجدول */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50 hover:bg-transparent">
                  <TableHead className="text-xs font-semibold">المعرّف</TableHead>
                  <TableHead className="text-xs font-semibold">عنوان API</TableHead>
                  <TableHead className="text-xs font-semibold">الحالة</TableHead>
                  <TableHead className="text-xs font-semibold">الحد الأقصى</TableHead>
                  <TableHead className="text-xs font-semibold">المهلة</TableHead>
                  <TableHead className="text-xs font-semibold">مفتاح API</TableHead>
                  <TableHead className="text-xs font-semibold">السر</TableHead>
                  {isAdmin && <TableHead className="text-xs font-semibold">الإجراءات</TableHead>}
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={isAdmin ? 8 : 7} className="h-48 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
                        <span className="text-sm text-muted-foreground">جارٍ التحميل...</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : wallets.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={isAdmin ? 8 : 7} className="h-48 text-center">
                      <span className="text-sm text-muted-foreground">لا توجد محافظ مسجّلة</span>
                    </TableCell>
                  </TableRow>
                ) : (
                  wallets.map((w) => (
                    <TableRow key={w.walletId} className="border-border/30 transition-colors hover:bg-muted/30">
                      <TableCell>
                        <span className="font-mono text-sm font-semibold text-foreground">{w.walletId}</span>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground" dir="ltr">{w.baseUrl}</span>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={w.isActive
                          ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                          : "bg-red-500/10 text-red-400 border-red-500/20"}>
                          {w.isActive ? "مفعّلة" : "معطّلة"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs font-semibold">{formatAmount(w.maxPayerLimit)}</span>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground" dir="ltr">{w.timeoutMs}ms</span>
                      </TableCell>
                      <TableCell>
                        {isSuperAdmin && w.apiKey !== undefined ? (
                          <div className="flex items-center gap-1">
                            <span className="font-mono text-xs text-muted-foreground" dir="ltr">
                              {revealedKeys[w.walletId] ? (w.apiKey || "—") : "••••••••"}
                            </span>
                            <button onClick={() => setRevealedKeys(p => ({...p, [w.walletId]: !p[w.walletId]}))}
                              className="text-muted-foreground hover:text-foreground transition-colors">
                              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                {revealedKeys[w.walletId]
                                  ? <><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></>
                                  : <><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></>}
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <span className="font-mono text-xs text-muted-foreground">••••••••</span>
                        )}
                      </TableCell>
                      <TableCell>
                        {isSuperAdmin && w.secret !== undefined ? (
                          <div className="flex items-center gap-1">
                            <span className="font-mono text-xs text-muted-foreground" dir="ltr">
                              {revealedSecrets[w.walletId] ? (w.secret || "—") : "••••••••"}
                            </span>
                            <button onClick={() => setRevealedSecrets(p => ({...p, [w.walletId]: !p[w.walletId]}))}
                              className="text-muted-foreground hover:text-foreground transition-colors">
                              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                                {revealedSecrets[w.walletId]
                                  ? <><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></>
                                  : <><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></>}
                              </svg>
                            </button>
                          </div>
                        ) : (
                          <span className="font-mono text-xs text-muted-foreground">••••••••</span>
                        )}
                      </TableCell>
                      {isAdmin && (
                        <TableCell>
                          <div className="flex items-center gap-1 flex-wrap">
                            {isSuperAdmin && (
                              <>
                                <button onClick={() => openEdit(w)}
                                  className="rounded-md bg-blue-500/10 px-2.5 py-1.5 text-[10px] font-medium text-blue-400 transition-colors hover:bg-blue-500/20">
                                  تعديل
                                </button>
                                <button onClick={() => toggleActive(w)}
                                  className={`rounded-md px-2.5 py-1.5 text-[10px] font-medium transition-colors ${
                                    w.isActive
                                      ? "bg-amber-500/10 text-amber-400 hover:bg-amber-500/20"
                                      : "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20"
                                  }`}>
                                  {w.isActive ? "تعطيل" : "تفعيل"}
                                </button>
                              </>
                            )}
                            <button onClick={() => testConnection(w.walletId)}
                              disabled={testingId === w.walletId}
                              className="rounded-md bg-violet-500/10 px-2.5 py-1.5 text-[10px] font-medium text-violet-400 transition-colors hover:bg-violet-500/20 disabled:opacity-50">
                              {testingId === w.walletId ? "⏳" : "اختبار"}
                            </button>
                            {testResult[w.walletId] === "success" && (
                              <span className="text-[10px] text-emerald-400">✓ متصل</span>
                            )}
                            {testResult[w.walletId] === "error" && (
                              <span className="text-[10px] text-red-400">✗ فشل</span>
                            )}
                          </div>
                        </TableCell>
                      )}
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* ═══ حوار إضافة/تعديل محفظة ═══ */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="border-border/50 max-w-lg">
          <DialogHeader>
            <DialogTitle>{dialogMode === "add" ? "إضافة محفظة جديدة" : "تعديل المحفظة"}</DialogTitle>
            <DialogDescription>
              {dialogMode === "add" ? "أدخل بيانات المحفظة الجديدة" : `تعديل إعدادات محفظة ${editingWallet}`}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 py-2">
            {/* معرّف المحفظة */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">معرّف المحفظة</label>
              <input type="text" value={form.walletId} dir="ltr"
                onChange={(e) => setField("walletId", e.target.value)}
                disabled={dialogMode === "edit"} placeholder="jawali"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50" />
            </div>
            {/* عنوان API */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-muted-foreground">عنوان API</label>
              <input type="text" value={form.baseUrl} dir="ltr"
                onChange={(e) => setField("baseUrl", e.target.value)}
                placeholder="https://api.jawali.ye"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
            </div>
            {/* مفتاح API + السر */}
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">
                  مفتاح API {dialogMode === "edit" && <span className="text-[10px]">(اتركه فارغاً للإبقاء)</span>}
                </label>
                <input type="password" value={form.apiKey} dir="ltr"
                  onChange={(e) => setField("apiKey", e.target.value)}
                  placeholder={dialogMode === "edit" ? "••••••••" : "api_key_..."}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">
                  السر {dialogMode === "edit" && <span className="text-[10px]">(اتركه فارغاً للإبقاء)</span>}
                </label>
                <input type="password" value={form.secret} dir="ltr"
                  onChange={(e) => setField("secret", e.target.value)}
                  placeholder={dialogMode === "edit" ? "••••••••" : "secret_..."}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
              </div>
            </div>
            {/* الحد + المهلة + المحاولات */}
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">الحد الأقصى</label>
                <input type="number" value={form.maxPayerLimit} dir="ltr"
                  onChange={(e) => setField("maxPayerLimit", e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">المهلة (ms)</label>
                <input type="number" value={form.timeoutMs} dir="ltr"
                  onChange={(e) => setField("timeoutMs", e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-medium text-muted-foreground">المحاولات</label>
                <input type="number" value={form.maxRetries} dir="ltr"
                  onChange={(e) => setField("maxRetries", e.target.value)}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" />
              </div>
            </div>
            {formError && (
              <div className="rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm text-red-400">{formError}</div>
            )}
          </div>
          <DialogFooter className="gap-2">
            <button onClick={() => setDialogOpen(false)}
              className="rounded-lg border border-border/50 px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted/50">
              إلغاء
            </button>
            <button onClick={handleSubmit} disabled={formLoading}
              className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20 transition-all hover:shadow-blue-600/30 disabled:opacity-50">
              {formLoading ? "جارٍ الحفظ..." : dialogMode === "add" ? "إضافة" : "حفظ التعديلات"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
