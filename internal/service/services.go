package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/repository"
)

// EnrollParams contains enrollment request parameters
type EnrollParams struct {
	WalletID           string
	AccountID          string
	ECPublicKey        string
	AttestationPubKey  string
	AttestationCert    string
	PlayIntegrityToken string
}

// EnrollResult contains enrollment result data
type EnrollResult struct {
	DeviceID         string
	DeviceSeedEncoded string
	AttestationLevel model.AttestationLevel
}

// EnrollmentService handles device enrollment with hardware attestation
type EnrollmentService struct {
	deviceRepo *repository.DeviceRepository
}

// NewEnrollmentService creates a new EnrollmentService
func NewEnrollmentService(deviceRepo *repository.DeviceRepository) *EnrollmentService {
	return &EnrollmentService{deviceRepo: deviceRepo}
}

// Enroll registers a new device with hardware attestation
// Called when SDK executes: AtheerSdk.enroll(accountId)
func (s *EnrollmentService) Enroll(ctx context.Context, params *EnrollParams) (*EnrollResult, error) {
	// 1. TODO: Verify Play Integrity token via Google API
	//    playIntegrityResult, err := s.attestationVerifier.VerifyPlayIntegrity(ctx, params.PlayIntegrityToken)

	// 2. TODO: Verify attestation certificate chain
	//    attestationLevel, err := s.attestationVerifier.VerifyCertificateChain(params.AttestationCert)

	// For now, default to TEE level
	attestationLevel := model.AttestationLevelTEE

	// 3. Generate device seed (256-bit random)
	deviceSeed := make([]byte, 32)
	if _, err := rand.Read(deviceSeed); err != nil {
		return nil, fmt.Errorf("failed to generate device seed: %w", err)
	}

	// 4. TODO: Encrypt device seed with KMS before storage
	//    encryptedSeed, err := s.kms.Encrypt(deviceSeed)
	// For now, store as-is (MUST use KMS in production!)
	encryptedSeed := deviceSeed

	// 5. Generate device ID
	deviceID := uuid.New().String()

	// 6. Store device record
	device := &model.Device{
		DeviceID:             deviceID,
		WalletID:             params.WalletID,
		AccountID:            params.AccountID,
		DeviceSeed:           encryptedSeed,
		Ctr:                  0,
		ECPublicKey:          params.ECPublicKey,
		AttestationPublicKey: params.AttestationPubKey,
		AttestationLevel:     attestationLevel,
		Status:               model.DeviceStatusActive,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	slog.Info("Device enrolled successfully",
		"deviceId", deviceID,
		"walletId", params.WalletID,
		"attestationLevel", attestationLevel,
	)

	return &EnrollResult{
		DeviceID:          deviceID,
		DeviceSeedEncoded: base64.StdEncoding.EncodeToString(deviceSeed),
		AttestationLevel:  attestationLevel,
	}, nil
}

// DeviceService handles device management operations
type DeviceService struct {
	deviceRepo *repository.DeviceRepository
}

func NewDeviceService(deviceRepo *repository.DeviceRepository) *DeviceService {
	return &DeviceService{deviceRepo: deviceRepo}
}

func (s *DeviceService) GetDevice(ctx context.Context, deviceID string) (*model.Device, error) {
	return s.deviceRepo.GetByDeviceID(ctx, deviceID)
}

func (s *DeviceService) UpdateStatus(ctx context.Context, deviceID string, status model.DeviceStatus) error {
	return s.deviceRepo.UpdateStatus(ctx, deviceID, status)
}

// TransactionService handles transaction processing
type TransactionService struct {
	txRepo       *repository.TransactionRepository
	deviceRepo   *repository.DeviceRepository
}

func NewTransactionService(
	txRepo *repository.TransactionRepository,
	deviceRepo *repository.DeviceRepository,
) *TransactionService {
	return &TransactionService{
		txRepo:     txRepo,
		deviceRepo: deviceRepo,
	}
}

func (s *TransactionService) ProcessTransaction(ctx context.Context, req *model.PayerTlvPacket, channel string) (*model.Transaction, error) {
	txID := uuid.New().String()

	amount := float64(req.Amount)

	tx := &model.Transaction{
		TxID:            txID,
		PayerPublicID:   req.PublicID,
		PayerUserID:     "unknown", // To be fetched
		PayerType:       model.UserTypePersonal,
		PayeeID:         req.ReceiverID,
		PayeeType:       req.PayeeType,
		TransactionType: model.TxP2P, // Default
		Amount:          req.Amount,
		Currency:        "SAR",
		WalletID:        "unknown",
		Status:          model.TxStatusPending,
		Counter:         req.Counter,
	}

	// Create transaction record
	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// TODO Phase 3: Execute via Saga (Temporal)
	//   1. Debit Side A via Adapter
	//   2. Credit Side B via Adapter
	//   3. Send SMS notifications
	//   4. Update status to COMPLETED

	// For now, mark as completed (will be replaced with Saga)
	completedStatus := model.TxStatusCompleted
	s.txRepo.UpdateStatus(ctx, txID, completedStatus, nil, nil)
	tx.Status = completedStatus

	slog.Info("Transaction processed",
		"txId", txID,
		"amount", amount,
		"channel", channel,
	)

	return tx, nil
}

// GetStatus returns the status of a transaction
func (s *TransactionService) GetStatus(ctx context.Context, txID string) (*model.Transaction, error) {
	return s.txRepo.GetByTxID(ctx, txID)
}

// DisputeService handles dispute operations
type DisputeService struct {
	disputeRepo *repository.DisputeRepository
	txRepo      *repository.TransactionRepository
}

func NewDisputeService(disputeRepo *repository.DisputeRepository, txRepo *repository.TransactionRepository) *DisputeService {
	return &DisputeService{disputeRepo: disputeRepo, txRepo: txRepo}
}

func (s *DisputeService) OpenDispute(ctx context.Context, txID, reason, openedBy string) (*model.Dispute, error) {
	dispute := &model.Dispute{
		TxID:     txID,
		Reason:   reason,
		Status:   model.DisputeStatusOpen,
		OpenedBy: openedBy,
	}
	if err := s.disputeRepo.Create(ctx, dispute); err != nil {
		return nil, err
	}
	// Mark transaction as disputed
	s.txRepo.UpdateStatus(ctx, txID, model.TxStatusDisputed, nil, nil)
	return dispute, nil
}

func (s *DisputeService) ListDisputes(ctx context.Context) ([]*model.Dispute, error) {
	return s.disputeRepo.ListOpen(ctx)
}

// LimitsService handles limits checking
type LimitsService struct {
	limitsRepo *repository.LimitsMatrixRepository
}

func NewLimitsService(limitsRepo *repository.LimitsMatrixRepository) *LimitsService {
	return &LimitsService{limitsRepo: limitsRepo}
}

func (s *LimitsService) GetLimits(ctx context.Context, walletID, opType string) (interface{}, error) {
	if walletID == "" {
		return s.limitsRepo.ListAll(ctx)
	}
	return s.limitsRepo.GetLimits(ctx, walletID, opType, "", "basic")
}

func (s *LimitsService) UpdateLimits(ctx context.Context, lm *model.LimitsMatrix) error {
	return s.limitsRepo.Upsert(ctx, lm)
}
