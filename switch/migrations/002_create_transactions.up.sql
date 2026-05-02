-- إنشاء جدول المعاملات
-- يُرجى الرجوع إلى SPEC §4
CREATE TABLE transactions (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transaction_id    UUID NOT NULL DEFAULT gen_random_uuid(),
    payer_public_id   VARCHAR(36) NOT NULL,
    merchant_id       VARCHAR(36) NOT NULL,
    payer_wallet_id   VARCHAR(36) NOT NULL,
    merchant_wallet_id VARCHAR(36) NOT NULL,
    amount            BIGINT NOT NULL,
    currency          VARCHAR(3) NOT NULL DEFAULT 'YER',
    counter           BIGINT NOT NULL,
    status            VARCHAR(20) NOT NULL,
    error_code        VARCHAR(50),
    duration_ms       INTEGER,
    debit_ref         VARCHAR(100),
    credit_ref        VARCHAR(100),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_transaction_id UNIQUE (transaction_id),
    CONSTRAINT chk_tx_status CHECK (status IN ('SUCCESS', 'FAILED', 'PENDING', 'REVERSED'))
);

-- فهارس لتسريع البحث والتصفية
CREATE INDEX idx_transactions_payer ON transactions(payer_public_id, created_at DESC);
CREATE INDEX idx_transactions_merchant ON transactions(merchant_id, created_at DESC);
CREATE INDEX idx_transactions_created ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);
