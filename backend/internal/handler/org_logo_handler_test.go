// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

type stubOrgLogos struct {
	logos map[string]string
	err   error
	asked []string
}

func (s *stubOrgLogos) LogosByDomains(_ context.Context, domains []string) (map[string]string, error) {
	s.asked = domains
	if s.err != nil {
		return nil, s.err
	}
	out := map[string]string{}
	for _, d := range domains {
		if l, ok := s.logos[d]; ok {
			out[d] = l
		}
	}
	return out, nil
}

func getOrgLogos(t *testing.T, h *OrgLogoHandler, query string) (*httptest.ResponseRecorder, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/crowdfunding/me/org-logos?"+query, nil))
	if rec.Code != http.StatusOK {
		return rec, nil
	}
	var body struct {
		Logos map[string]string `json:"logos"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return rec, body.Logos
}

func TestOrgLogoHandler_List(t *testing.T) {
	repo := &stubOrgLogos{logos: map[string]string{"acme.com": "https://cdn.example/acme.png"}}
	_, logos := getOrgLogos(t, NewOrgLogoHandler(repo), "domains=acme.com,www.Acme.com,other.org,not%20a%20domain,")

	// Looked up once each, normalized; answered under the caller's spelling.
	if !slices.Equal(repo.asked, []string{"acme.com", "other.org"}) {
		t.Errorf("asked = %v", repo.asked)
	}
	want := map[string]string{"acme.com": "https://cdn.example/acme.png", "www.Acme.com": "https://cdn.example/acme.png"}
	if len(logos) != len(want) || logos["acme.com"] != want["acme.com"] || logos["www.Acme.com"] != want["www.Acme.com"] {
		t.Errorf("logos = %v, want %v", logos, want)
	}
}

func TestOrgLogoHandler_List_NoDomains(t *testing.T) {
	repo := &stubOrgLogos{}
	rec, logos := getOrgLogos(t, NewOrgLogoHandler(repo), "")
	if rec.Code != http.StatusOK || len(logos) != 0 || repo.asked != nil {
		t.Errorf("status %d, logos %v, asked %v; want 200, empty, no query", rec.Code, logos, repo.asked)
	}
	if !strings.Contains(rec.Body.String(), `"logos":{}`) {
		t.Errorf("body = %s, want an empty logos object", rec.Body.String())
	}
}

func TestOrgLogoHandler_List_NilRepo(t *testing.T) {
	rec, logos := getOrgLogos(t, NewOrgLogoHandler(nil), "domains=acme.com")
	if rec.Code != http.StatusOK || len(logos) != 0 {
		t.Errorf("status %d, logos %v; want 200 and none", rec.Code, logos)
	}
}

func TestOrgLogoHandler_List_TooManyDomains(t *testing.T) {
	domains := make([]string, maxOrgLogoDomains+1)
	for i := range domains {
		domains[i] = "a.com"
	}
	rec, _ := getOrgLogos(t, NewOrgLogoHandler(&stubOrgLogos{}), "domains="+strings.Join(domains, ","))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestOrgLogoHandler_List_RepoError(t *testing.T) {
	rec, _ := getOrgLogos(t, NewOrgLogoHandler(&stubOrgLogos{err: errors.New("db down")}), "domains=acme.com")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
