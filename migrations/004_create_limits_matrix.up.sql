-- ============================================================
-- 004: Limits Matrix — مصفوفة الحدود المركزية
-- ============================================================

CREATE TABLE IF NOT EXISTS limits_matrix (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id      VARCHAR(32) NOT NULL,
    operation_type VARCHAR(16) NOT NULL,
    currency       VARCHAR(3) NOT NULL,
    max_tx_amount  DECIMAL(15,2) NOT NULL,
    max_daily      DECIMAL(15,2) NOT NULL,
    max_monthly    DECIMAL(15,2),
    tier           VARCHAR(16) NOT NULL DEFAULT 'basic',
    is_active      BOOLEAN NOT NULL DEFAULT true,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(wallet_id, operation_type, currency, tier)
);

-- Seed default limits for JEEP wallet
INSERT INTO limits_matrix (wallet_id, operation_type, currency, max_tx_amount, max_daily, tier)
VALUES
    ('JEEP', 'P2P_SAME', 'YER', 500000.00, 2000000.00, 'basic'),
    ('JEEP', 'P2M_SAME', 'YER', 1000000.00, 5000000.00, 'basic'),
    ('JEEP', 'P2P_SAME', 'SAR', 5000.00, 20000.00, 'basic'),
    ('JEEP', 'P2M_SAME', 'SAR', 10000.00, 50000.00, 'basic'),
    ('JEEP', 'P2P_SAME', 'USD', 2000.00, 10000.00, 'basic'),
    ('JEEP', 'P2M_SAME', 'USD', 5000.00, 20000.00, 'basic')
ON CONFLICT DO NOTHING;
