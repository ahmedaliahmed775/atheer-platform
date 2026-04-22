"use client";

import { useState } from "react";
import {
  ArrowLeftRight,
  Search,
  Filter,
  Download,
  ChevronLeft,
  ChevronRight,
  Eye,
  RefreshCw,
} from "lucide-react";

const mockTransactions = Array.from({ length: 25 }, (_, i) => ({
  id: `TX-${Math.random().toString(36).substr(2, 8).toUpperCase()}`,
  sideAWallet: "JEEP",
  sideADevice: `DEV-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
  sideBWallet: i % 3 === 0 ? "WENET" : i % 3 === 1 ? "WASEL" : "JEEP",
  sideBDevice: `DEV-${Math.random().toString(36).substr(2, 6).toUpperCase()}`,
  operationType: ["P2P_SAME", "P2M_SAME", "P2P_CROSS", "P2M_CROSS"][i % 4],
  amount: (Math.random() * 50000 + 100).toFixed(2),
  currency: "YER",
  status: ["COMPLETED", "COMPLETED", "COMPLETED", "PENDING", "FAILED", "REVERSED"][i % 6],
  channel: ["JEEP", "WENET", "WASEL"][i % 3],
  ctr: (i + 1) * 3,
  nonce: `${Math.random().toString(36).substr(2, 8)}-${Math.random().toString(36).substr(2, 4)}`,
  createdAt: new Date(Date.now() - i * 300000).toLocaleString("ar-SA"),
  latency: Math.floor(Math.random() * 80 + 20),
}));

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

export default function TransactionsPage() {
  const [search, setSearch] = useState("");
  const [filterStatus, setFilterStatus] = useState("ALL");
  const [page, setPage] = useState(1);
  const perPage = 10;

  const filtered = mockTransactions.filter((tx) => {
    if (filterStatus !== "ALL" && tx.status !== filterStatus) return false;
    if (search && !tx.id.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  });

  const totalPages = Math.ceil(filtered.length / perPage);
  const paginated = filtered.slice((page - 1) * perPage, page * perPage);

  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>المعاملات</h2>
          <p>إدارة ومراقبة جميع المعاملات عبر الـ Pipeline</p>
        </div>
        <div className="page-header-right">
          <button className="btn btn-outline btn-sm">
            <Download style={{ width: 14, height: 14 }} />
            تصدير CSV
          </button>
          <button className="btn btn-primary btn-sm">
            <RefreshCw style={{ width: 14, height: 14 }} />
            تحديث
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="content-card" style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
          <div style={{
            display: "flex", alignItems: "center", gap: 8,
            background: "var(--bg-primary)", borderRadius: 8, padding: "6px 12px",
            border: "1px solid var(--border-color)", flex: 1, minWidth: 200, maxWidth: 320,
          }}>
            <Search style={{ width: 15, height: 15, color: "var(--text-muted)" }} />
            <input
              type="text"
              placeholder="بحث بمعرّف المعاملة..."
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              style={{
                background: "transparent", border: "none", outline: "none",
                color: "var(--text-primary)", fontSize: 13, width: "100%",
              }}
            />
          </div>

          <div style={{ display: "flex", gap: 6 }}>
            {["ALL", "COMPLETED", "PENDING", "FAILED", "REVERSED"].map((s) => (
              <button
                key={s}
                onClick={() => { setFilterStatus(s); setPage(1); }}
                className={`btn btn-sm ${filterStatus === s ? "btn-primary" : "btn-outline"}`}
                style={{ fontSize: 11 }}
              >
                {s === "ALL" ? "الكل" : s === "COMPLETED" ? "مكتمل" : s === "PENDING" ? "معلق" : s === "FAILED" ? "فشل" : "معكوس"}
              </button>
            ))}
          </div>

          <span style={{ marginLeft: "auto", fontSize: 12, color: "var(--text-muted)" }}>
            {filtered.length} معاملة
          </span>
        </div>
      </div>

      {/* Table */}
      <div className="content-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>المعرّف</th>
              <th>النوع</th>
              <th>Side A</th>
              <th>Side B</th>
              <th>المبلغ</th>
              <th>CTR</th>
              <th>الحالة</th>
              <th>زمن الاستجابة</th>
              <th>التاريخ</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {paginated.map((tx) => (
              <tr key={tx.id}>
                <td className="mono" style={{ fontWeight: 600 }}>{tx.id}</td>
                <td><span className="badge info">{tx.operationType}</span></td>
                <td style={{ fontSize: 12 }}>
                  <div>{tx.sideAWallet}</div>
                  <div className="mono" style={{ color: "var(--text-muted)", fontSize: 10 }}>{tx.sideADevice}</div>
                </td>
                <td style={{ fontSize: 12 }}>
                  <div>{tx.sideBWallet}</div>
                  <div className="mono" style={{ color: "var(--text-muted)", fontSize: 10 }}>{tx.sideBDevice}</div>
                </td>
                <td style={{ fontWeight: 700 }}>{tx.amount} <span style={{ fontSize: 10, color: "var(--text-muted)" }}>{tx.currency}</span></td>
                <td className="mono">{tx.ctr}</td>
                <td><StatusBadge status={tx.status} /></td>
                <td className="mono" style={{ color: parseInt(tx.latency.toString()) < 50 ? "var(--accent-emerald)" : "var(--accent-amber)" }}>
                  {tx.latency}ms
                </td>
                <td style={{ fontSize: 11, color: "var(--text-muted)" }}>{tx.createdAt}</td>
                <td>
                  <button className="btn btn-outline btn-sm" style={{ padding: "4px 6px" }}>
                    <Eye style={{ width: 13, height: 13 }} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* Pagination */}
        <div style={{
          display: "flex", alignItems: "center", justifyContent: "space-between",
          padding: "16px 0 0", borderTop: "1px solid var(--border-color)", marginTop: 8,
        }}>
          <span style={{ fontSize: 12, color: "var(--text-muted)" }}>
            صفحة {page} من {totalPages}
          </span>
          <div style={{ display: "flex", gap: 6 }}>
            <button className="btn btn-outline btn-sm" onClick={() => setPage(Math.max(1, page - 1))} disabled={page === 1}>
              <ChevronRight style={{ width: 14, height: 14 }} />
            </button>
            <button className="btn btn-outline btn-sm" onClick={() => setPage(Math.min(totalPages, page + 1))} disabled={page === totalPages}>
              <ChevronLeft style={{ width: 14, height: 14 }} />
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
