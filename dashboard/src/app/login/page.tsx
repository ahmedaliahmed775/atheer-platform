// شاشة تسجيل الدخول — البريد الإلكتروني + كلمة المرور + رمز TOTP
// تصميم dark mode أنيق مع دعم RTL
"use client";

import { useState, useCallback, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { login, setAuthCookie } from "@/lib/auth";
import { cn } from "@/lib/utils";

/** حالات خطأ تسجيل الدخول */
type LoginError = {
  message: string;
  requiresTOTP?: boolean;
};

/** محتوى نموذج تسجيل الدخول — يُغلّف بـ Suspense بسبب useSearchParams */
function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectPath = searchParams.get("redirect") || "/";

  // حالة النموذج
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [totpCode, setTotpCode] = useState("");
  const [showTOTP, setShowTOTP] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<LoginError | null>(null);

  /** معالجة تسجيل الدخول */
  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      setIsLoading(true);

      try {
        await login(email, password, showTOTP ? totpCode : undefined);
        // تسجيل الدخول ناجح — التوجيه للصفحة المطلوبة
        router.push(redirectPath);
        router.refresh();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "حدث خطأ غير متوقع";

        // إذا كان الخطأ يطلب رمز TOTP
        if (message.includes("TOTP") || message.includes("التحقق الثنائي")) {
          setShowTOTP(true);
          setError({ message: "يرجى إدخال رمز التحقق الثنائي", requiresTOTP: true });
        } else {
          setError({ message });
        }
      } finally {
        setIsLoading(false);
      }
    },
    [email, password, totpCode, showTOTP, router, redirectPath]
  );

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <div className="w-full max-w-md space-y-8">
        {/* الشعار والعنوان */}
        <div className="text-center">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <span className="text-2xl font-bold text-primary-foreground">A</span>
          </div>
          <h1 className="text-3xl font-bold text-foreground">سويتش Atheer</h1>
          <p className="mt-2 text-muted-foreground">
            لوحة تحكم إدارة المدفوعات
          </p>
        </div>

        {/* نموذج تسجيل الدخول */}
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* بطاقة النموذج */}
          <div className="rounded-lg border border-border bg-card p-6 shadow-lg space-y-4">
            {/* خطأ تسجيل الدخول */}
            {error && (
              <div
                className={cn(
                  "rounded-md p-3 text-sm",
                  error.requiresTOTP
                    ? "bg-yellow-500/10 text-yellow-500 border border-yellow-500/20"
                    : "bg-destructive/10 text-destructive border border-destructive/20"
                )}
              >
                {error.message}
              </div>
            )}

            {/* البريد الإلكتروني */}
            <div className="space-y-2">
              <label
                htmlFor="email"
                className="text-sm font-medium text-foreground"
              >
                البريد الإلكتروني
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@atheer.ye"
                required
                dir="ltr"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              />
            </div>

            {/* كلمة المرور */}
            <div className="space-y-2">
              <label
                htmlFor="password"
                className="text-sm font-medium text-foreground"
              >
                كلمة المرور
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                required
                dir="ltr"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              />
            </div>

            {/* رمز التحقق الثنائي TOTP — يظهر عند الحاجة */}
            {showTOTP && (
              <div className="space-y-2">
                <label
                  htmlFor="totp"
                  className="text-sm font-medium text-foreground"
                >
                  رمز التحقق الثنائي (TOTP)
                </label>
                <input
                  id="totp"
                  type="text"
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                  placeholder="000000"
                  maxLength={6}
                  dir="ltr"
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-center text-lg tracking-[0.5em] ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
                <p className="text-xs text-muted-foreground">
                  أدخل الرمز المكون من 6 أرقام من تطبيق المصادقة
                </p>
              </div>
            )}

            {/* زر تسجيل الدخول */}
            <button
              type="submit"
              disabled={isLoading || !email || !password}
              className="flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground ring-offset-background transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"
            >
              {isLoading ? (
                <span className="flex items-center gap-2">
                  <svg
                    className="h-4 w-4 animate-spin"
                    xmlns="http://www.w3.org/2000/svg"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                  جارٍ تسجيل الدخول...
                </span>
              ) : (
                "تسجيل الدخول"
              )}
            </button>
          </div>

          {/* معلومات إضافية */}
          <p className="text-center text-xs text-muted-foreground">
            سويتش Atheer — نظام مدفوعات NFC آمن
          </p>
        </form>
      </div>
    </div>
  );
}

/** صفحة تسجيل الدخول — تغليف المحتوى بـ Suspense لاستخدام useSearchParams */
export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-background">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      }
    >
      <LoginForm />
    </Suspense>
  );
}
