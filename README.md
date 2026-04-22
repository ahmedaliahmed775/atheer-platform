<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/Next.js-16-black?style=for-the-badge&logo=next.js" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql&logoColor=white" />
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" />
</p>

# 🏗️ Atheer Switch — NFC Payment Platform

> **Modular Monolith** backend for the Atheer SDK V3.0 — a secure, NFC-based mobile payment system supporting P2P and P2M transactions across multiple wallet providers.

---

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Transaction Pipeline](#transaction-pipeline)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Dashboard](#dashboard)
- [Security](#security)
- [Testing](#testing)
- [Configuration](#configuration)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

Atheer Switch is the central payment processing engine that:

- **Processes NFC tap-to-pay transactions** between mobile devices
- **Enforces a 10-layer security pipeline** (rate limiting → HMAC verification → Saga execution)
- **Supports 3 wallet providers**: JEEP, WENET, WASEL
- **Implements the Saga pattern** for reliable cross-wallet transactions with automatic compensation
- **Provides a real-time dashboard** for monitoring and administration

### Transaction Flow

```
┌──────────┐    NFC Tap    ┌──────────┐
│  Payer   │──────────────▶│ Merchant │
│ (SDK)    │               │ (SDK)    │
└────┬─────┘               └────┬─────┘
     │ Side A                   │ Side B
     │ Payload                  │ Payload
     └──────────┬───────────────┘
                ▼
     ┌─────────────────────┐
     │   Atheer Switch     │
     │   10-Layer Pipeline │
     │                     │
     │ 1. Rate Limiter     │
     │ 2. Request Logger   │
     │ 3. Anti-Replay      │
     │ 4. Attestation      │
     │ 5. HMAC Side A      │
     │ 6. HMAC Side B      │
     │ 7. Cross-Validator  │
     │ 8. Limits Checker   │
     │ 9. Idempotency      │
     │ 10. Saga Executor   │
     └─────────┬───────────┘
               ▼
     ┌─────────────────────┐
     │  Payment Adapters   │
     │  ┌─────┬─────┬────┐ │
     │  │JEEP │WENET│WASEL│ │
     │  └─────┴─────┴────┘ │
     └─────────────────────┘
```

---

## Architecture

The platform follows a **Modular Monolith** pattern:

```
┌─────────────────────────────────────────────────────────┐
│                      API Layer                           │
│              Chi Router + Middleware Pipeline             │
├──────────────┬──────────────┬───────────────────────────┤
│   Handlers   │   Services   │        Adapters            │
│  (HTTP I/O)  │  (Business)  │   (JEEP/WENET/WASEL)      │
├──────────────┴──────────────┴───────────────────────────┤
│                   Repository Layer                        │
│            (PostgreSQL via pgx + Redis)                   │
├─────────────────────────────────────────────────────────┤
│                    Crypto Layer                           │
│          HMAC-SHA256 + ECDSA P-256 + Anti-Replay         │
└─────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Modular Monolith | Faster iteration than microservices, easy to split later |
| Pipeline Pattern | Each security layer is an independent, testable middleware |
| Saga Pattern | Guarantees atomicity for cross-wallet Debit→Credit→Notify |
| Constant-time comparison | Prevents timing attacks on HMAC verification |
| Redis Lua scripts | Atomic anti-replay counter checks |

---

## Transaction Pipeline

Every transaction passes through **10 security layers** before execution:

| Layer | Name | Purpose | Implementation |
|-------|------|---------|----------------|
| 1 | **Rate Limiter** | Flood protection per device/wallet/IP | Redis sliding window |
| 2 | **Request Logger** | Structured audit logging (no sensitive data) | slog JSON |
| 3 | **Anti-Replay** | Reject duplicate/replayed counters | Redis Lua (atomic) |
| 4 | **Attestation** | Verify TEE/StrongBox ECDSA signature | ECDSA P-256 |
| 5 | **HMAC Side A** | Verify payer's signature | HMAC-SHA256 + LUK |
| 6 | **HMAC Side B** | Verify merchant's signature | HMAC-SHA256 + LUK |
| 7 | **Cross-Validator** | 7 consistency rules between Side A & B | In-memory rules |
| 8 | **Limits Checker** | Transaction/daily/monthly limit enforcement | DB + Adapter query |
| 9 | **Idempotency** | Prevent double-charging via nonce cache | Redis TTL |
| 10 | **Saga Executor** | Debit → Credit → Notify with auto-reversal | Saga pattern |

### LUK (Limited Use Key) Derivation

```
LUK = HMAC-SHA256(deviceSeed, BigEndian(counter))
Signature = HMAC-SHA256(LUK, PayloadHash)
PayloadHash = SHA256(walletId|deviceId|ctr|opType|currency|amount|nonce|timestamp)
```

This matches the SDK's `SignatureUtils.kt` implementation exactly.

---

## Tech Stack

### Backend
- **Language**: Go 1.26
- **Router**: Chi v5
- **Database**: PostgreSQL 16 (via pgxpool)
- **Cache**: Redis 7 (via go-redis)
- **Config**: Viper
- **Decimal**: shopspring/decimal
- **UUID**: google/uuid

### Dashboard
- **Framework**: Next.js 16 (App Router + Turbopack)
- **Language**: TypeScript
- **Styling**: Tailwind CSS 4
- **Charts**: Recharts
- **Icons**: Lucide React

### Infrastructure
- **Containers**: Docker + Docker Compose
- **DB Admin**: pgAdmin 4
- **Redis Admin**: Redis Commander

---

## Project Structure

```
atheer-platform/
├── cmd/
│   ├── server/main.go              # Production entry point
│   └── testserver/main.go          # Test server (in-memory, no DB required)
│
├── internal/
│   ├── config/
│   │   ├── config.go               # Viper-based configuration
│   │   ├── database.go             # PostgreSQL connection pool
│   │   └── redis.go                # Redis client setup
│   │
│   ├── handler/
│   │   ├── handlers.go             # API handlers (enroll, transaction, device, etc.)
│   │   └── health_handler.go       # Health check with DB/Redis status
│   │
│   ├── middleware/
│   │   ├── rate_limiter.go         # Layer 1: Tiered rate limiting
│   │   ├── request_logger.go       # Layer 2: Structured logging
│   │   ├── anti_replay.go          # Layer 3: Redis Lua anti-replay
│   │   ├── attestation_verifier.go # Layer 4: ECDSA attestation
│   │   ├── signature_verifier.go   # Layer 5-6: HMAC Side A & B
│   │   ├── cross_validator.go      # Layer 7: Cross-validation rules
│   │   ├── limits_checker.go       # Layer 8: Limit enforcement
│   │   └── idempotency.go          # Layer 9: Nonce-based idempotency
│   │
│   ├── service/
│   │   ├── services.go             # Core business logic
│   │   └── saga_service.go         # Saga: Debit→Credit→Notify with reversal
│   │
│   ├── adapter/
│   │   ├── payment_adapter.go      # PaymentAdapter interface + Registry
│   │   ├── jeep_adapter.go         # JEEP wallet adapter
│   │   └── wallet_adapters.go      # WENET + WASEL adapters
│   │
│   ├── model/
│   │   ├── device.go               # Device model
│   │   ├── transaction.go          # Transaction + SDK payloads
│   │   ├── pending_operation.go    # Saga operation tracking
│   │   ├── limits_matrix.go        # Limit rules
│   │   └── dispute.go              # Disputes + audit logs
│   │
│   ├── repository/
│   │   ├── device_repo.go          # Device CRUD + counter increment
│   │   ├── transaction_repo.go     # Transaction CRUD + status management
│   │   ├── limits_repo.go          # Limits + disputes + audit repos
│   │   └── pending_operation_repo.go # Saga state tracking
│   │
│   └── router/
│       └── router.go               # Chi router + pipeline wiring
│
├── pkg/
│   ├── crypto/
│   │   ├── hmac.go                 # HMAC-SHA256: DeriveLUK, SignPayload, Verify
│   │   └── ecdsa.go                # ECDSA P-256: attestation verification
│   │
│   └── response/
│       └── api_response.go         # Standardized API response + error codes
│
├── dashboard/                      # Next.js 16 Admin Dashboard
│   └── src/
│       ├── app/
│       │   ├── page.tsx            # Dashboard home (live data from Switch)
│       │   ├── transactions/       # Transaction management
│       │   ├── devices/            # Device management
│       │   ├── limits/             # Limits matrix editor
│       │   ├── disputes/           # Dispute tracking
│       │   ├── pipeline/           # Pipeline visualization
│       │   ├── security/           # Security config + audit log
│       │   └── settings/           # Server settings
│       └── components/
│           └── Sidebar.tsx         # Navigation + health indicator
│
├── tests/unit/
│   └── crypto_test.go              # 11 crypto tests (HMAC + ECDSA)
│
├── migrations/                     # PostgreSQL migrations (5 files, 8 tables)
├── redis/                          # Redis Lua scripts
├── docker-compose.yml              # Full stack (PostgreSQL + Redis + Switch)
├── Dockerfile                      # Multi-stage Go build
├── .env.example                    # Environment template
└── go.mod                          # Go module definition
```

---

## Quick Start

### Option 1: Test Server (No Dependencies)

The fastest way to run — no PostgreSQL or Redis needed:

```bash
# Build and run the test server
cd atheer-platform
go build -o atheer-test ./cmd/testserver/
./atheer-test

# Server starts on :8080 with 12 devices + 20 transactions
```

### Option 2: Full Stack with Docker

```bash
# Copy env file
cp .env.example .env

# Start everything (PostgreSQL + Redis + Switch)
docker-compose up -d

# Run migrations
docker exec -i atheer-db psql -U atheer -d atheer < migrations/001_create_devices.up.sql
docker exec -i atheer-db psql -U atheer -d atheer < migrations/002_create_transactions.up.sql
docker exec -i atheer-db psql -U atheer -d atheer < migrations/003_create_pending_operations.up.sql
docker exec -i atheer-db psql -U atheer -d atheer < migrations/004_create_limits_matrix.up.sql
docker exec -i atheer-db psql -U atheer -d atheer < migrations/005_create_remaining_tables.up.sql
```

### Dashboard

```bash
cd dashboard
npm install
npm run dev
# Dashboard on http://localhost:3000
# Switch on http://localhost:8080
```

---

## API Reference

### Base URL: `http://localhost:8080/api/v2`

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "3.0.0",
  "pipeline": "10 layers active",
  "db": "connected",
  "redis": "connected"
}
```

### Enrollment

```http
POST /api/v2/enroll
Content-Type: application/json

{
  "walletId": "JEEP",
  "deviceModel": "Samsung Galaxy S24",
  "attestationLevel": "STRONG_BOX",
  "attestationPublicKey": "<base64-encoded-public-key>"
}
```

**Response:**
```json
{
  "success": true,
  "code": "S000",
  "deviceId": "DEV-a1b2c3d4",
  "deviceSeed": "<base64-encoded-seed>"
}
```

### Process Transaction

```http
POST /api/v2/transaction
Content-Type: application/json

{
  "sideA": {
    "walletId": "JEEP",
    "deviceId": "DEV-a1b2c3d4",
    "ctr": 42,
    "operationType": "P2P_SAME",
    "currency": "YER",
    "amount": 15000.50,
    "nonce": "unique-nonce-uuid",
    "timestamp": 1714000000,
    "signature": "<hmac-sha256-signature>"
  },
  "sideB": {
    "walletId": "WENET",
    "deviceId": "DEV-e5f6g7h8",
    "operationType": "P2P_SAME",
    "currency": "YER",
    "amount": 15000.50,
    "accountId": "ACC-001",
    "timestamp": 1714000000,
    "signature": "<hmac-sha256-signature>"
  }
}
```

**Response:**
```json
{
  "success": true,
  "code": "S000",
  "transactionId": "TX-1a490143-609",
  "status": "COMPLETED",
  "latencyMs": 47
}
```

### Error Codes

| Code | Description |
|------|-------------|
| S000 | Success |
| E001 | Invalid request / validation error |
| E002 | Device not found |
| E003 | Signature verification failed |
| E004 | Anti-replay violation |
| E005 | Rate limit exceeded |
| E006 | Transaction not found |
| E007 | Limit exceeded |
| E008 | Adapter error (debit/credit failed) |
| E009 | Saga failure (reversal triggered) |
| E010 | Attestation verification failed |
| E011 | Cross-validation mismatch |
| E012 | Duplicate transaction (idempotency) |
| E013 | Device suspended/revoked |
| E014 | Channel not configured |
| E015 | Internal server error |

---

## Dashboard

The admin dashboard provides **real-time monitoring** with live data from the Switch:

| Page | Features |
|------|----------|
| **Dashboard** | KPI cards, transaction volume chart, channel distribution, pipeline health |
| **Transactions** | Searchable/filterable table, status badges, latency indicators |
| **Devices** | Device list, attestation levels (StrongBox/TEE), suspend/revoke |
| **Limits** | Editable limits matrix per wallet/operation/currency |
| **Disputes** | Dispute tracking with priority levels |
| **Pipeline** | 10-layer visualization with pass/fail rates and configs |
| **Security** | Crypto settings display, KMS/TLS status, audit log |
| **Settings** | Server, database, rate limits, adapter configuration |

### Live Data Integration

The dashboard polls the Switch API every **5 seconds** and displays:
- ✅ Connection status (green/red banner)
- 📊 Real-time transaction count and total value
- 📱 Active device count
- ⚡ Average response latency
- 🔐 Pipeline layer pass rates

---

## Security

### Cryptographic Primitives

| Primitive | Algorithm | Usage |
|-----------|-----------|-------|
| Key Derivation | HMAC-SHA256 | `LUK = HMAC(seed, BigEndian(ctr))` |
| Payload Signing | HMAC-SHA256 | `Sig = HMAC(LUK, PayloadHash)` |
| Attestation | ECDSA P-256 | TEE/StrongBox device verification |
| Anti-Replay | Redis Lua | Atomic counter comparison |
| Comparison | `hmac.Equal()` | Constant-time (timing attack prevention) |

### Production Security Checklist

- [ ] Enable KMS for `deviceSeed` encryption at rest
- [ ] Enable mTLS between Switch and Adapters
- [ ] Configure Play Integrity API verification
- [ ] Set up HSM for key management
- [ ] Enable audit log persistence to database
- [ ] Configure TLS termination (nginx/traefik)

---

## Testing

### Unit Tests

```bash
go test ./... -v -count=1
```

**Current Coverage (11 tests):**

| Test | Description | Status |
|------|-------------|--------|
| TestDeriveLUK_Deterministic | LUK derivation produces consistent results | ✅ PASS |
| TestDeriveLUK_DifferentCounters | Different counters → different LUKs | ✅ PASS |
| TestSignPayload_Deterministic | Payload signing is deterministic | ✅ PASS |
| TestTimingSafeEqual | Constant-time comparison works correctly | ✅ PASS |
| TestBuildSideAPayloadHash_Consistency | Payload hash is consistent | ✅ PASS |
| TestBuildSideAPayloadHash_DifferentAmount | Different amounts → different hashes | ✅ PASS |
| TestBuildSideAPayloadHash_NilAmount | Nil amount handled correctly | ✅ PASS |
| TestFullSignatureFlow_EndToEnd | SDK → Switch signature verification | ✅ PASS |
| TestFullSignatureFlow_TamperedPayload | Tampered payloads rejected | ✅ PASS |
| TestVerifyECDSA_ValidSignature | ECDSA verification works | ✅ PASS |
| TestVerifyECDSA_InvalidSignature | Invalid ECDSA signatures rejected | ✅ PASS |

### Integration Test (Manual)

```bash
# Start test server
go run ./cmd/testserver/

# Send a transaction
curl -X POST http://localhost:8080/api/v2/transaction \
  -H "Content-Type: application/json" \
  -d '{"sideA":{"walletId":"JEEP","deviceId":"DEV-001","ctr":1,"operationType":"P2P_SAME","currency":"YER","amount":1000,"nonce":"test","timestamp":1714000000,"signature":"test"},"sideB":{"walletId":"WENET","deviceId":"DEV-002","operationType":"P2P_SAME","currency":"YER","amount":1000,"accountId":"ACC","timestamp":1714000000,"signature":"test"}}'
```

---

## Configuration

### Environment Variables

```env
# Server
SERVER_PORT=8080
SERVER_ENV=development
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=15s
SERVER_SHUTDOWN_TIMEOUT=30s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=atheer
DB_PASSWORD=atheer_secure_pass
DB_NAME=atheer
DB_POOL_MIN=5
DB_POOL_MAX=20

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Security
HMAC_ALGORITHM=SHA256
ECDSA_CURVE=P-256
ANTI_REPLAY_TTL=86400
IDEMPOTENCY_TTL=3600

# Rate Limits
RATE_LIMIT_DEVICE=10
RATE_LIMIT_WALLET=100
RATE_LIMIT_IP=50

# Dashboard
NEXT_PUBLIC_SWITCH_API_URL=http://localhost:8080/api/v2
```

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style
- Go: `gofmt` + `golangci-lint`
- TypeScript: ESLint + Prettier
- Commits: Conventional Commits (`feat:`, `fix:`, `docs:`)

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>Built with ❤️ for Yemen's digital payment future</strong>
</p>
