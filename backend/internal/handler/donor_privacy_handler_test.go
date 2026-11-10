// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v85"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// hPrivacy is a domain.DonorPrivacyRepository stub for handler tests.
type hPrivacy struct {
	anonymous bool
	err       error
	setSub    func(id, userID string, anonymous bool) error
}

func (p *hPrivacy) MatchLedgerRefs(context.Context, []string, []string) (*models.LedgerRefMatches, error) {
	return &models.LedgerRefMatches{}, nil
}
func (p *hPrivacy) AnonymousDonors(context.Context, []string) (map[string]map[string]bool, error) {
	return nil, nil
}
func (p *hPrivacy) AnonymousLedgerUsers(context.Context, []string) (map[string]bool, error) {
	return nil, nil
}
func (p *hPrivacy) IsPaymentIntentAnonymous(context.Context, string) (bool, error) {
	return p.anonymous, p.err
}
func (p *hPrivacy) IsStripeSubscriptionAnonymous(context.Context, string) (bool, error) {
	return p.anonymous, p.err
}
func (p *hPrivacy) SetDonationAnonymous(context.Context, string, string, bool) error { return nil }
func (p *hPrivacy) SetSubscriptionAnonymous(_ context.Context, id, userID string, anonymous bool) error {
	return p.setSub(id, userID, anonymous)
}

var _ domain.DonorPrivacyRepository = (*hPrivacy)(nil)

func TestWebhookHandler_AdminEmailHidesAnonymousDonor(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		payload   string
		privacy   *hPrivacy
		wantName  string
		wantEmail string
	}{
		{"one-time anonymous", "payment_intent.succeeded", piV2JSON, &hPrivacy{anonymous: true}, anonymousDonorName, ""},
		{"recurring anonymous", "invoice.payment_succeeded", invV2JSON, &hPrivacy{anonymous: true}, anonymousDonorName, ""},
		{"lookup failure hides donor", "payment_intent.succeeded", piV2JSON, &hPrivacy{err: errors.New("db down")}, anonymousDonorName, ""},
		{"not anonymous", "payment_intent.succeeded", piV2JSON, &hPrivacy{}, "", "jane@example.com"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var adminName, adminEmail, receiptTo string
			es := &wbEmailService{
				onConfirmation: func(toEmail, _, _, _, _ string) { receiptTo = toEmail },
				onAdminNotify: func(_, donorName, donorEmail, _, _, _ string) {
					adminName, adminEmail = donorName, donorEmail
				},
			}
			event := buildEvent(c.eventType, c.payload)
			sc := &wbStripeClient{onConstruct: func([]byte, string, string) (stripe.Event, error) { return event, nil }}
			h := newTestWebhookHandlerFull(sc, &wbDonationRepo{}, &wbSubscriptionRepo{}, &wbLedgerClient{}, es)
			h.SetDonorPrivacyRepo(c.privacy)

			if rr := postWebhook(t, h, "t=1,v1=sig", `{}`); rr.Code != http.StatusOK {
				t.Fatalf("status = %d", rr.Code)
			}
			if c.wantName != "" && adminName != c.wantName {
				t.Errorf("admin donor name = %q, want %q", adminName, c.wantName)
			}
			if c.wantName == "" && adminName == anonymousDonorName {
				t.Errorf("non-anonymous donor hidden in admin email")
			}
			if adminEmail != c.wantEmail {
				t.Errorf("admin donor email = %q, want %q", adminEmail, c.wantEmail)
			}
			if receiptTo != "jane@example.com" {
				t.Errorf("donor receipt to = %q; the donor's own receipt must not change", receiptTo)
			}
		})
	}
}

func subscriptionAnonymityReq(id, body string, principal *models.Principal) *http.Request {
	r := httptest.NewRequest(http.MethodPatch, "/crowdfunding/me/subscriptions/"+id, strings.NewReader(body))
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestSubscriptionHandler_SetAnonymous(t *testing.T) {
	const subID = "6f1b0b8e-3a52-4c2e-9a49-1f3f5d6b7c80"
	principal := &models.Principal{Username: "alice"}
	user := &models.User{ID: "00000000-0000-0000-0000-0000000000aa", Username: "alice"}

	newHandler := func(setSub func(id, userID string, anonymous bool) error) *SubscriptionHandler {
		h := newSubscriptionHandler(&subscriptionRepo{}, &subscriptionInitiativeRepo{}, &subscriptionUserRepo{user: user}, &subscriptionStripeClient{})
		h.svc.SetDonorPrivacyRepo(&hPrivacy{setSub: setSub})
		return h
	}

	t.Run("updates the caller's subscription", func(t *testing.T) {
		var gotID, gotUser string
		var gotAnon bool
		h := newHandler(func(id, userID string, anonymous bool) error {
			gotID, gotUser, gotAnon = id, userID, anonymous
			return nil
		})
		w := httptest.NewRecorder()
		h.SetAnonymous(w, subscriptionAnonymityReq(subID, `{"anonymous":true}`, principal))
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
		}
		var state models.AnonymityState
		if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil || state.ID != subID || !state.Anonymous {
			t.Errorf("body = %s (%v)", w.Body.String(), err)
		}
		if gotID != subID || gotUser != user.ID || !gotAnon {
			t.Errorf("stored (%s, %s, %v)", gotID, gotUser, gotAnon)
		}
	})

	t.Run("someone else's subscription is 404", func(t *testing.T) {
		h := newHandler(func(string, string, bool) error { return domain.ErrSubscriptionNotFound })
		w := httptest.NewRecorder()
		h.SetAnonymous(w, subscriptionAnonymityReq(subID, `{"anonymous":false}`, principal))
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})

	t.Run("missing field is 400", func(t *testing.T) {
		h := newHandler(func(string, string, bool) error { t.Error("store called"); return nil })
		w := httptest.NewRecorder()
		h.SetAnonymous(w, subscriptionAnonymityReq(subID, `{}`, principal))
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("no principal is 401", func(t *testing.T) {
		h := newHandler(func(string, string, bool) error { t.Error("store called"); return nil })
		w := httptest.NewRecorder()
		h.SetAnonymous(w, subscriptionAnonymityReq(subID, `{"anonymous":true}`, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})
}

func TestDonationHandler_SetAnonymous_RequiresField(t *testing.T) {
	h := NewDonationHandler(nil)
	r := httptest.NewRequest(http.MethodPatch, "/crowdfunding/me/donations/x", strings.NewReader(`{"anonymous":"yes"}`))
	r = r.WithContext(auth.ContextWithPrincipal(r.Context(), &models.Principal{Username: "alice"}))
	w := httptest.NewRecorder()
	h.SetAnonymous(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
