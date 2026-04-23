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

// ═══════════════════════════════════════════════
// WENET Adapter
// ═══════════════════════════════════════════════

type WENETAdapter struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type WENETConfig struct {
	BaseURL    string
	APIKey     string
	TimeoutSec int
}

func NewWENETAdapter(cfg WENETConfig) *WENETAdapter {
	timeout := 10 * time.Second
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	return &WENETAdapter{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (a *WENETAdapter) WalletID() string { return "WENET" }

func (a *WENETAdapter) BuildRequest(dto model.TransactionDTO) (*WalletAPIRequest, error) {
    body, _ := json.Marshal(map[string]interface{}{
        "payerUserId":     dto.PayerUserID,
        "payerType":       dto.PayerType,
        "payeeId":         dto.PayeeID,
        "payeeType":       dto.PayeeType,
        "transactionType": dto.TransactionType,
        "amount":          dto.Amount,
        "currency":        dto.Currency,
    })
    return &WalletAPIRequest{
        URL:    a.baseURL + "/transfer/transaction",
        Method: "POST",
        Headers: map[string]string{
            "X-API-Key":     a.apiKey,
            "Content-Type":  "application/json",
        },
        Body: body,
    }, nil
}

func (a *WENETAdapter) ParseResponse(raw []byte) (*model.AtheerResult, error) {
    var resp map[string]interface{}
    json.Unmarshal(raw, &resp)
    return &model.AtheerResult{
        Success:       resp["status"] == "OK",
        TransactionID: fmt.Sprintf("%v", resp["ref"]),
    }, nil
}

func (a *WENETAdapter) ExecuteTransaction(ctx context.Context, dto model.TransactionDTO) (*model.AtheerResult, error) {
    debitResult, err := a.Debit(ctx, dto.WalletID, dto.PayerUserID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
    if err != nil || !debitResult.Success {
        return &model.AtheerResult{Success: false, ErrorCode: "ERR_BALANCE"}, err
    }

    creditResult, err := a.Credit(ctx, dto.WalletID, dto.PayeeID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
    if err != nil || !creditResult.Success {
        _ = a.ReverseDebit(ctx, dto.WalletID, dto.PayerUserID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
        return &model.AtheerResult{Success: false, ErrorCode: "ERR_WALLET_DOWN"}, err
    }

    go func() {
        _ = a.SendSMS(ctx, dto.PayerUserID, fmt.Sprintf("تم خصم %d %s", dto.Amount, dto.Currency))
        _ = a.SendSMS(ctx, dto.PayeeID, fmt.Sprintf("تم إيداع %d %s", dto.Amount, dto.Currency))
    }()

    return &model.AtheerResult{Success: true, TransactionID: creditResult.TransactionID}, nil
}

func (a *WENETAdapter) Debit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*DebitResult, error) {
	resp, err := a.postJSON(ctx, "/transfer/debit", map[string]interface{}{
		"wallet": walletID, "account": accountID,
		"amount": amount.StringFixed(2), "ref": txID,
	})
	if err != nil {
		return nil, fmt.Errorf("WENET debit: %w", err)
	}
	slog.Info("WENET debit", "txId", txID, "amount", amount)
	return &DebitResult{
		Success:       resp["status"] == "OK",
		TransactionID: fmt.Sprintf("%v", resp["ref"]),
		NewBalance:    decimalFromResp(resp, "balance"),
	}, nil
}

func (a *WENETAdapter) Credit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*CreditResult, error) {
	resp, err := a.postJSON(ctx, "/transfer/credit", map[string]interface{}{
		"wallet": walletID, "account": accountID,
		"amount": amount.StringFixed(2), "ref": txID,
	})
	if err != nil {
		return nil, fmt.Errorf("WENET credit: %w", err)
	}
	slog.Info("WENET credit", "txId", txID, "amount", amount)
	return &CreditResult{
		Success:       resp["status"] == "OK",
		TransactionID: fmt.Sprintf("%v", resp["ref"]),
		NewBalance:    decimalFromResp(resp, "balance"),
	}, nil
}

func (a *WENETAdapter) ReverseDebit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, originalTxID string) error {
	_, err := a.postJSON(ctx, "/transfer/reverse", map[string]interface{}{
		"wallet": walletID, "account": accountID,
		"amount": amount.StringFixed(2), "originalRef": originalTxID,
	})
	return err
}

func (a *WENETAdapter) CheckBalance(ctx context.Context, walletID, accountID string) (*BalanceResult, error) {
	resp, err := a.getJSON(ctx, fmt.Sprintf("/account/%s/%s", walletID, accountID))
	if err != nil {
		return nil, err
	}
	return &BalanceResult{Available: decimalFromResp(resp, "balance"), Currency: fmt.Sprintf("%v", resp["currency"])}, nil
}

func (a *WENETAdapter) GetTransactionStatus(ctx context.Context, txID string) (*StatusResult, error) {
	resp, err := a.getJSON(ctx, "/transaction/"+txID)
	if err != nil {
		return nil, err
	}
	return &StatusResult{TxID: txID, Status: fmt.Sprintf("%v", resp["status"])}, nil
}

func (a *WENETAdapter) SendSMS(ctx context.Context, phone, message string) error {
	_, err := a.postJSON(ctx, "/notify/sms", map[string]interface{}{"to": phone, "body": message})
	return err
}

func (a *WENETAdapter) GetLimits(ctx context.Context, walletID, accountID string, opType model.TransactionType) (*model.LimitsResult, error) {
	resp, err := a.getJSON(ctx, fmt.Sprintf("/limits/%s/%s", walletID, accountID))
	if err != nil {
		return &model.LimitsResult{
			MaxTxAmount: decimal.NewFromInt(999999999), RemainingDaily: decimal.NewFromInt(999999999), MaxDaily: decimal.NewFromInt(999999999),
		}, nil
	}
	return &model.LimitsResult{
		MaxTxAmount: decimalFromResp(resp, "maxTx"), RemainingDaily: decimalFromResp(resp, "remainingDaily"), MaxDaily: decimalFromResp(resp, "maxDaily"),
	}, nil
}

func (a *WENETAdapter) postJSON(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", a.baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", a.apiKey)
	return doHTTP(a.httpClient, req)
}

func (a *WENETAdapter) getJSON(ctx context.Context, path string) (map[string]interface{}, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.baseURL+path, nil)
	req.Header.Set("X-API-Key", a.apiKey)
	return doHTTP(a.httpClient, req)
}

// ═══════════════════════════════════════════════
// WASEL Adapter
// ═══════════════════════════════════════════════

type WASELAdapter struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type WASELConfig struct {
	BaseURL    string
	APIKey     string
	TimeoutSec int
}

func NewWASELAdapter(cfg WASELConfig) *WASELAdapter {
	timeout := 10 * time.Second
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	return &WASELAdapter{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (a *WASELAdapter) WalletID() string { return "WASEL" }

func (a *WASELAdapter) BuildRequest(dto model.TransactionDTO) (*WalletAPIRequest, error) {
    body, _ := json.Marshal(map[string]interface{}{
        "payerUserId":     dto.PayerUserID,
        "payerType":       dto.PayerType,
        "payeeId":         dto.PayeeID,
        "payeeType":       dto.PayeeType,
        "transactionType": dto.TransactionType,
        "amount":          dto.Amount,
        "currency":        dto.Currency,
    })
    return &WalletAPIRequest{
        URL:    a.baseURL + "/v2/transaction",
        Method: "POST",
        Headers: map[string]string{
            "Authorization": "Bearer " + a.apiKey,
            "Content-Type":  "application/json",
        },
        Body: body,
    }, nil
}

func (a *WASELAdapter) ParseResponse(raw []byte) (*model.AtheerResult, error) {
    var resp map[string]interface{}
    json.Unmarshal(raw, &resp)
    return &model.AtheerResult{
        Success:       resp["code"] == "00",
        TransactionID: fmt.Sprintf("%v", resp["txRef"]),
    }, nil
}

func (a *WASELAdapter) ExecuteTransaction(ctx context.Context, dto model.TransactionDTO) (*model.AtheerResult, error) {
    debitResult, err := a.Debit(ctx, dto.WalletID, dto.PayerUserID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
    if err != nil || !debitResult.Success {
        return &model.AtheerResult{Success: false, ErrorCode: "ERR_BALANCE"}, err
    }

    creditResult, err := a.Credit(ctx, dto.WalletID, dto.PayeeID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
    if err != nil || !creditResult.Success {
        _ = a.ReverseDebit(ctx, dto.WalletID, dto.PayerUserID, decimal.NewFromInt(dto.Amount), fmt.Sprintf("TX-%d", dto.Timestamp))
        return &model.AtheerResult{Success: false, ErrorCode: "ERR_WALLET_DOWN"}, err
    }

    go func() {
        _ = a.SendSMS(ctx, dto.PayerUserID, fmt.Sprintf("تم خصم %d %s", dto.Amount, dto.Currency))
        _ = a.SendSMS(ctx, dto.PayeeID, fmt.Sprintf("تم إيداع %d %s", dto.Amount, dto.Currency))
    }()

    return &model.AtheerResult{Success: true, TransactionID: creditResult.TransactionID}, nil
}

func (a *WASELAdapter) Debit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*DebitResult, error) {
	resp, err := a.postJSON(ctx, "/v2/debit", map[string]interface{}{
		"walletId": walletID, "accountId": accountID,
		"amount": amount.StringFixed(2), "txRef": txID,
	})
	if err != nil {
		return nil, fmt.Errorf("WASEL debit: %w", err)
	}
	slog.Info("WASEL debit", "txId", txID, "amount", amount)
	return &DebitResult{
		Success: resp["code"] == "00", TransactionID: fmt.Sprintf("%v", resp["txRef"]), NewBalance: decimalFromResp(resp, "balance"),
	}, nil
}

func (a *WASELAdapter) Credit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, txID string) (*CreditResult, error) {
	resp, err := a.postJSON(ctx, "/v2/credit", map[string]interface{}{
		"walletId": walletID, "accountId": accountID,
		"amount": amount.StringFixed(2), "txRef": txID,
	})
	if err != nil {
		return nil, fmt.Errorf("WASEL credit: %w", err)
	}
	slog.Info("WASEL credit", "txId", txID, "amount", amount)
	return &CreditResult{
		Success: resp["code"] == "00", TransactionID: fmt.Sprintf("%v", resp["txRef"]), NewBalance: decimalFromResp(resp, "balance"),
	}, nil
}

func (a *WASELAdapter) ReverseDebit(ctx context.Context, walletID, accountID string, amount decimal.Decimal, originalTxID string) error {
	_, err := a.postJSON(ctx, "/v2/reverse", map[string]interface{}{
		"walletId": walletID, "accountId": accountID,
		"amount": amount.StringFixed(2), "originalTxRef": originalTxID,
	})
	return err
}

func (a *WASELAdapter) CheckBalance(ctx context.Context, walletID, accountID string) (*BalanceResult, error) {
	resp, err := a.getJSON(ctx, fmt.Sprintf("/v2/balance/%s/%s", walletID, accountID))
	if err != nil {
		return nil, err
	}
	return &BalanceResult{Available: decimalFromResp(resp, "balance"), Currency: fmt.Sprintf("%v", resp["currency"])}, nil
}

func (a *WASELAdapter) GetTransactionStatus(ctx context.Context, txID string) (*StatusResult, error) {
	resp, err := a.getJSON(ctx, "/v2/status/"+txID)
	if err != nil {
		return nil, err
	}
	return &StatusResult{TxID: txID, Status: fmt.Sprintf("%v", resp["status"])}, nil
}

func (a *WASELAdapter) SendSMS(ctx context.Context, phone, message string) error {
	_, err := a.postJSON(ctx, "/v2/sms", map[string]interface{}{"phone": phone, "text": message})
	return err
}

func (a *WASELAdapter) GetLimits(ctx context.Context, walletID, accountID string, opType model.TransactionType) (*model.LimitsResult, error) {
	resp, err := a.getJSON(ctx, fmt.Sprintf("/v2/limits/%s/%s", walletID, accountID))
	if err != nil {
		return &model.LimitsResult{
			MaxTxAmount: decimal.NewFromInt(999999999), RemainingDaily: decimal.NewFromInt(999999999), MaxDaily: decimal.NewFromInt(999999999),
		}, nil
	}
	return &model.LimitsResult{
		MaxTxAmount: decimalFromResp(resp, "maxTx"), RemainingDaily: decimalFromResp(resp, "remainingDaily"), MaxDaily: decimalFromResp(resp, "maxDaily"),
	}, nil
}

func (a *WASELAdapter) postJSON(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", a.baseURL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return doHTTP(a.httpClient, req)
}

func (a *WASELAdapter) getJSON(ctx context.Context, path string) (map[string]interface{}, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", a.baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return doHTTP(a.httpClient, req)
}

// ═══════════════════════════════════════════════
// Shared HTTP utility
// ═══════════════════════════════════════════════

func doHTTP(client *http.Client, req *http.Request) (map[string]interface{}, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}
