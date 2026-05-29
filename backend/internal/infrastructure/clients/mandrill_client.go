// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var mandrillTracer = otel.Tracer("mandrill-client")

// ErrMandrillNotConfigured is returned by SendTemplate when the API key is empty.
// The email service propagates this sentinel; callers should log at WARN and continue.
var ErrMandrillNotConfigured = errors.New("mandrill: API key not configured")

// MandrillTemplateName is a strongly-typed Mandrill template slug.
type MandrillTemplateName string

const (
	// MandrillTemplateSubmittedForReview is sent to the reviewer inbox when a new
	// initiative is submitted and is awaiting approval.
	MandrillTemplateSubmittedForReview MandrillTemplateName = "communitybridge-review-mentorship-submission"
	// MandrillTemplateApproved is sent to the initiative owner when their
	// initiative is approved and goes live.
	MandrillTemplateApproved MandrillTemplateName = "admin-mentorship-submission-approved"
	// MandrillTemplateDeclined is sent to the initiative owner when their
	// initiative is declined.
	MandrillTemplateDeclined MandrillTemplateName = "admin-mentorship-submission-rejected"
)

// MandrillClient is the interface consumed by the email service layer.
type MandrillClient interface {
	// SendTemplate sends a Mandrill transactional template to a single recipient.
	// templateName must match the slug configured in Mandrill.
	// mergeVars is the map of UPPERCASE merge-tag names to their string values.
	SendTemplate(ctx context.Context, templateName MandrillTemplateName, toEmail, toName string, mergeVars map[string]string) error
}

// MandrillConfig holds Mandrill transactional email settings.
type MandrillConfig struct {
	APIKey    string
	FromEmail string        // e.g. "noreply@lfx.linuxfoundation.org"
	FromName  string        // e.g. "LFX Crowdfunding"
	Timeout   time.Duration // default 10s
}

type mandrillClient struct {
	cfg        MandrillConfig
	httpClient *http.Client
}

// NewMandrillClient creates a Mandrill HTTP client with OTel-traced transport.
func NewMandrillClient(cfg MandrillConfig) MandrillClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &mandrillClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// mandrillMergeVar is the wire format for a single global merge variable.
type mandrillMergeVar struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// mandrillRecipient is the wire format for an email recipient.
type mandrillRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

// mandrillMessage is the wire format for the Mandrill message object.
type mandrillMessage struct {
	FromEmail       string              `json:"from_email"`
	FromName        string              `json:"from_name"`
	To              []mandrillRecipient `json:"to"`
	GlobalMergeVars []mandrillMergeVar  `json:"global_merge_vars"`
}

// mandrillSendTemplateRequest is the full Mandrill send-template request body.
type mandrillSendTemplateRequest struct {
	Key             string          `json:"key"`
	TemplateName    string          `json:"template_name"`
	TemplateContent []struct{}      `json:"template_content"`
	Message         mandrillMessage `json:"message"`
}

const mandrillAPIURL = "https://mandrillapp.com/api/1.0/messages/send-template.json"

// mandrillSendResult is the per-recipient result object returned inside the
// Mandrill send-template response array.
type mandrillSendResult struct {
	Email        string `json:"email"`
	Status       string `json:"status"`        // "sent", "queued", "scheduled", "rejected", "invalid"
	RejectReason string `json:"reject_reason"` // non-empty when status == "rejected"
}

// SendTemplate sends a transactional template via the Mandrill API.
func (c *mandrillClient) SendTemplate(ctx context.Context, templateName MandrillTemplateName, toEmail, toName string, mergeVars map[string]string) error {
	if c.cfg.APIKey == "" {
		return ErrMandrillNotConfigured
	}

	ctx, span := mandrillTracer.Start(ctx, "mandrill.send-template")
	defer span.End()
	span.SetAttributes(attribute.String("mandrill.template", string(templateName)))

	globalMergeVars := make([]mandrillMergeVar, 0, len(mergeVars))
	for k, v := range mergeVars {
		globalMergeVars = append(globalMergeVars, mandrillMergeVar{Name: k, Content: v})
	}

	payload := mandrillSendTemplateRequest{
		Key:             c.cfg.APIKey,
		TemplateName:    string(templateName),
		TemplateContent: []struct{}{},
		Message: mandrillMessage{
			FromEmail:       c.cfg.FromEmail,
			FromName:        c.cfg.FromName,
			To:              []mandrillRecipient{{Email: toEmail, Name: toName, Type: "to"}},
			GlobalMergeVars: globalMergeVars,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mandrill: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mandrillAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mandrill: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mandrill: execute request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// Bound and drain the body so the underlying TCP connection can be reused.
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if readErr != nil {
		return fmt.Errorf("mandrill: read response body: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mandrill: unexpected status %d for template %q: %s", resp.StatusCode, templateName, respBody)
	}

	// Mandrill can return HTTP 200 while flagging per-recipient delivery failures
	// (e.g. status "rejected" or "invalid") inside the JSON array. Parse and surface them.
	// An unmarshal failure means an unexpected response format (e.g. error object or proxy HTML);
	// surface it so the caller can log a warning rather than silently treating it as success.
	var results []mandrillSendResult
	if err := json.Unmarshal(respBody, &results); err != nil {
		trunc := respBody
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		return fmt.Errorf("mandrill: unexpected response format for template %q: %s", templateName, trunc)
	}
	for _, r := range results {
		switch r.Status {
		case "sent", "queued", "scheduled":
			// accepted
		default:
			return fmt.Errorf("mandrill: recipient %q status %q (reason: %q) for template %q",
				r.Email, r.Status, r.RejectReason, templateName)
		}
	}

	return nil
}
