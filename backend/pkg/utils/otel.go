// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package utils provides shared OpenTelemetry initialisation helpers.
package utils

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

// OTelConfig holds the parameters needed to initialise OTel tracing.
type OTelConfig struct {
	ServiceName    string
	ServiceVersion string
	// Endpoint is the OTLP HTTP endpoint (e.g. "http://localhost:4318").
	// If empty, a no-op tracer is used.
	Endpoint string
}

// InitOTel configures the global OpenTelemetry tracer and returns a shutdown func.
func InitOTel(ctx context.Context, cfg OTelConfig) (func(), error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	var tp *sdktrace.TracerProvider
	if cfg.Endpoint != "" {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(cfg.Endpoint),
		)
		if err != nil {
			return nil, fmt.Errorf("otel exporter: %w", err)
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(res),
		)
	} else {
		// No endpoint configured — use a no-op tracer (still creates valid spans for tests).
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		_ = tp.Shutdown(context.Background())
	}, nil
}
