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

	// Pipeline middleware (Layers 1-9)
	RateLimiter         *middleware.RateLimiter
	RequestLogger       *middleware.RequestLogger
	AntiReplay          *middleware.AntiReplay
	AttestationVerifier *middleware.AttestationVerifier
	SigVerifierA        *middleware.SignatureVerifierA
	SigVerifierB        *middleware.SignatureVerifierB
	CrossValidator      *middleware.CrossValidator
	LimitsChecker       *middleware.LimitsChecker
	Idempotency         *middleware.Idempotency
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
		ExposedHeaders:   []string{"X-Request-ID", "X-Idempotent"},
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
		// Layers execute in order: 1→2→3→4→5→6→7→8→9→Handler
		r.Route("/transaction", func(r chi.Router) {
			// Layer 1: Rate Limiter
			r.Use(deps.RateLimiter.Middleware)
			// Layer 2: Request Logger
			r.Use(deps.RequestLogger.Middleware)
			// Layer 3: Anti-Replay (parses body, stores in context)
			r.Use(deps.AntiReplay.Middleware)
			// Layer 4: Attestation Verifier (ECDSA from TEE)
			r.Use(deps.AttestationVerifier.Middleware)
			// Layer 5: Signature Verifier A (HMAC)
			r.Use(deps.SigVerifierA.Middleware)
			// Layer 6: Signature Verifier B (HMAC)
			r.Use(deps.SigVerifierB.Middleware)
			// Layer 7: Cross-Validator
			r.Use(deps.CrossValidator.Middleware)
			// Layer 8: Limits Checker
			r.Use(deps.LimitsChecker.Middleware)
			// Layer 9: Idempotency
			r.Use(deps.Idempotency.Middleware)

			// Layer 10: Transaction Handler (Router → Adapter → Saga)
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
