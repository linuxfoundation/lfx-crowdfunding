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
)

// b2bOrgLookupByWebsiteSubject is lfx-v2-member-service's domain-to-b2b_org lookup.
const b2bOrgLookupByWebsiteSubject = "lfx.member.b2b_org_lookup_by_website"

// b2bOrgNotFound is member-service's reply when no single b2b_org owns the domain.
// It covers both "not found" and "ambiguous", which are indistinguishable by design.
const b2bOrgNotFound = "b2b org not found"

// b2bOrgLookupTimeout bounds one lookup. It runs after the donation is saved,
// so it never holds up checkout.
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

// LookupB2BOrgByWebsite returns the 18-char Salesforce Account SFID of the b2b_org
// that owns domain, matched by primary domain, domain alias or Account website.
// It returns "" with a nil error when there is no single match, and an error when
// the lookup itself failed and is worth retrying.
func (c *MemberClient) LookupB2BOrgByWebsite(ctx context.Context, domain string) (string, error) {
	// An empty name skips member-service's name-only tier: only domain matches count.
	data, err := json.Marshal(struct {
		Name    string `json:"name"`
		Website string `json:"website"`
	}{Website: domain})
	if err != nil {
		return "", fmt.Errorf("marshal b2b org lookup request: %w", err)
	}

	ctx, span := memberTracer.Start(ctx, "nats.request "+b2bOrgLookupByWebsiteSubject,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", b2bOrgLookupByWebsiteSubject),
		),
	)
	defer span.End()

	sfid, err := c.lookup(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	span.SetAttributes(attribute.Bool("b2b_org.matched", sfid != ""))
	return sfid, nil
}

func (c *MemberClient) lookup(ctx context.Context, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, b2bOrgLookupTimeout)
	defer cancel()

	req := nats.NewMsg(b2bOrgLookupByWebsiteSubject)
	req.Data = data
	// NATS headers are case-sensitive; http.Header canonicalisation would hide traceparent from the consumer.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		req.Header.Set(k, v)
	}

	reply, err := c.conn.RequestMsgWithContext(ctx, req)
	if err != nil {
		return "", fmt.Errorf("member service request: %w", err)
	}

	var resp struct {
		ID    string `json:"id"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return "", fmt.Errorf("decode b2b org lookup reply: %w", err)
	}
	switch {
	case resp.Error == b2bOrgNotFound:
		return "", nil
	case resp.Error != "":
		return "", fmt.Errorf("b2b org lookup: %s", resp.Error)
	case resp.ID == "":
		return "", errors.New("decode b2b org lookup reply: no id")
	}
	return resp.ID, nil
}
