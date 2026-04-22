package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// DeviceRepository handles database operations for devices
type DeviceRepository struct {
	db *pgxpool.Pool
}

// NewDeviceRepository creates a new DeviceRepository
func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create inserts a new device record
func (r *DeviceRepository) Create(ctx context.Context, device *model.Device) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO devices (
			device_id, wallet_id, account_id, device_seed, ctr,
			ec_public_key, attestation_public_key, attestation_level, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		device.DeviceID, device.WalletID, device.AccountID,
		device.DeviceSeed, device.Ctr,
		device.ECPublicKey, device.AttestationPublicKey,
		device.AttestationLevel, device.Status,
	)
	return err
}

// GetByDeviceID retrieves a device by its device ID
func (r *DeviceRepository) GetByDeviceID(ctx context.Context, deviceID string) (*model.Device, error) {
	device := &model.Device{}
	err := r.db.QueryRow(ctx, `
		SELECT id, device_id, wallet_id, account_id, device_seed, ctr,
		       ec_public_key, attestation_public_key, attestation_level,
		       status, enrolled_at, last_tx_at
		FROM devices WHERE device_id = $1`, deviceID,
	).Scan(
		&device.ID, &device.DeviceID, &device.WalletID, &device.AccountID,
		&device.DeviceSeed, &device.Ctr,
		&device.ECPublicKey, &device.AttestationPublicKey, &device.AttestationLevel,
		&device.Status, &device.EnrolledAt, &device.LastTxAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}
	return device, err
}

// GetByWalletAndAccount retrieves devices by wallet and account
func (r *DeviceRepository) GetByWalletAndAccount(ctx context.Context, walletID, accountID string) ([]*model.Device, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, device_id, wallet_id, account_id, device_seed, ctr,
		       ec_public_key, attestation_public_key, attestation_level,
		       status, enrolled_at, last_tx_at
		FROM devices WHERE wallet_id = $1 AND account_id = $2 AND status = 'ACTIVE'`,
		walletID, accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		d := &model.Device{}
		if err := rows.Scan(
			&d.ID, &d.DeviceID, &d.WalletID, &d.AccountID,
			&d.DeviceSeed, &d.Ctr,
			&d.ECPublicKey, &d.AttestationPublicKey, &d.AttestationLevel,
			&d.Status, &d.EnrolledAt, &d.LastTxAt,
		); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// UpdateStatus updates the status of a device
func (r *DeviceRepository) UpdateStatus(ctx context.Context, deviceID string, status model.DeviceStatus) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE devices SET status = $1 WHERE device_id = $2`,
		status, deviceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device not found: %s", deviceID)
	}
	return nil
}

// IncrementCounter atomically increments the counter and updates last_tx_at
func (r *DeviceRepository) IncrementCounter(ctx context.Context, deviceID string, newCtr int64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE devices SET ctr = $1, last_tx_at = NOW()
		WHERE device_id = $2 AND ctr < $1 AND status = 'ACTIVE'`,
		newCtr, deviceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("counter update failed: device=%s newCtr=%d (replay or inactive)", deviceID, newCtr)
	}
	return nil
}
