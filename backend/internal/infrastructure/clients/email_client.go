// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package clients provides outbound HTTP clients for external services.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	emailapi "github.com/linuxfoundation/lfx-v2-email-service/pkg/api"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// emailSendTimeout bounds one send. Donation emails go out inline in the Stripe
// webhook, so a stalled email service must not hold the webhook for long.
const emailSendTimeout = 5 * time.Second

var emailTracer = otel.Tracer("email-client")

// ErrEmailRejected marks a send the email service refused; the relay does not retry it.
var ErrEmailRejected = errors.New("email service rejected send")

// EmailRequester is the subset of *nats.Conn the email client needs.
type EmailRequester interface {
	RequestMsgWithContext(ctx context.Context, msg *nats.Msg) (*nats.Msg, error)
}

// EmailMessage is one pre-rendered email to a single recipient.
type EmailMessage struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// EmailSender delivers one rendered EmailMessage.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// EmailClient sends email through lfx-v2-email-service's NATS request/reply subject.
type EmailClient struct {
	conn            EmailRequester
	fromDisplayName string
}

// NewEmailClient returns an EmailClient that sends over conn. fromDisplayName
// is the sender name shown in the inbox; the address stays the platform default.
func NewEmailClient(conn EmailRequester, fromDisplayName string) *EmailClient {
	return &EmailClient{conn: conn, fromDisplayName: fromDisplayName}
}

// Send delivers msg to the email service and returns once it is accepted.
// Acceptance is not delivery: bounces arrive later on a separate subject.
func (c *EmailClient) Send(ctx context.Context, msg EmailMessage) error {
	data, err := json.Marshal(emailapi.SendEmailRequest{
		To:              msg.To,
		Subject:         msg.Subject,
		HTML:            msg.HTML,
		Text:            msg.Text,
		FromDisplayName: c.fromDisplayName,
	})
	if err != nil {
		return fmt.Errorf("marshal send email request: %w", err)
	}

	ctx, span := emailTracer.Start(ctx, "nats.request "+emailapi.SendEmailSubject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", emailapi.SendEmailSubject),
			attribute.Int("messaging.message.body.size", len(data)),
		),
	)
	defer span.End()

	emailID, err := c.request(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetAttributes(attribute.String("email.email_id", emailID))
	return nil
}

func (c *EmailClient) request(ctx context.Context, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, emailSendTimeout)
	defer cancel()

	req := nats.NewMsg(emailapi.SendEmailSubject)
	req.Data = data
	// NATS headers are case-sensitive; http.Header canonicalisation would hide traceparent from the consumer.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		req.Header.Set(k, v)
	}

	reply, err := c.conn.RequestMsgWithContext(ctx, req)
	if err != nil {
		return "", fmt.Errorf("email service request: %w", err)
	}

	var errResp emailapi.SendEmailErrorResponse
	if err := json.Unmarshal(reply.Data, &errResp); err == nil && errResp.Error != "" {
		return "", fmt.Errorf("%w: %s", ErrEmailRejected, errResp.Error)
	}
	// Decode email_id as a pointer: a recipient blocked by the non-prod allowlist
	// gets "" (counted as sent), but a reply without the field is not a send reply.
	var resp struct {
		EmailID *string `json:"email_id"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return "", fmt.Errorf("decode email service reply: %w", err)
	}
	if resp.EmailID == nil {
		return "", errors.New("decode email service reply: no email_id")
	}
	return *resp.EmailID, nil
}
