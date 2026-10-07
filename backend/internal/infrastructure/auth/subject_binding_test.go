// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
)

func TestRequireSubjectBinding(t *testing.T) {
	legacy := &models.User{ID: "u1", Username: "456789", LegacyUserID: "github|456789"}
	tests := []struct {
		name string
		p    *models.Principal
		row  *models.User
		err  error
		want int
	}{
		{"attacker sub vs legacy row", &models.Principal{Username: "456789", UserID: "auth0|attacker"}, legacy, nil, http.StatusForbidden},
		{"matching sub", &models.Principal{Username: "456789", UserID: "github|456789"}, legacy, nil, http.StatusOK},
		{"row without legacy id", &models.Principal{Username: "bob", UserID: "auth0|bob"}, &models.User{Username: "bob"}, nil, http.StatusOK},
		{"first login, no row", &models.Principal{Username: "new", UserID: "auth0|new"}, nil, domain.ErrUserNotFound, http.StatusOK},
		{"heimdall token exempt", &models.Principal{Username: "456789", UserID: "456789", IsHeimdallIssued: true}, legacy, nil, http.StatusOK},
		{"lookup outage is 503, not a pass", &models.Principal{Username: "bob", UserID: "auth0|bob"}, nil, errors.New("db down"), http.StatusServiceUnavailable},
		{"no principal", nil, nil, nil, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(context.Context, string) (*models.User, error) { return tt.row, tt.err }
			h := RequireSubjectBinding(lookup, slog.Default())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			r := httptest.NewRequest(http.MethodGet, "/crowdfunding/me/initiatives", nil)
			if tt.p != nil {
				r = r.WithContext(ContextWithPrincipal(r.Context(), tt.p))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Errorf("got %d, want %d", w.Code, tt.want)
			}
		})
	}
}
