// إدارة JWT — تسجيل الدخول، التجديد، الخروج، الجلسة
// يُخزّن الرموز في localStorage مع دعم SSR

/** مفاتيح التخزين */
const STORAGE_KEYS = {
  accessToken: "atheer_access_token",
  refreshToken: "atheer_refresh_token",
  userRole: "atheer_user_role",
  userScope: "atheer_user_scope",
  userEmail: "atheer_user_email",
} as const;

/** بيانات الجلسة */
export interface Session {
  accessToken: string;
  refreshToken: string;
  role: string;
  scope: string;
  email: string;
}

/** استجابة تسجيل الدخول من السويتش */
interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  role: string;
  scope: string;
}

/** عنوان API الخلفي */
const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// ── عمليات التخزين ──

/** حفظ قيمة في localStorage بأمان */
function setItem(key: string, value: string): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(key, value);
  } catch {
    // تجاهل أخطاء التخزين (مثل الوضع الخاص)
  }
}

/** قراءة قيمة من localStorage بأمان */
function getItem(key: string): string | null {
  if (typeof window === "undefined") return null;
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

/** حذف قيمة من localStorage بأمان */
function removeItem(key: string): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.removeItem(key);
  } catch {
    // تجاهل
  }
}

// ── الوصول للرموز ──

/** الحصول على رمز الوصول */
export function getAccessToken(): string | null {
  return getItem(STORAGE_KEYS.accessToken);
}

/** الحصول على رمز التجديد */
export function getRefreshToken(): string | null {
  return getItem(STORAGE_KEYS.refreshToken);
}

/** الحصول على دور المستخدم */
export function getUserRole(): string | null {
  return getItem(STORAGE_KEYS.userRole);
}

/** الحصول على نطاق المستخدم */
export function getUserScope(): string | null {
  return getItem(STORAGE_KEYS.userScope);
}

/** الحصول على بريد المستخدم */
export function getUserEmail(): string | null {
  return getItem(STORAGE_KEYS.userEmail);
}

// ── الجلسة ──

/** الحصول على بيانات الجلسة الكاملة */
export function getSession(): Session | null {
  const accessToken = getAccessToken();
  if (!accessToken) return null;

  return {
    accessToken,
    refreshToken: getRefreshToken() || "",
    role: getUserRole() || "",
    scope: getUserScope() || "",
    email: getUserEmail() || "",
  };
}

/** هل المستخدم مسجّل الدخول؟ */
export function isAuthenticated(): boolean {
  return !!getAccessToken();
}

/** هل المستخدم يملك دوراً معيّناً على الأقل؟ */
export function hasRole(minRole: string): boolean {
  const role = getUserRole();
  if (!role) return false;

  const roleLevels: Record<string, number> = {
    SUPER_ADMIN: 4,
    ADMIN: 3,
    WALLET_ADMIN: 2,
    VIEWER: 1,
  };

  const userLevel = roleLevels[role] || 0;
  const requiredLevel = roleLevels[minRole] || 0;

  return userLevel >= requiredLevel;
}

// ── حفظ الجلسة ──

/** حفظ بيانات الجلسة بعد تسجيل الدخول */
function saveSession(data: LoginResponse): void {
  setItem(STORAGE_KEYS.accessToken, data.accessToken);
  setItem(STORAGE_KEYS.refreshToken, data.refreshToken);
  setItem(STORAGE_KEYS.userRole, data.role);
  setItem(STORAGE_KEYS.userScope, data.scope);
  // حفظ كوكي المصادقة — يُستخدم من وسيط المسارات
  setAuthCookie(data.accessToken);
  // استخراج البريد من الرمز (JWT payload)
  try {
    const payload = JSON.parse(atob(data.accessToken.split(".")[1]));
    setItem(STORAGE_KEYS.userEmail, payload.email || "");
  } catch {
    // تجاهل خطأ تحليل الرمز
  }
}

/** حفظ علم المصادقة في كوكي — يُستخدم من وسيط المسارات */
export function setAuthCookie(token: string): void {
  if (typeof document === "undefined") return;
  document.cookie = `atheer_auth_token=${token}; path=/; max-age=28800; SameSite=Strict`;
}

/** حذف كوكي المصادقة */
function removeAuthCookie(): void {
  if (typeof document === "undefined") return;
  document.cookie = "atheer_auth_token=; path=/; max-age=0";
}

// ── تسجيل الدخول ──

/** تسجيل الدخول بالبريد وكلمة المرور ورمز TOTP */
export async function login(
  email: string,
  password: string,
  totpCode?: string
): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE_URL}/admin/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      email,
      password,
      totpCode: totpCode || "",
    }),
  });

  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.message || "فشل تسجيل الدخول");
  }

  // حفظ الجلسة
  saveSession(data);

  return data as LoginResponse;
}

// ── تجديد الرمز ──

/** تجديد رمز الوصول باستخدام رمز التجديد */
export async function refreshSession(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const response = await fetch(`${API_BASE_URL}/admin/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });

    if (!response.ok) {
      clearSession();
      return false;
    }

    const data = await response.json();
    saveSession(data as LoginResponse);
    return true;
  } catch {
    return false;
  }
}

// ── تسجيل الخروج ──

/** تسجيل الخروج — حذف الجلسة محلياً وإعلام السويتش */
export async function logout(): Promise<void> {
  const accessToken = getAccessToken();

  try {
    if (accessToken) {
      await fetch(`${API_BASE_URL}/admin/v1/auth/logout`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
      });
    }
  } catch {
    // تجاهل خطأ الشبكة — الحذف المحلي أهم
  }

  clearSession();
}

/** حذف بيانات الجلسة من التخزين */
export function clearSession(): void {
  Object.values(STORAGE_KEYS).forEach((key) => removeItem(key));
}
