"use client";

import { useEffect, useState, useCallback } from "react";
import {
  ArrowLeftRight,
  Smartphone,
  TrendingUp,
  Zap,
  ChevronRight,
  ArrowUpRight,
  ArrowDownRight,
  Shield,
  CheckCircle,
  XCircle,
  Clock,
  RefreshCw,
  Wifi,
  WifiOff,
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  Cell,
} from "recharts";

const API = "http://localhost:8080/api/v2";

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { cls: string; label: string }> = {
    COMPLETED: { cls: "success", label: "مكتمل" },
    PENDING: { cls: "warning", label: "قيد المعالجة" },
    FAILED: { cls: "error", label: "فشل" },
    REVERSED: { cls: "purple", label: "مُعكوس" },
  };
  const s = map[status] || { cls: "info", label: status };
  return (
    <span className={`badge ${s.cls}`}>
      <span className="badge-dot" />
      {s.label}
    </span>
  );
}

export default function DashboardPage() {
  const [stats, setStats] = useState<any>(null);
  const [transactions, setTransactions] = useState<any[]>([]);
  const [pipeline, setPipeline] = useState<any[]>([]);
  const [channels, setChannels] = useState<any[]>([]);
  const [volume, setVolume] = useState<any[]>([]);
  const [connected, setConnected] = useState(false);
  const [lastUpdate, setLastUpdate] = useState("");

  const fetchData = useCallback(async () => {
    try {
      const [statsRes, txRes, pipeRes, chanRes, volRes] = await Promise.all([
        fetch(`${API}/stats`).then(r => r.json()),
        fetch(`${API}/transaction`).then(r => r.json()),
        fetch(`${API}/stats/pipeline`).then(r => r.json()),
        fetch(`${API}/stats/channels`).then(r => r.json()),
        fetch(`${API}/stats/volume`).then(r => r.json()),
      ]);
      setStats(statsRes);
      setTransactions(txRes.transactions?.slice(0, 6) || []);
      setPipeline(pipeRes.layers || []);
      
      const chanData = chanRes.channels || {};
      setChannels([
        { name: "JEEP", count: chanData.JEEP || 0, color: "#3b82f6" },
        { name: "WENET", count: chanData.WENET || 0, color: "#8b5cf6" },
        { name: "WASEL", count: chanData.WASEL || 0, color: "#06b6d4" },
      ]);
      
      setVolume(volRes.hours || []);
      setConnected(true);
      setLastUpdate(new Date().toLocaleTimeString("ar-SA"));
    } catch {
      setConnected(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [fetchData]);

  return (
    <>
      {/* Connection Banner */}
      {connected ? (
        <div style={{
          display: "flex", alignItems: "center", gap: 8, padding: "8px 14px",
          background: "rgba(16, 185, 129, 0.08)", border: "1px solid rgba(16, 185, 129, 0.2)",
          borderRadius: 10, marginBottom: 16, fontSize: 12, color: "var(--accent-emerald)",
        }}>
          <Wifi style={{ width: 14, height: 14 }} />
          متصل بـ Atheer Switch (localhost:8080) — بيانات حية • آخر تحديث: {lastUpdate}
        </div>
      ) : (
        <div style={{
          display: "flex", alignItems: "center", gap: 8, padding: "8px 14px",
          background: "rgba(239, 68, 68, 0.08)", border: "1px solid rgba(239, 68, 68, 0.2)",
          borderRadius: 10, marginBottom: 16, fontSize: 12, color: "var(--accent-red)",
        }}>
          <WifiOff style={{ width: 14, height: 14 }} />
          غير متصل بالسويتش — تأكد من تشغيل الخادم على المنفذ 8080
        </div>
      )}

      {/* Page Header */}
      <div className="page-header">
        <div className="page-header-left">
          <h2>لوحة المؤشرات</h2>
          <p>بيانات حية من Atheer Switch — تحديث تلقائي كل 5 ثوانٍ</p>
        </div>
        <div className="page-header-right">
          <button className="btn btn-outline btn-sm" onClick={fetchData}>
            <RefreshCw style={{ width: 14, height: 14 }} />
            تحديث
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="stat-grid">
        <div className="stat-card blue">
          <div className="stat-card-header">
            <div className="stat-card-icon blue"><ArrowLeftRight style={{ width: 18, height: 18 }} /></div>
            <span className="stat-card-change up"><ArrowUpRight style={{ width: 12, height: 12 }} /> live</span>
          </div>
          <div className="stat-card-value">{stats?.totalTx ?? "—"}</div>
          <div className="stat-card-label">المعاملات ({stats?.completedTx ?? 0} مكتمل / {stats?.failedTx ?? 0} فشل)</div>
        </div>

        <div className="stat-card emerald">
          <div className="stat-card-header">
            <div className="stat-card-icon emerald"><TrendingUp style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats?.totalAmount ? Number(stats.totalAmount).toLocaleString("en", { maximumFractionDigits: 0 }) : "—"}</div>
          <div className="stat-card-label">إجمالي القيمة ({stats?.currency || "YER"})</div>
        </div>

        <div className="stat-card amber">
          <div className="stat-card-header">
            <div className="stat-card-icon amber"><Smartphone style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats?.activeDevices ?? "—"} / {stats?.totalDevices ?? "—"}</div>
          <div className="stat-card-label">أجهزة نشطة / إجمالي</div>
        </div>

        <div className="stat-card purple">
          <div className="stat-card-header">
            <div className="stat-card-icon purple"><Zap style={{ width: 18, height: 18 }} /></div>
          </div>
          <div className="stat-card-value">{stats?.avgLatencyMs ?? "—"}ms</div>
          <div className="stat-card-label">متوسط زمن الاستجابة</div>
        </div>
      </div>

      {/* Charts */}
      <div className="content-grid">
        <div className="content-card">
          <div className="content-card-header">
            <h3>حجم المعاملات — 24 ساعة (حي)</h3>
          </div>
          <div style={{ height: 260 }}>
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={volume}>
                <defs>
                  <linearGradient id="blueGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
                <XAxis dataKey="hour" stroke="#64748b" fontSize={11} />
                <YAxis stroke="#64748b" fontSize={11} />
                <Tooltip contentStyle={{ background: "#1a2236", border: "1px solid #1e293b", borderRadius: 8, fontSize: 12 }} />
                <Area type="monotone" dataKey="volume" stroke="#3b82f6" strokeWidth={2} fill="url(#blueGrad)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="content-card">
          <div className="content-card-header"><h3>توزيع القنوات (حي)</h3></div>
          <div style={{ height: 260 }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={channels} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" horizontal={false} />
                <XAxis type="number" stroke="#64748b" fontSize={11} />
                <YAxis type="category" dataKey="name" stroke="#64748b" fontSize={12} width={60} />
                <Tooltip contentStyle={{ background: "#1a2236", border: "1px solid #1e293b", borderRadius: 8, fontSize: 12 }} />
                <Bar dataKey="count" radius={[0, 6, 6, 0]} barSize={24}>
                  {channels.map((entry, idx) => (
                    <Cell key={idx} fill={entry.color} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Transactions + Pipeline */}
      <div className="content-grid">
        <div className="content-card">
          <div className="content-card-header">
            <h3>آخر المعاملات (من السويتش)</h3>
            <span className="badge success"><span className="badge-dot" /> حي</span>
          </div>
          <table className="data-table">
            <thead>
              <tr>
                <th>المعرّف</th>
                <th>النوع</th>
                <th>المبلغ</th>
                <th>القناة</th>
                <th>الحالة</th>
                <th>زمن</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((tx: any) => (
                <tr key={tx.id}>
                  <td className="mono" style={{ fontSize: 11 }}>{tx.id}</td>
                  <td><span className="badge info" style={{ fontSize: 10 }}>{tx.operationType}</span></td>
                  <td style={{ fontWeight: 600 }}>{Number(tx.amount).toLocaleString("en", { maximumFractionDigits: 2 })} <span style={{ fontSize: 10, color: "var(--text-muted)" }}>{tx.currency}</span></td>
                  <td>{tx.channel}</td>
                  <td><StatusBadge status={tx.status} /></td>
                  <td className="mono" style={{ color: tx.latencyMs < 50 ? "var(--accent-emerald)" : "var(--accent-amber)" }}>{tx.latencyMs}ms</td>
                </tr>
              ))}
              {transactions.length === 0 && (
                <tr><td colSpan={6} style={{ textAlign: "center", color: "var(--text-muted)", padding: 20 }}>لا توجد معاملات</td></tr>
              )}
            </tbody>
          </table>
        </div>

        <div className="content-card">
          <div className="content-card-header">
            <h3>Pipeline (من السويتش)</h3>
            <Shield style={{ width: 16, height: 16, color: "var(--accent-emerald)" }} />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {pipeline.map((layer: any) => {
              const total = layer.pass + layer.fail;
              const pct = total > 0 ? ((layer.pass / total) * 100).toFixed(1) : "100.0";
              return (
                <div key={layer.num} style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <span style={{ fontSize: 11, color: "var(--text-muted)", width: 90, flexShrink: 0 }}>
                    {layer.num}. {layer.name}
                  </span>
                  <div style={{ flex: 1, height: 6, background: "var(--border-color)", borderRadius: 3, overflow: "hidden" }}>
                    <div style={{
                      width: `${pct}%`, height: "100%", borderRadius: 3,
                      background: layer.fail === 0 ? "var(--accent-emerald)" : "var(--accent-amber)",
                    }} />
                  </div>
                  <span style={{ fontSize: 11, fontWeight: 600, color: "var(--text-secondary)", width: 45, textAlign: "right" }}>
                    {pct}%
                  </span>
                  {layer.fail > 0 ? (
                    <XCircle style={{ width: 13, height: 13, color: "var(--accent-red)" }} />
                  ) : (
                    <CheckCircle style={{ width: 13, height: 13, color: "var(--accent-emerald)" }} />
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </>
  );
}
