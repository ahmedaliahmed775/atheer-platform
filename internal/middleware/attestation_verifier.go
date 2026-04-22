package middleware

import (
	"encoding/base64"
	"log/slog"
	"net/http"

	"github.com/atheer-payment/atheer-platform/internal/repository"
	"github.com/atheer-payment/atheer-platform/pkg/crypto"
	"github.com/atheer-payment/atheer-platform/pkg/response"
)

// AttestationVerifier is Layer 4 of the transaction pipeline
// Verifies the ECDSA attestation signature from the device's TEE/StrongBox
// Must match SDK's AttestationManager.signTransactionRequest()
type AttestationVerifier struct {
	deviceRepo *repository.DeviceRepository
}

func NewAttestationVerifier(deviceRepo *repository.DeviceRepository) *AttestationVerifier {
	return &AttestationVerifier{deviceRepo: deviceRepo}
}

func (av *AttestationVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		sideA := &req.SideA

		// 1. Get device record
		device, err := av.deviceRepo.GetByDeviceID(r.Context(), sideA.DeviceID)
		if err != nil {
			response.Forbidden(w, response.ErrDeviceNotRegistered, "Device not found")
			return
		}

		// 2. Verify device is active
		if !device.IsActive() {
			response.Forbidden(w, response.ErrDeviceNotRegistered,
				"Device is "+string(device.Status))
			return
		}

		// Store device in context for downstream layers
		ctx := SetDeviceA(r.Context(), device)

		// 3. Decode attestation signature
		attSigBytes, err := base64.StdEncoding.DecodeString(sideA.AttestationSignature)
		if err != nil {
			response.BadRequest(w, response.ErrInvalidECDSA, "Invalid attestation signature encoding")
			return
		}

		// 4. Build payload hash (same fields as HMAC)
		payloadHash := crypto.BuildSideAPayloadHash(
			sideA.WalletID, sideA.DeviceID, sideA.Ctr,
			sideA.OperationType, sideA.Currency,
			sideA.Amount, sideA.Nonce, sideA.Timestamp,
		)

		// 5. Verify ECDSA signature using device's attestation public key
		err = crypto.VerifyECDSASignature(device.AttestationPublicKey, payloadHash, attSigBytes)
		if err != nil {
			slog.Warn("Attestation ECDSA verification failed",
				"device_id", sideA.DeviceID,
				"attestation_level", device.AttestationLevel,
				"error", err,
			)
			response.Forbidden(w, response.ErrInvalidECDSA,
				"Hardware attestation signature invalid")
			return
		}

		slog.Debug("Attestation verified",
			"device_id", sideA.DeviceID,
			"level", device.AttestationLevel,
		)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
