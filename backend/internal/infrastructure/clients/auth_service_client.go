// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// userMetadataReadSubject is lfx-v2-auth-service's profile lookup. It takes a
// bare username (or sub, or token) as the payload, not JSON.
const userMetadataReadSubject = "lfx.auth-service.user_metadata.read"

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
	ctx, span := authServiceTracer.Start(ctx, "nats.request "+userMetadataReadSubject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", userMetadataReadSubject),
		),
	)
	defer span.End()

	md, err := c.request(ctx, username)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return md, nil
}

func (c *AuthServiceClient) request(ctx context.Context, username string) (*UserMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, authServiceTimeout)
	defer cancel()

	reply, err := c.conn.RequestWithContext(ctx, userMetadataReadSubject, []byte(username))
	if err != nil {
		return nil, fmt.Errorf("auth-service user_metadata request: %w", err)
	}

	var resp struct {
		Success bool          `json:"success"`
		Error   string        `json:"error"`
		Data    *UserMetadata `json:"data"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return nil, fmt.Errorf("decode auth-service user_metadata reply: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("auth-service user_metadata: %s", resp.Error)
	}
	if resp.Data == nil {
		return nil, errors.New("auth-service user_metadata: no data")
	}
	return resp.Data, nil
}
