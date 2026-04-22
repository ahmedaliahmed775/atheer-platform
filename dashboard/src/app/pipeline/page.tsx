"use client";

import {
  Layers,
  Shield,
  Lock,
  Fingerprint,
  Key,
  GitCompare,
  Gauge,
  Hash,
  CheckCircle,
  XCircle,
  Zap,
  ArrowRight,
} from "lucide-react";

const pipelineLayers = [
  {
    num: 1, name: "Rate Limiter", icon: Zap,
    desc: "حماية ضد الفيض — حد لكل جهاز/محفظة/IP",
    pass: 12470, fail: 23, avgMs: 0.5,
    config: { perDevice: 10, perWallet: 100, perIP: 50, window: "1 min" },
  },
  {
    num: 2, name: "Request Logger", icon: Hash,
    desc: "تسجيل بيانات الطلب بدون بيانات مالية حساسة",
    pass: 12447, fail: 0, avgMs: 0.1,
    config: { format: "JSON structured", sensitive: "EXCLUDED" },
  },
  {
    num: 3, name: "Anti-Replay", icon: Lock,
    desc: "Redis Lua — رفض العدادات غير المتزايدة",
    pass: 12447, fail: 8, avgMs: 1.2,
    config: { store: "Redis Lua Script", rule: "ctr(new) > ctr(stored)", ttl: "24h" },
  },
  {
    num: 4, name: "Attestation (ECDSA)", icon: Fingerprint,
    desc: "التحقق من توقيع TEE/StrongBox — P-256",
    pass: 12439, fail: 12, avgMs: 3.5,
    config: { algorithm: "ECDSA P-256", source: "TEE/StrongBox", comparison: "timing-safe" },
  },
  {
    num: 5, name: "HMAC Side A", icon: Key,
    desc: "التحقق من توقيع المرسل — LUK = HMAC(seed, ctr)",
    pass: 12427, fail: 5, avgMs: 2.1,
    config: { algorithm: "HMAC-SHA256", key: "LUK = DeriveLUK(seed, ctr)", comparison: "constant-time" },
  },
  {
    num: 6, name: "HMAC Side B", icon: Key,
    desc: "التحقق من توقيع المستقبل — نفس الخوارزمية",
    pass: 12422, fail: 3, avgMs: 2.0,
    config: { algorithm: "HMAC-SHA256", side: "Receiver", comparison: "constant-time" },
  },
  {
    num: 7, name: "Cross-Validator", icon: GitCompare,
    desc: "7 قواعد تحقق بين Side A و Side B",
    pass: 12419, fail: 15, avgMs: 0.8,
    config: { rules: 7, checks: "currency, opType, timestamp, amount, walletId, merchantId, channel" },
  },
  {
    num: 8, name: "Limits Checker", icon: Gauge,
    desc: "فحص الحد الأقصى = min(Matrix, Adapter)",
    pass: 12404, fail: 7, avgMs: 4.2,
    config: { source: "DB + Adapter API", daily: "SUM(today)", formula: "min(Switch, Adapter)" },
  },
  {
    num: 9, name: "Idempotency", icon: Shield,
    desc: "Redis nonce cache — يمنع الخصم المزدوج",
    pass: 12397, fail: 0, avgMs: 0.9,
    config: { store: "Redis", key: "nonce", ttl: "3600s", onHit: "return cached result" },
  },
  {
    num: 10, name: "Saga Executor", icon: Layers,
    desc: "Debit → Credit → Notify مع تراجع تلقائي",
    pass: 12397, fail: 2, avgMs: 45.0,
    config: { pattern: "Saga", steps: "Debit → Credit → Notify", compensation: "ReverseDebit" },
  },
];

export default function PipelinePage() {
  const totalRequests = pipelineLayers[0].pass + pipelineLayers[0].fail;
  const successRate = ((pipelineLayers[9].pass / totalRequests) * 100).toFixed(2);

  return (
    <>
      <div className="page-header">
        <div className="page-header-left">
          <h2>Transaction Pipeline</h2>
          <p>عرض مفصّل للـ 10 طبقات ومؤشرات أدائها في الوقت الحقيقي</p>
        </div>
        <div className="page-header-right">
          <div style={{
            background: "rgba(16, 185, 129, 0.1)", border: "1px solid rgba(16, 185, 129, 0.2)",
            borderRadius: 10, padding: "8px 16px", display: "flex", alignItems: "center", gap: 8,
          }}>
            <CheckCircle style={{ width: 16, height: 16, color: "var(--accent-emerald)" }} />
            <span style={{ fontSize: 13, fontWeight: 700, color: "var(--accent-emerald)" }}>
              {successRate}% نسبة النجاح الكلية
            </span>
          </div>
        </div>
      </div>

      {/* Pipeline Flow Visualization */}
      <div className="content-card" style={{ marginBottom: 16 }}>
        <h3 style={{ fontSize: 13, marginBottom: 12, color: "var(--text-muted)" }}>تدفق المعالجة</h3>
        <div className="pipeline-flow">
          {pipelineLayers.map((layer, i) => (
            <div key={layer.num} style={{ display: "flex", alignItems: "center", gap: 4 }}>
              <div className={`pipeline-node ${layer.fail === 0 ? "pass" : layer.fail > 10 ? "fail" : "pending"}`}>
                <layer.icon style={{ width: 12, height: 12 }} />
                {layer.num}
              </div>
              {i < pipelineLayers.length - 1 && (
                <ArrowRight className="pipeline-arrow" style={{ width: 12, height: 12 }} />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Layer Cards */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 12 }}>
        {pipelineLayers.map((layer) => {
          const total = layer.pass + layer.fail;
          const pct = total > 0 ? ((layer.pass / total) * 100).toFixed(2) : "100.00";
          return (
            <div key={layer.num} className="content-card" style={{ padding: 16 }}>
              <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 12 }}>
                <div style={{
                  width: 32, height: 32, borderRadius: 8,
                  background: layer.fail === 0 ? "rgba(16, 185, 129, 0.12)" : "rgba(245, 158, 11, 0.12)",
                  display: "flex", alignItems: "center", justifyContent: "center",
                  color: layer.fail === 0 ? "var(--accent-emerald)" : "var(--accent-amber)",
                }}>
                  <layer.icon style={{ width: 16, height: 16 }} />
                </div>
                <div>
                  <div style={{ fontSize: 13, fontWeight: 700 }}>Layer {layer.num}: {layer.name}</div>
                  <div style={{ fontSize: 11, color: "var(--text-muted)" }}>{layer.desc}</div>
                </div>
                <div style={{ marginLeft: "auto", textAlign: "right" }}>
                  <div style={{ fontSize: 16, fontWeight: 800, color: parseFloat(pct) === 100 ? "var(--accent-emerald)" : "var(--text-primary)" }}>
                    {pct}%
                  </div>
                  <div style={{ fontSize: 10, color: "var(--text-muted)" }}>{layer.avgMs}ms avg</div>
                </div>
              </div>

              {/* Progress bar */}
              <div style={{ height: 4, background: "var(--border-color)", borderRadius: 2, marginBottom: 10, overflow: "hidden" }}>
                <div style={{
                  width: `${pct}%`, height: "100%", borderRadius: 2,
                  background: layer.fail === 0 ? "var(--accent-emerald)" : "var(--accent-amber)",
                }} />
              </div>

              {/* Stats */}
              <div style={{ display: "flex", gap: 16, fontSize: 11 }}>
                <span style={{ color: "var(--accent-emerald)" }}>
                  <CheckCircle style={{ width: 11, height: 11, display: "inline", verticalAlign: "middle" }} /> {layer.pass.toLocaleString()} نجاح
                </span>
                <span style={{ color: layer.fail > 0 ? "var(--accent-red)" : "var(--text-muted)" }}>
                  <XCircle style={{ width: 11, height: 11, display: "inline", verticalAlign: "middle" }} /> {layer.fail} رفض
                </span>
              </div>

              {/* Config */}
              <div style={{
                marginTop: 10, padding: "8px 10px", background: "var(--bg-primary)",
                borderRadius: 6, fontSize: 11, color: "var(--text-muted)",
                fontFamily: "'IBM Plex Mono', monospace",
              }}>
                {Object.entries(layer.config).map(([k, v]) => (
                  <div key={k}><span style={{ color: "var(--accent-blue)" }}>{k}:</span> {v}</div>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </>
  );
}
