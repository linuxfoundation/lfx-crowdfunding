// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package models

import "testing"

func TestUserDisplayName(t *testing.T) {
	for _, tc := range []struct {
		user User
		want string
	}{
		{User{Name: "Dana Doe", GivenName: "D", FamilyName: "X"}, "Dana Doe"},
		{User{Name: "  ", GivenName: "Eve", FamilyName: "Lin"}, "Eve Lin"},
		{User{GivenName: "Eve"}, "Eve"},
		{User{FamilyName: " Lin "}, "Lin"},
		{User{Email: "never@example.com"}, ""},
	} {
		if got := tc.user.DisplayName(); got != tc.want {
			t.Errorf("DisplayName(%+v) = %q, want %q", tc.user, got, tc.want)
		}
	}
}
