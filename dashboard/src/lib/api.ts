// عميل API للسويتش — fetch wrapper مع JWT
// يُضيف رمز المصادقة تلقائياً ويعالج أخطاء 401

import { getAccessToken, refreshSession, clearSession } from "./auth";

/** عنوان API الخلفي */
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/** خيارات الطلب */
export interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  params?: Record<string, string>;
  /** طلب بدون مصادقة (مثل تسجيل الدخول) */
  noAuth?: boolean;
  /** نوع المحتوى — افتراضي application/json */
  contentType?: string;
}

/** خطأ API */
export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: unknown
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** بناء سلسلة الاستعلام من المعاملات */
function buildQueryString(params: Record<string, string>): string {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      searchParams.append(key, value);
    }
  });
  const qs = searchParams.toString();
  return qs ? `?${qs}` : "";
}

/** تنفيذ طلب API مع إعادة المحاولة عند انتهاء الرمز */
export async function apiRequest<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const {
    method = "GET",
    body,
    headers = {},
    params,
    noAuth = false,
    contentType = "application/json",
  } = options;

  // بناء الرابط
  let url = `${API_BASE_URL}${path}`;
  if (params) {
    url += buildQueryString(params);
  }

  // بناء الرؤوس
  const reqHeaders: Record<string, string> = {
    "Content-Type": contentType,
    ...headers,
  };

  // إضافة رمز المصادقة
  if (!noAuth) {
    const token = getAccessToken();
    if (token) {
      reqHeaders["Authorization"] = `Bearer ${token}`;
    }
  }

  // بناء الجسم
  let reqBody: string | undefined;
  if (body !== undefined) {
    reqBody = JSON.stringify(body);
  }

  // تنفيذ الطلب
  const response = await fetch(url, {
    method,
    headers: reqHeaders,
    body: reqBody,
  });

  // معالجة انتهاء الرمز — محاولة تجديد واحدة
  if (response.status === 401 && !noAuth) {
    const refreshed = await refreshSession();
    if (refreshed) {
      // إعادة الطلب بالرمز الجديد
      const newToken = getAccessToken();
      reqHeaders["Authorization"] = `Bearer ${newToken}`;
      const retryResponse = await fetch(url, {
        method,
        headers: reqHeaders,
        body: reqBody,
      });
      return handleResponse<T>(retryResponse);
    }
    // فشل التجديد — تسجيل الخروج
    clearSession();
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
    throw new ApiError(401, "TOKEN_EXPIRED", "انتهت صلاحية الجلسة");
  }

  return handleResponse<T>(response);
}

/** معالجة الاستجابة */
async function handleResponse<T>(response: Response): Promise<T> {
  // لا محتوى
  if (response.status === 204) {
    return undefined as T;
  }

  // تصدير CSV — إرجاع النص مباشرة
  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("text/csv")) {
    const text = await response.text();
    return text as T;
  }

  // استجابة JSON
  const data = await response.json();

  if (!response.ok) {
    throw new ApiError(
      response.status,
      data.code || "UNKNOWN",
      data.message || "حدث خطأ غير متوقع",
      data.details
    );
  }

  return data as T;
}

/** اختصار GET */
export function apiGet<T>(
  path: string,
  params?: Record<string, string>
): Promise<T> {
  return apiRequest<T>(path, { method: "GET", params });
}

/** اختصار POST */
export function apiPost<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiRequest<T>(path, { method: "POST", body });
}

/** اختصار PUT */
export function apiPut<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiRequest<T>(path, { method: "PUT", body });
}

/** اختصار PATCH */
export function apiPatch<T>(
  path: string,
  body?: unknown
): Promise<T> {
  return apiRequest<T>(path, { method: "PATCH", body });
}

/** اختصار DELETE */
export function apiDelete<T>(path: string): Promise<T> {
  return apiRequest<T>(path, { method: "DELETE" });
}

/** تحميل CSV كملف */
export async function downloadCsv(
  path: string,
  params?: Record<string, string>,
  filename?: string
): Promise<void> {
  const csvContent = await apiRequest<string>(path, {
    method: "GET",
    params,
    contentType: "text/csv",
  });

  // إنشاء رابط تحميل
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename || "export.csv";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
