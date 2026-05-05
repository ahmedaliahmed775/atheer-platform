// شريط التنقل الجانبي — التصميم الرئيسي للوحة التحكم
// يتضمّن القوائم والروابط وتسجيل الخروج مع دعم RTL
"use client";

import React, { useCallback, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { logout, getUserEmail, getUserRole, getUserScope } from "@/lib/auth";
import { cn } from "@/lib/utils";

/** عنصر قائمة التنقل */
interface NavItem {
  label: string;
  href: string;
  icon: React.ReactNode;
  minRole?: string;
}

/** أيقونات SVG مدمجة — لتجنب أي تبعيات خارجية غير مطلوبة */
const Icons = {
  dashboard: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect width="7" height="9" x="3" y="3" rx="1"/><rect width="7" height="5" x="14" y="3" rx="1"/><rect width="7" height="9" x="14" y="12" rx="1"/><rect width="7" height="5" x="3" y="16" rx="1"/></svg>
  ),
  transactions: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
  ),
  users: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
  ),
  wallets: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/></svg>
  ),
  analytics: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
  ),
  reconciliation: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M16 3h5v5"/><path d="M8 3H3v5"/><path d="M12 22v-8.3a4 4 0 0 0-1.172-2.872L3 3"/><path d="m15 9 6-6"/></svg>
  ),
  commission: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8"/><path d="M12 18V6"/></svg>
  ),
  settings: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>
  ),
  terminal: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" x2="20" y1="19" y2="19"/></svg>
  ),
  logout: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" x2="9" y1="12" y2="12"/></svg>
  ),
  menu: (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="4" x2="20" y1="12" y2="12"/><line x1="4" x2="20" y1="6" y2="6"/><line x1="4" x2="20" y1="18" y2="18"/></svg>
  ),
};

/** عناصر القائمة الرئيسية */
const NAV_ITEMS: NavItem[] = [
  { label: "لوحة القيادة", href: "/dashboard", icon: Icons.dashboard },
  { label: "المعاملات", href: "/transactions", icon: Icons.transactions },
  { label: "المستخدمون", href: "/users", icon: Icons.users, minRole: "ADMIN" },
  { label: "المحافظ", href: "/wallets", icon: Icons.wallets, minRole: "ADMIN" },
  { label: "الإحصائيات", href: "/analytics", icon: Icons.analytics },
  { label: "العمولات", href: "/commissions", icon: Icons.commission, minRole: "ADMIN" },
  { label: "التسوية", href: "/reconciliation", icon: Icons.reconciliation, minRole: "ADMIN" },
  { label: "الطرفية", href: "/terminal", icon: Icons.terminal, minRole: "SUPER_ADMIN" },
  { label: "الإعدادات", href: "/settings", icon: Icons.settings },
];

/** مستويات الأدوار */
const ROLE_LEVELS: Record<string, number> = {
  SUPER_ADMIN: 4,
  ADMIN: 3,
  WALLET_ADMIN: 2,
  VIEWER: 1,
};

/** ترجمة الأدوار بالعربية */
const ROLE_LABELS: Record<string, string> = {
  SUPER_ADMIN: "مدير أعلى",
  ADMIN: "مدير",
  WALLET_ADMIN: "مدير محفظة",
  VIEWER: "مشاهد",
};

export default function AppSidebar({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  const email = getUserEmail() || "";
  const role = getUserRole() || "VIEWER";
  const scope = getUserScope() || "";

  /** تسجيل الخروج */
  const handleLogout = useCallback(async () => {
    await logout();
    router.push("/login");
  }, [router]);

  /** هل العنصر مرئي للدور الحالي */
  const isVisible = (item: NavItem) => {
    if (!item.minRole) return true;
    return (ROLE_LEVELS[role] || 0) >= (ROLE_LEVELS[item.minRole] || 0);
  };

  /** هل المسار نشط */
  const isActive = (href: string) => {
    if (href === "/dashboard") return pathname === "/dashboard" || pathname === "/";
    return pathname.startsWith(href);
  };

  /** محتوى الشريط الجانبي */
  const sidebarContent = (
    <div className="flex h-full flex-col">
      {/* الشعار */}
      <div className="flex h-16 items-center gap-3 border-b border-border/50 px-4">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-violet-600 shadow-lg shadow-blue-500/20">
          <span className="text-sm font-bold text-white">A</span>
        </div>
        {!collapsed && (
          <div className="flex flex-col">
            <span className="text-sm font-bold text-foreground">Atheer</span>
            <span className="text-[10px] text-muted-foreground">لوحة التحكم</span>
          </div>
        )}
      </div>

      {/* القوائم */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
        {NAV_ITEMS.filter(isVisible).map((item) => (
          <Link
            key={item.href}
            href={item.href}
            onClick={() => setMobileOpen(false)}
            className={cn(
              "group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-200",
              isActive(item.href)
                ? "bg-gradient-to-l from-blue-500/15 to-violet-500/10 text-blue-400 shadow-sm"
                : "text-muted-foreground hover:bg-muted/50 hover:text-foreground"
            )}
          >
            <span className={cn(
              "flex-shrink-0 transition-colors",
              isActive(item.href) ? "text-blue-400" : "text-muted-foreground group-hover:text-foreground"
            )}>
              {item.icon}
            </span>
            {!collapsed && <span>{item.label}</span>}
            {isActive(item.href) && (
              <span className="mr-auto h-1.5 w-1.5 rounded-full bg-blue-400 shadow-sm shadow-blue-400/50" />
            )}
          </Link>
        ))}
      </nav>

      {/* معلومات المستخدم + تسجيل الخروج */}
      <div className="border-t border-border/50 p-3">
        {!collapsed && (
          <div className="mb-2 rounded-lg bg-muted/30 px-3 py-2">
            <p className="truncate text-xs font-medium text-foreground" dir="ltr">{email}</p>
            <p className="text-[10px] text-muted-foreground">
              {ROLE_LABELS[role] || role}
              {scope && scope !== "global" ? ` • ${scope}` : ""}
            </p>
          </div>
        )}
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
        >
          {Icons.logout}
          {!collapsed && <span>تسجيل الخروج</span>}
        </button>
      </div>
    </div>
  );

  return (
    <div className="flex min-h-screen bg-background">
      {/* التراكب على الجوال */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm lg:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* الشريط الجانبي — سطح المكتب */}
      <aside
        className={cn(
          "fixed inset-y-0 right-0 z-50 hidden border-l border-border/50 bg-card/80 backdrop-blur-xl transition-all duration-300 lg:block",
          collapsed ? "w-[68px]" : "w-64"
        )}
      >
        {sidebarContent}
      </aside>

      {/* الشريط الجانبي — جوال */}
      <aside
        className={cn(
          "fixed inset-y-0 right-0 z-50 w-64 border-l border-border/50 bg-card/95 backdrop-blur-xl transition-transform duration-300 lg:hidden",
          mobileOpen ? "translate-x-0" : "translate-x-full"
        )}
      >
        {sidebarContent}
      </aside>

      {/* المحتوى الرئيسي */}
      <main className={cn(
        "flex-1 transition-all duration-300",
        collapsed ? "lg:mr-[68px]" : "lg:mr-64"
      )}>
        {/* شريط علوي */}
        <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b border-border/50 bg-background/80 px-4 backdrop-blur-xl lg:px-6">
          {/* زر القائمة — جوال */}
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="rounded-lg p-2 text-muted-foreground hover:bg-muted/50 lg:hidden"
          >
            {Icons.menu}
          </button>
          {/* زر طي الشريط — سطح مكتب */}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="hidden rounded-lg p-2 text-muted-foreground hover:bg-muted/50 lg:block"
          >
            {Icons.menu}
          </button>
          <div className="flex-1" />
        </header>
        <div className="p-4 lg:p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
