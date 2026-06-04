// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/domain/models"
	"github.com/linuxfoundation/lfx-v2-initiatives-service/internal/infrastructure/auth"
)

// UserHandler holds Chi handlers for the /v1/me resource.
type UserHandler struct {
	userRepo domain.UserRepository
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(userRepo domain.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// SyncProfile handles PATCH /v1/me.
//
// Called immediately after a successful login to ensure the authenticated
// user's profile is persisted (or updated) in the users table. All profile
// data is sourced from the verified JWT claims already present in the
// request context — the LF SSO username is used as the stable identifier.
func (h *UserHandler) SyncProfile(w http.ResponseWriter, r *http.Request) {
	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil || principal.Username == "" {
		Error(w, domain.ErrUnauthorized)
		return
	}

	user := &models.User{
		Username:   principal.Username,
		Email:      principal.Email,
		Name:       principal.Name,
		GivenName:  principal.GivenName,
		FamilyName: principal.FamilyName,
		AvatarURL:  principal.Picture,
	}

	result, err := h.userRepo.Upsert(r.Context(), user)
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}
