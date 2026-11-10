// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// lfx-v2-auth-service profile subjects. All accept a bare username as the
// user identifier; user_metadata.read and username_to_sub take it as the raw payload.
const (
	userMetadataReadSubject = "lfx.auth-service.user_metadata.read"
	userEmailsReadSubject   = "lfx.auth-service.user_emails.read"
	usernameToSubSubject    = "lfx.auth-service.username_to_sub"
)

// authServiceTimeout bounds one lookup. It runs inline in the login sync, so a
// stalled auth-service must not hold the request for long.
const authServiceTimeout = 3 * time.Second

var authServiceTracer = otel.Tracer("auth-service-client")

// AuthServiceRequester is the subset of *nats.Conn the auth-service client needs.
type AuthServiceRequester interface {
	RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error)
}

// UserMetadata is the profile subset Crowdfunding stores from auth-service.
type UserMetadata struct {
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
}

// AuthServiceClient reads user profiles from lfx-v2-auth-service over NATS.
type AuthServiceClient struct {
	conn AuthServiceRequester
}

// NewAuthServiceClient returns an AuthServiceClient that requests over conn.
func NewAuthServiceClient(conn AuthServiceRequester) *AuthServiceClient {
	return &AuthServiceClient{conn: conn}
}

// UserMetadata returns the profile auth-service holds for username.
func (c *AuthServiceClient) UserMetadata(ctx context.Context, username string) (*UserMetadata, error) {
	var md UserMetadata
	if err := c.request(ctx, userMetadataReadSubject, []byte(username), &md); err != nil {
		return nil, err
	}
	return &md, nil
}

// PrimaryEmail returns the primary email auth-service holds for username.
// Only call it for the authenticated user's own username: auth-service does not
// authorize username lookups itself.
func (c *AuthServiceClient) PrimaryEmail(ctx context.Context, username string) (string, error) {
	payload, err := json.Marshal(map[string]any{"user": map[string]string{"auth_token": username}})
	if err != nil {
		return "", fmt.Errorf("marshal user_emails request: %w", err)
	}
	var data struct {
		PrimaryEmail string `json:"primary_email"`
	}
	if err := c.request(ctx, userEmailsReadSubject, payload, &data); err != nil {
		return "", err
	}
	if data.PrimaryEmail == "" {
		return "", errors.New("auth-service user_emails: no primary_email")
	}
	return data.PrimaryEmail, nil
}

// UsernameToSub returns the Auth0 subject (e.g. "auth0|jdoe") auth-service maps
// username to. Ledger rows and Stripe metadata key users by it (#468). Unlike the
// other subjects, a success reply is the bare subject as plain text; only an
// error is a JSON envelope.
func (c *AuthServiceClient) UsernameToSub(ctx context.Context, username string) (sub string, err error) {
	ctx, span := authServiceTracer.Start(ctx, "nats.request "+usernameToSubSubject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", usernameToSubSubject),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, authServiceTimeout)
	defer cancel()

	reply, err := c.conn.RequestWithContext(ctx, usernameToSubSubject, []byte(username))
	if err != nil {
		return "", fmt.Errorf("auth-service %s request: %w", usernameToSubSubject, err)
	}
	body := strings.TrimSpace(string(reply.Data))
	if strings.HasPrefix(body, "{") {
		var resp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(reply.Data, &resp)
		return "", fmt.Errorf("auth-service %s: %s", usernameToSubSubject, cmp.Or(resp.Error, "error reply"))
	}
	// A subject is "<connection>|<id>"; anything else is not one to store.
	if !strings.Contains(body, "|") || strings.ContainsAny(body, " \t\n") {
		return "", fmt.Errorf("auth-service %s: unexpected reply %q", usernameToSubSubject, body)
	}
	return body, nil
}

// request sends payload to subject and decodes the reply's data into out.
func (c *AuthServiceClient) request(ctx context.Context, subject string, payload []byte, out any) (err error) {
	ctx, span := authServiceTracer.Start(ctx, "nats.request "+subject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", subject),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, authServiceTimeout)
	defer cancel()

	reply, err := c.conn.RequestWithContext(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("auth-service %s request: %w", subject, err)
	}

	var resp struct {
		Success bool            `json:"success"`
		Error   string          `json:"error"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return fmt.Errorf("decode auth-service %s reply: %w", subject, err)
	}
	if !resp.Success {
		return fmt.Errorf("auth-service %s: %s", subject, resp.Error)
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return fmt.Errorf("auth-service %s: no data", subject)
	}
	if err := json.Unmarshal(resp.Data, out); err != nil {
		return fmt.Errorf("decode auth-service %s data: %w", subject, err)
	}
	return nil
}
