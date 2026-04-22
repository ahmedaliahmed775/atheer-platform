"use client";

import { useState } from "react";
import {
  Smartphone,
  Search,
  ShieldCheck,
  ShieldAlert,
  ShieldOff,
  Fingerprint,
  Key,
  Clock,
  MoreVertical,
} from "lucide-react";

const mockDevices = Array.from({ length: 15 }, (_, i) => ({
  id: `DEV-${Math.random().toString(36).substr(2, 8).toUpperCase()}`,
  walletId: ["JEEP", "WENET", "WASEL"][i % 3],
  deviceModel: ["Samsung Galaxy S24", "Pixel 8 Pro", "Xiaomi 14", "OnePlus 12"][i % 4],
  attestationLevel: ["STRONG_BOX", "TEE", "SOFTWARE"][i % 3] as string,
  status: ["ACTIVE", "ACTIVE", "ACTIVE", "SUSPENDED", "REVOKED"][i % 5],
  ctr: Math.floor(Math.random() * 500),
  enrolledAt: new Date(Date.now() - i * 86400000 * 3).toLocaleDateString("ar-SA"),
  lastTxAt: new Date(Date.now() - i * 3600000).toLocaleString("ar-SA"),
  txCount: Math.floor(Math.random() * 200 + 10),
}));

function AttestationBadge({ level }: { level: string }) {
  if (level === "STRONG_BOX") return <span className="badge success"><Fingerprint style={{ width: 11, height: 11 }} /> StrongBox</span>;
  if (level === "TEE") return <span className="badge info"><ShieldCheck style={{ width: 11, height: 11 }} /> TEE</span>;
  return <span className="badge warning"><ShieldAlert style={{ width: 11, height: 11 }} /> Software</span>;
}

function StatusBadge({ status }: { status: string }) {
  if (status === "ACTIVE") return <span className="badge success"><span className="badge-dot" /> نشط</span>;
  if (status === "SUSPENDED") return <span className="badge warning"><span className="badge-dot" /> معلّق</span>;
  return <span className="badge error"><span className="badge-dot" /> ملغى</span>;
}

export default function DevicesPage() {
  const [search, setSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("ALL");

  const filtered = mockDevices.filter((d) => {
    if (filterStatus !== "ALL" && d.status !== filterStatus) return false;
    if (search && !d.id.toLowerCase().includes(search.toLowerCase()) && !d.deviceModel.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const stats = {
    total: mockDevices.length,
    active: mockDevices.filter((d) => d.status === "ACTIVE").length,
    strongbox: mockDevices.filter((d) => d.attestationLevel === "STRONG_BOX").length,
  };

  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>الأجهزة</h2>
          <p>إدارة الأجهزة المسجلة والتحقق من مستوى الأمان</p>
        </div>
      </div>

      {/* Stats */}
      <div className="stat-grid" style={{ gridTemplateColumns: "repeat(3, 1fr)" }}>
        <div className="stat-card blue">
          <div className="stat-card-header">
            <div className="stat-card-icon blue"><Smartphone style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats.total}</div>
          <div className="stat-card-label">إجمالي الأجهزة</div>
        </div>
        <div className="stat-card emerald">
          <div className="stat-card-header">
            <div className="stat-card-icon emerald"><ShieldCheck style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats.active}</div>
          <div className="stat-card-label">الأجهزة النشطة</div>
        </div>
        <div className="stat-card purple">
          <div className="stat-card-header">
            <div className="stat-card-icon purple"><Fingerprint style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats.strongbox}</div>
          <div className="stat-card-label">StrongBox أجهزة</div>
        </div>
      </div>

      {/* Filters */}
      <div className="content-card" style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <div style={{
            display: "flex", alignItems: "center", gap: 8,
            background: "var(--bg-primary)", borderRadius: 8, padding: "6px 12px",
            border: "1px solid var(--border-color)", flex: 1, maxWidth: 300,
          }}>
            <Search style={{ width: 15, height: 15, color: "var(--text-muted)" }} />
            <input
              type="text" placeholder="بحث بالمعرّف أو الطراز..."
              value={search} onChange={(e) => setSearch(e.target.value)}
              style={{ background: "transparent", border: "none", outline: "none", color: "var(--text-primary)", fontSize: 13, width: "100%" }}
            />
          </div>
          <div style={{ display: "flex", gap: 6 }}>
            {["ALL", "ACTIVE", "SUSPENDED", "REVOKED"].map((s) => (
              <button key={s} onClick={() => setFilterStatus(s)}
                className={`btn btn-sm ${filterStatus === s ? "btn-primary" : "btn-outline"}`} style={{ fontSize: 11 }}>
                {s === "ALL" ? "الكل" : s === "ACTIVE" ? "نشط" : s === "SUSPENDED" ? "معلق" : "ملغى"}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="content-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>المعرّف</th>
              <th>المحفظة</th>
              <th>الطراز</th>
              <th>مستوى الأمان</th>
              <th>الحالة</th>
              <th>العدّاد</th>
              <th>المعاملات</th>
              <th>آخر نشاط</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((dev) => (
              <tr key={dev.id}>
                <td className="mono" style={{ fontWeight: 600 }}>{dev.id}</td>
                <td><span className="badge info">{dev.walletId}</span></td>
                <td>{dev.deviceModel}</td>
                <td><AttestationBadge level={dev.attestationLevel} /></td>
                <td><StatusBadge status={dev.status} /></td>
                <td className="mono">{dev.ctr}</td>
                <td style={{ fontWeight: 600 }}>{dev.txCount}</td>
                <td style={{ fontSize: 11, color: "var(--text-muted)" }}>{dev.lastTxAt}</td>
                <td>
                  <button className="btn btn-outline btn-sm" style={{ padding: "4px 6px" }}>
                    <MoreVertical style={{ width: 13, height: 13 }} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
