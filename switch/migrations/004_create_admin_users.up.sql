-- إنشاء جدول المستخدمين الإداريين
-- يُرجى الرجوع إلى SPEC §4
CREATE TABLE admin_users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    totp_secret   VARCHAR(64),
    role          VARCHAR(20) NOT NULL,
    scope         VARCHAR(64) NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_admin_email UNIQUE (email),
    CONSTRAINT chk_admin_role CHECK (role IN ('SUPER_ADMIN', 'ADMIN', 'WALLET_ADMIN', 'VIEWER'))
);
