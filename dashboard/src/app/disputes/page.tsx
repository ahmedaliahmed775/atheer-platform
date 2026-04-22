"use client";

import { useState } from "react";
import { AlertTriangle, MessageSquare, Clock, CheckCircle } from "lucide-react";

const mockDisputes = [
  { id: "DSP-001", txId: "TX-8F2A...E1D4", type: "AMOUNT_MISMATCH", status: "OPEN", priority: "HIGH", description: "العميل يؤكد خصم مبلغ مختلف عن المعروض", createdAt: "2026-04-22", assignedTo: "فريق التسوية" },
  { id: "DSP-002", txId: "TX-3C7B...A9F2", type: "DUPLICATE_CHARGE", status: "INVESTIGATING", priority: "CRITICAL", description: "خصم مزدوج بسبب انقطاع الشبكة أثناء المعالجة", createdAt: "2026-04-21", assignedTo: "فريق التقنية" },
  { id: "DSP-003", txId: "TX-1D5E...B3C8", type: "UNAUTHORIZED", status: "RESOLVED", priority: "MEDIUM", description: "العميل ينكر إجراء المعاملة — تم التحقق من التوقيعات", createdAt: "2026-04-20", assignedTo: "فريق الأمان" },
  { id: "DSP-004", txId: "TX-9A4F...D7E1", type: "SERVICE_NOT_RECEIVED", status: "OPEN", priority: "LOW", description: "تم الخصم لكن الخدمة لم تُقدم من التاجر", createdAt: "2026-04-19", assignedTo: "فريق التسوية" },
];

function PriorityBadge({ p }: { p: string }) {
  const m: Record<string, string> = { CRITICAL: "error", HIGH: "warning", MEDIUM: "info", LOW: "success" };
  return <span className={`badge ${m[p] || "info"}`}>{p}</span>;
}

function StatusBadge({ s }: { s: string }) {
  const m: Record<string, { c: string; l: string }> = {
    OPEN: { c: "warning", l: "مفتوح" },
    INVESTIGATING: { c: "info", l: "قيد التحقيق" },
    RESOLVED: { c: "success", l: "تم الحل" },
    CLOSED: { c: "purple", l: "مغلق" },
  };
  const st = m[s] || { c: "info", l: s };
  return <span className={`badge ${st.c}`}><span className="badge-dot" />{st.l}</span>;
}

export default function DisputesPage() {
  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>النزاعات</h2>
          <p>متابعة وحل النزاعات على المعاملات</p>
        </div>
        <div className="page-header-right">
          <button className="btn btn-primary btn-sm">
            <AlertTriangle style={{ width: 14, height: 14 }} />
            فتح نزاع جديد
          </button>
        </div>
      </div>

      <div className="stat-grid" style={{ gridTemplateColumns: "repeat(4, 1fr)" }}>
        <div className="stat-card amber">
          <div className="stat-card-value">2</div>
          <div className="stat-card-label">نزاعات مفتوحة</div>
        </div>
        <div className="stat-card blue">
          <div className="stat-card-value">1</div>
          <div className="stat-card-label">قيد التحقيق</div>
        </div>
        <div className="stat-card emerald">
          <div className="stat-card-value">1</div>
          <div className="stat-card-label">تم الحل</div>
        </div>
        <div className="stat-card purple">
          <div className="stat-card-value">48h</div>
          <div className="stat-card-label">متوسط وقت الحل</div>
        </div>
      </div>

      <div className="content-card">
        <table className="data-table">
          <thead>
            <tr>
              <th>المعرّف</th>
              <th>المعاملة</th>
              <th>النوع</th>
              <th>الأولوية</th>
              <th>الحالة</th>
              <th>الوصف</th>
              <th>المُكلف</th>
              <th>التاريخ</th>
            </tr>
          </thead>
          <tbody>
            {mockDisputes.map((d) => (
              <tr key={d.id}>
                <td className="mono" style={{ fontWeight: 600 }}>{d.id}</td>
                <td className="mono">{d.txId}</td>
                <td style={{ fontSize: 11 }}>{d.type}</td>
                <td><PriorityBadge p={d.priority} /></td>
                <td><StatusBadge s={d.status} /></td>
                <td style={{ fontSize: 12, maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{d.description}</td>
                <td style={{ fontSize: 12 }}>{d.assignedTo}</td>
                <td style={{ fontSize: 11, color: "var(--text-muted)" }}>{d.createdAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
