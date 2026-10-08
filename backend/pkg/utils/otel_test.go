// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestInitOTel_ExportsToTracesPath(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		wantPath string
	}{
		{name: "base URL", suffix: "", wantPath: "/v1/traces"},
		{name: "trailing slash", suffix: "/", wantPath: "/v1/traces"},
		{name: "base path", suffix: "/otlp", wantPath: "/otlp/v1/traces"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make(chan string, 10)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths <- r.URL.Path
			}))
			defer srv.Close()

			shutdown, err := InitOTel(context.Background(), OTelConfig{
				ServiceName: "test",
				Endpoint:    srv.URL + tt.suffix,
			})
			if err != nil {
				t.Fatalf("InitOTel error: %v", err)
			}
			_, span := otel.Tracer("test").Start(context.Background(), "span")
			span.End()
			shutdown() // flushes the batch

			select {
			case got := <-paths:
				if got != tt.wantPath {
					t.Errorf("export path = %q, want %q", got, tt.wantPath)
				}
			default:
				t.Fatal("no export request received")
			}
		})
	}
}

func TestInitOTel_InvalidEndpoint(t *testing.T) {
	if _, err := InitOTel(context.Background(), OTelConfig{ServiceName: "test", Endpoint: "http://[::1"}); err == nil {
		t.Fatal("InitOTel error = nil, want error for unparseable endpoint")
	}
}
