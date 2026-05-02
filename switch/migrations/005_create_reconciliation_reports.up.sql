-- إنشاء جدول تقارير التسوية
-- يُستخدم لتتبع حالات التسوية اليومية بين السويتش والمحافظ
CREATE TABLE reconciliation_reports (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    report_date     DATE NOT NULL,
    wallet_id       VARCHAR(36) NOT NULL,
    total_tx_count  INTEGER NOT NULL DEFAULT 0,
    total_amount    BIGINT NOT NULL DEFAULT 0,
    success_count   INTEGER NOT NULL DEFAULT 0,
    failed_count    INTEGER NOT NULL DEFAULT 0,
    disputed_count  INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_report_date_wallet UNIQUE (report_date, wallet_id),
    CONSTRAINT chk_report_status CHECK (status IN ('PENDING', 'VERIFIED', 'DISPUTED', 'RESOLVED'))
);

CREATE INDEX idx_reconciliation_date ON reconciliation_reports(report_date DESC);
CREATE INDEX idx_reconciliation_wallet ON reconciliation_reports(wallet_id);
