package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/atheer-payment/atheer-platform/internal/model"
)

// JEEPAdapter implements PaymentAdapter for the JEEP wallet system
type JEEPAdapter struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// JEEPConfig holds configuration for the JEEP adapter
type JEEPConfig struct {
	BaseURL    string
	APIKey     string
	TimeoutSec int
}

func NewJEEPAdapter(cfg JEEPConfig) *JEEPAdapter {
	timeout := 10 * time.Second
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	return &JEEPAdapter{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (a *JEEPAdapter) ID() string { return "JEEP" }

func (a *JEEPAdapter) Debit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*DebitResult, error) {
	body := map[string]interface{}{
		"walletId":      walletID,
		"accountId":     accountID,
		"amount":        amount.StringFixed(2),
		"transactionId": txID,
		"type":          "DEBIT",
	}

	resp, err := a.post(ctx, "/api/v1/transaction/debit", body)
	if err != nil {
		return nil, fmt.Errorf("JEEP debit failed: %w", err)
	}

	slog.Info("JEEP debit executed",
		"txId", txID, "amount", amount, "accountId", accountID)

	return &DebitResult{
		Success:       resp["success"] == true,
		TransactionID: fmt.Sprintf("%v", resp["transactionId"]),
		NewBalance:    decimalFromResp(resp, "newBalance"),
	}, nil
}

func (a *JEEPAdapter) Credit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*CreditResult, error) {
	body := map[string]interface{}{
		"walletId":      walletID,
		"accountId":     accountID,
		"amount":        amount.StringFixed(2),
		"transactionId": txID,
		"type":          "CREDIT",
	}

	resp, err := a.post(ctx, "/api/v1/transaction/credit", body)
	if err != nil {
		return nil, fmt.Errorf("JEEP credit failed: %w", err)
	}

	slog.Info("JEEP credit executed",
		"txId", txID, "amount", amount, "accountId", accountID)

	return &CreditResult{
		Success:       resp["success"] == true,
		TransactionID: fmt.Sprintf("%v", resp["transactionId"]),
		NewBalance:    decimalFromResp(resp, "newBalance"),
	}, nil
}

func (a *JEEPAdapter) ReverseDebit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, originalTxID string) error {
	body := map[string]interface{}{
		"walletId":          walletID,
		"accountId":         accountID,
		"amount":            amount.StringFixed(2),
		"originalTransactionId": originalTxID,
		"type":              "REVERSAL",
	}

	_, err := a.post(ctx, "/api/v1/transaction/reverse", body)
	if err != nil {
		return fmt.Errorf("JEEP reversal failed: %w", err)
	}

	slog.Info("JEEP reversal executed",
		"originalTxId", originalTxID, "amount", amount)
	return nil
}

func (a *JEEPAdapter) CheckBalance(ctx context.Context, walletID, accountID string) (*BalanceResult, error) {
	resp, err := a.get(ctx, fmt.Sprintf("/api/v1/account/%s/%s/balance", walletID, accountID))
	if err != nil {
		return nil, fmt.Errorf("JEEP balance check failed: %w", err)
	}

	return &BalanceResult{
		Available: decimalFromResp(resp, "available"),
		Currency:  fmt.Sprintf("%v", resp["currency"]),
	}, nil
}

func (a *JEEPAdapter) GetTransactionStatus(ctx context.Context, txID string) (*StatusResult, error) {
	resp, err := a.get(ctx, fmt.Sprintf("/api/v1/transaction/%s/status", txID))
	if err != nil {
		return nil, fmt.Errorf("JEEP status check failed: %w", err)
	}

	return &StatusResult{
		TxID:   txID,
		Status: fmt.Sprintf("%v", resp["status"]),
	}, nil
}

func (a *JEEPAdapter) SendSMS(ctx context.Context, phone, message string) error {
	body := map[string]interface{}{
		"phone":   phone,
		"message": message,
	}
	_, err := a.post(ctx, "/api/v1/notification/sms", body)
	return err
}

func (a *JEEPAdapter) GetLimits(ctx context.Context, walletID, accountID string, opType model.OperationType) (*model.LimitsResult, error) {
	resp, err := a.get(ctx, fmt.Sprintf("/api/v1/limits/%s/%s?opType=%s", walletID, accountID, opType))
	if err != nil {
		// Return high defaults if adapter limits unavailable
		return &model.LimitsResult{
			MaxTxAmount:    decimal.NewFromInt(999999999),
			RemainingDaily: decimal.NewFromInt(999999999),
			MaxDaily:       decimal.NewFromInt(999999999),
		}, nil
	}

	return &model.LimitsResult{
		MaxTxAmount:    decimalFromResp(resp, "maxTxAmount"),
		RemainingDaily: decimalFromResp(resp, "remainingDaily"),
		MaxDaily:       decimalFromResp(resp, "maxDaily"),
	}, nil
}

// === HTTP helpers ===

func (a *JEEPAdapter) post(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+path, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("X-Adapter", "JEEP")

	return a.doRequest(req)
}

func (a *JEEPAdapter) get(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("X-Adapter", "JEEP")

	return a.doRequest(req)
}

func (a *JEEPAdapter) doRequest(req *http.Request) (map[string]interface{}, error) {
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

func decimalFromResp(resp map[string]interface{}, key string) decimal.Decimal {
	if val, ok := resp[key]; ok {
		switch v := val.(type) {
		case float64:
			return decimal.NewFromFloat(v)
		case string:
			d, _ := decimal.NewFromString(v)
			return d
		}
	}
	return decimal.Zero
}
