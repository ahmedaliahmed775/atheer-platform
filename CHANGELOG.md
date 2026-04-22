# Changelog

All notable changes to this project will be documented in this file.

## [3.0.0] — 2026-04-23

### 🏗️ Foundation
- Go module with Chi, pgx, go-redis, Viper, decimal
- PostgreSQL 16 schema with 8 tables (devices, transactions, pending_operations, limits_matrix, attestation_records, disputes, audit_logs, channel_configs)
- Docker Compose with PostgreSQL + Redis + Switch + pgAdmin + Redis Commander
- Configuration management via Viper with environment variable support

### 🔐 Security & Crypto
- HMAC-SHA256 key derivation (DeriveLUK) matching SDK's SignatureUtils.kt
- ECDSA P-256 attestation verification for TEE/StrongBox
- Constant-time signature comparison (timing attack prevention)
- Redis Lua script for atomic anti-replay counter validation
- 11 unit tests covering all crypto operations

### 🔧 Transaction Pipeline (10 Layers)
- Layer 1: Tiered rate limiter (per device/wallet/IP)
- Layer 2: Structured request logger (sensitive data excluded)
- Layer 3: Anti-replay via Redis Lua atomic counter
- Layer 4: ECDSA attestation verifier
- Layer 5: HMAC Side A (payer) signature verification
- Layer 6: HMAC Side B (merchant) signature verification
- Layer 7: Cross-validator with 7 consistency rules
- Layer 8: Limits checker (matrix + daily totals)
- Layer 9: Idempotency via Redis nonce cache
- Layer 10: Transaction handler → Saga executor

### 💳 Payment Adapters
- PaymentAdapter interface with 7 methods (Debit, Credit, ReverseDebit, CheckBalance, GetTransactionStatus, SendSMS, GetLimits)
- JEEP adapter implementation
- WENET adapter implementation
- WASEL adapter implementation
- Adapter Registry for dynamic routing by walletId

### 🔄 Saga Pattern
- Saga service: Debit → Credit → Notify
- Automatic reversal (ReverseDebit) on Credit failure
- Pending operation tracking in database
- Best-effort SMS notifications via goroutine

### 📊 Dashboard (Next.js 16)
- Real-time dashboard with live data from Switch API (5s polling)
- 7 pages: Dashboard, Transactions, Devices, Limits, Disputes, Pipeline, Security, Settings
- Dark mode design with glassmorphism and gradients
- Interactive charts (Recharts): transaction volume, channel distribution
- Pipeline visualization with 10-layer pass/fail rates
- Connection status indicator (green/red)
- Arabic language support

### 🧪 Test Server
- Standalone test server (no PostgreSQL/Redis required)
- In-memory data store with 12 devices + 20 transactions
- Full API compatibility for dashboard integration testing
