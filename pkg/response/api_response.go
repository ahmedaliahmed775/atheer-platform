package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard response format for all API endpoints
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorCode defines all error codes matching SDK's ErrorCode enum
type ErrorCode string

const (
	ErrInvalidCounter       ErrorCode = "E001" // عداد غير صالح
	ErrInvalidSignature     ErrorCode = "E002" // توقيع غير صالح أو نافذة زمنية
	ErrCrossValidationFail  ErrorCode = "E003" // عدم تطابق تحديثات متقاطعة
	ErrAmountExceedsLimit   ErrorCode = "E004" // مبلغ تجاوز الحد الفعلي
	ErrDeviceNotRegistered  ErrorCode = "E005" // جهاز غير مسّجل أو معّلق
	ErrTimestampExpired     ErrorCode = "E006" // نافذة زمنية منتهية
	ErrDuplicateTransaction ErrorCode = "E007" // معاملة مكررة (nonce)
	ErrInsufficientBalance  ErrorCode = "E008" // رصيد غير كافٍ
	ErrAdapterFailure       ErrorCode = "E009" // فشل المحول الخارجي
	ErrInvalidAttestation   ErrorCode = "E010" // تهيئة عتادية غير صالحة
	ErrPlayIntegrityFail    ErrorCode = "E011" // Play Integrity فشل
	ErrRootedDevice         ErrorCode = "E012" // جهاز مروّض
	ErrInvalidECDSA         ErrorCode = "E013" // توقيع عتادي غير صالح
	ErrRateLimited          ErrorCode = "E014" // تجاوز حد المعدل
	ErrInternalError        ErrorCode = "E500" // خطأ داخلي
)

// Error descriptions matching SDK's ErrorCode descriptions
var errorDescriptions = map[ErrorCode]string{
	ErrInvalidCounter:       "Invalid counter value — replay attempt detected",
	ErrInvalidSignature:     "Invalid HMAC signature or expired time window",
	ErrCrossValidationFail:  "Cross-validation failed — currency, operation type, or amount mismatch",
	ErrAmountExceedsLimit:   "Amount exceeds the actual limit (Limits Matrix + Adapter Query)",
	ErrDeviceNotRegistered:  "Device not registered, suspended, or revoked",
	ErrTimestampExpired:     "Timestamp expired — exceeds 5-minute window",
	ErrDuplicateTransaction: "Duplicate transaction — same nonce already processed",
	ErrInsufficientBalance:  "Insufficient balance in source wallet",
	ErrAdapterFailure:       "External adapter failure — Saga reversal initiated",
	ErrInvalidAttestation:   "Invalid hardware attestation certificate",
	ErrPlayIntegrityFail:    "Play Integrity verification failed — device compromised",
	ErrRootedDevice:         "Rooted/tampered device detected",
	ErrInvalidECDSA:         "Invalid ECDSA attestation signature",
	ErrRateLimited:          "Rate limit exceeded — try again later",
	ErrInternalError:        "Internal server error",
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

// Err sends an error JSON response
func Err(w http.ResponseWriter, status int, code ErrorCode, message string) {
	if message == "" {
		message = errorDescriptions[code]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    string(code),
			Message: message,
		},
	})
}

// OK sends a 200 success response
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// Created sends a 201 created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// BadRequest sends a 400 error
func BadRequest(w http.ResponseWriter, code ErrorCode, message string) {
	Err(w, http.StatusBadRequest, code, message)
}

// Unauthorized sends a 401 error
func Unauthorized(w http.ResponseWriter, message string) {
	Err(w, http.StatusUnauthorized, ErrInternalError, message)
}

// Forbidden sends a 403 error
func Forbidden(w http.ResponseWriter, code ErrorCode, message string) {
	Err(w, http.StatusForbidden, code, message)
}

// NotFound sends a 404 error
func NotFound(w http.ResponseWriter, message string) {
	Err(w, http.StatusNotFound, ErrDeviceNotRegistered, message)
}

// Conflict sends a 409 error (duplicate nonce)
func Conflict(w http.ResponseWriter, data interface{}) {
	// For idempotency: return the original result with 200
	JSON(w, http.StatusOK, data)
}

// TooManyRequests sends a 429 error
func TooManyRequests(w http.ResponseWriter) {
	Err(w, http.StatusTooManyRequests, ErrRateLimited, "")
}

// InternalError sends a 500 error
func InternalError(w http.ResponseWriter, message string) {
	Err(w, http.StatusInternalServerError, ErrInternalError, message)
}

// GetDescription returns the description for an error code
func GetDescription(code ErrorCode) string {
	if desc, ok := errorDescriptions[code]; ok {
		return desc
	}
	return "Unknown error"
}
