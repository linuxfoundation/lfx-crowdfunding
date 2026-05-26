// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package handler provides HTTP handlers for the initiatives API.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/clients"
)

// allowedImageTypes is the set of MIME types accepted for logo uploads.
// Only raster image formats are permitted; SVG is excluded because browsers
// execute embedded scripts in SVG files served with image/svg+xml, which
// would make public S3 objects a stored-XSS vector.
var allowedImageTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/gif":  {},
	"image/webp": {},
}

// UploadHandler holds Chi handlers for logo presigned-URL generation.
type UploadHandler struct {
	s3 clients.S3PresignClient
}

// NewUploadHandler creates an UploadHandler.
func NewUploadHandler(s3 clients.S3PresignClient) *UploadHandler {
	return &UploadHandler{s3: s3}
}

// presignedURLRequest is the JSON body for POST /v1/presigned-url.
type presignedURLRequest struct {
	// ContentType is the MIME type of the file to be uploaded (e.g. "image/png").
	ContentType string `json:"content_type"`
}

// presignedURLResponse is returned on success.
type presignedURLResponse struct {
	// UploadURL is the short-lived presigned PUT URL the frontend uses to upload directly to S3.
	UploadURL string `json:"upload_url"`
	// DestinationURL is the permanent public URL of the object once uploaded.
	DestinationURL string `json:"destination_url"`
}

// CreatePresignedURL handles POST /v1/presigned-url.
// Returns a presigned S3 PUT URL and the resulting permanent object URL.
// Requires JWT authentication.
func (h *UploadHandler) CreatePresignedURL(w http.ResponseWriter, r *http.Request) {
	if auth.PrincipalFromContext(r.Context()) == nil {
		Error(w, domain.ErrUnauthorized)
		return
	}

	var body presignedURLRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		Error(w, domain.ErrInvalidInput)
		return
	}

	contentType := strings.TrimSpace(body.ContentType)
	if contentType == "" {
		Error(w, domain.ErrInvalidInput)
		return
	}
	// Validate against the allowlist to prevent arbitrary content-type injection.
	if _, ok := allowedImageTypes[contentType]; !ok {
		Error(w, domain.ErrInvalidInput)
		return
	}

	uploadURL, destinationURL, err := h.s3.PresignLogoUpload(r.Context(), contentType)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, presignedURLResponse{
		UploadURL:      uploadURL,
		DestinationURL: destinationURL,
	})
}
