-- ============================================================
-- 002: Transactions — سجل المعاملات
-- ============================================================

CREATE TABLE IF NOT EXISTS transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_id             VARCHAR(64) UNIQUE NOT NULL,
    nonce             VARCHAR(64) UNIQUE NOT NULL,
    side_a_wallet_id  VARCHAR(32) NOT NULL,
    side_a_device_id  VARCHAR(64) NOT NULL,
    side_a_account_id VARCHAR(64) NOT NULL,
    side_b_wallet_id  VARCHAR(32) NOT NULL,
    side_b_device_id  VARCHAR(64) NOT NULL,
    side_b_account_id VARCHAR(64) NOT NULL,
    merchant_id       VARCHAR(64),
    operation_type    VARCHAR(16) NOT NULL,
    currency          VARCHAR(3) NOT NULL,
    amount            DECIMAL(15,2) NOT NULL,
    channel           VARCHAR(16) NOT NULL DEFAULT 'APN',
    status            VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    error_code        VARCHAR(8),
    error_message     TEXT,
    side_a_ctr        BIGINT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,

    CONSTRAINT chk_tx_op_type CHECK (operation_type IN ('P2P_SAME','P2M_SAME','P2M_CROSS','P2P_CROSS')),
    CONSTRAINT chk_tx_status CHECK (status IN ('PENDING','COMPLETED','FAILED','REVERSED','DISPUTED')),
    CONSTRAINT chk_tx_channel CHECK (channel IN ('APN','INTERNET'))
);

CREATE INDEX IF NOT EXISTS idx_tx_nonce ON transactions(nonce);
CREATE INDEX IF NOT EXISTS idx_tx_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_tx_side_a ON transactions(side_a_wallet_id, side_a_device_id);
CREATE INDEX IF NOT EXISTS idx_tx_side_b ON transactions(side_b_wallet_id, side_b_device_id);
CREATE INDEX IF NOT EXISTS idx_tx_created ON transactions(created_at DESC);
