-- بيانات أولية لإعدادات المحافظ — جوالي وفلوسك
-- يُرجى الرجوع إلى SPEC §8.6

-- محفظة جوالي (Jawali) — محفظة يمنية رئيسية
INSERT INTO wallet_configs (wallet_id, base_url, api_key, secret, max_payer_limit, timeout_ms, max_retries, is_active)
VALUES (
    'jawali',
    'https://api.jawali.ye/v1',
    '',
    '',
    50000,
    10000,
    2,
    TRUE
);

-- محفظة فلوسك (Flousk) — محفظة يمنية إضافية
INSERT INTO wallet_configs (wallet_id, base_url, api_key, secret, max_payer_limit, timeout_ms, max_retries, is_active)
VALUES (
    'flousk',
    'https://api.flousk.ye/v1',
    '',
    '',
    50000,
    10000,
    2,
    TRUE
);
