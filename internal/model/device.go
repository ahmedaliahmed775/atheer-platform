package model

import (
	"time"

	"github.com/google/uuid"
)

// DeviceStatus represents the status of a registered device
type DeviceStatus string

const (
	DeviceStatusActive    DeviceStatus = "ACTIVE"
	DeviceStatusSuspended DeviceStatus = "SUSPENDED"
	DeviceStatusRevoked   DeviceStatus = "REVOKED"
)

// AttestationLevel represents the hardware security level
type AttestationLevel string

const (
	AttestationLevelStrongBox AttestationLevel = "STRONGBOX"
	AttestationLevelTEE       AttestationLevel = "TEE"
	AttestationLevelSoftware  AttestationLevel = "SOFTWARE"
)

// Device represents a registered device in the system
type Device struct {
	ID                   uuid.UUID        `json:"id"                    db:"id"`
	DeviceID             string           `json:"deviceId"              db:"device_id"`
	WalletID             string           `json:"walletId"              db:"wallet_id"`
	AccountID            string           `json:"accountId"             db:"account_id"`
	DeviceSeed           []byte           `json:"-"                     db:"device_seed"` // KMS-encrypted — never exposed
	Ctr                  int64            `json:"ctr"                   db:"ctr"`
	ECPublicKey          string           `json:"ecPublicKey"           db:"ec_public_key"`
	AttestationPublicKey string           `json:"attestationPublicKey"  db:"attestation_public_key"`
	AttestationLevel     AttestationLevel `json:"attestationLevel"      db:"attestation_level"`
	Status               DeviceStatus     `json:"status"                db:"status"`
	EnrolledAt           time.Time        `json:"enrolledAt"            db:"enrolled_at"`
	LastTxAt             *time.Time       `json:"lastTxAt,omitempty"    db:"last_tx_at"`
}

// IsActive returns true if the device can process transactions
func (d *Device) IsActive() bool {
	return d.Status == DeviceStatusActive
}
