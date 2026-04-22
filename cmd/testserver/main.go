package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════
// Atheer Switch — Test Server (In-Memory)
// No PostgreSQL or Redis required
// ═══════════════════════════════════════════════

var (
	mu           sync.RWMutex
	devices      = map[string]*TestDevice{}
	transactions = []*TestTransaction{}
	disputes     = []*TestDispute{}
)

type TestDevice struct {
	ID                 string `json:"id"`
	DeviceID           string `json:"deviceId"`
	WalletID           string `json:"walletId"`
	DeviceModel        string `json:"deviceModel"`
	AttestationLevel   string `json:"attestationLevel"`
	Status             string `json:"status"`
	Ctr                int64  `json:"ctr"`
	TxCount            int    `json:"txCount"`
	EnrolledAt         string `json:"enrolledAt"`
	LastTxAt           string `json:"lastTxAt"`
}

type TestTransaction struct {
	ID            string  `json:"id"`
	SideAWallet   string  `json:"sideAWallet"`
	SideADevice   string  `json:"sideADevice"`
	SideBWallet   string  `json:"sideBWallet"`
	SideBDevice   string  `json:"sideBDevice"`
	OperationType string  `json:"operationType"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	Channel       string  `json:"channel"`
	Ctr           int64   `json:"ctr"`
	Nonce         string  `json:"nonce"`
	LatencyMs     int     `json:"latencyMs"`
	CreatedAt     string  `json:"createdAt"`
	ErrorCode     string  `json:"errorCode,omitempty"`
}

type TestDispute struct {
	ID          string `json:"id"`
	TxID        string `json:"txId"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	slog.Info("═══════════════════════════════════════════════")
	slog.Info("  Atheer Switch V3.0 — Test Server (In-Memory)")
	slog.Info("═══════════════════════════════════════════════")

	// Seed test data
	seedTestData()

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	// Health
	r.Get("/health", healthHandler)

	// API v2
	r.Route("/api/v2", func(r chi.Router) {
		// Enrollment
		r.Post("/enroll", enrollHandler)

		// Transactions
		r.Post("/transaction", processTransactionHandler)
		r.Get("/transaction", listTransactionsHandler)
		r.Get("/transaction/{txId}", getTransactionHandler)

		// Devices
		r.Get("/device", listDevicesHandler)
		r.Get("/device/{deviceId}", getDeviceHandler)
		r.Post("/device/{deviceId}/suspend", suspendDeviceHandler)
		r.Post("/device/{deviceId}/revoke", revokeDeviceHandler)

		// Limits
		r.Get("/limits", getLimitsHandler)
		r.Put("/limits", updateLimitsHandler)

		// Disputes
		r.Post("/dispute", openDisputeHandler)
		r.Get("/dispute", listDisputesHandler)

		// Stats (for dashboard)
		r.Get("/stats", statsHandler)
		r.Get("/stats/pipeline", pipelineStatsHandler)
		r.Get("/stats/channels", channelStatsHandler)
		r.Get("/stats/volume", volumeStatsHandler)
	})

	port := "8080"
	slog.Info("Switch listening", "port", port, "mode", "TEST", "pipeline", "10 layers (simulated)")
	slog.Info("Dashboard can connect at", "url", "http://localhost:"+port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		slog.Error("Server failed", "error", err)
	}
}

// ═══════════════════════════════════════════════
// Handlers
// ═══════════════════════════════════════════════

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"status":    "healthy",
		"version":   "3.0.0",
		"mode":      "test",
		"uptime":    time.Since(startTime).String(),
		"db":        "in-memory",
		"redis":     "in-memory",
		"pipeline":  "10 layers active",
		"devices":   len(devices),
		"transactions": len(transactions),
	})
}

func enrollHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WalletID         string `json:"walletId"`
		DeviceModel      string `json:"deviceModel"`
		AttestationLevel string `json:"attestationLevel"`
		PublicKey        string `json:"attestationPublicKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "E001", "Invalid request body")
		return
	}

	deviceID := fmt.Sprintf("DEV-%s", uuid.New().String()[:8])
	seed := make([]byte, 32)
	rand.Read(seed)

	dev := &TestDevice{
		ID:               uuid.New().String(),
		DeviceID:         deviceID,
		WalletID:         req.WalletID,
		DeviceModel:      req.DeviceModel,
		AttestationLevel: req.AttestationLevel,
		Status:           "ACTIVE",
		Ctr:              0,
		TxCount:          0,
		EnrolledAt:       time.Now().Format(time.RFC3339),
		LastTxAt:         "",
	}

	mu.Lock()
	devices[deviceID] = dev
	mu.Unlock()

	slog.Info("Device enrolled", "deviceId", deviceID, "wallet", req.WalletID, "model", req.DeviceModel)

	writeJSON(w, 200, map[string]interface{}{
		"success":    true,
		"code":       "S000",
		"deviceId":   deviceID,
		"deviceSeed": base64.StdEncoding.EncodeToString(seed),
		"message":    "Device enrolled successfully",
	})
}

func processTransactionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SideA struct {
			WalletID      string   `json:"walletId"`
			DeviceID      string   `json:"deviceId"`
			Ctr           int64    `json:"ctr"`
			OperationType string   `json:"operationType"`
			Currency      string   `json:"currency"`
			Amount        *float64 `json:"amount"`
			Nonce         string   `json:"nonce"`
			Timestamp     int64    `json:"timestamp"`
			Signature     string   `json:"signature"`
		} `json:"sideA"`
		SideB struct {
			WalletID      string   `json:"walletId"`
			DeviceID      string   `json:"deviceId"`
			OperationType string   `json:"operationType"`
			Currency      string   `json:"currency"`
			Amount        *float64 `json:"amount"`
			AccountID     string   `json:"accountId"`
			Timestamp     int64    `json:"timestamp"`
			Signature     string   `json:"signature"`
		} `json:"sideB"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "E001", "Invalid request body")
		return
	}

	start := time.Now()
	txID := fmt.Sprintf("TX-%s", uuid.New().String()[:12])

	// Simulate pipeline
	slog.Info("Pipeline: Layer 1 — Rate Limiter ✓")
	slog.Info("Pipeline: Layer 2 — Request Logger ✓")
	slog.Info("Pipeline: Layer 3 — Anti-Replay ✓", "ctr", req.SideA.Ctr)
	slog.Info("Pipeline: Layer 4 — Attestation (ECDSA) ✓")
	slog.Info("Pipeline: Layer 5 — HMAC Side A ✓")
	slog.Info("Pipeline: Layer 6 — HMAC Side B ✓")
	slog.Info("Pipeline: Layer 7 — Cross-Validator ✓")
	slog.Info("Pipeline: Layer 8 — Limits Checker ✓")
	slog.Info("Pipeline: Layer 9 — Idempotency ✓")
	slog.Info("Pipeline: Layer 10 — Saga Executor (Debit → Credit → Notify)")

	amount := 0.0
	if req.SideA.Amount != nil {
		amount = *req.SideA.Amount
	}

	latency := int(time.Since(start).Milliseconds()) + 15 // Add simulated adapter latency

	tx := &TestTransaction{
		ID:            txID,
		SideAWallet:   req.SideA.WalletID,
		SideADevice:   req.SideA.DeviceID,
		SideBWallet:   req.SideB.WalletID,
		SideBDevice:   req.SideB.DeviceID,
		OperationType: req.SideA.OperationType,
		Amount:        amount,
		Currency:      req.SideA.Currency,
		Status:        "COMPLETED",
		Channel:       req.SideA.WalletID,
		Ctr:           req.SideA.Ctr,
		Nonce:         req.SideA.Nonce,
		LatencyMs:     latency,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	mu.Lock()
	transactions = append([]*TestTransaction{tx}, transactions...)
	// Update device counters
	if dev, ok := devices[req.SideA.DeviceID]; ok {
		dev.Ctr = req.SideA.Ctr
		dev.TxCount++
		dev.LastTxAt = time.Now().Format(time.RFC3339)
	}
	mu.Unlock()

	slog.Info("Transaction completed",
		"txId", txID, "amount", amount, "currency", req.SideA.Currency,
		"opType", req.SideA.OperationType, "latency_ms", latency)

	writeJSON(w, 200, map[string]interface{}{
		"success":       true,
		"code":          "S000",
		"transactionId": txID,
		"status":        "COMPLETED",
		"latencyMs":     latency,
		"message":       "Transaction processed successfully via 10-layer pipeline",
	})
}

func listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	writeJSON(w, 200, map[string]interface{}{
		"success":      true,
		"transactions": transactions,
		"total":        len(transactions),
	})
}

func getTransactionHandler(w http.ResponseWriter, r *http.Request) {
	txID := chi.URLParam(r, "txId")
	mu.RLock()
	defer mu.RUnlock()
	for _, tx := range transactions {
		if tx.ID == txID {
			writeJSON(w, 200, map[string]interface{}{"success": true, "transaction": tx})
			return
		}
	}
	writeError(w, 404, "E006", "Transaction not found")
}

func listDevicesHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	devList := make([]*TestDevice, 0, len(devices))
	for _, d := range devices {
		devList = append(devList, d)
	}
	writeJSON(w, 200, map[string]interface{}{
		"success": true,
		"devices": devList,
		"total":   len(devList),
	})
}

func getDeviceHandler(w http.ResponseWriter, r *http.Request) {
	devID := chi.URLParam(r, "deviceId")
	mu.RLock()
	defer mu.RUnlock()
	if dev, ok := devices[devID]; ok {
		writeJSON(w, 200, map[string]interface{}{"success": true, "device": dev})
		return
	}
	writeError(w, 404, "E002", "Device not found")
}

func suspendDeviceHandler(w http.ResponseWriter, r *http.Request) {
	devID := chi.URLParam(r, "deviceId")
	mu.Lock()
	defer mu.Unlock()
	if dev, ok := devices[devID]; ok {
		dev.Status = "SUSPENDED"
		slog.Info("Device suspended", "deviceId", devID)
		writeJSON(w, 200, map[string]interface{}{"success": true, "message": "Device suspended"})
		return
	}
	writeError(w, 404, "E002", "Device not found")
}

func revokeDeviceHandler(w http.ResponseWriter, r *http.Request) {
	devID := chi.URLParam(r, "deviceId")
	mu.Lock()
	defer mu.Unlock()
	if dev, ok := devices[devID]; ok {
		dev.Status = "REVOKED"
		slog.Info("Device revoked", "deviceId", devID)
		writeJSON(w, 200, map[string]interface{}{"success": true, "message": "Device revoked"})
		return
	}
	writeError(w, 404, "E002", "Device not found")
}

func getLimitsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"success": true,
		"limits": []map[string]interface{}{
			{"wallet": "JEEP", "opType": "P2P_SAME", "currency": "YER", "tier": "basic", "maxTx": 50000, "maxDaily": 500000, "maxMonthly": 5000000},
			{"wallet": "JEEP", "opType": "P2M_SAME", "currency": "YER", "tier": "basic", "maxTx": 100000, "maxDaily": 1000000, "maxMonthly": 10000000},
			{"wallet": "WENET", "opType": "P2P_SAME", "currency": "YER", "tier": "basic", "maxTx": 40000, "maxDaily": 400000, "maxMonthly": 4000000},
			{"wallet": "WASEL", "opType": "P2P_SAME", "currency": "YER", "tier": "basic", "maxTx": 30000, "maxDaily": 300000, "maxMonthly": 3000000},
		},
	})
}

func updateLimitsHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Limits updated")
	writeJSON(w, 200, map[string]interface{}{"success": true, "message": "Limits updated"})
}

func openDisputeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxID        string `json:"txId"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	dispute := &TestDispute{
		ID: fmt.Sprintf("DSP-%03d", len(disputes)+1),
		TxID: req.TxID, Type: req.Type, Status: "OPEN",
		Priority: "MEDIUM", Description: req.Description,
		CreatedAt: time.Now().Format("2006-01-02"),
	}
	mu.Lock()
	disputes = append(disputes, dispute)
	mu.Unlock()
	slog.Info("Dispute opened", "id", dispute.ID, "txId", req.TxID)
	writeJSON(w, 200, map[string]interface{}{"success": true, "dispute": dispute})
}

func listDisputesHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	writeJSON(w, 200, map[string]interface{}{"success": true, "disputes": disputes, "total": len(disputes)})
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	totalAmount := 0.0
	completed := 0
	failed := 0
	avgLatency := 0
	for _, tx := range transactions {
		totalAmount += tx.Amount
		if tx.Status == "COMPLETED" { completed++ }
		if tx.Status == "FAILED" { failed++ }
		avgLatency += tx.LatencyMs
	}
	if len(transactions) > 0 { avgLatency /= len(transactions) }

	activeDevices := 0
	for _, d := range devices {
		if d.Status == "ACTIVE" { activeDevices++ }
	}

	writeJSON(w, 200, map[string]interface{}{
		"success":       true,
		"totalTx":       len(transactions),
		"completedTx":   completed,
		"failedTx":      failed,
		"totalAmount":   totalAmount,
		"currency":      "YER",
		"activeDevices": activeDevices,
		"totalDevices":  len(devices),
		"avgLatencyMs":  avgLatency,
		"pipelineLayers": 10,
	})
}

func pipelineStatsHandler(w http.ResponseWriter, r *http.Request) {
	total := len(transactions) + 23 // Add some baseline
	writeJSON(w, 200, map[string]interface{}{
		"success": true,
		"layers": []map[string]interface{}{
			{"num": 1, "name": "Rate Limiter", "pass": total, "fail": 3, "avgMs": 0.5},
			{"num": 2, "name": "Request Logger", "pass": total - 3, "fail": 0, "avgMs": 0.1},
			{"num": 3, "name": "Anti-Replay", "pass": total - 3, "fail": 1, "avgMs": 1.2},
			{"num": 4, "name": "Attestation", "pass": total - 4, "fail": 2, "avgMs": 3.5},
			{"num": 5, "name": "HMAC Side A", "pass": total - 6, "fail": 0, "avgMs": 2.1},
			{"num": 6, "name": "HMAC Side B", "pass": total - 6, "fail": 0, "avgMs": 2.0},
			{"num": 7, "name": "Cross-Validator", "pass": total - 6, "fail": 1, "avgMs": 0.8},
			{"num": 8, "name": "Limits Checker", "pass": total - 7, "fail": 0, "avgMs": 4.2},
			{"num": 9, "name": "Idempotency", "pass": total - 7, "fail": 0, "avgMs": 0.9},
			{"num": 10, "name": "Saga Executor", "pass": total - 7, "fail": 0, "avgMs": 15.0},
		},
	})
}

func channelStatsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	channels := map[string]int{"JEEP": 0, "WENET": 0, "WASEL": 0}
	for _, tx := range transactions {
		channels[tx.Channel]++
	}
	writeJSON(w, 200, map[string]interface{}{
		"success":  true,
		"channels": channels,
	})
}

func volumeStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Generate hourly volume data
	hours := make([]map[string]interface{}, 24)
	for i := 0; i < 24; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(80))
		hours[i] = map[string]interface{}{
			"hour":   fmt.Sprintf("%02d:00", i),
			"volume": n.Int64() + 5,
		}
	}
	writeJSON(w, 200, map[string]interface{}{"success": true, "hours": hours})
}

// ═══════════════════════════════════════════════
// Seed Test Data
// ═══════════════════════════════════════════════

var startTime = time.Now()

func seedTestData() {
	wallets := []string{"JEEP", "WENET", "WASEL"}
	models := []string{"Samsung Galaxy S24", "Pixel 8 Pro", "Xiaomi 14", "OnePlus 12"}
	attestLevels := []string{"STRONG_BOX", "TEE", "SOFTWARE"}
	opTypes := []string{"P2P_SAME", "P2M_SAME", "P2P_CROSS", "P2M_CROSS"}

	// Create 12 devices
	for i := 0; i < 12; i++ {
		devID := fmt.Sprintf("DEV-%s", uuid.New().String()[:8])
		devices[devID] = &TestDevice{
			ID:               uuid.New().String(),
			DeviceID:         devID,
			WalletID:         wallets[i%3],
			DeviceModel:      models[i%4],
			AttestationLevel: attestLevels[i%3],
			Status:           "ACTIVE",
			Ctr:              int64(i * 10),
			TxCount:          i * 5,
			EnrolledAt:       time.Now().Add(-time.Duration(i*24) * time.Hour).Format(time.RFC3339),
			LastTxAt:         time.Now().Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
		}
	}

	// Create 20 transactions
	devKeys := make([]string, 0, len(devices))
	for k := range devices {
		devKeys = append(devKeys, k)
	}

	for i := 0; i < 20; i++ {
		statuses := []string{"COMPLETED", "COMPLETED", "COMPLETED", "COMPLETED", "PENDING", "FAILED"}
		nBig, _ := rand.Int(rand.Reader, big.NewInt(50000))
		amount := float64(nBig.Int64()+100) + 0.50

		sideADev := devKeys[i%len(devKeys)]
		sideBDev := devKeys[(i+1)%len(devKeys)]

		transactions = append(transactions, &TestTransaction{
			ID:            fmt.Sprintf("TX-%s", uuid.New().String()[:12]),
			SideAWallet:   devices[sideADev].WalletID,
			SideADevice:   sideADev,
			SideBWallet:   devices[sideBDev].WalletID,
			SideBDevice:   sideBDev,
			OperationType: opTypes[i%4],
			Amount:        amount,
			Currency:      "YER",
			Status:        statuses[i%6],
			Channel:       devices[sideADev].WalletID,
			Ctr:           int64(i + 1),
			Nonce:         uuid.New().String(),
			LatencyMs:     20 + i*3,
			CreatedAt:     time.Now().Add(-time.Duration(i*5) * time.Minute).Format(time.RFC3339),
		})
	}

	// Seed disputes
	disputes = append(disputes,
		&TestDispute{ID: "DSP-001", TxID: transactions[4].ID, Type: "AMOUNT_MISMATCH", Status: "OPEN", Priority: "HIGH", Description: "مبلغ مختلف عن المتوقع", CreatedAt: "2026-04-22"},
		&TestDispute{ID: "DSP-002", TxID: transactions[8].ID, Type: "DUPLICATE_CHARGE", Status: "INVESTIGATING", Priority: "CRITICAL", Description: "خصم مزدوج", CreatedAt: "2026-04-21"},
	)

	slog.Info("Test data seeded",
		"devices", len(devices),
		"transactions", len(transactions),
		"disputes", len(disputes),
	)
}

// ═══════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"success": false,
		"code":    code,
		"message": message,
	})
}
