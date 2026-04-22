package crypto_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/atheer-payment/atheer-platform/pkg/crypto"
)

// TestDeriveLUK_Deterministic verifies that the same inputs always produce the same LUK
func TestDeriveLUK_Deterministic(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}

	luk1 := crypto.DeriveLUK(seed, 0)
	luk2 := crypto.DeriveLUK(seed, 0)

	if hex.EncodeToString(luk1) != hex.EncodeToString(luk2) {
		t.Fatalf("LUK derivation not deterministic:\n  luk1=%s\n  luk2=%s",
			hex.EncodeToString(luk1), hex.EncodeToString(luk2))
	}
	t.Logf("LUK(ctr=0) = %s", hex.EncodeToString(luk1))
}

// TestDeriveLUK_DifferentCounters verifies different counters produce different LUKs
func TestDeriveLUK_DifferentCounters(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}

	luk0 := crypto.DeriveLUK(seed, 0)
	luk1 := crypto.DeriveLUK(seed, 1)
	luk99 := crypto.DeriveLUK(seed, 99)

	if hex.EncodeToString(luk0) == hex.EncodeToString(luk1) {
		t.Fatal("LUK for ctr=0 and ctr=1 should differ")
	}
	if hex.EncodeToString(luk1) == hex.EncodeToString(luk99) {
		t.Fatal("LUK for ctr=1 and ctr=99 should differ")
	}

	t.Logf("LUK(ctr=0)  = %s", hex.EncodeToString(luk0))
	t.Logf("LUK(ctr=1)  = %s", hex.EncodeToString(luk1))
	t.Logf("LUK(ctr=99) = %s", hex.EncodeToString(luk99))
}

// TestSignPayload_Deterministic verifies consistent signing
func TestSignPayload_Deterministic(t *testing.T) {
	luk := make([]byte, 32)
	for i := range luk {
		luk[i] = byte(i + 100)
	}

	hash := sha256.Sum256([]byte("test-payload"))

	sig1 := crypto.SignPayload(luk, hash[:])
	sig2 := crypto.SignPayload(luk, hash[:])

	if !crypto.TimingSafeEqual(sig1, sig2) {
		t.Fatal("Signature should be deterministic")
	}
	t.Logf("SIG = %s", hex.EncodeToString(sig1))
}

// TestTimingSafeEqual verifies constant-time comparison
func TestTimingSafeEqual(t *testing.T) {
	a := []byte{1, 2, 3, 4, 5}
	b := []byte{1, 2, 3, 4, 5}
	c := []byte{1, 2, 3, 4, 6}
	d := []byte{1, 2, 3}

	if !crypto.TimingSafeEqual(a, b) {
		t.Fatal("Equal slices should return true")
	}
	if crypto.TimingSafeEqual(a, c) {
		t.Fatal("Different slices should return false")
	}
	if crypto.TimingSafeEqual(a, d) {
		t.Fatal("Different length slices should return false")
	}
}

// TestBuildSideAPayloadHash_Consistency verifies payload hash consistency
func TestBuildSideAPayloadHash_Consistency(t *testing.T) {
	amount := 500.0
	hash1 := crypto.BuildSideAPayloadHash(
		"JEEP", "device-001", 5,
		"P2P_SAME", "YER", &amount,
		"nonce-uuid-1234", 1700000000000,
	)
	hash2 := crypto.BuildSideAPayloadHash(
		"JEEP", "device-001", 5,
		"P2P_SAME", "YER", &amount,
		"nonce-uuid-1234", 1700000000000,
	)

	if hex.EncodeToString(hash1) != hex.EncodeToString(hash2) {
		t.Fatal("Same inputs should produce same hash")
	}
	t.Logf("PayloadHash = %s", hex.EncodeToString(hash1))
}

// TestBuildSideAPayloadHash_DifferentAmount verifies different amounts produce different hashes
func TestBuildSideAPayloadHash_DifferentAmount(t *testing.T) {
	amount1 := 500.0
	amount2 := 500.5

	hash1 := crypto.BuildSideAPayloadHash("W", "D", 0, "P2P_SAME", "YER", &amount1, "n", 0)
	hash2 := crypto.BuildSideAPayloadHash("W", "D", 0, "P2P_SAME", "YER", &amount2, "n", 0)

	if hex.EncodeToString(hash1) == hex.EncodeToString(hash2) {
		t.Fatal("Different amounts should produce different hashes")
	}
}

// TestBuildSideAPayloadHash_NilAmount verifies nil amount handling
func TestBuildSideAPayloadHash_NilAmount(t *testing.T) {
	hashNil := crypto.BuildSideAPayloadHash("W", "D", 0, "P2P_SAME", "YER", nil, "n", 0)
	amount := 0.0
	hashZero := crypto.BuildSideAPayloadHash("W", "D", 0, "P2P_SAME", "YER", &amount, "n", 0)

	if hex.EncodeToString(hashNil) == hex.EncodeToString(hashZero) {
		t.Fatal("nil amount and 0.0 amount should produce different hashes")
	}
}

// TestFullSignatureFlow_EndToEnd simulates the complete SDK → Switch signature flow
func TestFullSignatureFlow_EndToEnd(t *testing.T) {
	// 1. Simulate enrollment: generate device seed
	deviceSeed := make([]byte, 32)
	if _, err := rand.Read(deviceSeed); err != nil {
		t.Fatal(err)
	}

	// 2. Simulate SDK side: derive LUK and sign
	ctr := int64(42)
	amount := 1500.75
	walletId := "JEEP"
	deviceId := "test-device-abc"
	opType := "P2M_SAME"
	currency := "YER"
	nonce := "550e8400-e29b-41d4-a716-446655440000"
	timestamp := int64(1700000000000)

	sdkLUK := crypto.DeriveLUK(deviceSeed, ctr)
	sdkHash := crypto.BuildSideAPayloadHash(walletId, deviceId, ctr, opType, currency, &amount, nonce, timestamp)
	sdkSignature := crypto.SignPayload(sdkLUK, sdkHash)

	// 3. Simulate Switch side: verify signature
	err := crypto.VerifyHMACSignature(deviceSeed, ctr,
		walletId, deviceId, opType, currency,
		&amount, nonce, timestamp,
		sdkSignature)

	if err != nil {
		t.Fatalf("Switch should accept SDK's signature: %v", err)
	}
	t.Log("✅ End-to-end signature flow: SDK → Switch verified successfully")
}

// TestFullSignatureFlow_TamperedPayload verifies tampered payloads are rejected
func TestFullSignatureFlow_TamperedPayload(t *testing.T) {
	deviceSeed := make([]byte, 32)
	rand.Read(deviceSeed)

	amount := 1000.0
	luk := crypto.DeriveLUK(deviceSeed, 1)
	hash := crypto.BuildSideAPayloadHash("JEEP", "dev1", 1, "P2P_SAME", "YER", &amount, "nonce1", 1700000000000)
	sig := crypto.SignPayload(luk, hash)

	// Tamper the amount
	tamperedAmount := 9999.0
	err := crypto.VerifyHMACSignature(deviceSeed, 1,
		"JEEP", "dev1", "P2P_SAME", "YER",
		&tamperedAmount, "nonce1", 1700000000000,
		sig)

	if err == nil {
		t.Fatal("Tampered payload should be rejected")
	}
	t.Log("✅ Tampered payload correctly rejected")
}

// TestVerifyECDSA_ValidSignature generates a real ECDSA key pair and verifies a signature
func TestVerifyECDSA_ValidSignature(t *testing.T) {
	// Generate a test P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Sign data
	data := []byte("test attestation payload")
	hash := sha256.Sum256(data)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	if !ecdsa.VerifyASN1(&privateKey.PublicKey, hash[:], signature) {
		t.Fatal("ECDSA signature should be valid")
	}
	t.Log("✅ ECDSA P-256 signature verified")
}

// TestVerifyECDSA_InvalidSignature verifies that wrong data fails ECDSA check
func TestVerifyECDSA_InvalidSignature(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	data := []byte("original data")
	hash := sha256.Sum256(data)
	sig, _ := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])

	// Verify with wrong data
	wrongHash := sha256.Sum256([]byte("tampered data"))
	if ecdsa.VerifyASN1(&privateKey.PublicKey, wrongHash[:], sig) {
		t.Fatal("ECDSA should reject signature for different data")
	}
	t.Log("✅ ECDSA correctly rejected tampered data")
}
