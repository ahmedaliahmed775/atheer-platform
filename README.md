<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/Next.js-16-black?style=for-the-badge&logo=next.js" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql&logoColor=white" />
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" />
</p>

# 🏗️ Atheer Switch — NFC Payment Platform

> **Modular Monolith** backend for the Atheer SDK V3.0 — a secure, NFC-based mobile payment system supporting P2P, P2M, M2P, and M2M transactions across multiple wallet providers.

---

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Transaction Types](#transaction-types)
- [Transaction Pipeline](#transaction-pipeline)
- [SwitchRecord](#switchrecord)
- [HMAC Formula](#hmac-formula)
- [Adapter Pattern](#adapter-pattern)
- [Error Codes](#error-codes)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Dashboard](#dashboard)
- [Security](#security)
- [Configuration](#configuration)
- [License](#license)

---

## Overview

Atheer Switch is the central payment processing engine that:

- **Processes NFC tap-to-pay transactions** between mobile devices (Dual Tap protocol)
- **Enforces a 7-layer security pipeline** (Rate Limiting → HMAC Verification → PayeeType Verification → TransactionType Resolution)
- **Supports multiple wallet providers** via the Adapter Pattern (`WalletAdapter` interface)
- **Maintains a SwitchRecord per user**: `PublicID + Seed + UserID + UserType + WalletID`
- **Determines TransactionType automatically**: `PayerType + PayeeType → P2P | P2M | M2P | M2M`
- **Provides a real-time dashboard** for monitoring and administration

### Transaction Flow — Dual Tap Protocol

```
┌──────────────┐    NFC Tap 1     ┌──────────────┐
│  Party A     │◄────────────────│  Party B     │
│  (Payer)     │  Payee sends:   │  (Payee)     │
│  Offline     │  ReceiverID,    │  Online      │
│  HCE Mode   │  Amount,        │  Reader Mode │
│              │  PayeeType      │              │
│ TEE: HMAC    │                 │              │
│ + Auth       │                 │              │
│              │    NFC Tap 2    │              │
│              │────────────────►│              │
│              │  TLV Packet:    │  mTLS to     │
│              │  PublicID,      │  Switch      │
│              │  Amount,        │              │
│              │  ReceiverID,    │              │
│              │  PayeeType,     │              │
│              │  Counter, HMAC  │              │
└──────────────┘                 └──────┬───────┘
                                       │ mTLS
                                       ▼
                              ┌────────────────┐
                              │  Atheer Switch  │
                              │  7-Layer Pipeline│
                              │                 │
                              │ 1. Rate Limiter │
                              │ 2. Request Logger│
                              │ 3. Anti-Replay  │
                              │ 4. Limits Check │
                              │ 5. HMAC Verify  │
                              │ 6. PayeeType    │
                              │    Verify       │
                              │ 7. TxType       │
                              │    Resolver     │
                              └───────┬─────────┘
                                      ▼
                              ┌────────────────┐
                              │ Adapter Registry│
                              │ ┌────┬────┬───┐│
                              │ │JEEP│WNET│WSL││
                              │ └────┴────┴───┘│
                              └───────┬────────┘
                                      ▼
                              ┌────────────────┐
                              │ Wallet Server   │
                              │ Debit + Credit  │
                              │ + SMS           │
                              └────────────────┘
```

---

## Transaction Types

> **Reference: Document Section 2**

The Switch determines the transaction type **automatically** — no external party sets it.

| Type | Payer | Payee | Scenario |
|------|-------|-------|----------|
| **P2P** | Personal (P) | Personal (P) | Transfer between individuals |
| **P2M** | Personal (P) | Merchant (M) | Payment for goods/services |
| **M2P** | Merchant (M) | Personal (P) | Refund or reverse payment |
| **M2M** | Merchant (M) | Merchant (M) | Transfer between merchants/branches |

```go
TransactionType = DetermineTransactionType(PayerType, PayeeType)
//   PayerType ← from SwitchRecord (by PublicID)
//   PayeeType ← verified against SwitchRecord (by ReceiverID)
```

🔴 **Critical**: If `ReceiverID` is not registered in the Switch, or if `PayeeType` from the packet doesn't match `UserType` in the SwitchRecord → **immediate rejection**.

---

## SwitchRecord

> **Reference: Document Section 4 — Phase 1**

Each enrolled user has one record in the Switch:

```go
type SwitchRecord struct {
    PublicID  string     // Public identifier (not tied to real identity)
    Seed      []byte     // Cryptographic seed (HSM-protected)
    UserID    string     // Wallet user ID
    UserType  UserType   // "P" (Personal) | "M" (Merchant)
    WalletID  string     // Wallet identifier for adapter routing
    Counter   uint64     // Monotonic counter
    Status    string     // ACTIVE | SUSPENDED
}
```

🟡 **Important**: `UserType` is determined **exclusively** by the Switch — never sent from the wallet app.

---

## HMAC Formula

> **Reference: Document Section 4 — Step 3**

```
LUK   = HMAC-SHA256(Seed, Counter)
Token = HMAC-SHA256(LUK, Amount || ReceiverID || PayeeType || WalletID || Counter)
```

| Input | Source |
|-------|--------|
| `Seed`, `Counter`, `WalletID` | SwitchRecord (via PublicID lookup) |
| `Amount`, `ReceiverID`, `PayeeType` | TLV Packet from Party A |

🔴 **Critical**: `WalletID` is included in the HMAC but **not sent** in the TLV packet. The Switch extracts it from the SwitchRecord via `PublicID`. This prevents routing manipulation.

---

## Transaction Pipeline

> **Reference: Document Section 4 — Step 5**

Every transaction passes through **7 security layers**:

| Layer | Name | Purpose | File |
|-------|------|---------|------|
| 1 | **Rate Limiter** | Flood protection per device/wallet/IP | `rate_limiter.go` |
| 2 | **Request Logger** | Structured audit logging | `request_logger.go` |
| 3 | **Anti-Replay** | Reject replayed counters (Redis Lua) | `anti_replay.go` |
| 4 | **Limits Checker** | Spending limits & risk thresholds | `limits_checker.go` |
| 5 | **HMAC Verifier** | Verify payer's HMAC signature | `signature_verifier.go` |
| 6 | **PayeeType Verifier** | Verify PayeeType matches SwitchRecord | `payee_type_verifier.go` |
| 7 | **TxType Resolver** | Determine P2P/P2M/M2P/M2M automatically | `transaction_type_resolver.go` |

---

## Adapter Pattern

> **Reference: Document Section 5**

```
Atheer Switch
    ├── AdapterRegistry
    │       ├── WalletAdapter [WalletID: "JEEP"]   → Wallet Server A
    │       ├── WalletAdapter [WalletID: "WENET"]  → Wallet Server B
    │       └── WalletAdapter [WalletID: "WASEL"]  → Wallet Server C
    └── HSM (shared across all adapters)
```

### WalletAdapter Interface

```go
type WalletAdapter interface {
    WalletID()                                      string
    BuildRequest(dto TransactionDTO) (*WalletAPIRequest, error)
    ParseResponse(raw []byte)        (*AtheerResult, error)
}
```

Adding a new wallet partner = writing a new adapter only. No changes to the security layer or HSM.

### TransactionDTO

```go
type TransactionDTO struct {
    PayerUserID     string
    PayerType       UserType        // P | M
    PayeeID         string
    PayeeType       UserType        // P | M (verified)
    TransactionType TransactionType // P2P | P2M | M2P | M2M
    Amount          int64
    Currency        string
    WalletID        string
    Counter         uint64
    Timestamp       int64
}
```

---

## Error Codes

> **Reference: Document Section 5 — Error Mapping**

| Switch Code | SDK ErrorCode | Cause |
|-------------|---------------|-------|
| `ERR_HMAC_MISMATCH` | `SIGNATURE_MISMATCH` | Signature doesn't match |
| `ERR_COUNTER_REPLAY` | `SIGNATURE_MISMATCH` | Counter reuse detected |
| `ERR_PAYEE_TYPE_MISMATCH` | `TRANSACTION_REJECTED` | PayeeType doesn't match record |
| `ERR_SPEND_LIMIT` | `SPENDING_LIMIT_EXCEEDED` | Spending limit exceeded |
| `ERR_BALANCE` | `INSUFFICIENT_BALANCE` | Insufficient balance |
| `ERR_WALLET_DOWN` | `WALLET_SERVER_ERROR` | Wallet server unavailable |
| `ERR_UNKNOWN_WALLET` | `INVALID_WALLET_ID` | WalletID not registered |

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

### Infrastructure
- **Containers**: Docker + Docker Compose

---

## Project Structure

```
atheer-platform/
├── cmd/
│   ├── server/main.go              # Production entry point
│   └── testserver/main.go          # Test server (in-memory)
│
├── internal/
│   ├── config/                     # Viper config + DB + Redis
│   ├── handler/                    # API handlers
│   │
│   ├── middleware/
│   │   ├── rate_limiter.go         # Layer 1: Flood protection
│   │   ├── request_logger.go       # Layer 2: Audit logging
│   │   ├── anti_replay.go          # Layer 3: Counter replay protection
│   │   ├── limits_checker.go       # Layer 4: Spending limits
│   │   ├── signature_verifier.go   # Layer 5: HMAC verification
│   │   ├── payee_type_verifier.go  # Layer 6: PayeeType verification
│   │   ├── transaction_type_resolver.go # Layer 7: TxType determination
│   │   └── context_helpers.go      # Context key storage
│   │
│   ├── service/
│   │   └── services.go             # Core business logic
│   │
│   ├── adapter/
│   │   ├── payment_adapter.go      # WalletAdapter interface + Registry
│   │   ├── jeep_adapter.go         # JEEP wallet adapter
│   │   └── wallet_adapters.go      # WENET + WASEL adapters
│   │
│   ├── model/
│   │   ├── switch_record.go        # SwitchRecord + UserType (P|M)
│   │   ├── transaction.go          # Transaction + TransactionType + PayerTlvPacket
│   │   ├── limits_matrix.go        # Limit rules
│   │   └── dispute.go              # Disputes
│   │
│   ├── repository/
│   │   ├── switch_record_repo.go   # SwitchRecord CRUD
│   │   ├── transaction_repo.go     # Transaction CRUD
│   │   └── ...
│   │
│   └── router/
│       └── router.go               # Chi router + 7-layer pipeline
│
├── pkg/
│   ├── crypto/
│   │   └── hmac.go                 # DeriveLUK + GenerateTransactionHMAC + VerifyTransactionHMAC
│   └── response/
│       └── api_response.go         # Standardized API responses
│
├── dashboard/                      # Next.js Admin Dashboard
├── migrations/                     # PostgreSQL migrations
├── docker-compose.yml
├── Dockerfile
└── go.mod
```

---

## Quick Start

### Option 1: Test Server (No Dependencies)

```bash
cd atheer-platform
go build -o atheer-test ./cmd/testserver/
./atheer-test
# Server starts on :8080
```

### Option 2: Full Stack with Docker

```bash
cp .env.example .env
docker-compose up -d
```

### Dashboard

```bash
cd dashboard
npm install
npm run dev
# Dashboard on http://localhost:3000
```

---

## API Reference

### Base URL: `http://localhost:8080/api/v2`

### Enrollment

```http
POST /api/v2/enroll
Content-Type: application/json

{
  "walletId": "JEEP",
  "userId": "user_001",
  "attestationResult": "<base64>"
}
```

**Response:**
```json
{
  "success": true,
  "publicId": "PUB-a1b2c3d4",
  "encryptedSeed": "<base64>",
  "counter": 0,
  "userType": "P"
}
```

### Process Transaction

```http
POST /api/v2/transaction
Content-Type: application/json

{
  "publicId": "PUB-a1b2c3d4",
  "amount": 15000,
  "receiverId": "user_002",
  "payeeType": "P",
  "counter": 42,
  "hmac": "<base64-hmac>"
}
```

**Response:**
```json
{
  "success": true,
  "transactionId": "TX-1a490143",
  "transactionType": "P2P",
  "status": "COMPLETED"
}
```

---

## Security

### Cryptographic Primitives

| Primitive | Algorithm | Usage |
|-----------|-----------|-------|
| Key Derivation | HMAC-SHA256 | `LUK = HMAC(Seed, BigEndian(Counter))` |
| Transaction Signing | HMAC-SHA256 | `Token = HMAC(LUK, Amount\|ReceiverID\|PayeeType\|WalletID\|Counter)` |
| Anti-Replay | Redis Lua | Atomic counter comparison |
| Comparison | `hmac.Equal()` | Constant-time (timing attack prevention) |
| Party B ↔ Switch | mTLS | Authenticated connection binding ReceiverID |

### Security Guarantees

- **WalletID** is in HMAC but not in packet — prevents routing manipulation
- **PayeeType** is verified against SwitchRecord — prevents type spoofing
- **TransactionType** is determined server-side — prevents external manipulation
- **UserType** authority is in the Switch — never sent from wallet app
- **TTL = 30 seconds** — prevents counter desync on expired sessions
- **mTLS** between Party B and Switch — prevents packet interception

---

## Configuration

```env
SERVER_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=atheer
DB_PASSWORD=atheer_secure_pass
DB_NAME=atheer
REDIS_HOST=localhost
REDIS_PORT=6379
HMAC_ALGORITHM=SHA256
ANTI_REPLAY_TTL=86400
```

---

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>Built with ❤️ for Yemen's digital payment future</strong>
</p>
