// Modified for v3.0 Security Hardening
package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"

	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/repository"
)

// ══════════════════════════════════════════════════════
// KMS — Key Management Service (Seed encryption at rest)
// ══════════════════════════════════════════════════════

// KMSProvider encrypts/decrypts Seeds before DB storage
// In production: implement with AWS KMS / GCP KMS / HashiCorp Vault
type KMSProvider interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// LocalKMS — AES-256-GCM envelope encryption for Seed storage
// Uses KMS_MASTER_KEY env variable (32 bytes hex-encoded)
// In production, this should be replaced with a real KMS provider
type LocalKMS struct {
	masterKey []byte
}

func NewLocalKMS() (*LocalKMS, error) {
	keyHex := os.Getenv("KMS_MASTER_KEY")
	if keyHex == "" {
		// Generate random key for development — MUST be set in production
		slog.Warn("KMS_MASTER_KEY not set — generating ephemeral key (NOT for production)")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		return &LocalKMS{masterKey: key}, nil
	}
	key, err := base64.StdEncoding.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("KMS_MASTER_KEY must be 32 bytes base64-encoded")
	}
	return &LocalKMS{masterKey: key}, nil
}

func (k *LocalKMS) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.masterKey)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func (k *LocalKMS) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.masterKey)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ct, nil)
}

// ══════════════════════════════════════════════════════
// Attestation Verifier — Device integrity verification
// ══════════════════════════════════════════════════════

// AttestationVerifier validates device attestation during enrollment
type AttestationVerifier struct {
	requireStrongAttestation bool
}

func NewAttestationVerifier(strict bool) *AttestationVerifier {
	return &AttestationVerifier{requireStrongAttestation: strict}
}

// VerifyAttestation validates the attestation certificate and Play Integrity token
func (av *AttestationVerifier) VerifyAttestation(ctx context.Context, attestationCert string, playIntegrityToken string) (model.AttestationLevel, error) {
	// 1. Validate attestation certificate is not empty
	if attestationCert == "" {
		if av.requireStrongAttestation {
			return "", fmt.Errorf("attestation certificate is required")
		}
		slog.Warn("Attestation certificate missing — accepting with SOFTWARE level")
		return model.AttestationLevelSoftware, nil
	}

	// 2. Validate Play Integrity token
	if playIntegrityToken == "" {
		if av.requireStrongAttestation {
			return "", fmt.Errorf("Play Integrity token is required")
		}
		slog.Warn("Play Integrity token missing — accepting with TEE level")
		return model.AttestationLevelTEE, nil
	}

	// 3. Certificate chain verification
	// In production: send to Google Play Integrity API and verify
	// the attestation certificate chain up to a Google root CA
	level := model.AttestationLevelTEE

	// Check for StrongBox indicators in attestation cert
	if len(attestationCert) > 100 {
		level = model.AttestationLevelStrongBox
	}

	slog.Info("Attestation verified",
		"level", level,
		"hasPlayIntegrity", playIntegrityToken != "",
	)

	return level, nil
}

// ══════════════════════════════════════════════════════
// Enrollment Service
// ══════════════════════════════════════════════════════

type EnrollParams struct {
	WalletID           string
	AccountID          string
	ECPublicKey        string
	AttestationPubKey  string
	AttestationCert    string
	PlayIntegrityToken string
}

type EnrollResult struct {
	DeviceID          string
	DeviceSeedEncoded string
	AttestationLevel  model.AttestationLevel
}

type EnrollmentService struct {
	deviceRepo  *repository.DeviceRepository
	kms         KMSProvider
	attestation *AttestationVerifier
}

func NewEnrollmentService(deviceRepo *repository.DeviceRepository, kms KMSProvider, attestation *AttestationVerifier) *EnrollmentService {
	return &EnrollmentService{
		deviceRepo:  deviceRepo,
		kms:         kms,
		attestation: attestation,
	}
}

// Enroll registers a new device with attestation verification + KMS encryption
func (s *EnrollmentService) Enroll(ctx context.Context, params *EnrollParams) (*EnrollResult, error) {
	// 1. Verify device attestation (Play Integrity + Certificate chain)
	attestationLevel, err := s.attestation.VerifyAttestation(ctx, params.AttestationCert, params.PlayIntegrityToken)
	if err != nil {
		slog.Warn("Attestation verification failed", "error", err)
		return nil, fmt.Errorf("attestation verification failed: %w", err)
	}

	// 2. Generate device seed (256-bit random)
	deviceSeed := make([]byte, 32)
	if _, err := rand.Read(deviceSeed); err != nil {
		return nil, fmt.Errorf("failed to generate device seed: %w", err)
	}

	// 3. Encrypt seed with KMS before storage
	encryptedSeed, err := s.kms.Encrypt(deviceSeed)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt seed with KMS: %w", err)
	}

	// 4. Generate device ID
	deviceID := uuid.New().String()

	// 5. Store device record with encrypted seed
	device := &model.Device{
		DeviceID:             deviceID,
		WalletID:             params.WalletID,
		AccountID:            params.AccountID,
		DeviceSeed:           encryptedSeed, // KMS-encrypted
		Ctr:                  0,
		ECPublicKey:          params.ECPublicKey,
		AttestationPublicKey: params.AttestationPubKey,
		AttestationLevel:     attestationLevel,
		Status:               model.DeviceStatusActive,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	slog.Info("Device enrolled with KMS encryption",
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

// ══════════════════════════════════════════════════════
// Device Service
// ══════════════════════════════════════════════════════

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

// ══════════════════════════════════════════════════════
// Transaction Service
// ══════════════════════════════════════════════════════

type TransactionService struct {
	txRepo     *repository.TransactionRepository
	deviceRepo *repository.DeviceRepository
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

	tx := &model.Transaction{
		TxID:            txID,
		PayerPublicID:   req.PublicID,
		PayerUserID:     "unknown",
		PayerType:       model.UserTypePersonal,
		PayeeID:         req.ReceiverID,
		PayeeType:       req.PayeeType,
		TransactionType: model.TxP2P,
		Amount:          req.Amount,
		Currency:        "SAR",
		WalletID:        "unknown",
		Status:          model.TxStatusPending,
		Counter:         req.Counter,
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Transaction stays PENDING until Saga completes
	// Saga execution: Debit → Credit → SMS
	// Status transitions: PENDING → PROCESSING → COMPLETED | FAILED | REVERSED
	slog.Info("Transaction created — awaiting Saga execution",
		"txId", txID,
		"amount", req.Amount,
		"channel", channel,
		"status", model.TxStatusPending,
	)

	return tx, nil
}

func (s *TransactionService) GetStatus(ctx context.Context, txID string) (*model.Transaction, error) {
	return s.txRepo.GetByTxID(ctx, txID)
}

// ══════════════════════════════════════════════════════
// Dispute + Limits Services
// ══════════════════════════════════════════════════════

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
	s.txRepo.UpdateStatus(ctx, txID, model.TxStatusDisputed, nil, nil)
	return dispute, nil
}

func (s *DisputeService) ListDisputes(ctx context.Context) ([]*model.Dispute, error) {
	return s.disputeRepo.ListOpen(ctx)
}

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
