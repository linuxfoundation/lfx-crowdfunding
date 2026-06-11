// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main starts the initiatives API server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/handler"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/db"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Server wraps the Chi router and all service dependencies.
type Server struct {
	router  *chi.Mux
	pool    *pgxpool.Pool
	cfg     *Config
	logger  *slog.Logger
	httpSrv *http.Server
}

// NewServer wires up all dependencies and builds the Chi router.
func NewServer(ctx context.Context, cfg *Config, logger *slog.Logger) (*Server, error) {
	// Database pool
	pool, err := db.NewPool(ctx, db.PoolConfig{
		DSN:             cfg.Database.DSN,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("database pool: %w", err)
	}

	// Repositories
	initiativeRepo := db.NewInitiativeRepository(pool)
	donationRepo := db.NewDonationRepository(pool)
	subscriptionRepo := db.NewSubscriptionRepository(pool)
	statisticsRepo := db.NewStatisticsRepository(pool)
	userRepo := db.NewUserRepository(pool)

	// Clients
	ledgerClient := clients.NewLedgerClient(clients.LedgerConfig{
		BaseURL: cfg.Ledger.BaseURL,
		APIKey:  cfg.Ledger.APIKey,
		Timeout: cfg.Ledger.Timeout,
	})
	stripeClient := clients.NewStripeClient(clients.StripeConfig{
		SecretKey: cfg.Stripe.SecretKey,
		Timeout:   cfg.Stripe.Timeout,
		ReturnURL: cfg.Stripe.ReturnURL,
	})
	s3Client, err := clients.NewS3PresignClient(ctx, clients.S3Config{
		BucketName:    cfg.S3.BucketName,
		Region:        cfg.S3.Region,
		PresignExpiry: cfg.S3.PresignExpiry,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}

	// Mandrill email client
	mandrillClient := clients.NewMandrillClient(clients.MandrillConfig{
		APIKey:    cfg.Mandrill.APIKey,
		FromEmail: cfg.Mandrill.FromEmail,
		FromName:  cfg.Mandrill.FromName,
		Timeout:   cfg.Mandrill.Timeout,
	})
	emailSvc := clients.NewEmailService(mandrillClient, cfg.Mandrill.FrontendBase, cfg.Mandrill.NotificationEmails)

	// Reimbursement Service client — nil when REIMBURSEMENTS_API_URL is unset
	// (integration disabled; no sync calls are made).
	// Fail fast when URL is set but KEY is missing — the service would otherwise
	// silently fail every sync with 401/403.
	if err := validateReimbursementConfig(cfg.Reimbursement); err != nil {
		return nil, fmt.Errorf("reimbursement config: %w", err)
	}
	reimbursementClient := clients.NewReimbursementClient(clients.ReimbursementConfig{
		APIURL:       cfg.Reimbursement.APIURL,
		APIKey:       cfg.Reimbursement.APIKey,
		FrontendBase: cfg.Mandrill.FrontendBase,
		Timeout:      cfg.Reimbursement.Timeout,
	})
	if reimbursementClient == nil {
		logger.Warn("REIMBURSEMENTS_API_URL is not set — Reimbursement Service sync is disabled")
	}

	// Services
	initiativeSvc := service.NewInitiativeService(initiativeRepo, userRepo, ledgerClient, stripeClient, emailSvc, reimbursementClient, logger)
	donationSvc := service.NewDonationService(donationRepo, initiativeRepo, userRepo, stripeClient)
	subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, initiativeRepo, userRepo, stripeClient)
	paymentSvc := service.NewPaymentService(userRepo, stripeClient)
	statisticsSvc := service.NewStatisticsService(statisticsRepo, ledgerClient)

	// JWT authenticator
	jwtAuth, err := auth.NewJWTAuthenticator(ctx, auth.JWTAuthConfig{
		JWKSURL:                    cfg.JWT.JWKSURL,
		Audience:                   cfg.JWT.Audience,
		Issuer:                     cfg.JWT.Issuer,
		ClockSkew:                  cfg.JWT.ClockSkew,
		AllowMockPrincipalBypass:   cfg.Local.AllowMockLocalPrincipalBypass,
		DisabledMockLocalPrincipal: cfg.Local.DisabledMockLocalPrincipal,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("jwt authenticator: %w", err)
	}
	if jwtAuth.IsBypassActive() {
		logger.Warn("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		logger.Warn("!!! JWT AUTHENTICATION IS DISABLED — ALL REQUESTS ARE    !!!")
		logger.Warn("!!! TREATED AS AUTHENTICATED. NEVER USE IN PRODUCTION.   !!!")
		logger.Warn("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	}

	// Handlers
	initiativeH := handler.NewInitiativeHandler(initiativeSvc, cfg.Approval.AllowedApprovers, logger)
	donationH := handler.NewDonationHandler(donationSvc)
	subscriptionH := handler.NewSubscriptionHandler(subscriptionSvc)
	paymentH := handler.NewPaymentHandler(paymentSvc)
	statisticsH := handler.NewStatisticsHandler(statisticsSvc)
	webhookH := handler.NewWebhookHandler(stripeClient, ledgerClient, donationRepo, subscriptionRepo, emailSvc, cfg.Stripe.WebhookSecret, logger, cfg.Stripe.AckUnimplementedWebhooks)
	uploadH := handler.NewUploadHandler(s3Client)

	// UserInfo client — fetches full profile from Auth0 on login sync.
	// In bypass mode (local dev) there is no real Auth0, so use a mock fetcher.
	var userInfoFetcher auth.UserInfoFetcher
	if jwtAuth.IsBypassActive() {
		userInfoFetcher = auth.NewMockUserInfoFetcher(cfg.Local.DisabledMockLocalPrincipal)
	} else {
		userInfoFetcher, err = auth.NewUserInfoClient(cfg.JWT.Issuer, nil)
		if err != nil {
			return nil, fmt.Errorf("userinfo client: %w", err)
		}
	}
	userH := handler.NewUserHandler(userRepo, userInfoFetcher)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	// Chi's context timeout must be shorter than the HTTP WriteTimeout so the
	// handler has time to write a graceful 504 before the server closes the
	// connection. 80% of WriteTimeout is a safe margin.
	// When WriteTimeout is 0 (disabled) the computed value is also 0, which
	// causes chimiddleware.Timeout to panic. Skip the middleware in that case.
	if chiTimeout := cfg.Server.WriteTimeout * 4 / 5; chiTimeout > 0 {
		r.Use(chimiddleware.Timeout(chiTimeout))
	}

	// Health endpoints (no auth)
	r.Get("/livez", handleLivez)
	r.Get("/healthz", handleLivez)
	r.Get("/readyz", handleReadyz(pool))

	// Stripe webhook (no JWT — uses its own HMAC signature validation)
	r.Post("/v1/stripe/webhook", webhookH.Handle)

	// Public API (no auth)
	r.Get("/v1/statistics", statisticsH.GetPlatform)
	r.Get("/v1/statistics/platform", statisticsH.GetPlatformDetails)
	r.Get("/v1/statistics/monthly", statisticsH.GetPlatformMonthly)
	r.Get("/v1/statistics/recent-donations", statisticsH.GetRecentDonations)
	r.Get("/v1/initiatives", initiativeH.List)
	r.Get("/v1/initiatives/{id}/transactions", initiativeH.GetTransactions)

	// Initiative detail — public for published initiatives; approvers may also
	// view non-published initiatives if a valid token is supplied.
	r.With(jwtAuth.OptionalMiddleware).Get("/v1/initiatives/{id}", initiativeH.GetByID)

	// Protected API — requires a valid bearer token with access:me scope.
	// All routes are under /v1/me/* to make the identity-scoped contract explicit.
	r.Route("/v1/me", func(r chi.Router) {
		r.Use(jwtAuth.Middleware)
		r.Use(jwtAuth.RequireScope(auth.ScopeMe))

		// Profile sync — calls Auth0 UserInfo, writes to DB.
		r.Patch("/", userH.SyncProfile)

		// Caller's own initiatives, donations, and subscriptions across all initiatives.
		r.Get("/initiatives", initiativeH.ListForUser)
		r.Get("/donations", donationH.ListForUser)
		r.Get("/subscriptions", subscriptionH.ListForUser)

		// Payment account (saved card for 3DS flows).
		r.Post("/setup-intent", paymentH.CreateSetupIntent)
		r.Post("/payment-method", paymentH.AttachPaymentMethod)
		r.Get("/payment-account", paymentH.GetPaymentAccount)
		r.Delete("/payment-method", paymentH.DeletePaymentMethod)

		// Logo uploads (used during initiative creation by the owning user).
		r.Post("/presigned-url", uploadH.CreatePresignedURL)

		// Owner-checked initiative read + mutations. The detail read returns the
		// caller's own initiative in any status (the public detail endpoint hides
		// non-published initiatives from non-approvers).
		r.Get("/initiatives/{id}", initiativeH.GetForUser)
		r.Get("/initiatives/{id}/transactions", initiativeH.GetTransactionsForUser)
		r.Post("/initiatives", initiativeH.Create)
		r.Patch("/initiatives/{id}", initiativeH.Update)
		r.Delete("/initiatives/{id}", initiativeH.Delete)

		// Donations and subscriptions on a specific initiative (caller is the donor).
		r.Get("/initiatives/{id}/donations", donationH.List)
		r.Post("/initiatives/{id}/donations", donationH.Create)
		r.Get("/initiatives/{id}/subscriptions", subscriptionH.List)
		r.Post("/initiatives/{id}/subscriptions", subscriptionH.Create)
		r.Delete("/subscriptions/{id}", subscriptionH.Cancel)
	})

	// Approval route — caller is an approver (allowlist check), not the resource owner.
	// Lives outside /v1/me because the URL is initiative-scoped rather than
	// identity-scoped; the handler enforces its own approver allowlist check.
	// TODO: when M2M approver tokens are issued, switch to RequireScope(auth.ScopeManage).
	// For now all callers hold user tokens with access:me, so that scope is used.
	r.With(jwtAuth.Middleware, jwtAuth.RequireScope(auth.ScopeMe)).
		Post("/v1/initiatives/{id}/process-approval/{action}", initiativeH.ProcessApproval)

	// M2M routes — require a valid bearer token with access:manage scope.
	// These endpoints are for service-to-service callers, not end users.
	r.With(jwtAuth.Middleware, jwtAuth.RequireScope(auth.ScopeManage)).
		Get("/v1/initiatives/{slug}/owner-info", initiativeH.GetOwnerEmail)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      otelhttp.NewHandler(r, "initiatives-api"),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	return &Server{
		router:  r,
		pool:    pool,
		cfg:     cfg,
		logger:  logger,
		httpSrv: httpSrv,
	}, nil
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	s.logger.Info("server listening", "addr", s.httpSrv.Addr)
	return s.httpSrv.ListenAndServe()
}

// Shutdown performs a graceful shutdown with the configured timeout.
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		s.logger.Error("graceful shutdown error", "error", err)
	}
	s.pool.Close()
}

func handleLivez(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleReadyz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, `{"status":"unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}
}
