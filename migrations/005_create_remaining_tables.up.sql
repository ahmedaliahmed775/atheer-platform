-- ============================================================
-- 005: Attestation Records — سجلات التهيئة العتادية
-- ============================================================

CREATE TABLE IF NOT EXISTS attestation_records (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id             VARCHAR(64) NOT NULL,
    certificate_chain     TEXT[] NOT NULL,
    play_integrity_token  TEXT NOT NULL,
    attestation_level     VARCHAR(16) NOT NULL,
    is_verified           BOOLEAN NOT NULL DEFAULT false,
    verified_at           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_att_device ON attestation_records(device_id);

-- ============================================================
-- 006: Disputes — النزاعات
-- ============================================================

CREATE TABLE IF NOT EXISTS disputes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_id       VARCHAR(64) NOT NULL,
    reason      TEXT NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'OPEN',
    opened_by   VARCHAR(64) NOT NULL,
    resolved_by VARCHAR(64),
    resolution  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,

    CONSTRAINT chk_dispute_status CHECK (status IN ('OPEN','INVESTIGATING','RESOLVED','REJECTED'))
);

CREATE INDEX IF NOT EXISTS idx_dispute_status ON disputes(status);
CREATE INDEX IF NOT EXISTS idx_dispute_tx ON disputes(tx_id);

-- ============================================================
-- 007: Audit Logs — سجلات التدقيق
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action        VARCHAR(32) NOT NULL,
    actor         VARCHAR(64) NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    resource_id   VARCHAR(64),
    details       JSONB,
    ip_address    INET,
    channel       VARCHAR(16),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id);

-- ============================================================
-- 008: Channel Configs — إعدادات القنوات
-- ============================================================

CREATE TABLE IF NOT EXISTS channel_configs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_type      VARCHAR(16) NOT NULL UNIQUE,
    endpoint_url      TEXT NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    rate_limit        INT NOT NULL DEFAULT 100,
    timeout_ms        INT NOT NULL DEFAULT 5000,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO channel_configs (channel_type, endpoint_url)
VALUES
    ('APN', 'https://apn.atheer.local'),
    ('INTERNET', 'https://api.atheer.local')
ON CONFLICT DO NOTHING;
