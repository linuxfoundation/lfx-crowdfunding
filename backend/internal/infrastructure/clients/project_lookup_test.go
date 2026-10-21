// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
)

func TestProjectLookup_IsAttributable(t *testing.T) {
	t.Parallel()
	const uid = "7cad5a8d-19d0-41a4-81a6-043453daf9ee"
	doc := func(id, public, stage string) string {
		return `{"resources":[{"type":"project","id":"` + id + `","data":{"public":` + public + `,"stage":"` + stage + `"}}]}`
	}
	cases := []struct {
		name, body string
		want       bool
	}{
		{"public active", doc("project:"+uid, "true", "Active"), true},
		{"public no stage", doc("project:"+uid, "true", ""), true},
		{"formation", doc("project:"+uid, "true", "Formation - Exploratory"), false},
		{"prospect", doc("project:"+uid, "true", "Prospect"), false},
		{"private", doc("project:"+uid, "false", "Active"), false},
		{"other project", doc("project:00000000-0000-0000-0000-000000000000", "true", "Active"), false},
		{"no result", `{"resources":[]}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/query/resources" || r.URL.Query().Get("type") != "project" ||
					r.URL.Query().Get("filters") != "uid:"+uid || r.URL.Query().Get("v") != "1" {
					t.Errorf("unexpected request %s", r.URL)
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("lookup must be anonymous")
				}
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			got, err := NewProjectLookup(srv.URL+"/", 2*time.Second).IsAttributable(context.Background(), uid)
			if err != nil || got != tc.want {
				t.Fatalf("got (%v, %v), want (%v, nil)", got, err, tc.want)
			}
		})
	}
}

func TestProjectLookup_Outage(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := NewProjectLookup(srv.URL, 2*time.Second).IsAttributable(context.Background(), "x"); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("status error: got %v, want ErrUpstreamUnavailable", err)
	}

	srv.Close() // transport failure
	if _, err := NewProjectLookup(srv.URL, 2*time.Second).IsAttributable(context.Background(), "x"); !errors.Is(err, domain.ErrUpstreamUnavailable) {
		t.Fatalf("transport error: got %v, want ErrUpstreamUnavailable", err)
	}
}
