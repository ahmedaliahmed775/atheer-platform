package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/adapter"
	"github.com/atheer-payment/atheer-platform/internal/model"
	"github.com/atheer-payment/atheer-platform/internal/repository"
)

// SagaService implements the Saga pattern for distributed transactions
// Ensures atomicity across Debit → Credit with compensation (reversal) on failure
type SagaService struct {
	adapterRegistry *adapter.Registry
	txRepo          *repository.TransactionRepository
	pendingRepo     *repository.PendingOperationRepository
	deviceRepo      *repository.DeviceRepository
}

func NewSagaService(
	registry *adapter.Registry,
	txRepo *repository.TransactionRepository,
	pendingRepo *repository.PendingOperationRepository,
	deviceRepo *repository.DeviceRepository,
) *SagaService {
	return &SagaService{
		adapterRegistry: registry,
		txRepo:          txRepo,
		pendingRepo:     pendingRepo,
		deviceRepo:      deviceRepo,
	}
}

// SagaRequest contains all info needed to execute a Saga
type SagaRequest struct {
	TxID           string
	Nonce          string
	SideAWalletID  string
	SideAAccountID string
	SideADeviceID  string
	SideBWalletID  string
	SideBAccountID string
	SideBDeviceID  string
	MerchantID     *string
	TransactionType  model.TransactionType
	Currency       string
	Amount         decimal.Decimal
	Channel        string
	SideACtr       int64
}

// SagaResult contains the outcome of a Saga execution
type SagaResult struct {
	TxID    string                `json:"txId"`
	Status  model.TransactionStatus `json:"status"`
	Error   string                `json:"error,omitempty"`
}

// Execute runs the full Saga: Debit → Credit → Notify
// If Credit fails, it compensates by reversing the Debit
func (s *SagaService) Execute(ctx context.Context, req *SagaRequest) (*SagaResult, error) {
	slog.Info("Saga started",
		"txId", req.TxID,
		"txType", req.TransactionType,
		"amount", req.Amount,
	)

	// Determine which adapter to use for each side
	adapterABase, ok := s.adapterRegistry.GetAdapter(req.SideAWalletID)
	if !ok {
		return s.failTx(ctx, req.TxID, "E009", "No adapter for wallet: "+req.SideAWalletID)
	}

	adapterBBase, ok := s.adapterRegistry.GetAdapter(req.SideBWalletID)
	if !ok {
		return s.failTx(ctx, req.TxID, "E009", "No adapter for wallet: "+req.SideBWalletID)
	}

	// Assert SagaExecutor capability (internal interface — not in document)
	adapterA, ok := adapterABase.(adapter.SagaExecutor)
	if !ok {
		return s.failTx(ctx, req.TxID, "E009", "Adapter does not support Saga: "+req.SideAWalletID)
	}
	adapterB, ok := adapterBBase.(adapter.SagaExecutor)
	if !ok {
		return s.failTx(ctx, req.TxID, "E009", "Adapter does not support Saga: "+req.SideBWalletID)
	}

	// ─── Step 1: DEBIT Side A ───
	debitOp := &model.PendingOperation{
		ID:        uuid.New(),
		TxID:      req.TxID,
		OpType:    model.PendingOpDebit,
		AdapterID: adapterABase.WalletID(),
		WalletID:  req.SideAWalletID,
		AccountID: req.SideAAccountID,
		Amount:    req.Amount,
		Status:    model.PendingOpStatusPending,
	}
	s.pendingRepo.Create(ctx, debitOp)

	debitResult, err := adapterA.Debit(ctx, req.SideAWalletID, req.SideAAccountID, req.Amount, req.TxID)
	if err != nil || !debitResult.Success {
		errMsg := "Debit failed"
		if err != nil {
			errMsg = err.Error()
		}
		s.pendingRepo.UpdateStatus(ctx, debitOp.ID, model.PendingOpStatusFailed, errMsg)
		return s.failTx(ctx, req.TxID, "E008", errMsg)
	}
	s.pendingRepo.UpdateStatus(ctx, debitOp.ID, model.PendingOpStatusDone, "")

	slog.Info("Saga: Debit completed", "txId", req.TxID)

	// ─── Step 2: CREDIT Side B ───
	creditOp := &model.PendingOperation{
		ID:        uuid.New(),
		TxID:      req.TxID,
		OpType:    model.PendingOpCredit,
		AdapterID: adapterBBase.WalletID(),
		WalletID:  req.SideBWalletID,
		AccountID: req.SideBAccountID,
		Amount:    req.Amount,
		Status:    model.PendingOpStatusPending,
	}
	s.pendingRepo.Create(ctx, creditOp)

	creditResult, err := adapterB.Credit(ctx, req.SideBWalletID, req.SideBAccountID, req.Amount, req.TxID)
	if err != nil || !creditResult.Success {
		errMsg := "Credit failed"
		if err != nil {
			errMsg = err.Error()
		}
		s.pendingRepo.UpdateStatus(ctx, creditOp.ID, model.PendingOpStatusFailed, errMsg)

		// ─── COMPENSATION: Reverse Debit ───
		slog.Warn("Saga: Credit failed, reversing debit",
			"txId", req.TxID, "error", errMsg)

		reversalOp := &model.PendingOperation{
			ID:        uuid.New(),
			TxID:      req.TxID,
			OpType:    model.PendingOpReversal,
			AdapterID: adapterABase.WalletID(),
			WalletID:  req.SideAWalletID,
			AccountID: req.SideAAccountID,
			Amount:    req.Amount,
			Status:    model.PendingOpStatusPending,
		}
		s.pendingRepo.Create(ctx, reversalOp)

		if reverseErr := adapterA.ReverseDebit(ctx, req.SideAWalletID, req.SideAAccountID, req.Amount, req.TxID); reverseErr != nil {
			slog.Error("Saga: CRITICAL — Reversal failed! Manual intervention required",
				"txId", req.TxID, "error", reverseErr)
			s.pendingRepo.UpdateStatus(ctx, reversalOp.ID, model.PendingOpStatusFailed, reverseErr.Error())
			return s.failTx(ctx, req.TxID, "E009", "Credit failed + reversal failed: "+reverseErr.Error())
		}
		s.pendingRepo.UpdateStatus(ctx, reversalOp.ID, model.PendingOpStatusDone, "")

		// Mark tx as REVERSED
		errCode := "E009"
		s.txRepo.UpdateStatus(ctx, req.TxID, model.TxStatusReversed, &errCode, &errMsg)
		return &SagaResult{TxID: req.TxID, Status: model.TxStatusReversed, Error: errMsg}, nil
	}
	s.pendingRepo.UpdateStatus(ctx, creditOp.ID, model.PendingOpStatusDone, "")

	slog.Info("Saga: Credit completed", "txId", req.TxID)

	// ─── Step 3: NOTIFY (best effort) ───
	go func() {
		notifyCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = adapterA.SendSMS(notifyCtx, req.SideAAccountID, fmt.Sprintf(
			"Atheer: تم خصم %s %s من حسابك. المرجع: %s",
			req.Amount.StringFixed(2), req.Currency, req.TxID[:8]))

		_ = adapterB.SendSMS(notifyCtx, req.SideBAccountID, fmt.Sprintf(
			"Atheer: تم إيداع %s %s في حسابك. المرجع: %s",
			req.Amount.StringFixed(2), req.Currency, req.TxID[:8]))
	}()

	// ─── Step 4: Update counter + Mark COMPLETED ───
	s.deviceRepo.IncrementCounter(ctx, req.SideADeviceID, req.SideACtr)

	now := time.Now()
	_ = now
	s.txRepo.UpdateStatus(ctx, req.TxID, model.TxStatusCompleted, nil, nil)

	slog.Info("Saga completed successfully", "txId", req.TxID)

	return &SagaResult{
		TxID:   req.TxID,
		Status: model.TxStatusCompleted,
	}, nil
}

func (s *SagaService) failTx(ctx context.Context, txID, errCode, errMsg string) (*SagaResult, error) {
	s.txRepo.UpdateStatus(ctx, txID, model.TxStatusFailed, &errCode, &errMsg)
	return &SagaResult{
		TxID:   txID,
		Status: model.TxStatusFailed,
		Error:  errMsg,
	}, nil
}
