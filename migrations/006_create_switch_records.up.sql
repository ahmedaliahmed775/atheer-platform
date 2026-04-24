-- ============================================================
-- 006: Switch Records — سجلات المستخدمين في السويتش
-- حسب القسم 4 — المرحلة الأولى من الوثيقة المرجعية v3.0
-- السويتش هو المصدر الوحيد للحقيقة لـ UserType و WalletID
-- ============================================================

CREATE TABLE IF NOT EXISTS switch_records (
    public_id   VARCHAR(64) PRIMARY KEY,           -- معرّف عام غير مرتبط بهوية
    seed        BYTEA NOT NULL,                     -- البذرة التشفيرية (KMS-encrypted)
    user_id     VARCHAR(64) NOT NULL,               -- معرّف المستخدم في المحفظة
    user_type   VARCHAR(1) NOT NULL,                -- P | M — يحدده السويتش فقط
    wallet_id   VARCHAR(32) NOT NULL,               -- معرّف المحفظة
    counter     BIGINT NOT NULL DEFAULT 0,          -- العداد التصاعدي
    status      VARCHAR(16) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE | SUSPENDED
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_user_type CHECK (user_type IN ('P', 'M')),
    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'SUSPENDED'))
);

CREATE INDEX IF NOT EXISTS idx_switch_user_id ON switch_records(user_id);
CREATE INDEX IF NOT EXISTS idx_switch_wallet_id ON switch_records(wallet_id);
CREATE INDEX IF NOT EXISTS idx_switch_status ON switch_records(status);
