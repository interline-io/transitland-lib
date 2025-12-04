package authn

import (
	"testing"
)

type Role string

const (
	RoleAnon  Role = "ANON"
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

func TestUser_HasRole(t *testing.T) {
	testcases := []struct {
		name    string
		user    User
		role    Role
		hasRole bool
	}{
		{"anon", NewCtxUser("", "", ""), RoleAnon, true},
		{"anon", NewCtxUser("test", "", ""), RoleAnon, true},
		{"anon", NewCtxUser("test", "", "").WithRoles(string(RoleAdmin)), RoleAnon, true},

		{"user", NewCtxUser("", "", ""), RoleUser, false},
		{"user", NewCtxUser("test", "", ""), RoleUser, true},
		{"user", NewCtxUser("test", "", "").WithRoles(string(RoleAnon)), RoleUser, true},

		{"admin", NewCtxUser("", "", ""), RoleAdmin, false},
		{"admin", NewCtxUser("", "", ""), RoleAdmin, false},
		{"admin", NewCtxUser("test", "", ""), RoleAdmin, false},
		{"admin", NewCtxUser("test", "", ""), RoleAdmin, false},
		{"admin", NewCtxUser("test", "", "").WithRoles(string(RoleAdmin)), RoleAdmin, true},

		{"other roles", NewCtxUser("test", "", "").WithRoles(string(Role("tlv2-admin"))), Role("tlv2-admin"), true},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.user.HasRole(string(tc.role)) != tc.hasRole {
				t.Errorf("expected role %s to be %t", tc.role, tc.hasRole)
			}
		})
	}
}
