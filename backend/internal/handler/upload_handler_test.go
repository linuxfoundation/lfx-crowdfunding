// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// ── stub ──────────────────────────────────────────────────────────────────────

// mockS3PresignClient is a configurable stub for clients.S3PresignClient.
type mockS3PresignClient struct {
	uploadURL       string
	destinationURL  string
	requiredHeaders map[string]string
	err             error
}

func (m *mockS3PresignClient) PresignLogoUpload(_ context.Context, _ string) (string, string, map[string]string, error) {
	return m.uploadURL, m.destinationURL, m.requiredHeaders, m.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

// uploadReq builds a POST request for POST /v1/presigned-url with an optional principal.
func uploadReq(contentType string, principal *models.Principal) *http.Request {
	body := ""
	if contentType != "" {
		body = `{"content_type":"` + contentType + `"}`
	} else {
		body = `{}`
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/presigned-url", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if principal != nil {
		r = r.WithContext(auth.ContextWithPrincipal(r.Context(), principal))
	}
	return r
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestUploadHandler_NoPrincipal(t *testing.T) {
	h := NewUploadHandler(&mockS3PresignClient{})
	w := httptest.NewRecorder()
	h.CreatePresignedURL(w, uploadReq("image/png", nil))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUploadHandler_EmptyContentType(t *testing.T) {
	principal := &models.Principal{Username: "user1"}
	h := NewUploadHandler(&mockS3PresignClient{})
	w := httptest.NewRecorder()
	h.CreatePresignedURL(w, uploadReq("", principal))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUploadHandler_DisallowedContentType(t *testing.T) {
	principal := &models.Principal{Username: "user1"}
	h := NewUploadHandler(&mockS3PresignClient{})
	w := httptest.NewRecorder()
	h.CreatePresignedURL(w, uploadReq("image/svg+xml", principal))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUploadHandler_Success(t *testing.T) {
	principal := &models.Principal{Username: "user1"}
	stub := &mockS3PresignClient{
		uploadURL:      "https://bucket.s3.us-east-1.amazonaws.com/key?X-Amz-Signature=abc",
		destinationURL: "https://bucket.s3.us-east-1.amazonaws.com/key",
		requiredHeaders: map[string]string{
			"Content-Type": "image/png",
			"x-amz-acl":    "public-read",
		},
	}
	h := NewUploadHandler(stub)
	w := httptest.NewRecorder()
	h.CreatePresignedURL(w, uploadReq("image/png", principal))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp presignedURLResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.UploadURL != stub.uploadURL {
		t.Errorf("upload_url: got %q, want %q", resp.UploadURL, stub.uploadURL)
	}
	if resp.DestinationURL != stub.destinationURL {
		t.Errorf("destination_url: got %q, want %q", resp.DestinationURL, stub.destinationURL)
	}
	if resp.RequiredHeaders["x-amz-acl"] != "public-read" {
		t.Errorf("required_headers[x-amz-acl]: got %q, want %q", resp.RequiredHeaders["x-amz-acl"], "public-read")
	}
	if resp.RequiredHeaders["Content-Type"] != "image/png" {
		t.Errorf("required_headers[Content-Type]: got %q, want %q", resp.RequiredHeaders["Content-Type"], "image/png")
	}
}
