// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package clients

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendTemplate_HandlebarsPassthrough(t *testing.T) {
	var got mandrillSendTemplateRequest
	c := &mandrillClient{
		cfg: MandrillConfig{APIKey: "k"},
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"email":"a@b.c","status":"sent"}]`))}, nil
		})},
	}

	err := c.SendTemplate(context.Background(), MandrillTemplateSubmittedForReview, "a@b.c", "A",
		map[string]string{"SUBMISSION_NAME": `Foo</a><a href="https://evil.example">Approve</a>`})
	if err != nil {
		t.Fatalf("SendTemplate: %v", err)
	}

	// Templates escape via {{safehtml}}; we must send the raw value and pin handlebars.
	raw := `Foo</a><a href="https://evil.example">Approve</a>`
	vars := got.Message.GlobalMergeVars
	if len(vars) != 1 || vars[0].Content != raw {
		t.Errorf("merge vars = %+v, want raw content %q", vars, raw)
	}
	if got.Message.MergeLanguage != "handlebars" {
		t.Errorf("merge_language = %q, want handlebars", got.Message.MergeLanguage)
	}
}
