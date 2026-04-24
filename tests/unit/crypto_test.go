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

// TestGenerateTransactionHMAC_Deterministic verifies consistent HMAC generation
func TestGenerateTransactionHMAC_Deterministic(t *testing.T) {
        seed := make([]byte, 32)
        for i := range seed {
                seed[i] = byte(i + 100)
        }

        sig1 := crypto.GenerateTransactionHMAC(seed, 500, "user-b", "YER", "JEEP", 1)
        sig2 := crypto.GenerateTransactionHMAC(seed, 500, "user-b", "YER", "JEEP", 1)

        if !crypto.TimingSafeEqual(sig1, sig2) {
                t.Fatal("HMAC should be deterministic")
        }
        t.Logf("HMAC = %s", hex.EncodeToString(sig1))
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

// TestFullSignatureFlow_EndToEnd simulates the complete SDK -> Switch signature flow
// v3.0 formula: LUK = HMAC-SHA256(Seed, Counter), Token = HMAC-SHA256(LUK, Amount||ReceiverID||Currency||WalletID||Counter)
func TestFullSignatureFlow_EndToEnd(t *testing.T) {
        deviceSeed := make([]byte, 32)
        if _, err := rand.Read(deviceSeed); err != nil {
                t.Fatal(err)
        }

        ctr := uint64(42)
        amount := int64(150075)
        receiverID := "user-merchant-123"
        currency := "YER"
        walletID := "JEEP"

        sdkHMAC := crypto.GenerateTransactionHMAC(deviceSeed, amount, receiverID, currency, walletID, ctr)

        err := crypto.VerifyTransactionHMAC(deviceSeed, amount, receiverID, currency, walletID, ctr, sdkHMAC)

        if err != nil {
                t.Fatalf("Switch should accept SDK's HMAC: %v", err)
        }
        t.Log("End-to-end signature flow: SDK -> Switch verified successfully")
}

// TestFullSignatureFlow_TamperedAmount verifies tampered amounts are rejected
func TestFullSignatureFlow_TamperedAmount(t *testing.T) {
        deviceSeed := make([]byte, 32)
        rand.Read(deviceSeed)

        ctr := uint64(1)
        sig := crypto.GenerateTransactionHMAC(deviceSeed, 100000, "user-b", "SAR", "WENET", ctr)

        tamperedAmount := int64(999900)
        err := crypto.VerifyTransactionHMAC(deviceSeed, tamperedAmount, "user-b", "SAR", "WENET", ctr, sig)

        if err == nil {
                t.Fatal("Tampered amount should be rejected")
        }
        t.Log("Tampered amount correctly rejected")
}

// TestFullSignatureFlow_TamperedCurrency verifies currency swap attacks are rejected
func TestFullSignatureFlow_TamperedCurrency(t *testing.T) {
        deviceSeed := make([]byte, 32)
        rand.Read(deviceSeed)

        sig := crypto.GenerateTransactionHMAC(deviceSeed, 1000, "user-b", "YER", "JEEP", 1)

        err := crypto.VerifyTransactionHMAC(deviceSeed, 1000, "user-b", "SAR", "JEEP", 1, sig)

        if err == nil {
                t.Fatal("Currency swap attack should be rejected")
        }
        t.Log("Currency swap attack correctly rejected")
}

// TestVerifyECDSA_ValidSignature generates a real ECDSA key pair and verifies a signature
func TestVerifyECDSA_ValidSignature(t *testing.T) {
        privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
        if err != nil {
                t.Fatal(err)
        }

        data := []byte("test attestation payload")
        hash := sha256.Sum256(data)
        signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
        if err != nil {
                t.Fatal(err)
        }

        if !ecdsa.VerifyASN1(&privateKey.PublicKey, hash[:], signature) {
                t.Fatal("ECDSA signature should be valid")
        }
        t.Log("ECDSA P-256 signature verified")
}

// TestVerifyECDSA_InvalidSignature verifies that wrong data fails ECDSA check
func TestVerifyECDSA_InvalidSignature(t *testing.T) {
        privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
        data := []byte("original data")
        hash := sha256.Sum256(data)
        sig, _ := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])

        wrongHash := sha256.Sum256([]byte("tampered data"))
        if ecdsa.VerifyASN1(&privateKey.PublicKey, wrongHash[:], sig) {
                t.Fatal("ECDSA should reject signature for different data")
        }
        t.Log("ECDSA correctly rejected tampered data")
}
