// قائمة المستخدمين المسجلين — جدول مع بحث وإجراءات إدارية
// يشمل: تعليق/تفعيل/إلغاء حساب + تعديل حد الدافع
"use client";

import React, { useEffect, useState, useCallback } from "react";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { apiGet, apiPatch } from "@/lib/api";
import { hasRole, getUserScope } from "@/lib/auth";

// ── الأنواع ──

interface UserInfo {
  publicId: string;
  walletId: string;
  deviceId: string;
  counter: number;
  payerLimit: number;
  status: string;
  userType: string;
  createdAt: string;
  updatedAt: string;
}

interface UserListResponse {
  users: UserInfo[];
  totalCount: number;
  page: number;
  pageSize: number;
}

/** تنسيق المبلغ */
function formatAmount(amount: number): string {
  const val = amount / 100;
  return new Intl.NumberFormat("ar-YE", {
    style: "currency",
    currency: "YER",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(val);
}

/** تنسيق التاريخ */
function formatDate(dateStr: string): string {
  if (!dateStr) return "—";
  const d = new Date(dateStr);
  if (isNaN(d.getTime())) return dateStr;
  return new Intl.DateTimeFormat("ar-YE", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** شارة حالة المستخدم */
function UserStatusBadge({ status }: { status: string }) {
  const config: Record<string, { label: string; className: string }> = {
    ACTIVE: {
      label: "نشط",
      className: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
    },
    SUSPENDED: {
      label: "معلّق",
      className: "bg-amber-500/10 text-amber-400 border-amber-500/20",
    },
    REVOKED: {
      label: "ملغى",
      className: "bg-red-500/10 text-red-400 border-red-500/20",
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

/** شارة نوع المستخدم */
function UserTypeBadge({ type }: { type: string }) {
  return (
    <Badge
      variant="secondary"
      className={
        type === "P"
          ? "bg-blue-500/10 text-blue-400 border-blue-500/20 text-[10px]"
          : "bg-violet-500/10 text-violet-400 border-violet-500/20 text-[10px]"
      }
    >
      {type === "P" ? "دافع" : "تاجر"}
    </Badge>
  );
}

const PAGE_SIZE = 20;

export default function UsersPage() {
  // حالة البيانات
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

  // حالة البحث
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  // حالة الإجراءات
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  // حالة حوار تعديل الحد
  const [limitDialogOpen, setLimitDialogOpen] = useState(false);
  const [limitDialogUser, setLimitDialogUser] = useState<UserInfo | null>(null);
  const [newLimit, setNewLimit] = useState("");
  const [limitLoading, setLimitLoading] = useState(false);
  const [limitError, setLimitError] = useState<string | null>(null);

  // حالة حوار تأكيد الإجراء
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [confirmAction, setConfirmAction] = useState<{
    publicId: string;
    status: string;
    label: string;
  } | null>(null);

  const isAdmin = hasRole("ADMIN");
  const scope = getUserScope();
  const effectiveWalletId = scope && scope !== "global" ? scope : "";

  /** جلب المستخدمين */
  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {
        page: String(page),
        pageSize: String(PAGE_SIZE),
      };
      if (statusFilter) params.status = statusFilter;
      if (effectiveWalletId) params.walletId = effectiveWalletId;
      if (search) {
        // البحث بـ publicId أو walletId — نُرسل كمعامل
        params.search = search;
      }

      const res = await apiGet<UserListResponse>("/admin/v1/users", params);
      setUsers(res.users || []);
      setTotalCount(res.totalCount || 0);
    } catch {
      setUsers([]);
      setTotalCount(0);
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter, effectiveWalletId, search]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  /** تعديل حالة المستخدم */
  const handleUpdateStatus = useCallback(
    async (publicId: string, status: string) => {
      setActionLoading(publicId);
      setActionError(null);
      try {
        await apiPatch(`/admin/v1/users/${publicId}/status`, { status });
        // تحديث القائمة محلياً
        setUsers((prev) =>
          prev.map((u) =>
            u.publicId === publicId ? { ...u, status } : u
          )
        );
        setConfirmDialogOpen(false);
        setConfirmAction(null);
      } catch (err) {
        setActionError(
          err instanceof Error ? err.message : "فشل تحديث الحالة"
        );
      } finally {
        setActionLoading(null);
      }
    },
    []
  );

  /** فتح حوار التأكيد */
  const openConfirmDialog = (
    publicId: string,
    status: string,
    label: string
  ) => {
    setConfirmAction({ publicId, status, label });
    setConfirmDialogOpen(true);
    setActionError(null);
  };

  /** فتح حوار تعديل الحد */
  const openLimitDialog = (user: UserInfo) => {
    setLimitDialogUser(user);
    setNewLimit(String(user.payerLimit / 100));
    setLimitDialogOpen(true);
    setLimitError(null);
  };

  /** تعديل حد الدافع */
  const handleUpdateLimit = useCallback(async () => {
    if (!limitDialogUser || !newLimit) return;
    setLimitLoading(true);
    setLimitError(null);
    try {
      const limitValue = Math.round(parseFloat(newLimit) * 100);
      if (isNaN(limitValue) || limitValue <= 0) {
        setLimitError("يرجى إدخال قيمة صحيحة أكبر من صفر");
        setLimitLoading(false);
        return;
      }
      await apiPatch(
        `/admin/v1/users/${limitDialogUser.publicId}/limit`,
        { payerLimit: limitValue }
      );
      // تحديث القائمة محلياً
      setUsers((prev) =>
        prev.map((u) =>
          u.publicId === limitDialogUser.publicId
            ? { ...u, payerLimit: limitValue }
            : u
        )
      );
      setLimitDialogOpen(false);
      setLimitDialogUser(null);
    } catch (err) {
      setLimitError(
        err instanceof Error ? err.message : "فشل تحديث الحد"
      );
    } finally {
      setLimitLoading(false);
    }
  }, [limitDialogUser, newLimit]);

  const totalPages = Math.ceil(totalCount / PAGE_SIZE);

  // تصفية محلية بالبحث (publicId أو walletId)
  const filteredUsers = search
    ? users.filter(
        (u) =>
          u.publicId.toLowerCase().includes(search.toLowerCase()) ||
          u.walletId.toLowerCase().includes(search.toLowerCase())
      )
    : users;

  return (
    <div className="space-y-6">
      {/* العنوان */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">المستخدمون</h1>
        <p className="text-sm text-muted-foreground">
          إدارة المستخدمين المسجلين في سويتش Atheer
        </p>
      </div>

      {/* شريط البحث والفلترة */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-1 gap-3">
          {/* بحث */}
          <div className="relative flex-1 max-w-sm">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.3-4.3" />
            </svg>
            <input
              type="text"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              placeholder="بحث بـ publicId أو walletId..."
              dir="ltr"
              className="flex h-10 w-full rounded-lg border border-input bg-background pr-10 pl-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
          </div>

          {/* فلتر الحالة */}
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="flex h-10 rounded-lg border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value="">كل الحالات</option>
            <option value="ACTIVE">نشط</option>
            <option value="SUSPENDED">معلّق</option>
          </select>
        </div>

        {/* إحصائية */}
        <p className="text-sm text-muted-foreground">
          إجمالي {totalCount.toLocaleString("ar-YE")} مستخدم
        </p>
      </div>

      {/* الجدول */}
      <Card className="border-border/50">
        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50 hover:bg-transparent">
                  <TableHead className="text-xs font-semibold">المعرّف العام</TableHead>
                  <TableHead className="text-xs font-semibold">المحفظة</TableHead>
                  <TableHead className="text-xs font-semibold">النوع</TableHead>
                  <TableHead className="text-xs font-semibold">الحالة</TableHead>
                  <TableHead className="text-xs font-semibold">الجهاز</TableHead>
                  <TableHead className="text-xs font-semibold">العداد</TableHead>
                  <TableHead className="text-xs font-semibold">حد الدافع</TableHead>
                  <TableHead className="text-xs font-semibold">تاريخ التسجيل</TableHead>
                  {isAdmin && (
                    <TableHead className="text-xs font-semibold">الإجراءات</TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading ? (
                  <TableRow>
                    <TableCell colSpan={isAdmin ? 9 : 8} className="h-48 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-500 border-t-transparent" />
                        <span className="text-sm text-muted-foreground">جارٍ التحميل...</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : filteredUsers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={isAdmin ? 9 : 8} className="h-48 text-center">
                      <div className="flex flex-col items-center gap-2">
                        <svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground/30"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>
                        <span className="text-sm text-muted-foreground">لا يوجد مستخدمون</span>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredUsers.map((user) => (
                    <TableRow
                      key={user.publicId}
                      className="border-border/30 transition-colors hover:bg-muted/30"
                    >
                      <TableCell>
                        <span className="font-mono text-xs text-blue-400" dir="ltr">
                          {user.publicId}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary" className="text-[10px]">
                          {user.walletId}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <UserTypeBadge type={user.userType} />
                      </TableCell>
                      <TableCell>
                        <UserStatusBadge status={user.status} />
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-[10px] text-muted-foreground" dir="ltr">
                          {user.deviceId?.slice(0, 12)}…
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs">{user.counter}</span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs font-semibold">
                          {formatAmount(user.payerLimit)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-xs text-muted-foreground">
                          {formatDate(user.createdAt)}
                        </span>
                      </TableCell>
                      {isAdmin && (
                        <TableCell>
                          <div className="flex items-center gap-1">
                            {/* أزرار الإجراءات */}
                            {user.status === "ACTIVE" && (
                              <button
                                onClick={() =>
                                  openConfirmDialog(
                                    user.publicId,
                                    "SUSPENDED",
                                    "تعليق"
                                  )
                                }
                                disabled={actionLoading === user.publicId}
                                className="rounded-md bg-amber-500/10 px-2.5 py-1.5 text-[10px] font-medium text-amber-400 transition-colors hover:bg-amber-500/20 disabled:opacity-50"
                                title="تعليق الحساب"
                              >
                                تعليق
                              </button>
                            )}
                            {user.status === "SUSPENDED" && (
                              <button
                                onClick={() =>
                                  openConfirmDialog(
                                    user.publicId,
                                    "ACTIVE",
                                    "إلغاء التعليق"
                                  )
                                }
                                disabled={actionLoading === user.publicId}
                                className="rounded-md bg-emerald-500/10 px-2.5 py-1.5 text-[10px] font-medium text-emerald-400 transition-colors hover:bg-emerald-500/20 disabled:opacity-50"
                                title="إلغاء التعليق"
                              >
                                تفعيل
                              </button>
                            )}
                            {user.status !== "REVOKED" && (
                              <button
                                onClick={() =>
                                  openConfirmDialog(
                                    user.publicId,
                                    "REVOKED",
                                    "إلغاء الحساب"
                                  )
                                }
                                disabled={actionLoading === user.publicId}
                                className="rounded-md bg-red-500/10 px-2.5 py-1.5 text-[10px] font-medium text-red-400 transition-colors hover:bg-red-500/20 disabled:opacity-50"
                                title="إلغاء الحساب نهائياً"
                              >
                                إلغاء
                              </button>
                            )}
                            <button
                              onClick={() => openLimitDialog(user)}
                              disabled={actionLoading === user.publicId}
                              className="rounded-md bg-blue-500/10 px-2.5 py-1.5 text-[10px] font-medium text-blue-400 transition-colors hover:bg-blue-500/20 disabled:opacity-50"
                              title="تعديل حد الدافع"
                            >
                              الحد
                            </button>
                          </div>
                        </TableCell>
                      )}
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* شريط الصفحات */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between border-t border-border/50 px-4 py-3">
              <p className="text-sm text-muted-foreground">
                صفحة {page} من {totalPages}
              </p>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  className="flex h-8 items-center rounded-md border border-border/50 bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/50 disabled:pointer-events-none disabled:opacity-50"
                >
                  السابقة
                </button>
                <button
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  disabled={page >= totalPages}
                  className="flex h-8 items-center rounded-md border border-border/50 bg-card px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted/50 disabled:pointer-events-none disabled:opacity-50"
                >
                  التالية
                </button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* ═══ حوار تأكيد تعديل الحالة ═══ */}
      <Dialog open={confirmDialogOpen} onOpenChange={setConfirmDialogOpen}>
        <DialogContent className="border-border/50">
          <DialogHeader>
            <DialogTitle>تأكيد {confirmAction?.label}</DialogTitle>
            <DialogDescription>
              هل أنت متأكد من {confirmAction?.label} للمستخدم{" "}
              <span className="font-mono text-foreground" dir="ltr">
                {confirmAction?.publicId}
              </span>
              ؟
            </DialogDescription>
          </DialogHeader>

          {actionError && (
            <div className="rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm text-red-400">
              {actionError}
            </div>
          )}

          <DialogFooter className="gap-2">
            <button
              onClick={() => setConfirmDialogOpen(false)}
              className="rounded-lg border border-border/50 px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted/50"
            >
              إلغاء
            </button>
            <button
              onClick={() => {
                if (confirmAction) {
                  handleUpdateStatus(
                    confirmAction.publicId,
                    confirmAction.status
                  );
                }
              }}
              disabled={actionLoading !== null}
              className={`rounded-lg px-4 py-2 text-sm font-medium text-white transition-all disabled:opacity-50 ${
                confirmAction?.status === "REVOKED"
                  ? "bg-red-600 hover:bg-red-700 shadow-lg shadow-red-600/20"
                  : confirmAction?.status === "SUSPENDED"
                  ? "bg-amber-600 hover:bg-amber-700 shadow-lg shadow-amber-600/20"
                  : "bg-emerald-600 hover:bg-emerald-700 shadow-lg shadow-emerald-600/20"
              }`}
            >
              {actionLoading ? (
                <span className="flex items-center gap-2">
                  <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  جارٍ التنفيذ...
                </span>
              ) : (
                `تأكيد ${confirmAction?.label}`
              )}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ═══ حوار تعديل حد الدافع ═══ */}
      <Dialog open={limitDialogOpen} onOpenChange={setLimitDialogOpen}>
        <DialogContent className="border-border/50">
          <DialogHeader>
            <DialogTitle>تعديل حد الدافع</DialogTitle>
            <DialogDescription>
              تعديل الحد الأقصى للمعاملة الواحدة للمستخدم{" "}
              <span className="font-mono text-foreground" dir="ltr">
                {limitDialogUser?.publicId}
              </span>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* الحد الحالي */}
            <div className="rounded-lg bg-muted/30 px-4 py-3">
              <p className="text-xs text-muted-foreground">الحد الحالي</p>
              <p className="text-lg font-bold text-foreground">
                {formatAmount(limitDialogUser?.payerLimit ?? 0)}
              </p>
            </div>

            {/* الحد الجديد */}
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">
                الحد الجديد (بالريال اليمني)
              </label>
              <input
                type="number"
                value={newLimit}
                onChange={(e) => setNewLimit(e.target.value)}
                placeholder="مثال: 50000"
                dir="ltr"
                min="1"
                className="flex h-10 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              />
              <p className="text-xs text-muted-foreground">
                أدخل المبلغ بالريال اليمني (سيُحوّل تلقائياً للوحدة الصغرى)
              </p>
            </div>

            {limitError && (
              <div className="rounded-md border border-red-500/20 bg-red-500/5 px-3 py-2 text-sm text-red-400">
                {limitError}
              </div>
            )}
          </div>

          <DialogFooter className="gap-2">
            <button
              onClick={() => setLimitDialogOpen(false)}
              className="rounded-lg border border-border/50 px-4 py-2 text-sm text-foreground transition-colors hover:bg-muted/50"
            >
              إلغاء
            </button>
            <button
              onClick={handleUpdateLimit}
              disabled={limitLoading || !newLimit}
              className="rounded-lg bg-gradient-to-l from-blue-600 to-blue-700 px-4 py-2 text-sm font-medium text-white shadow-lg shadow-blue-600/20 transition-all hover:shadow-blue-600/30 disabled:opacity-50"
            >
              {limitLoading ? (
                <span className="flex items-center gap-2">
                  <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  جارٍ التحديث...
                </span>
              ) : (
                "حفظ الحد الجديد"
              )}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
