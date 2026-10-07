// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

type fakeResolver struct {
	ok    bool
	err   error
	calls int
}

func (f *fakeResolver) CanManage(context.Context, models.AttributionType, string, string) (bool, error) {
	f.calls++
	return f.ok, f.err
}

func TestCanManage(t *testing.T) {
	org := models.Attribution{Type: models.AttributionOrganization, EntityUID: "001abc"}
	outage := errors.Join(domain.ErrUpstreamUnavailable, errors.New("nats down"))
	tests := []struct {
		name      string
		caller    string
		attr      models.Attribution
		resolver  *fakeResolver // nil → no resolver wired
		want      bool
		wantErr   error
		wantCalls int
	}{
		{"creator, no FGA call", "u1", org, &fakeResolver{ok: false}, true, nil, 0},
		{"writer on attributed entity", "u2", org, &fakeResolver{ok: true}, true, nil, 1},
		{"non-writer", "u2", org, &fakeResolver{ok: false}, false, nil, 1},
		{"resolver outage is an error, not false", "u2", org, &fakeResolver{err: outage}, false, domain.ErrUpstreamUnavailable, 1},
		{"personal never consults resolver", "u2", models.Attribution{Type: models.AttributionPersonal}, &fakeResolver{ok: true}, false, nil, 0},
		{"no resolver wired is creator-only", "u2", org, nil, false, nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &models.Initiative{OwnerID: "u1", Attribution: tt.attr}
			var got bool
			var err error
			caller := &models.User{ID: tt.caller, Username: tt.caller}
			if tt.resolver == nil {
				got, err = canManage(context.Background(), nil, caller, i)
			} else {
				got, err = canManage(context.Background(), tt.resolver, caller, i)
				if tt.resolver.calls != tt.wantCalls {
					t.Errorf("resolver calls = %d, want %d", tt.resolver.calls, tt.wantCalls)
				}
			}
			if got != tt.want || !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, %v; want %v, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

// A writer's edit must publish the creator as owner, never the writer.
func TestOwnerUsername_WriterEditKeepsCreator(t *testing.T) {
	s := &InitiativeService{userRepo: &mockUserRepository{user: &models.User{ID: "u1", Username: "creator"}}}
	got, err := s.ownerUsername(context.Background(), "u1", &models.User{ID: "u2", Username: "writer"})
	if err != nil || got != "creator" {
		t.Errorf("got %q, %v; want creator", got, err)
	}
	got, _ = s.ownerUsername(context.Background(), "u2", &models.User{ID: "u2", Username: "writer"})
	if got != "writer" {
		t.Errorf("creator edit: got %q, want writer", got)
	}
}
