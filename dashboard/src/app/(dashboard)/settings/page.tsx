// الإعدادات — اتصال السويتش + حسابات الداشبورد + إعدادات النظام + الملف الشخصي
"use client";
import React, { useEffect, useState, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { apiGet, apiPost, apiPatch, getSwitchUrl, setSwitchUrl, resetSwitchUrl } from "@/lib/api";
import { hasRole, getUserEmail } from "@/lib/auth";

// ── الأنواع ──
interface AdminAccount { id: number; email: string; role: string; scope: string; isActive: boolean; lastLoginAt: string; createdAt: string; }
interface AdminListRes { admins: AdminAccount[]; }
interface NewAdminForm { email: string; password: string; role: string; scope: string; }

const ROLES = [
  { value: "SUPER_ADMIN", label: "مدير أعلى" },
  { value: "ADMIN", label: "مدير" },
  { value: "WALLET_ADMIN", label: "مدير محفظة" },
  { value: "VIEWER", label: "مشاهد" },
];
const ROLE_LABELS: Record<string, string> = { SUPER_ADMIN: "مدير أعلى", ADMIN: "مدير", WALLET_ADMIN: "مدير محفظة", VIEWER: "مشاهد" };

function formatDate(s: string) {
  if (!s) return "—";
  const d = new Date(s);
  return isNaN(d.getTime()) ? s : new Intl.DateTimeFormat("ar-YE", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(d);
}

// ════════════════════════════════════════
// تبويب 1: اتصال السويتش
// ════════════════════════════════════════
function SwitchConnectionTab() {
  const [url, setUrl] = useState("");
  const [savedUrl, setSavedUrl] = useState("");
  const [testing, setTesting] = useState(false);
  const [status, setStatus] = useState<"idle" | "connected" | "failed" | "testing">("idle");
  const [healthData, setHealthData] = useState<{ status: string; version: string; dbStatus: string } | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // تحميل العنوان المحفوظ عند التحميل
  useEffect(() => {
    const current = getSwitchUrl();
    setUrl(current);
    setSavedUrl(current);
  }, []);

  // اختبار الاتصال بالسويتش
  const testConnection = useCallback(async (testUrl: string) => {
    setTesting(true);
    setStatus("testing");
    setHealthData(null);
    try {
      const res = await fetch(`${testUrl}/health`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setHealthData({ status: data.status, version: data.version || "—", dbStatus: data.dbStatus || "—" });
      setStatus("connected");
    } catch {
      setStatus("failed");
      setHealthData(null);
    } finally {
      setTesting(false);
    }
  }, []);

  // حفظ العنوان
  const handleSave = () => {
    const trimmed = url.replace(/\/+$/, ""); // إزالة الشرطات الأخيرة
    if (!trimmed) { setMsg("العنوان مطلوب"); return; }
    // السماح بالمسارات النسبية (مثل /api) أو العناوين الكاملة
    const isRelative = trimmed.startsWith("/");
    if (!isRelative) {
      try {
        new URL(trimmed); // التحقق من صحة العنوان الكامل
      } catch {
        setMsg("العنوان غير صالح — تأكد من أنه يبدأ بـ http:// أو https:// أو /");
        return;
      }
    }
    setSwitchUrl(trimmed);
    setSavedUrl(trimmed);
    setMsg("تم حفظ العنوان بنجاح — سيُستخدم في جميع الطلبات القادمة");
    testConnection(trimmed);
  };

  // إعادة العنوان للقيمة الافتراضية
  const handleReset = () => {
    resetSwitchUrl();
    // المتصفح يستخدم /api كمسار نسبي عبر Nginx
    const defaultUrl = "/api";
    setUrl(defaultUrl);
    setSavedUrl(defaultUrl);
    setMsg("تم إعادة العنوان للقيمة الافتراضية");
    testConnection(defaultUrl);
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <h2 className="text-lg font-semibold">اتصال السويتش</h2>

      {/* حقل العنوان */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-sm">عنوان سويتش Atheer</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">
              عنوان API السويتش (URL)
            </label>
            <div className="flex gap-2">
              <input
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                dir="ltr"
                placeholder="http://192.168.1.100:8080"
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm font-mono focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
              <button
                onClick={() => testConnection(url)}
                disabled={testing || !url}
                className="shrink-0 rounded-lg border border-border/50 px-4 py-2 text-sm font-medium hover:bg-muted/50 disabled:opacity-50"
              >
                {testing ? (
                  <span className="flex items-center gap-1.5">
                    <span className="h-3 w-3 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
                    اختبار
                  </span>
                ) : (
                  "اختبار"
                )}
              </button>
            </div>
            <p className="text-[10px] text-muted-foreground">
              العنوان الحالي: <code dir="ltr" className="text-xs bg-muted/50 px-1 rounded">{savedUrl}</code>
            </p>
          </div>

          {/* أزرار الحفظ والإعادة */}
          <div className="flex gap-2">
            <button
              onClick={handleSave}
              className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20"
            >
              حفظ العنوان
            </button>
            <button
              onClick={handleReset}
              className="rounded-lg border border-border/50 px-4 py-2 text-sm font-medium hover:bg-muted/50"
            >
              إعادة للقيمة الافتراضية
            </button>
          </div>

          {msg && (
            <div
              className={`rounded-md px-3 py-2 text-sm ${msg.includes("نجاح")
                  ? "border border-emerald-500/20 bg-emerald-500/5 text-emerald-400"
                  : "border border-red-500/20 bg-red-500/5 text-red-400"
                }`}
            >
              {msg}
            </div>
          )}
        </CardContent>
      </Card>

      {/* حالة الاتصال */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-sm">حالة الاتصال</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-3">
            {/* مؤشر الحالة */}
            <div
              className={`h-3 w-3 rounded-full ${status === "connected"
                  ? "bg-emerald-500 shadow-lg shadow-emerald-500/50"
                  : status === "failed"
                    ? "bg-red-500 shadow-lg shadow-red-500/50"
                    : status === "testing"
                      ? "bg-yellow-500 animate-pulse"
                      : "bg-muted-foreground/30"
                }`}
            />
            <span className="text-sm">
              {status === "connected"
                ? "متصل بالسويتش"
                : status === "failed"
                  ? "فشل الاتصال"
                  : status === "testing"
                    ? "جارٍ الاختبار..."
                    : "لم يتم الاختبار بعد"}
            </span>
          </div>

          {/* بيانات فحص الصحة */}
          {healthData && (
            <div className="grid grid-cols-3 gap-3 rounded-lg border border-border/30 bg-muted/20 p-3">
              <div>
                <p className="text-[10px] text-muted-foreground">الحالة</p>
                <Badge
                  variant="outline"
                  className={
                    healthData.status === "OK"
                      ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                      : "bg-red-500/10 text-red-400 border-red-500/20"
                  }
                >
                  {healthData.status}
                </Badge>
              </div>
              <div>
                <p className="text-[10px] text-muted-foreground">الإصدار</p>
                <p className="text-sm font-mono" dir="ltr">
                  {healthData.version}
                </p>
              </div>
              <div>
                <p className="text-[10px] text-muted-foreground">قاعدة البيانات</p>
                <Badge
                  variant="outline"
                  className={
                    healthData.dbStatus === "OK"
                      ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                      : "bg-red-500/10 text-red-400 border-red-500/20"
                  }
                >
                  {healthData.dbStatus}
                </Badge>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// ════════════════════════════════════════
// تبويب 2: حسابات الداشبورد
// ════════════════════════════════════════
function AccountsTab() {
  const [admins, setAdmins] = useState<AdminAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [addOpen, setAddOpen] = useState(false);
  const [form, setForm] = useState<NewAdminForm>({ email: "", password: "", role: "VIEWER", scope: "global" });
  const [formLoading, setFormLoading] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    setLoading(true);
    try { const r = await apiGet<AdminListRes>("/admin/v1/admins"); setAdmins(r.admins || []); }
    catch { setAdmins([]); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetch(); }, [fetch]);

  const handleAdd = async () => {
    if (!form.email || !form.password) { setFormError("البريد وكلمة المرور مطلوبان"); return; }
    setFormLoading(true); setFormError(null);
    try {
      await apiPost("/admin/v1/admins", form);
      setAddOpen(false); fetch();
    } catch (e) { setFormError(e instanceof Error ? e.message : "خطأ"); }
    finally { setFormLoading(false); }
  };

  const toggleActive = async (a: AdminAccount) => {
    try { await apiPatch(`/admin/v1/admins/${a.id}`, { isActive: !a.isActive }); fetch(); } catch { }
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold">حسابات الداشبورد</h2>
        <button onClick={() => { setForm({ email: "", password: "", role: "VIEWER", scope: "global" }); setFormError(null); setAddOpen(true); }}
          className="flex items-center gap-2 rounded-lg bg-gradient-to-l from-blue-600 to-violet-600 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14" /><path d="M12 5v14" /></svg>
          إضافة مستخدم
        </button>
      </div>
      <Card className="border-border/50">
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="border-border/50 hover:bg-transparent">
                <TableHead className="text-xs font-semibold">البريد</TableHead>
                <TableHead className="text-xs font-semibold">الدور</TableHead>
                <TableHead className="text-xs font-semibold">النطاق</TableHead>
                <TableHead className="text-xs font-semibold">الحالة</TableHead>
                <TableHead className="text-xs font-semibold">آخر دخول</TableHead>
                <TableHead className="text-xs font-semibold">الإجراءات</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow><TableCell colSpan={6} className="h-32 text-center"><div className="flex justify-center gap-2"><div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />جارٍ التحميل...</div></TableCell></TableRow>
              ) : admins.length === 0 ? (
                <TableRow><TableCell colSpan={6} className="h-32 text-center text-sm text-muted-foreground">لا يوجد حسابات</TableCell></TableRow>
              ) : admins.map(a => (
                <TableRow key={a.id} className="border-border/30 hover:bg-muted/30">
                  <TableCell><span className="text-sm" dir="ltr">{a.email}</span></TableCell>
                  <TableCell><Badge variant="secondary" className="text-[10px]">{ROLE_LABELS[a.role] || a.role}</Badge></TableCell>
                  <TableCell><span className="font-mono text-xs text-muted-foreground">{a.scope || "global"}</span></TableCell>
                  <TableCell><Badge variant="outline" className={a.isActive ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-red-500/10 text-red-400 border-red-500/20"}>{a.isActive ? "مفعّل" : "معطّل"}</Badge></TableCell>
                  <TableCell><span className="text-xs text-muted-foreground">{formatDate(a.lastLoginAt)}</span></TableCell>
                  <TableCell>
                    <button onClick={() => toggleActive(a)} className={`rounded-md px-2.5 py-1.5 text-[10px] font-medium transition-colors ${a.isActive ? "bg-red-500/10 text-red-400 hover:bg-red-500/20" : "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20"}`}>
                      {a.isActive ? "تعطيل" : "تفعيل"}
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      {/* حوار إضافة */}
      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent className="border-border/50">
          <DialogHeader><DialogTitle>إضافة مستخدم جديد</DialogTitle><DialogDescription>أنشئ حساب إداري للوحة التحكم</DialogDescription></DialogHeader>
          <div className="space-y-3 py-2">
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">البريد الإلكتروني</label><input type="email" value={form.email} onChange={e => setForm(p => ({ ...p, email: e.target.value }))} dir="ltr" placeholder="admin@atheer.ye" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">كلمة المرور</label><input type="password" value={form.password} onChange={e => setForm(p => ({ ...p, password: e.target.value }))} dir="ltr" placeholder="••••••••" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">الدور</label><select value={form.role} onChange={e => setForm(p => ({ ...p, role: e.target.value }))} className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">{ROLES.map(r => <option key={r.value} value={r.value}>{r.label}</option>)}</select></div>
              <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">النطاق</label><input type="text" value={form.scope} onChange={e => setForm(p => ({ ...p, scope: e.target.value }))} dir="ltr" placeholder="global أو wallet:jawali" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
            </div>
            {formError && <div className="rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm text-red-400">{formError}</div>}
          </div>
          <DialogFooter className="gap-2">
            <button onClick={() => setAddOpen(false)} className="rounded-lg border border-border/50 px-4 py-2 text-sm hover:bg-muted/50">إلغاء</button>
            <button onClick={handleAdd} disabled={formLoading} className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20 disabled:opacity-50">{formLoading ? "جارٍ الحفظ..." : "إضافة"}</button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ════════════════════════════════════════
// تبويب 2: إعدادات النظام
// ════════════════════════════════════════
function SystemTab() {
  const [lookAhead, setLookAhead] = useState("10");
  const [payerLimit, setPayerLimit] = useState("50000");
  const [tolerance, setTolerance] = useState("60");
  const [tgToken, setTgToken] = useState("");
  const [tgChatId, setTgChatId] = useState("");
  const [tgEnabled, setTgEnabled] = useState(false);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const handleSave = async () => {
    setSaving(true); setMsg(null);
    try {
      await apiPost("/admin/v1/settings", {
        lookAheadWindow: parseInt(lookAhead), defaultPayerLimit: parseInt(payerLimit),
        timestampTolerance: parseInt(tolerance),
        telegram: { botToken: tgToken, chatId: tgChatId, enabled: tgEnabled },
      });
      setMsg("تم حفظ الإعدادات بنجاح");
    } catch { setMsg("فشل حفظ الإعدادات"); }
    finally { setSaving(false); }
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <h2 className="text-lg font-semibold">إعدادات النظام</h2>
      {/* إعدادات الأمان */}
      <Card className="border-border/50">
        <CardHeader><CardTitle className="text-sm">إعدادات الأمان</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">نافذة التطلع (lookAheadWindow)</label><input type="number" value={lookAhead} onChange={e => setLookAhead(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /><p className="text-[10px] text-muted-foreground">عدد العدادات المسموح بها مقدماً</p></div>
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">حد الدافع الافتراضي</label><input type="number" value={payerLimit} onChange={e => setPayerLimit(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /><p className="text-[10px] text-muted-foreground">بالوحدة الصغرى (ريال × 100)</p></div>
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">تفاوت الطابع الزمني (ثانية)</label><input type="number" value={tolerance} onChange={e => setTolerance(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /><p className="text-[10px] text-muted-foreground">الفرق المسموح بالثواني</p></div>
          </div>
        </CardContent>
      </Card>
      {/* إعدادات تيليجرام */}
      <Card className="border-border/50">
        <CardHeader><CardTitle className="text-sm">إشعارات Telegram</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <button onClick={() => setTgEnabled(!tgEnabled)} className={`relative h-6 w-11 rounded-full transition-colors ${tgEnabled ? "bg-blue-500" : "bg-muted"}`}>
              <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform ${tgEnabled ? "right-0.5" : "right-[22px]"}`} />
            </button>
            <span className="text-sm">{tgEnabled ? "مفعّلة" : "معطّلة"}</span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">Bot Token</label><input type="password" value={tgToken} onChange={e => setTgToken(e.target.value)} dir="ltr" placeholder="123456:ABC-DEF..." className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">Chat ID</label><input type="text" value={tgChatId} onChange={e => setTgChatId(e.target.value)} dir="ltr" placeholder="-1001234567890" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
          </div>
        </CardContent>
      </Card>
      {msg && <div className={`rounded-md px-3 py-2 text-sm ${msg.includes("نجاح") ? "border border-emerald-500/20 bg-emerald-500/5 text-emerald-400" : "border border-red-500/20 bg-red-500/5 text-red-400"}`}>{msg}</div>}
      <button onClick={handleSave} disabled={saving} className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-6 py-2.5 text-sm font-medium text-white shadow-lg shadow-blue-600/20 disabled:opacity-50">{saving ? "جارٍ الحفظ..." : "حفظ الإعدادات"}</button>
    </div>
  );
}

// ════════════════════════════════════════
// تبويب 3: الملف الشخصي
// ════════════════════════════════════════
function ProfileTab() {
  const email = getUserEmail() || "";
  const [oldPass, setOldPass] = useState("");
  const [newPass, setNewPass] = useState("");
  const [confirmPass, setConfirmPass] = useState("");
  const [passLoading, setPassLoading] = useState(false);
  const [passMsg, setPassMsg] = useState<string | null>(null);
  const [totpLoading, setTotpLoading] = useState(false);
  const [totpQr, setTotpQr] = useState<string | null>(null);
  const [totpMsg, setTotpMsg] = useState<string | null>(null);

  const handleChangePass = async () => {
    if (newPass !== confirmPass) { setPassMsg("كلمتا المرور غير متطابقتين"); return; }
    if (newPass.length < 8) { setPassMsg("كلمة المرور يجب أن تكون 8 أحرف على الأقل"); return; }
    setPassLoading(true); setPassMsg(null);
    try {
      await apiPost("/admin/v1/auth/change-password", { oldPassword: oldPass, newPassword: newPass });
      setPassMsg("تم تغيير كلمة المرور بنجاح");
      setOldPass(""); setNewPass(""); setConfirmPass("");
    } catch (e) { setPassMsg(e instanceof Error ? e.message : "فشل تغيير كلمة المرور"); }
    finally { setPassLoading(false); }
  };

  const handleSetupTotp = async () => {
    setTotpLoading(true); setTotpMsg(null);
    try {
      const res = await apiPost<{ qrCode: string; secret: string }>("/admin/v1/auth/totp/setup");
      setTotpQr(res.qrCode || null);
      setTotpMsg("امسح رمز QR بتطبيق المصادقة");
    } catch (e) { setTotpMsg(e instanceof Error ? e.message : "فشل إعداد TOTP"); }
    finally { setTotpLoading(false); }
  };

  return (
    <div className="space-y-6 max-w-xl">
      <h2 className="text-lg font-semibold">الملف الشخصي</h2>
      {/* البريد */}
      <Card className="border-border/50">
        <CardContent className="p-4"><p className="text-xs text-muted-foreground">البريد الإلكتروني</p><p className="text-sm font-medium" dir="ltr">{email}</p></CardContent>
      </Card>
      {/* تغيير كلمة المرور */}
      <Card className="border-border/50">
        <CardHeader><CardTitle className="text-sm">تغيير كلمة المرور</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">كلمة المرور الحالية</label><input type="password" value={oldPass} onChange={e => setOldPass(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">كلمة المرور الجديدة</label><input type="password" value={newPass} onChange={e => setNewPass(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
            <div className="space-y-1"><label className="text-xs font-medium text-muted-foreground">تأكيد كلمة المرور</label><input type="password" value={confirmPass} onChange={e => setConfirmPass(e.target.value)} dir="ltr" className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" /></div>
          </div>
          {passMsg && <div className={`rounded-md px-3 py-2 text-sm ${passMsg.includes("نجاح") ? "border border-emerald-500/20 bg-emerald-500/5 text-emerald-400" : "border border-red-500/20 bg-red-500/5 text-red-400"}`}>{passMsg}</div>}
          <button onClick={handleChangePass} disabled={passLoading} className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20 disabled:opacity-50">{passLoading ? "جارٍ التغيير..." : "تغيير كلمة المرور"}</button>
        </CardContent>
      </Card>
      {/* TOTP */}
      <Card className="border-border/50">
        <CardHeader><CardTitle className="text-sm">التحقق الثنائي (TOTP)</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs text-muted-foreground">إعداد أو إعادة إعداد رمز التحقق الثنائي باستخدام تطبيق مصادقة مثل Google Authenticator</p>
          <button onClick={handleSetupTotp} disabled={totpLoading} className="rounded-lg border border-border/50 px-4 py-2 text-sm font-medium hover:bg-muted/50 disabled:opacity-50">{totpLoading ? "جارٍ الإعداد..." : "إعداد TOTP"}</button>
          {totpQr && (
            <div className="rounded-lg border border-border/50 bg-white p-4 text-center">
              <img src={totpQr} alt="QR Code" className="mx-auto h-48 w-48" />
            </div>
          )}
          {totpMsg && <div className={`rounded-md px-3 py-2 text-sm ${totpMsg.includes("امسح") ? "border border-blue-500/20 bg-blue-500/5 text-blue-400" : "border border-red-500/20 bg-red-500/5 text-red-400"}`}>{totpMsg}</div>}
        </CardContent>
      </Card>
    </div>
  );
}

// ════════════════════════════════════════
// الصفحة الرئيسية
// ════════════════════════════════════════
export default function SettingsPage() {
  const isSuperAdmin = hasRole("SUPER_ADMIN");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">الإعدادات</h1>
        <p className="text-sm text-muted-foreground">إدارة حسابات الداشبورد وإعدادات النظام</p>
      </div>
      <Tabs defaultValue="connection" dir="rtl">
        <TabsList className="w-full justify-start">
          <TabsTrigger value="connection">اتصال السويتش</TabsTrigger>
          <TabsTrigger value="profile">الملف الشخصي</TabsTrigger>
          {isSuperAdmin && <TabsTrigger value="accounts">حسابات الداشبورد</TabsTrigger>}
          {isSuperAdmin && <TabsTrigger value="system">إعدادات النظام</TabsTrigger>}
        </TabsList>
        <TabsContent value="connection"><SwitchConnectionTab /></TabsContent>
        <TabsContent value="profile"><ProfileTab /></TabsContent>
        {isSuperAdmin && <TabsContent value="accounts"><AccountsTab /></TabsContent>}
        {isSuperAdmin && <TabsContent value="system"><SystemTab /></TabsContent>}
      </Tabs>
    </div>
  );
}
