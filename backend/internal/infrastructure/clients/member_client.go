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
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// b2bOrgLookupByWebsiteSubject is lfx-v2-member-service's domain-to-b2b_org lookup.
const b2bOrgLookupByWebsiteSubject = "lfx.member.b2b_org_lookup_by_website"

// b2bOrgLookupSubject is lfx-v2-member-service's b2b_org lookup by SFID.
const b2bOrgLookupSubject = "lfx.member.b2b_org_lookup"

// b2bOrgNotFound is member-service's reply when no single b2b_org owns the domain.
// It covers both "not found" and "ambiguous", which are indistinguishable by design.
const b2bOrgNotFound = "b2b org not found"

// b2bOrgLookupTimeout bounds one lookup. Both callers run off the request path:
// after the donation is saved, or in a CronJob.
const b2bOrgLookupTimeout = 10 * time.Second

var memberTracer = otel.Tracer("member-client")

// MemberClient queries lfx-v2-member-service over NATS request/reply.
type MemberClient struct {
	// ponytail: EmailRequester is the generic *nats.Conn subset; both clients share the connection.
	conn EmailRequester
}

// NewMemberClient returns a MemberClient that sends over conn.
func NewMemberClient(conn EmailRequester) *MemberClient {
	return &MemberClient{conn: conn}
}

// LookupB2BOrgByWebsite returns the b2b_org that owns domain, matched by primary
// domain, domain alias or Account website: its 18-char Salesforce Account SFID
// and its logo URL (#472). It returns a zero match with a nil error when there is
// no single match, and an error when the lookup itself failed and is worth retrying.
func (c *MemberClient) LookupB2BOrgByWebsite(ctx context.Context, domain string) (models.B2BOrgMatch, error) {
	// An empty name skips member-service's name-only tier: only domain matches count.
	data, err := json.Marshal(struct {
		Name    string `json:"name"`
		Website string `json:"website"`
	}{Website: domain})
	if err != nil {
		return models.B2BOrgMatch{}, fmt.Errorf("marshal b2b org lookup request: %w", err)
	}
	return c.request(ctx, b2bOrgLookupByWebsiteSubject, data)
}

// GetB2BOrgLogo returns the logo URL of the b2b_org sfid, "" when it has none.
// found is false when member-service no longer knows the org.
func (c *MemberClient) GetB2BOrgLogo(ctx context.Context, sfid string) (logoURL string, found bool, err error) {
	data, err := json.Marshal(struct {
		ID string `json:"id"`
	}{ID: sfid})
	if err != nil {
		return "", false, fmt.Errorf("marshal b2b org lookup request: %w", err)
	}
	match, err := c.request(ctx, b2bOrgLookupSubject, data)
	if err != nil {
		return "", false, err
	}
	return match.LogoURL, match.SFID != "", nil
}

// request sends one lookup on subject and decodes member-service's reply.
func (c *MemberClient) request(ctx context.Context, subject string, data []byte) (models.B2BOrgMatch, error) {
	ctx, span := memberTracer.Start(ctx, "nats.request "+subject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", subject),
		),
	)
	defer span.End()

	match, err := c.lookup(ctx, subject, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return models.B2BOrgMatch{}, err
	}
	span.SetAttributes(attribute.Bool("b2b_org.matched", match.SFID != ""))
	return match, nil
}

func (c *MemberClient) lookup(ctx context.Context, subject string, data []byte) (models.B2BOrgMatch, error) {
	ctx, cancel := context.WithTimeout(ctx, b2bOrgLookupTimeout)
	defer cancel()

	req := nats.NewMsg(subject)
	req.Data = data
	// NATS headers are case-sensitive; http.Header canonicalisation would hide traceparent from the consumer.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		req.Header.Set(k, v)
	}

	reply, err := c.conn.RequestMsgWithContext(ctx, req)
	if err != nil {
		return models.B2BOrgMatch{}, fmt.Errorf("member service request: %w", err)
	}

	var resp struct {
		ID      string `json:"id"`
		LogoURL string `json:"logo_url"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return models.B2BOrgMatch{}, fmt.Errorf("decode b2b org lookup reply: %w", err)
	}
	switch {
	case resp.Error == b2bOrgNotFound:
		return models.B2BOrgMatch{}, nil
	case resp.Error != "":
		return models.B2BOrgMatch{}, fmt.Errorf("b2b org lookup: %s", resp.Error)
	case resp.ID == "":
		return models.B2BOrgMatch{}, errors.New("decode b2b org lookup reply: no id")
	}
	return models.B2BOrgMatch{SFID: resp.ID, LogoURL: resp.LogoURL}, nil
}
