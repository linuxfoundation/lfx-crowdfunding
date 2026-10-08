// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package fga provides a read-only client to the platform's fga-sync service,
// used to answer "can this principal manage this entity?" for CF's attributed
// initiatives. See backend/docs/rewrite/11-initiative-attribution-and-access.md
// (PR #230, pending merge) §2.2, §3.1 and lfx-v2-fga-sync/docs/fga-sync-contract.md
// for the NATS wire contract.
//
// This package is a consumer only — CF never publishes tuples, so it has no
// fga-contract.md and needs no fga-catalog.md entry (those are for
// publishers). It reaches fga-sync over direct on-network NATS (transit
// option C in the design doc, approved by the platform team) rather than the
// public HTTP /access-check endpoint used by the MCP-server precedent
// (lfx-mcp/internal/lfxv2/access_check.go) — different transport, same wire
// semantics: object#relation@user tuples, unordered replies correlated by
// the echoed request token.
package fga

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// AccessCheckSubject is the fga-sync NATS subject for batch authorization
// checks (lfx-v2-fga-sync/docs/fga-sync-contract.md).
const AccessCheckSubject = "lfx.access_check.request"

// EntityRoleResolver answers "can this principal manage this entity?" for a
// single attributed entity (a b2b_org or an LF project).
//
// This interface is the migration seam called out in design §3.4: when CF
// moves behind the Heimdall gateway, an implementation backed by a single
// crowdfunding_initiative#writer check can replace NATSResolver without any
// change to callers.
type EntityRoleResolver interface {
	// CanManage reports whether username is a writer on the given entity.
	// attrType must be models.AttributionOrganization or
	// models.AttributionProject — CanManage does not accept "personal"
	// (personal initiatives make no FGA call at all; see design §2.2).
	//
	// A definitive non-writer result returns (false, nil). A resolver/
	// transport error returns a non-nil error wrapping
	// domain.ErrUpstreamUnavailable — callers must never turn that into a
	// false/404; it must surface as 503 (design §2.2 "fail closed, but
	// distinguish the reason").
	CanManage(ctx context.Context, attrType models.AttributionType, entityUID, username string) (bool, error)
}

// natsRequester is the minimal subset of *nats.Conn the resolver needs.
// Abstracted out so tests can fake NATS request/reply without a live server;
// *nats.Conn satisfies it directly, so production callers pass one as-is.
type natsRequester interface {
	RequestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error)
}

// NATSResolver implements EntityRoleResolver over the fga-sync
// lfx.access_check.request subject.
type NATSResolver struct {
	conn natsRequester
	// timeout bounds each request/reply round trip. nats.Timeout on the
	// connection only covers initial connection setup, not individual
	// requests, so CanManage must apply this itself — otherwise a caller
	// context without a deadline could block indefinitely on a subscriber
	// that receives the request but never replies. Zero means "no bound",
	// which keeps existing test doubles constructed without it working
	// unchanged.
	timeout time.Duration
}

// NewNATSResolver wraps a NATS connection for access checks. Callers own the
// connection's lifecycle (created and closed in cmd/initiatives-api). timeout
// bounds each access-check request/reply round trip.
func NewNATSResolver(conn *nats.Conn, timeout time.Duration) *NATSResolver {
	return &NATSResolver{conn: conn, timeout: timeout}
}

// entityTypePrefix maps CF's attribution type to the OpenFGA object-type
// prefix used in tuple strings. b2b_org is member-service's object type for
// canonical platform organizations (fga-sync-contract.md); project is
// project-service's. The relation matches what the gateway checks: projects
// use the *_guard relations (writer_guard, auditor_guard, which include the
// global_writer/global_auditor holders), so global holders pass here exactly as
// they do at Heimdall; b2b_org has no guard relations. guard is the suffix to
// append to a base relation (writer, auditor).
func entityTypePrefix(attrType models.AttributionType) (prefix, guard string, err error) {
	switch attrType {
	case models.AttributionOrganization:
		return "b2b_org", "", nil
	case models.AttributionProject:
		return "project", "_guard", nil
	default:
		return "", "", fmt.Errorf("access check requires organization or project attribution, got %q", attrType)
	}
}

// CanManage implements EntityRoleResolver.
func (r *NATSResolver) CanManage(ctx context.Context, attrType models.AttributionType, entityUID, username string) (bool, error) {
	prefix, guard, err := entityTypePrefix(attrType)
	if err != nil {
		return false, err
	}
	request := fmt.Sprintf("%s:%s#%s@user:%s", prefix, entityUID, "writer"+guard, username)
	results, err := r.checks(ctx, request)
	if err != nil {
		return false, err
	}
	return results[request], nil
}

// checks sends one batched lfx.access_check.request and returns the result for
// every token. Transport, parse and missing-result failures wrap
// domain.ErrUpstreamUnavailable.
func (r *NATSResolver) checks(ctx context.Context, requests ...string) (map[string]bool, error) {
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	msg, err := r.conn.RequestWithContext(ctx, AccessCheckSubject, []byte(strings.Join(requests, "\n")))
	if err != nil {
		return nil, fmt.Errorf("%w: access check request: %w", domain.ErrUpstreamUnavailable, err)
	}

	results, err := parseAccessCheckReply(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrUpstreamUnavailable, err)
	}
	for _, req := range requests {
		if _, ok := results[req]; !ok {
			return nil, fmt.Errorf("%w: access check returned no result for %q", domain.ErrUpstreamUnavailable, req)
		}
	}
	return results, nil
}

// parseAccessCheckReply parses a lfx.access_check.request reply: one line per
// check, tab-delimited "{request}\t{true|false}". Order is not guaranteed —
// fga-sync may return cached results first — so results are returned keyed
// by the echoed request token, never by line position.
func parseAccessCheckReply(data []byte) (map[string]bool, error) {
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	results := make(map[string]bool, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed access-check result line (no tab delimiter): %q", line)
		}
		switch parts[1] {
		case "true":
			results[parts[0]] = true
		case "false":
			results[parts[0]] = false
		default:
			return nil, fmt.Errorf("malformed access-check result line (value not true/false): %q", line)
		}
	}
	return results, nil
}

// EntityAffiliationChecker answers "may this user attribute an initiative to
// this entity?" (lfx-crowdfunding#259). It mirrors Self Serve's lens view gate:
// a full FGA auditor check, so team grants and the b2b_org/project
// parent/child cascade count (lf-staff holds auditor via the model, so staff
// pass without an app-side persona check). Managing the initiative afterwards
// still needs writer (EntityRoleResolver).
type EntityAffiliationChecker interface {
	IsAffiliated(ctx context.Context, attrType models.AttributionType, entityUID, username string) (bool, error)
}

// IsAffiliated implements EntityAffiliationChecker with one access check.
// Like CanManage, a transport failure wraps domain.ErrUpstreamUnavailable and
// is never turned into false.
func (r *NATSResolver) IsAffiliated(ctx context.Context, attrType models.AttributionType, entityUID, username string) (bool, error) {
	prefix, guard, err := entityTypePrefix(attrType)
	if err != nil {
		return false, err
	}
	auditor := fmt.Sprintf("%s:%s#%s@user:%s", prefix, entityUID, "auditor"+guard, username)
	results, err := r.checks(ctx, auditor)
	if err != nil {
		return false, err
	}
	return results[auditor], nil
}
