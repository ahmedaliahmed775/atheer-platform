"use client";

import { Settings, Save, Server, Database, Shield } from "lucide-react";
import { useState } from "react";

export default function SettingsPage() {
  const [saved, setSaved] = useState(false);

  const handleSave = () => {
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>الإعدادات</h2>
          <p>تكوين الخادم وقاعدة البيانات والاتصالات</p>
        </div>
        <div className="page-header-right">
          <button className="btn btn-primary btn-sm" onClick={handleSave}>
            <Save style={{ width: 14, height: 14 }} />
            {saved ? "✓ تم الحفظ" : "حفظ"}
          </button>
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 16 }}>
        {/* Server */}
        <div className="content-card">
          <div className="content-card-header">
            <h3><Server style={{ width: 16, height: 16, display: "inline", verticalAlign: "middle" }} /> الخادم</h3>
          </div>
          {[
            { label: "المنفذ", value: "8080", key: "port" },
            { label: "البيئة", value: "development", key: "env" },
            { label: "Read Timeout", value: "10s", key: "readTimeout" },
            { label: "Write Timeout", value: "15s", key: "writeTimeout" },
            { label: "Shutdown Timeout", value: "30s", key: "shutdownTimeout" },
          ].map((item) => (
            <div key={item.key} style={{
              display: "flex", alignItems: "center", justifyContent: "space-between",
              padding: "10px 0", borderBottom: "1px solid var(--border-color)",
            }}>
              <span style={{ fontSize: 13, color: "var(--text-secondary)" }}>{item.label}</span>
              <input defaultValue={item.value} style={{
                background: "var(--bg-primary)", border: "1px solid var(--border-color)",
                borderRadius: 6, padding: "5px 10px", color: "var(--text-primary)",
                fontSize: 12, fontFamily: "'IBM Plex Mono', monospace", width: 140, textAlign: "right",
                outline: "none",
              }} />
            </div>
          ))}
        </div>

        {/* Database */}
        <div className="content-card">
          <div className="content-card-header">
            <h3><Database style={{ width: 16, height: 16, display: "inline", verticalAlign: "middle" }} /> قاعدة البيانات</h3>
          </div>
          {[
            { label: "PostgreSQL Host", value: "localhost:5432", key: "dbhost" },
            { label: "Database", value: "atheer", key: "dbname" },
            { label: "Pool Size Min", value: "5", key: "poolMin" },
            { label: "Pool Size Max", value: "20", key: "poolMax" },
            { label: "Redis Host", value: "localhost:6379", key: "redishost" },
          ].map((item) => (
            <div key={item.key} style={{
              display: "flex", alignItems: "center", justifyContent: "space-between",
              padding: "10px 0", borderBottom: "1px solid var(--border-color)",
            }}>
              <span style={{ fontSize: 13, color: "var(--text-secondary)" }}>{item.label}</span>
              <input defaultValue={item.value} style={{
                background: "var(--bg-primary)", border: "1px solid var(--border-color)",
                borderRadius: 6, padding: "5px 10px", color: "var(--text-primary)",
                fontSize: 12, fontFamily: "'IBM Plex Mono', monospace", width: 180, textAlign: "right",
                outline: "none",
              }} />
            </div>
          ))}
        </div>

        {/* Rate Limits */}
        <div className="content-card">
          <div className="content-card-header">
            <h3><Shield style={{ width: 16, height: 16, display: "inline", verticalAlign: "middle" }} /> حدود السرعة</h3>
          </div>
          {[
            { label: "Per Device (req/min)", value: "10", key: "rlDevice" },
            { label: "Per Wallet (req/min)", value: "100", key: "rlWallet" },
            { label: "Per IP (req/min)", value: "50", key: "rlIP" },
            { label: "Tx Time Window", value: "5 min", key: "txWindow" },
            { label: "Idempotency TTL", value: "3600s", key: "idemTTL" },
          ].map((item) => (
            <div key={item.key} style={{
              display: "flex", alignItems: "center", justifyContent: "space-between",
              padding: "10px 0", borderBottom: "1px solid var(--border-color)",
            }}>
              <span style={{ fontSize: 13, color: "var(--text-secondary)" }}>{item.label}</span>
              <input defaultValue={item.value} style={{
                background: "var(--bg-primary)", border: "1px solid var(--border-color)",
                borderRadius: 6, padding: "5px 10px", color: "var(--text-primary)",
                fontSize: 12, fontFamily: "'IBM Plex Mono', monospace", width: 120, textAlign: "right",
                outline: "none",
              }} />
            </div>
          ))}
        </div>

        {/* Adapters */}
        <div className="content-card">
          <div className="content-card-header">
            <h3>المحولات (Adapters)</h3>
          </div>
          {[
            { name: "JEEP", url: "https://api.jeep.ye", status: "مفعل" },
            { name: "WENET", url: "https://api.wenet.ye", status: "مفعل" },
            { name: "WASEL", url: "https://api.wasel.ye", status: "مفعل" },
          ].map((adapter) => (
            <div key={adapter.name} style={{
              display: "flex", alignItems: "center", gap: 12,
              padding: "10px 0", borderBottom: "1px solid var(--border-color)",
            }}>
              <span className="badge info">{adapter.name}</span>
              <span style={{ fontSize: 12, fontFamily: "'IBM Plex Mono', monospace", color: "var(--text-muted)", flex: 1 }}>
                {adapter.url}
              </span>
              <span className="badge success">{adapter.status}</span>
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
