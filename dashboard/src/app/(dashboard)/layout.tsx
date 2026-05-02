// تخطيط لوحة التحكم — يُغلّف كل صفحات الداشبورد بالشريط الجانبي
"use client";

import AppSidebar from "@/components/app-sidebar";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppSidebar>{children}</AppSidebar>;
}
