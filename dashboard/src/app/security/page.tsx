"use client";

import { Shield, Key, Lock, Fingerprint, AlertTriangle, CheckCircle, Eye, EyeOff } from "lucide-react";

const securityConfig = [
  { name: "HMAC Algorithm", value: "HMAC-SHA256", status: "active", icon: Key },
  { name: "ECDSA Curve", value: "P-256 (secp256r1)", status: "active", icon: Fingerprint },
  { name: "LUK Derivation", value: "HMAC(seed, BigEndian(ctr))", status: "active", icon: Lock },
  { name: "Anti-Replay", value: "Redis Lua — Atomic", status: "active", icon: Shield },
  { name: "Signature Comparison", value: "Constant-Time (hmac.Equal)", status: "active", icon: Shield },
  { name: "KMS Encryption", value: "غير مفعّل — مطلوب للإنتاج", status: "warning", icon: AlertTriangle },
  { name: "TLS (mTLS)", value: "غير مفعّل — مطلوب للإنتاج", status: "warning", icon: Lock },
  { name: "Play Integrity", value: "غير مفعّل — اختياري", status: "info", icon: Shield },
];

const auditLog = [
  { time: "22:41:03", user: "system", action: "Server started", detail: "Atheer Switch V3.0 — port 8080" },
  { time: "22:41:05", user: "system", action: "DB connected", detail: "PostgreSQL 16 — pool size 10" },
  { time: "22:41:05", user: "system", action: "Redis connected", detail: "Redis 7 — anti-replay ready" },
  { time: "22:41:06", user: "system", action: "Pipeline loaded", detail: "10 layers active" },
  { time: "22:42:15", user: "admin", action: "Limits updated", detail: "JEEP P2P_SAME maxTx → 50,000" },
  { time: "22:45:33", user: "system", action: "Replay blocked", detail: "DEV-A3F2 — ctr=42 (duplicate)" },
];

export default function SecurityPage() {
  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>الأمان</h2>
          <p>إعدادات التشفير والتوقيعات وسجل التدقيق</p>
        </div>
      </div>

      {/* Security Config */}
      <div className="content-card" style={{ marginBottom: 16 }}>
        <div className="content-card-header">
          <h3>إعدادات التشفير والتحقق</h3>
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 10 }}>
          {securityConfig.map((item) => (
            <div key={item.name} style={{
              display: "flex", alignItems: "center", gap: 12,
              padding: "12px 14px", borderRadius: 8,
              background: "var(--bg-primary)", border: "1px solid var(--border-color)",
            }}>
              <div style={{
                width: 32, height: 32, borderRadius: 8,
                background: item.status === "active" ? "rgba(16, 185, 129, 0.12)" : item.status === "warning" ? "rgba(245, 158, 11, 0.12)" : "rgba(59, 130, 246, 0.12)",
                color: item.status === "active" ? "var(--accent-emerald)" : item.status === "warning" ? "var(--accent-amber)" : "var(--accent-blue)",
                display: "flex", alignItems: "center", justifyContent: "center",
              }}>
                <item.icon style={{ width: 16, height: 16 }} />
              </div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 12, fontWeight: 600 }}>{item.name}</div>
                <div style={{ fontSize: 11, color: "var(--text-muted)", fontFamily: "'IBM Plex Mono', monospace" }}>{item.value}</div>
              </div>
              {item.status === "active" ? (
                <CheckCircle style={{ width: 16, height: 16, color: "var(--accent-emerald)" }} />
              ) : (
                <AlertTriangle style={{ width: 16, height: 16, color: "var(--accent-amber)" }} />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Audit Log */}
      <div className="content-card">
        <div className="content-card-header">
          <h3>سجل التدقيق</h3>
          <span style={{ fontSize: 11, color: "var(--text-muted)" }}>آخر 24 ساعة</span>
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>الوقت</th>
              <th>المستخدم</th>
              <th>الإجراء</th>
              <th>التفاصيل</th>
            </tr>
          </thead>
          <tbody>
            {auditLog.map((log, i) => (
              <tr key={i}>
                <td className="mono">{log.time}</td>
                <td><span className={`badge ${log.user === "system" ? "info" : "purple"}`}>{log.user}</span></td>
                <td style={{ fontWeight: 600 }}>{log.action}</td>
                <td style={{ fontSize: 12, color: "var(--text-muted)", fontFamily: "'IBM Plex Mono', monospace" }}>{log.detail}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
