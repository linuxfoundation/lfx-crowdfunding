// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package core provides shared HTTP client utilities with OTel trace propagation.
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPClient wraps http.Client with OTel tracing and a JSON helper.
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient returns an HTTPClient with an otelhttp transport and the given timeout.
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// GetJSON performs a GET request and unmarshals the JSON response body into dest.
// errHandler is called when the status code is >= 400; return a wrapped error from it.
func (c *HTTPClient) GetJSON(ctx context.Context, url string, headers map[string]string, dest any, errHandler func(*http.Response) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		if errHandler != nil {
			return errHandler(resp)
		}
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// PostJSON performs a POST request with a JSON-encoded body and unmarshals the
// JSON response into dest (may be nil to discard the response body).
// errHandler is called when the status code is >= 400.
func (c *HTTPClient) PostJSON(ctx context.Context, reqURL string, headers map[string]string, body any, dest any, errHandler func(*http.Response) error) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		if errHandler != nil {
			return errHandler(resp)
		}
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// PatchJSON performs a PATCH request with a JSON-encoded body and unmarshals the
// JSON response into dest (may be nil to discard the response body).
// errHandler is called when the status code is >= 400.
func (c *HTTPClient) PatchJSON(ctx context.Context, reqURL string, headers map[string]string, body any, dest any, errHandler func(*http.Response) error) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		if errHandler != nil {
			return errHandler(resp)
		}
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
