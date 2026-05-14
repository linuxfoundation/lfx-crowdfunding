// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

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
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	// Database pool
	pool, err := db.NewPool(context.Background(), db.PoolConfig{
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

	// Clients
	ledgerClient := clients.NewLedgerClient(clients.LedgerConfig{
		BaseURL: cfg.Ledger.BaseURL,
		APIKey:  cfg.Ledger.APIKey,
		Timeout: cfg.Ledger.Timeout,
	})
	stripeClient := clients.NewStripeClient(clients.StripeConfig{
		SecretKey:     cfg.Stripe.SecretKey,
		WebhookSecret: cfg.Stripe.WebhookSecret,
		Timeout:       cfg.Stripe.Timeout,
	})

	// Services
	initiativeSvc := service.NewInitiativeService(initiativeRepo, ledgerClient, stripeClient)
	donationSvc := service.NewDonationService(donationRepo, initiativeRepo, stripeClient)
	subscriptionSvc := service.NewSubscriptionService(subscriptionRepo, initiativeRepo, stripeClient)

	// JWT authenticator
	jwtAuth, err := auth.NewJWTAuthenticator(auth.JWTAuthConfig{
		JWKSURL:                    cfg.JWT.JWKSURL,
		Audience:                   cfg.JWT.Audience,
		Issuer:                     cfg.JWT.Issuer,
		ClockSkew:                  cfg.JWT.ClockSkew,
		DisabledMockLocalPrincipal: cfg.Local.DisabledMockLocalPrincipal,
	})
	if err != nil {
		return nil, fmt.Errorf("jwt authenticator: %w", err)
	}

	// Handlers
	initiativeH := handler.NewInitiativeHandler(initiativeSvc)
	donationH := handler.NewDonationHandler(donationSvc)
	subscriptionH := handler.NewSubscriptionHandler(subscriptionSvc)
	webhookH := handler.NewWebhookHandler(stripeClient, cfg.Stripe.WebhookSecret, logger)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(cfg.Server.WriteTimeout))

	// Health endpoints (no auth)
	r.Get("/livez", handleLivez)
	r.Get("/healthz", handleLivez)
	r.Get("/readyz", handleReadyz(pool))

	// Stripe webhook (no JWT — uses its own HMAC signature validation)
	r.Post("/v1/stripe/webhook", webhookH.Handle)

	// Protected API
	r.Route("/v1", func(r chi.Router) {
		r.Use(jwtAuth.Middleware)

		r.Route("/initiatives", func(r chi.Router) {
			r.Get("/", initiativeH.List)
			r.Post("/", initiativeH.Create)
			r.Get("/{id}", initiativeH.GetByID)
			r.Patch("/{id}", initiativeH.Update)
			r.Delete("/{id}", initiativeH.Delete)
			r.Get("/{id}/goals", initiativeH.ListGoals)
			r.Get("/{id}/donations", donationH.List)
			r.Post("/{id}/donations", donationH.Create)
			r.Get("/{id}/subscriptions", subscriptionH.List)
			r.Post("/{id}/subscriptions", subscriptionH.Create)
		})

		r.Delete("/subscriptions/{id}", subscriptionH.Cancel)
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
