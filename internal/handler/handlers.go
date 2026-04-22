package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/service"
	"github.com/atheer-payment/atheer-platform/pkg/response"
)

// EnrollRequest matches SDK's enrollment POST body
type EnrollRequest struct {
	WalletID           string `json:"walletId"            validate:"required"`
	AccountID          string `json:"accountId"           validate:"required"`
	ECPublicKey        string `json:"ecPublicKey"         validate:"required"`
	AttestationPubKey  string `json:"attestationPublicKey" validate:"required"`
	AttestationCert    string `json:"attestationCert"     validate:"required"`
	PlayIntegrityToken string `json:"playIntegrityToken"  validate:"required"`
}

// EnrollResponse is returned to SDK after successful enrollment
type EnrollResponse struct {
	DeviceSeed       string `json:"deviceSeed"`
	Ctr              int64  `json:"ctr"`
	DeviceID         string `json:"deviceId"`
	WalletID         string `json:"walletId"`
	AttestationLevel string `json:"attestationLevel"`
}

// EnrollHandler handles device enrollment
type EnrollHandler struct {
	enrollService *service.EnrollmentService
}

// NewEnrollHandler creates a new EnrollHandler
func NewEnrollHandler(es *service.EnrollmentService) *EnrollHandler {
	return &EnrollHandler{enrollService: es}
}

// Enroll handles POST /api/v2/enroll/
// Called by SDK: AtheerSdk.enroll(accountId)
func (h *EnrollHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	var req EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, response.ErrInternalError, "Invalid request body")
		return
	}

	// Validate required fields
	if req.WalletID == "" || req.AccountID == "" || req.ECPublicKey == "" ||
		req.AttestationPubKey == "" || req.AttestationCert == "" || req.PlayIntegrityToken == "" {
		response.BadRequest(w, response.ErrInternalError, "Missing required fields")
		return
	}

	result, err := h.enrollService.Enroll(r.Context(), &service.EnrollParams{
		WalletID:           req.WalletID,
		AccountID:          req.AccountID,
		ECPublicKey:        req.ECPublicKey,
		AttestationPubKey:  req.AttestationPubKey,
		AttestationCert:    req.AttestationCert,
		PlayIntegrityToken: req.PlayIntegrityToken,
	})
	if err != nil {
		slog.Error("Enrollment failed", "error", err, "walletId", req.WalletID)
		response.InternalError(w, "Enrollment failed")
		return
	}

	slog.Info("Device enrolled",
		"deviceId", result.DeviceID,
		"walletId", req.WalletID,
		"attestationLevel", result.AttestationLevel,
	)

	response.Created(w, EnrollResponse{
		DeviceSeed:       result.DeviceSeedEncoded,
		Ctr:              0,
		DeviceID:         result.DeviceID,
		WalletID:         req.WalletID,
		AttestationLevel: string(result.AttestationLevel),
	})
}

// RotateKey handles POST /api/v2/key/rotate/
func (h *EnrollHandler) RotateKey(w http.ResponseWriter, r *http.Request) {
	// TODO: Phase 2 — key rotation implementation
	response.OK(w, map[string]string{"status": "not_implemented"})
}

// TransactionHandler handles transaction processing
type TransactionHandler struct {
	txService *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(ts *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txService: ts}
}

// Process handles POST /api/v2/transaction/
// Called by SDK: SideBProcessor sends CombinedRequest after NFC tap
func (h *TransactionHandler) Process(w http.ResponseWriter, r *http.Request) {
	var req model.CombinedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, response.ErrInternalError, "Invalid request body")
		return
	}

	// Detect channel from header (SDK sends X-Channel: APN or INTERNET)
	channel := r.Header.Get("X-Channel")
	if channel == "" {
		channel = "APN"
	}

	result, err := h.txService.ProcessTransaction(r.Context(), &req, model.Channel(channel))
	if err != nil {
		slog.Error("Transaction processing failed",
			"error", err,
			"nonce", req.SideA.Nonce,
		)
		response.InternalError(w, "Transaction processing failed")
		return
	}

	response.OK(w, result)
}

// GetStatus handles GET /api/v2/transaction/{txId}
// Called by SDK: AtheerSdk.getTransactionStatus(txId)
func (h *TransactionHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	txId := r.PathValue("txId")
	if txId == "" {
		response.BadRequest(w, response.ErrInternalError, "txId is required")
		return
	}

	tx, err := h.txService.GetStatus(r.Context(), txId)
	if err != nil {
		response.NotFound(w, "Transaction not found")
		return
	}

	response.OK(w, tx)
}

// DeviceHandler handles device management
type DeviceHandler struct {
	deviceService *service.DeviceService
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(ds *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: ds}
}

// GetDevice handles GET /api/v2/device/{deviceId}
func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	deviceId := r.PathValue("deviceId")
	device, err := h.deviceService.GetDevice(r.Context(), deviceId)
	if err != nil {
		response.NotFound(w, "Device not found")
		return
	}
	response.OK(w, device)
}

// Suspend handles POST /api/v2/device/{deviceId}/suspend
func (h *DeviceHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	deviceId := r.PathValue("deviceId")
	if err := h.deviceService.UpdateStatus(r.Context(), deviceId, model.DeviceStatusSuspended); err != nil {
		response.InternalError(w, "Failed to suspend device")
		return
	}
	response.OK(w, map[string]string{"status": "suspended", "deviceId": deviceId})
}

// Revoke handles POST /api/v2/device/{deviceId}/revoke
func (h *DeviceHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	deviceId := r.PathValue("deviceId")
	if err := h.deviceService.UpdateStatus(r.Context(), deviceId, model.DeviceStatusRevoked); err != nil {
		response.InternalError(w, "Failed to revoke device")
		return
	}
	response.OK(w, map[string]string{"status": "revoked", "deviceId": deviceId})
}

// DisputeHandler handles dispute management
type DisputeHandler struct {
	disputeService *service.DisputeService
}

// NewDisputeHandler creates a new DisputeHandler
func NewDisputeHandler(ds *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{disputeService: ds}
}

// Open handles POST /api/v2/dispute/
func (h *DisputeHandler) Open(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxID     string `json:"txId"     validate:"required"`
		Reason   string `json:"reason"   validate:"required"`
		OpenedBy string `json:"openedBy" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, response.ErrInternalError, "Invalid request body")
		return
	}

	dispute, err := h.disputeService.OpenDispute(r.Context(), req.TxID, req.Reason, req.OpenedBy)
	if err != nil {
		response.InternalError(w, "Failed to open dispute")
		return
	}
	response.Created(w, dispute)
}

// List handles GET /api/v2/dispute/
func (h *DisputeHandler) List(w http.ResponseWriter, r *http.Request) {
	disputes, err := h.disputeService.ListDisputes(r.Context())
	if err != nil {
		response.InternalError(w, "Failed to list disputes")
		return
	}
	response.OK(w, disputes)
}

// Update handles PUT /api/v2/dispute/{disputeId}
func (h *DisputeHandler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: implement dispute status update
	response.OK(w, map[string]string{"status": "not_implemented"})
}

// LimitsHandler handles limits matrix management
type LimitsHandler struct {
	limitsService *service.LimitsService
}

// NewLimitsHandler creates a new LimitsHandler
func NewLimitsHandler(ls *service.LimitsService) *LimitsHandler {
	return &LimitsHandler{limitsService: ls}
}

// Get handles GET /api/v2/limits/
func (h *LimitsHandler) Get(w http.ResponseWriter, r *http.Request) {
	walletId := r.URL.Query().Get("walletId")
	opType := r.URL.Query().Get("operationType")

	limits, err := h.limitsService.GetLimits(r.Context(), walletId, opType)
	if err != nil {
		response.InternalError(w, "Failed to get limits")
		return
	}
	response.OK(w, limits)
}

// Update handles PUT /api/v2/limits/
func (h *LimitsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req model.LimitsMatrix
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, response.ErrInternalError, "Invalid request body")
		return
	}

	if err := h.limitsService.UpdateLimits(r.Context(), &req); err != nil {
		response.InternalError(w, "Failed to update limits")
		return
	}
	response.OK(w, req)
}
