// Modified for v3.0 Document Alignment
package router

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/atheer-payment/atheer-platform/internal/handler"
	"github.com/atheer-payment/atheer-platform/internal/middleware"
)

// Dependencies holds all injected dependencies for routes
type Dependencies struct {
	HealthHandler      *handler.HealthHandler
	EnrollHandler      *handler.EnrollHandler
	TransactionHandler *handler.TransactionHandler
	DeviceHandler      *handler.DeviceHandler
	DisputeHandler     *handler.DisputeHandler
	LimitsHandler      *handler.LimitsHandler

	// Pipeline middleware — v3.0 Document Alignment
	RateLimiter       *middleware.RateLimiter
	RequestLogger     *middleware.RequestLogger
	AntiReplay        *middleware.AntiReplay
	LimitsChecker     *middleware.LimitsChecker
	SigVerifierA      *middleware.SignatureVerifierA
	PayeeTypeVerifier *middleware.PayeeTypeVerifier
	TxTypeResolver    *middleware.TransactionTypeResolver
}

// New creates and configures the Chi router with all routes
func New(deps *Dependencies) chi.Router {
	r := chi.NewRouter()

	// === Global Middleware ===
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Channel", "X-Device-ID", "X-Wallet-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// === Health Check ===
	r.Get("/health", deps.HealthHandler.Check)

	// === API v2 Routes ===
	r.Route("/api/v2", func(r chi.Router) {
		// Enrollment — no pipeline needed
		r.Post("/enroll", deps.EnrollHandler.Enroll)

		// === Transaction Pipeline ===
		// v3.0 Pipeline: Rate → Log → AntiReplay → Limits → HMAC → PayeeType → TxType → Handler
		r.Route("/transaction", func(r chi.Router) {
			r.Use(deps.RateLimiter.Middleware)        // حماية عامة
			r.Use(deps.RequestLogger.Middleware)       // تدقيق
			r.Use(deps.AntiReplay.Middleware)          // counter check
			r.Use(deps.LimitsChecker.Middleware)       // قيود الإنفاق — البند 1
			r.Use(deps.SigVerifierA.Middleware)        // التحقق من HMAC — البند 3
			r.Use(deps.PayeeTypeVerifier.Middleware)   // التحقق من PayeeType — البند 4 [جديد]
			r.Use(deps.TxTypeResolver.Middleware)      // تحديد TransactionType — البند 5 [جديد]

			// Handler (Router → Adapter → Saga)
			r.Post("/", deps.TransactionHandler.Process)
		})

		// Transaction status — no pipeline needed
		r.Get("/transaction/{txId}", deps.TransactionHandler.GetStatus)

		// Device management
		r.Route("/device", func(r chi.Router) {
			r.Get("/{deviceId}", deps.DeviceHandler.GetDevice)
			r.Post("/{deviceId}/suspend", deps.DeviceHandler.Suspend)
			r.Post("/{deviceId}/revoke", deps.DeviceHandler.Revoke)
		})

		// Key rotation
		r.Post("/key/rotate", deps.EnrollHandler.RotateKey)

		// Disputes
		r.Route("/dispute", func(r chi.Router) {
			r.Post("/", deps.DisputeHandler.Open)
			r.Get("/", deps.DisputeHandler.List)
			r.Put("/{disputeId}", deps.DisputeHandler.Update)
		})

		// Limits
		r.Route("/limits", func(r chi.Router) {
			r.Get("/", deps.LimitsHandler.Get)
			r.Put("/", deps.LimitsHandler.Update)
		})
	})

	return r
}
