-- إزالة عمود مصدر الاتصال
DROP INDEX IF EXISTS idx_transactions_wallet_source;
DROP INDEX IF EXISTS idx_transactions_connection_source;
ALTER TABLE transactions DROP COLUMN IF EXISTS connection_source;
