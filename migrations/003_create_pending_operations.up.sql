-- ============================================================
-- 003: Pending Operations — عمليات Saga المعلقة
-- ============================================================

CREATE TABLE IF NOT EXISTS pending_operations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_id         VARCHAR(64) NOT NULL REFERENCES transactions(tx_id),
    op_type       VARCHAR(16) NOT NULL,
    adapter_id    VARCHAR(32) NOT NULL,
    wallet_id     VARCHAR(32) NOT NULL,
    account_id    VARCHAR(64) NOT NULL,
    amount        DECIMAL(15,2) NOT NULL,
    status        VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    retry_count   INT NOT NULL DEFAULT 0,
    max_retries   INT NOT NULL DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,

    CONSTRAINT chk_pop_type CHECK (op_type IN ('DEBIT','CREDIT','REVERSAL')),
    CONSTRAINT chk_pop_status CHECK (status IN ('PENDING','DONE','FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_pop_tx ON pending_operations(tx_id);
CREATE INDEX IF NOT EXISTS idx_pop_status ON pending_operations(status);
