package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"
)

// DeriveLUK derives a Limited Use Key from deviceSeed and counter
// LUK = HMAC-SHA256(deviceSeed, ctr)
// Must match SDK's KeyDerivation.deriveLUK() exactly
func DeriveLUK(deviceSeed []byte, ctr int64) []byte {
	ctrBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(ctrBytes, uint64(ctr))

	mac := hmac.New(sha256.New, deviceSeed)
	mac.Write(ctrBytes)
	return mac.Sum(nil)
}

// SignPayload signs a payload hash with a LUK
// SIG = HMAC-SHA256(LUK, payloadHash)
// Must match SDK's SignatureUtils.sign() exactly
func SignPayload(luk []byte, payloadHash []byte) []byte {
	mac := hmac.New(sha256.New, luk)
	mac.Write(payloadHash)
	return mac.Sum(nil)
}

// BuildSideAPayloadHash builds the SHA256 hash of Side A payload fields
// The field order MUST match SDK's SignatureUtils.buildPayloadString() exactly:
//   walletId + deviceId + ctr + operationType + currency + amount + nonce + timestamp
//
// ⚠️ CRITICAL: Any difference in order or formatting will reject ALL transactions
func BuildSideAPayloadHash(walletId, deviceId string, ctr int64,
	operationType, currency string, amount *float64, nonce string, timestamp int64) []byte {

	data := walletId + deviceId +
		strconv.FormatInt(ctr, 10) +
		operationType + currency +
		formatAmount(amount) +
		nonce +
		strconv.FormatInt(timestamp, 10)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// BuildSideBPayloadHash builds the SHA256 hash of Side B payload fields
// Must match SDK's SideBProcessor signature building
func BuildSideBPayloadHash(walletId, deviceId string, merchantId *string,
	operationType, currency string, amount *float64, accountId string, timestamp int64) []byte {

	merchantStr := ""
	if merchantId != nil {
		merchantStr = *merchantId
	}

	data := walletId + deviceId + merchantStr +
		operationType + currency +
		formatAmount(amount) +
		accountId +
		strconv.FormatInt(timestamp, 10)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// TimingSafeEqual performs a constant-time comparison of two byte slices
// Prevents timing attacks on signature verification (FR-KEY-002)
func TimingSafeEqual(a, b []byte) bool {
	return hmac.Equal(a, b)
}

// VerifyHMACSignature verifies that a signature matches the expected HMAC
func VerifyHMACSignature(deviceSeed []byte, ctr int64,
	walletId, deviceId string, operationType, currency string,
	amount *float64, nonce string, timestamp int64,
	signatureBytes []byte) error {

	luk := DeriveLUK(deviceSeed, ctr)
	payloadHash := BuildSideAPayloadHash(walletId, deviceId, ctr,
		operationType, currency, amount, nonce, timestamp)
	expectedSig := SignPayload(luk, payloadHash)

	if !TimingSafeEqual(expectedSig, signatureBytes) {
		return fmt.Errorf("HMAC signature mismatch")
	}
	return nil
}

// formatAmount formats amount consistently to match SDK's format
// SDK uses Double.toString() which produces "500.0", "1234.56", etc.
// null amount → empty string ""
func formatAmount(amount *float64) string {
	if amount == nil {
		return ""
	}
	return strconv.FormatFloat(*amount, 'f', -1, 64)
}
