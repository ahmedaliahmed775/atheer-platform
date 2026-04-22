package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
)

// VerifyECDSASignature verifies an ECDSA P-256 attestation signature
// Must match SDK's AttestationManager.signTransactionRequest()
// The payload signed is the same fields as HMAC but using ECDSA from TEE
func VerifyECDSASignature(publicKeyPEM string, payloadHash, signature []byte) error {
	pubKey, err := ParseECDSAPublicKey(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse ECDSA public key: %w", err)
	}

	// Verify ASN.1 DER encoded signature
	if !ecdsa.VerifyASN1(pubKey, payloadHash, signature) {
		return fmt.Errorf("ECDSA signature verification failed")
	}

	return nil
}

// VerifyAttestationSignature verifies the hardware attestation signature for a Side A payload
func VerifyAttestationSignature(publicKeyPEM string,
	walletId, deviceId string, ctr int64,
	operationType, currency string, amount *float64,
	nonce string, timestamp int64,
	signatureBytes []byte) error {

	payloadHash := BuildSideAPayloadHash(walletId, deviceId, ctr,
		operationType, currency, amount, nonce, timestamp)

	return VerifyECDSASignature(publicKeyPEM, payloadHash, signatureBytes)
}

// ParseECDSAPublicKey parses a PEM-encoded ECDSA P-256 public key
func ParseECDSAPublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		// Try parsing as raw base64 (SDK may send without PEM headers)
		return parseRawECDSAPublicKey([]byte(pemStr))
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
	}

	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA")
	}

	if ecdsaPub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("key is not P-256, got %s", ecdsaPub.Curve.Params().Name)
	}

	return ecdsaPub, nil
}

// parseRawECDSAPublicKey attempts to parse raw uncompressed EC point bytes
func parseRawECDSAPublicKey(raw []byte) (*ecdsa.PublicKey, error) {
	// Try X.509 SubjectPublicKeyInfo format first
	pub, err := x509.ParsePKIXPublicKey(raw)
	if err == nil {
		ecdsaPub, ok := pub.(*ecdsa.PublicKey)
		if ok {
			return ecdsaPub, nil
		}
	}

	// Try raw uncompressed point (0x04 || X || Y)
	curve := elliptic.P256()
	keyLen := (curve.Params().BitSize + 7) / 8

	if len(raw) == 1+2*keyLen && raw[0] == 0x04 {
		x := new(big.Int).SetBytes(raw[1 : 1+keyLen])
		y := new(big.Int).SetBytes(raw[1+keyLen:])
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
	}

	return nil, fmt.Errorf("unable to parse ECDSA public key (len=%d)", len(raw))
}

// HashPayload computes SHA-256 hash of arbitrary data
func HashPayload(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
