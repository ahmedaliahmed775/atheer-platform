"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  ArrowLeftRight,
  Smartphone,
  Shield,
  Settings,
  AlertTriangle,
  Gauge,
  Activity,
  Layers,
} from "lucide-react";
import { useEffect, useState } from "react";

const navItems = [
  { label: "لوحة المؤشرات", icon: LayoutDashboard, href: "/" },
  { label: "المعاملات", icon: ArrowLeftRight, href: "/transactions", badge: "live" },
  { label: "الأجهزة", icon: Smartphone, href: "/devices" },
  { label: "الحدود", icon: Gauge, href: "/limits" },
  { label: "النزاعات", icon: AlertTriangle, href: "/disputes" },
  { label: "Pipeline", icon: Layers, href: "/pipeline" },
];

const adminItems = [
  { label: "الأمان", icon: Shield, href: "/security" },
  { label: "الإعدادات", icon: Settings, href: "/settings" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [switchStatus, setSwitchStatus] = useState<"online" | "offline" | "checking">("checking");

  useEffect(() => {
    async function checkHealth() {
      try {
        const res = await fetch("http://localhost:8080/health", { signal: AbortSignal.timeout(3000) });
        setSwitchStatus(res.ok ? "online" : "offline");
      } catch {
        setSwitchStatus("offline");
      }
    }
    checkHealth();
    const interval = setInterval(checkHealth, 15000);
    return () => clearInterval(interval);
  }, []);

  return (
    <aside className="sidebar">
      {/* Logo */}
      <div className="sidebar-logo">
        <div className="sidebar-logo-icon">A</div>
        <div className="sidebar-logo-text">
          <h1>Atheer Switch</h1>
          <span>Control Panel v3.0</span>
        </div>
      </div>

      {/* Navigation */}
      <nav className="sidebar-nav">
        <span className="sidebar-section-label">الرئيسية</span>
        {navItems.map((item) => {
          const isActive = pathname === item.href || 
            (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`sidebar-link ${isActive ? "active" : ""}`}
            >
              <item.icon />
              <span>{item.label}</span>
              {item.badge && (
                <span className="sidebar-badge">{item.badge}</span>
              )}
            </Link>
          );
        })}

        <span className="sidebar-section-label">إدارة</span>
        {adminItems.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`sidebar-link ${isActive ? "active" : ""}`}
            >
              <item.icon />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="sidebar-footer">
        <div className="sidebar-status">
          <span className={`status-dot ${switchStatus === "online" ? "" : "offline"}`} />
          <span>
            Switch: {switchStatus === "checking" ? "..." : switchStatus === "online" ? "متصل" : "غير متصل"}
          </span>
          <Activity style={{ width: 12, height: 12, marginLeft: "auto" }} />
        </div>
      </div>
    </aside>
  );
}
