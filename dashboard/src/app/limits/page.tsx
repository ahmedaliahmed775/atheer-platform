"use client";

import { useState } from "react";
import { Gauge, Save, RotateCcw, AlertTriangle } from "lucide-react";

const defaultLimits = [
  { wallet: "JEEP", opType: "P2P_SAME", currency: "YER", tier: "basic", maxTx: 50000, maxDaily: 500000, maxMonthly: 5000000 },
  { wallet: "JEEP", opType: "P2P_CROSS", currency: "YER", tier: "basic", maxTx: 30000, maxDaily: 200000, maxMonthly: 2000000 },
  { wallet: "JEEP", opType: "P2M_SAME", currency: "YER", tier: "basic", maxTx: 100000, maxDaily: 1000000, maxMonthly: 10000000 },
  { wallet: "JEEP", opType: "P2M_CROSS", currency: "YER", tier: "basic", maxTx: 50000, maxDaily: 500000, maxMonthly: 5000000 },
  { wallet: "WENET", opType: "P2P_SAME", currency: "YER", tier: "basic", maxTx: 40000, maxDaily: 400000, maxMonthly: 4000000 },
  { wallet: "WENET", opType: "P2M_SAME", currency: "YER", tier: "basic", maxTx: 80000, maxDaily: 800000, maxMonthly: 8000000 },
  { wallet: "WASEL", opType: "P2P_SAME", currency: "YER", tier: "basic", maxTx: 30000, maxDaily: 300000, maxMonthly: 3000000 },
  { wallet: "WASEL", opType: "P2M_SAME", currency: "YER", tier: "basic", maxTx: 60000, maxDaily: 600000, maxMonthly: 6000000 },
];

function formatNum(n: number) {
  return n.toLocaleString("en-US");
}

export default function LimitsPage() {
  const [limits, setLimits] = useState(defaultLimits);
  const [editIdx, setEditIdx] = useState<number | null>(null);
  const [saved, setSaved] = useState(false);

  const handleSave = () => {
    setSaved(true);
    setEditIdx(null);
    setTimeout(() => setSaved(false), 2000);
  };

  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>مصفوفة الحدود</h2>
          <p>إدارة حدود المعاملات حسب المحفظة ونوع العملية والعملة</p>
        </div>
        <div className="page-header-right">
          <button className="btn btn-outline btn-sm" onClick={() => setLimits(defaultLimits)}>
            <RotateCcw style={{ width: 14, height: 14 }} />
            إعادة تعيين
          </button>
          <button className="btn btn-primary btn-sm" onClick={handleSave}>
            <Save style={{ width: 14, height: 14 }} />
            {saved ? "✓ تم الحفظ" : "حفظ التغييرات"}
          </button>
        </div>
      </div>

      {/* Warning */}
      <div className="content-card" style={{
        marginBottom: 16,
        borderColor: "rgba(245, 158, 11, 0.3)",
        background: "rgba(245, 158, 11, 0.05)",
      }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <AlertTriangle style={{ width: 16, height: 16, color: "var(--accent-amber)", flexShrink: 0 }} />
          <span style={{ fontSize: 12.5, color: "var(--accent-amber)" }}>
            تغيير الحدود يسري فوراً على جميع المعاملات الجديدة. الحد النهائي = min(Switch, Adapter).
          </span>
        </div>
      </div>

      {/* Limits Table */}
      <div className="content-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>المحفظة</th>
              <th>نوع العملية</th>
              <th>العملة</th>
              <th>المستوى</th>
              <th>حد المعاملة الواحدة</th>
              <th>الحد اليومي</th>
              <th>الحد الشهري</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {limits.map((lim, idx) => (
              <tr key={idx}>
                <td><span className="badge info">{lim.wallet}</span></td>
                <td><span className="badge purple">{lim.opType}</span></td>
                <td className="mono">{lim.currency}</td>
                <td style={{ textTransform: "capitalize" }}>{lim.tier}</td>
                <td>
                  {editIdx === idx ? (
                    <input type="number" defaultValue={lim.maxTx}
                      onChange={(e) => {
                        const updated = [...limits];
                        updated[idx] = { ...updated[idx], maxTx: parseInt(e.target.value) || 0 };
                        setLimits(updated);
                      }}
                      style={{
                        background: "var(--bg-primary)", border: "1px solid var(--accent-blue)",
                        borderRadius: 6, padding: "4px 8px", color: "var(--text-primary)",
                        fontSize: 13, width: 100, outline: "none",
                      }}
                    />
                  ) : (
                    <span style={{ fontWeight: 600 }}>{formatNum(lim.maxTx)} <span style={{ fontSize: 10, color: "var(--text-muted)" }}>YER</span></span>
                  )}
                </td>
                <td>
                  {editIdx === idx ? (
                    <input type="number" defaultValue={lim.maxDaily}
                      onChange={(e) => {
                        const updated = [...limits];
                        updated[idx] = { ...updated[idx], maxDaily: parseInt(e.target.value) || 0 };
                        setLimits(updated);
                      }}
                      style={{
                        background: "var(--bg-primary)", border: "1px solid var(--accent-blue)",
                        borderRadius: 6, padding: "4px 8px", color: "var(--text-primary)",
                        fontSize: 13, width: 120, outline: "none",
                      }}
                    />
                  ) : (
                    <span style={{ fontWeight: 600 }}>{formatNum(lim.maxDaily)} <span style={{ fontSize: 10, color: "var(--text-muted)" }}>YER</span></span>
                  )}
                </td>
                <td style={{ fontWeight: 600 }}>
                  {formatNum(lim.maxMonthly)} <span style={{ fontSize: 10, color: "var(--text-muted)" }}>YER</span>
                </td>
                <td>
                  <button className="btn btn-outline btn-sm" style={{ padding: "4px 8px", fontSize: 11 }}
                    onClick={() => setEditIdx(editIdx === idx ? null : idx)}>
                    {editIdx === idx ? "تم" : "تعديل"}
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
