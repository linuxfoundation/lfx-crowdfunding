// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main starts the initiatives API server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/pkg/utils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := LoadConfig()
	if err != nil {
		logger.Error("config error", "error", err)
		os.Exit(1)
	}

	// Initialise OpenTelemetry
	shutdownOTel, err := utils.InitOTel(context.Background(), utils.OTelConfig{
		ServiceName:    cfg.OTel.ServiceName,
		ServiceVersion: cfg.OTel.ServiceVersion,
		Endpoint:       cfg.OTel.Endpoint,
	})
	if err != nil {
		logger.Error("otel init error", "error", err)
		os.Exit(1)
	}
	defer shutdownOTel()

	srv, err := NewServer(cfg, logger)
	if err != nil {
		logger.Error("server init error", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGTERM / SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down…")
	srv.Shutdown()
	logger.Info("stopped")
}
