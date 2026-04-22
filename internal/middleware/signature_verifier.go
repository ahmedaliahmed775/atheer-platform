package middleware

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"

	"github.com/atheer-payment/atheer-platform/internal/repository"
	"github.com/atheer-payment/atheer-platform/pkg/crypto"
	"github.com/atheer-payment/atheer-platform/pkg/response"
)

// SignatureVerifierA is Layer 5 of the transaction pipeline
// Verifies Side A's HMAC-SHA256 signature using deviceSeed + counter
// Must match SDK's SignatureUtils.sign(luk, payloadHash) exactly
type SignatureVerifierA struct {
	deviceRepo *repository.DeviceRepository
}

func NewSignatureVerifierA(deviceRepo *repository.DeviceRepository) *SignatureVerifierA {
	return &SignatureVerifierA{deviceRepo: deviceRepo}
}

func (sv *SignatureVerifierA) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		sideA := &req.SideA

		// 1. Get device from DB (may already be in context from attestation layer)
		device := GetDeviceA(r.Context())
		if device == nil {
			var err error
			device, err = sv.deviceRepo.GetByDeviceID(r.Context(), sideA.DeviceID)
			if err != nil {
				response.Forbidden(w, response.ErrDeviceNotRegistered, "Device not found")
				return
			}
			r = r.WithContext(SetDeviceA(r.Context(), device))
		}

		// 2. Verify device is active
		if !device.IsActive() {
			response.Forbidden(w, response.ErrDeviceNotRegistered, "Device is "+string(device.Status))
			return
		}

		// 3. Decrypt device seed (TODO: use KMS in production)
		deviceSeed := device.DeviceSeed

		// 4. Decode the signature from base64
		sigBytes, err := base64.StdEncoding.DecodeString(sideA.Signature)
		if err != nil {
			response.BadRequest(w, response.ErrInvalidSignature, "Invalid signature encoding")
			return
		}

		// 5. Verify HMAC signature
		// ⚠️ CRITICAL: Field order MUST match SDK's SignatureUtils.buildPayloadString()
		err = crypto.VerifyHMACSignature(
			deviceSeed, sideA.Ctr,
			sideA.WalletID, sideA.DeviceID,
			sideA.OperationType, sideA.Currency,
			sideA.Amount, sideA.Nonce, sideA.Timestamp,
			sigBytes,
		)
		if err != nil {
			slog.Warn("Side A HMAC signature verification failed",
				"device_id", sideA.DeviceID,
				"ctr", sideA.Ctr,
				"error", err,
			)
			response.Forbidden(w, response.ErrInvalidSignature, "Side A signature invalid")
			return
		}

		slog.Debug("Side A signature verified",
			"device_id", sideA.DeviceID,
			"ctr", sideA.Ctr,
		)
		next.ServeHTTP(w, r)
	})
}

// SignatureVerifierB is Layer 6 of the transaction pipeline
// Verifies Side B's HMAC-SHA256 signature
type SignatureVerifierB struct {
	deviceRepo *repository.DeviceRepository
}

func NewSignatureVerifierB(deviceRepo *repository.DeviceRepository) *SignatureVerifierB {
	return &SignatureVerifierB{deviceRepo: deviceRepo}
}

func (sv *SignatureVerifierB) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := GetCombinedRequest(r.Context())
		if req == nil {
			response.BadRequest(w, response.ErrInternalError, "Request not parsed")
			return
		}

		sideB := &req.SideB

		// 1. Get Side B device
		deviceB, err := sv.deviceRepo.GetByDeviceID(r.Context(), sideB.DeviceID)
		if err != nil {
			response.Forbidden(w, response.ErrDeviceNotRegistered, "Side B device not found")
			return
		}

		if !deviceB.IsActive() {
			response.Forbidden(w, response.ErrDeviceNotRegistered, "Side B device is "+string(deviceB.Status))
			return
		}

		// Store device B in context
		ctx := r.Context()
		ctx = context.WithValue(ctx, deviceBKey, deviceB)

		// 2. Decode signature
		sigBytes, err := base64.StdEncoding.DecodeString(sideB.Signature)
		if err != nil {
			response.BadRequest(w, response.ErrInvalidSignature, "Invalid Side B signature encoding")
			return
		}

		// 3. Derive LUK for Side B (Side B doesn't send ctr, use device's current ctr)
		deviceSeed := deviceB.DeviceSeed
		luk := crypto.DeriveLUK(deviceSeed, deviceB.Ctr)

		// 4. Build Side B payload hash
		payloadHash := crypto.BuildSideBPayloadHash(
			sideB.WalletID, sideB.DeviceID, sideB.MerchantID,
			sideB.OperationType, sideB.Currency,
			sideB.Amount, sideB.AccountID, sideB.Timestamp,
		)

		// 5. Verify
		expectedSig := crypto.SignPayload(luk, payloadHash)
		if !crypto.TimingSafeEqual(expectedSig, sigBytes) {
			slog.Warn("Side B HMAC signature verification failed",
				"device_id", sideB.DeviceID,
			)
			response.Forbidden(w, response.ErrInvalidSignature, "Side B signature invalid")
			return
		}

		slog.Debug("Side B signature verified", "device_id", sideB.DeviceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
