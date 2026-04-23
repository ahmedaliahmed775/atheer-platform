// Modified for v3.0 Document Alignment
// Repository for SwitchRecord — replaces DeviceRepository for v3.0
package repository

import (
    "context"
    "fmt"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/atheer-payment/atheer-platform/internal/model"
)

// SwitchRecordRepository handles CRUD for SwitchRecord
type SwitchRecordRepository struct {
    db *pgxpool.Pool
}

func NewSwitchRecordRepository(db *pgxpool.Pool) *SwitchRecordRepository {
    return &SwitchRecordRepository{db: db}
}

// GetByPublicID retrieves a SwitchRecord by PublicID
func (r *SwitchRecordRepository) GetByPublicID(ctx context.Context, publicID string) (*model.SwitchRecord, error) {
    var record model.SwitchRecord
    err := r.db.QueryRow(ctx,
        `SELECT public_id, seed, user_id, user_type, wallet_id, counter, status
         FROM switch_records WHERE public_id = $1`, publicID).Scan(
        &record.PublicID, &record.Seed, &record.UserID,
        &record.UserType, &record.WalletID, &record.Counter, &record.Status,
    )
    if err != nil {
        return nil, fmt.Errorf("switch record not found for publicId=%s: %w", publicID, err)
    }
    return &record, nil
}

// GetByUserID retrieves a SwitchRecord by UserID
func (r *SwitchRecordRepository) GetByUserID(ctx context.Context, userID string) (*model.SwitchRecord, error) {
    var record model.SwitchRecord
    err := r.db.QueryRow(ctx,
        `SELECT public_id, seed, user_id, user_type, wallet_id, counter, status
         FROM switch_records WHERE user_id = $1`, userID).Scan(
        &record.PublicID, &record.Seed, &record.UserID,
        &record.UserType, &record.WalletID, &record.Counter, &record.Status,
    )
    if err != nil {
        return nil, fmt.Errorf("switch record not found for userId=%s: %w", userID, err)
    }
    return &record, nil
}

// IncrementCounter atomically increments the counter and returns the new value
func (r *SwitchRecordRepository) IncrementCounter(ctx context.Context, publicID string) (uint64, error) {
    var newCounter uint64
    err := r.db.QueryRow(ctx,
        `UPDATE switch_records SET counter = counter + 1 WHERE public_id = $1 RETURNING counter`,
        publicID).Scan(&newCounter)
    if err != nil {
        return 0, fmt.Errorf("failed to increment counter: %w", err)
    }
    return newCounter, nil
}
