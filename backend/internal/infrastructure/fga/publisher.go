// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package fga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

// deleteFlushTimeout bounds FlushWithContext when the caller's ctx has no
// deadline of its own. nats.go's FlushWithContext requires a deadline
// (returns ErrNoDeadlineContext otherwise, context.go:181-184 as of v1.52.0),
// and syncFGADelete's detached context has none.
const deleteFlushTimeout = 5 * time.Second

// Subjects fga-sync's generic sync handler listens on
// (lfx-v2-fga-sync/docs/fga-sync-contract.md). Fire-and-forget: these
// subjects send no reply, JetStream persistence + at-least-once redelivery
// is fga-sync's job, not the publisher's.
const (
	UpdateAccessSubject = "lfx.fga-sync.update_access"
	DeleteAccessSubject = "lfx.fga-sync.delete_access"
)

// InitiativeObjectType is the OpenFGA object type CF publishes tuples for
// (lfx-crowdfunding#269, model accepted in lfx-v2-helm#172).
const InitiativeObjectType = "crowdfunding_initiative"

// GenericFGAMessage is fga-sync's generic sync envelope, shared by every
// publisher (fga-sync-contract.md "Access Message Envelope").
type GenericFGAMessage struct {
	ObjectType string `json:"object_type"`
	Operation  string `json:"operation"`
	Data       any    `json:"data"`
}

// updateAccessData is the payload shape for operation "update_access".
type updateAccessData struct {
	UID string `json:"uid"`
	// Public becomes the viewer:[user:*] wildcard tuple — fga-sync treats it
	// as a fixed relation outside the Relations full-sync (handler.go's
	// processStandardAccessUpdate), so toggling it is how the wildcard is
	// both granted and withdrawn.
	Public     bool                `json:"public"`
	Relations  map[string][]string `json:"relations"`
	References map[string][]string `json:"references,omitempty"`
}

// deleteAccessData is the payload shape for operation "delete_access".
type deleteAccessData struct {
	UID string `json:"uid"`
}

// natsPublisher is the minimal subset of *nats.Conn the publisher needs.
type natsPublisher interface {
	Publish(subj string, data []byte) error
	FlushWithContext(ctx context.Context) error
}

// Publisher publishes crowdfunding_initiative access tuples to fga-sync.
// A nil *Publisher is valid and every method is a no-op — this is how
// callers behave when FGA_NATS_URL is unset, matching the nil-client
// pattern already used for the optional Reimbursement Service integration
// (InitiativeService.reimbursement).
type Publisher struct {
	conn natsPublisher
}

// NewPublisher wraps a NATS connection for fga-sync tuple publishing.
// Callers own the connection's lifecycle. A nil conn returns a Publisher
// whose methods are all no-ops — the explicit nil check here matters:
// storing a nil *nats.Conn directly into the conn field's interface type
// would produce a non-nil interface wrapping a nil pointer, silently
// defeating every method's own "p.conn == nil" guard.
func NewPublisher(conn *nats.Conn) *Publisher {
	if conn == nil {
		return &Publisher{}
	}
	return &Publisher{conn: conn}
}

// InitiativeAccess is the full desired access state for one initiative.
// Both the live write path and the one-time backfill job build this from
// current Postgres state and pass it to UpdateAccess, so both republish
// through the same shape — the backfill is just "replay current state,"
// which is what makes it idempotent and safe to rerun.
type InitiativeAccess struct {
	UID           string
	OwnerUsername string
	Attribution   models.Attribution
	Published     bool // true iff status == published; drives the viewer:[user:*] wildcard
}

// UpdateAccess publishes the given initiative's current access state as a
// full sync: any relation not present is removed by fga-sync. A nil
// receiver or nil connection is a no-op.
func (p *Publisher) UpdateAccess(ctx context.Context, access InitiativeAccess) error {
	if p == nil || p.conn == nil {
		return nil
	}
	if access.UID == "" {
		return fmt.Errorf("update access: uid is required")
	}
	if access.OwnerUsername == "" {
		return fmt.Errorf("update access: owner username is required")
	}

	data := updateAccessData{
		UID:       access.UID,
		Public:    access.Published,
		Relations: map[string][]string{"owner": {access.OwnerUsername}},
	}
	if access.Attribution.EntityUID != "" {
		switch access.Attribution.Type {
		case models.AttributionProject:
			data.References = map[string][]string{"project": {access.Attribution.EntityUID}}
		case models.AttributionOrganization:
			data.References = map[string][]string{"b2b_org": {access.Attribution.EntityUID}}
		}
	}

	return p.publish(UpdateAccessSubject, GenericFGAMessage{
		ObjectType: InitiativeObjectType,
		Operation:  "update_access",
		Data:       data,
	})
}

// DeleteAccess publishes removal of every publisher-managed tuple for the
// given initiative, then flushes the connection so the caller can detect a
// broker-level publish failure (not fga-sync processing) before returning —
// unlike UpdateAccess, a missed delete can't be reconstructed by the
// backfill job (it has no record of what no longer exists), so it gets this
// stronger guarantee. A nil receiver or nil connection is a no-op.
func (p *Publisher) DeleteAccess(ctx context.Context, uid string) error {
	if p == nil || p.conn == nil {
		return nil
	}
	if uid == "" {
		return fmt.Errorf("delete access: uid is required")
	}
	if err := p.publish(DeleteAccessSubject, GenericFGAMessage{
		ObjectType: InitiativeObjectType,
		Operation:  "delete_access",
		Data:       deleteAccessData{UID: uid},
	}); err != nil {
		return err
	}
	flushCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		flushCtx, cancel = context.WithTimeout(ctx, deleteFlushTimeout)
		defer cancel()
	}
	return p.conn.FlushWithContext(flushCtx)
}

func (p *Publisher) publish(subject string, msg GenericFGAMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal fga message: %w", err)
	}
	if err := p.conn.Publish(subject, payload); err != nil {
		return fmt.Errorf("publish fga message: %w", err)
	}
	return nil
}
