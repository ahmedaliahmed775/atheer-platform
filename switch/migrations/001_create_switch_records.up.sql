-- إنشاء جدول سجلات الدافعين
-- يُرجى الرجوع إلى SPEC §4 — التاجر لا يُسجّل في Atheer
CREATE TABLE switch_records (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id         VARCHAR(36) NOT NULL,
    wallet_id         VARCHAR(36) NOT NULL,
    device_id         VARCHAR(64) NOT NULL,
    seed_encrypted    BYTEA NOT NULL,
    seed_key_id       VARCHAR(64) NOT NULL,
    counter           BIGINT NOT NULL DEFAULT 0,
    payer_limit       BIGINT NOT NULL DEFAULT 5000,
    status            VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    user_type         VARCHAR(1) NOT NULL DEFAULT 'P',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_public_id UNIQUE (public_id),
    CONSTRAINT uq_device_id UNIQUE (device_id),
    CONSTRAINT chk_user_type CHECK (user_type = 'P'),
    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'SUSPENDED', 'REVOKED'))
);

-- فهارس لتسريع البحث
CREATE INDEX idx_switch_records_public_id ON switch_records(public_id);
CREATE INDEX idx_switch_records_device_id ON switch_records(device_id);
CREATE INDEX idx_switch_records_wallet_id ON switch_records(wallet_id);
