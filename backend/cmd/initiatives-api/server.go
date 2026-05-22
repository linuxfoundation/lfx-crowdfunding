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

	// Services
	initiativeSvc := service.NewInitiativeService(initiativeRepo, ledgerClient, stripeClient, logger)
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
		DisabledMockLocalPrincipal: cfg.Local.DisabledMockLocalPrincipal,
	})
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
	initiativeH := handler.NewInitiativeHandler(initiativeSvc, cfg.Approval.AllowedApprovers)
	donationH := handler.NewDonationHandler(donationSvc)
	subscriptionH := handler.NewSubscriptionHandler(subscriptionSvc)
	paymentH := handler.NewPaymentHandler(paymentSvc)
	statisticsH := handler.NewStatisticsHandler(statisticsSvc)
	webhookH := handler.NewWebhookHandler(stripeClient, donationRepo, subscriptionRepo, cfg.Stripe.WebhookSecret, logger, cfg.Stripe.AckUnimplementedWebhooks)

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
	r.Get("/v1/initiatives/{id}", initiativeH.GetByID)
	r.Get("/v1/initiatives/{id}/transactions", initiativeH.GetTransactions)

	// Protected API
	r.Route("/v1", func(r chi.Router) {
		r.Use(jwtAuth.Middleware)

		r.Route("/initiatives", func(r chi.Router) {
			r.Post("/", initiativeH.Create)
			r.Patch("/{id}", initiativeH.Update)
			r.Delete("/{id}", initiativeH.Delete)
			r.Post("/{id}/approval/{status}", initiativeH.Approval)
			r.Get("/{id}/donations", donationH.List)
			r.Post("/{id}/donations", donationH.Create)
			r.Get("/{id}/subscriptions", subscriptionH.List)
			r.Post("/{id}/subscriptions", subscriptionH.Create)
		})

		r.Delete("/subscriptions/{id}", subscriptionH.Cancel)
		r.Get("/me/donations", donationH.ListForUser)
		r.Get("/me/subscriptions", subscriptionH.ListForUser)

		// Payment account (saved card for 3DS flows).
		r.Post("/me/setup-intent", paymentH.CreateSetupIntent)
		r.Post("/me/payment-method", paymentH.AttachPaymentMethod)
		r.Get("/me/payment-account", paymentH.GetPaymentAccount)
		r.Delete("/me/payment-method", paymentH.DeletePaymentMethod)
	})

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
