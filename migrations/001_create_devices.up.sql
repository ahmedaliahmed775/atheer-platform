-- ============================================================
-- 001: Devices — الأجهزة المسّجلة مع التهيئة العتادية
-- ============================================================

CREATE TABLE IF NOT EXISTS devices (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id               VARCHAR(64) UNIQUE NOT NULL,
    wallet_id               VARCHAR(32) NOT NULL,
    account_id              VARCHAR(64) NOT NULL,
    device_seed             BYTEA NOT NULL,
    ctr                     BIGINT NOT NULL DEFAULT 0,
    ec_public_key           TEXT NOT NULL,
    attestation_public_key  TEXT NOT NULL,
    attestation_level       VARCHAR(16) NOT NULL DEFAULT 'TEE',
    status                  VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    enrolled_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_tx_at              TIMESTAMPTZ,

    CONSTRAINT chk_device_status CHECK (status IN ('ACTIVE','SUSPENDED','REVOKED')),
    CONSTRAINT chk_attestation_level CHECK (attestation_level IN ('TEE','STRONGBOX','SOFTWARE'))
);

CREATE INDEX IF NOT EXISTS idx_devices_wallet ON devices(wallet_id);
CREATE INDEX IF NOT EXISTS idx_devices_wallet_account ON devices(wallet_id, account_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
