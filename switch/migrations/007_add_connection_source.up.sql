-- إضافة عمود مصدر الاتصال لتصنيف المعاملات حسب نقطة الوصول
-- carrier = شبكة شركة الاتصالات (بدون رسوم بيانات على المستخدم)
-- internet = الإنترنت العام

ALTER TABLE transactions
    ADD COLUMN connection_source VARCHAR(20) NOT NULL DEFAULT 'internet';

-- فهرس لتسريع استعلامات إحصائيات العمولات
CREATE INDEX idx_transactions_connection_source ON transactions (connection_source);
CREATE INDEX idx_transactions_wallet_source ON transactions (payer_wallet_id, connection_source);
