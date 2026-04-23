# Changelog

All notable changes to this project are documented in this file.

## [3.0.0] — Document Alignment Release

### Architecture Changes
- **SwitchRecord** replaces Device model — `PublicID + Seed + UserID + UserType + WalletID`
- **TransactionType** changed from `P2P_SAME/P2M_CROSS` to `P2P/P2M/M2P/M2M`
- **UserType** (`P` | `M`) — determined exclusively by Switch, never sent from wallet
- **WalletAdapter** interface simplified to: `WalletID() + BuildRequest() + ParseResponse()`

### Security Pipeline
- **7-layer pipeline** (was 10): Rate Limiter → Logger → Anti-Replay → Limits → HMAC → PayeeType → TxType
- **Removed**: Attestation Verifier (moved to enrollment), SignatureVerifierB (replaced by mTLS), CrossValidator, Idempotency
- **Added**: PayeeType Verifier — validates PayeeType against SwitchRecord
- **Added**: TransactionType Resolver — determines P2P/P2M/M2P/M2M automatically

### Cryptography
- **HMAC formula** updated: `Token = HMAC-SHA256(LUK, Amount || ReceiverID || PayeeType || WalletID || Counter)`
- **WalletID** included in HMAC but extracted from SwitchRecord (not from packet)
- **LUK derivation** retained for forward secrecy: `LUK = HMAC-SHA256(Seed, Counter)`

### Error Codes
- Added standardized error code mapping: Switch (`ERR_*`) → SDK (`AtheerErrorCode`)
- `ERR_HMAC_MISMATCH`, `ERR_COUNTER_REPLAY`, `ERR_PAYEE_TYPE_MISMATCH`, `ERR_SPEND_LIMIT`, `ERR_BALANCE`, `ERR_WALLET_DOWN`, `ERR_UNKNOWN_WALLET`

### Removed
- `cross_validator.go` — not in document
- `attestation_verifier.go` — moved to enrollment phase
- `idempotency.go` — not in document
- `ecdsa.go` — moved to enrollment phase

## [2.0.0] — Initial Platform

- Initial Atheer Switch implementation
- 10-layer security pipeline
- Saga pattern for cross-wallet transactions
- Dashboard with real-time monitoring
