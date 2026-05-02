-- إنشاء جدول إعدادات المحافظ
-- يُرجى الرجوع إلى SPEC §4
CREATE TABLE wallet_configs (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    wallet_id     VARCHAR(36) NOT NULL,
    base_url      TEXT NOT NULL,
    api_key       VARCHAR(128),
    secret        VARCHAR(256),
    max_payer_limit BIGINT NOT NULL DEFAULT 50000,
    timeout_ms    INTEGER NOT NULL DEFAULT 10000,
    max_retries   INTEGER NOT NULL DEFAULT 2,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_wallet_id UNIQUE (wallet_id)
);
