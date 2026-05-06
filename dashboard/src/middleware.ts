// وسيط حماية المسارات — يُعيد التوجيه لـ /login إذا لا يوجد JWT
// يعمل على حافة Next.js (Edge Runtime)

import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

/** المسارات التي لا تحتاج مصادقة */
const PUBLIC_PATHS = ["/login", "/api", "/health"];

/** المسارات الثابتة */
const STATIC_PATHS = ["/_next", "/favicon.ico", "/fonts", "/images"];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // السماح بالمسارات العامة
  if (PUBLIC_PATHS.some((path) => pathname.startsWith(path))) {
    return NextResponse.next();
  }

  // السماح بالملفات الثابتة
  if (STATIC_PATHS.some((path) => pathname.startsWith(path))) {
    return NextResponse.next();
  }

  // السماح بالجذر — يُعاد توجيهه للوحة التحكم
  if (pathname === "/") {
    return NextResponse.next();
  }

  // فحص رمز JWT في الكوكيز (بديل عن localStorage للوسيط)
  // ملاحظة: localStorage لا يتوفر في Edge Runtime
  // نستخدم كوكي auth_token كعلم على وجود جلسة
  const authToken = request.cookies.get("atheer_auth_token")?.value;

  if (!authToken) {
    // إعادة التوجيه لصفحة تسجيل الدخول مع الرجوع
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  // تطبيق الوسيط على كل المسارات ما عدا الثوابت
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
