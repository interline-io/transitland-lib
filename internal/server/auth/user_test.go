package auth

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/server/model"
)

func TestUser_HasRole(t *testing.T) {
	testcases := []struct {
		name    string
		user    User
		role    model.Role
		hasRole bool
	}{
		{"anon", User{IsAnon: true}, model.RoleAnon, true},
		{"anon", User{IsAnon: true, IsUser: true}, model.RoleAnon, true},
		{"anon", User{IsAnon: true, IsUser: true, IsAdmin: true}, model.RoleAnon, true},
		{"anon", User{IsUser: true}, model.RoleAnon, true},
		{"anon", User{IsAdmin: true}, model.RoleAnon, true},

		{"user", User{IsAnon: true}, model.RoleUser, false},
		{"user", User{IsUser: true}, model.RoleUser, true},
		{"user", User{IsAdmin: true}, model.RoleUser, true},

		{"admin", User{IsAnon: true}, model.RoleAdmin, false},
		{"admin", User{IsAnon: true, IsUser: true}, model.RoleAdmin, false},
		{"admin", User{IsAnon: true, IsAdmin: false}, model.RoleAdmin, false},
		{"admin", User{IsUser: true}, model.RoleAdmin, false},
		{"admin", User{IsAdmin: true}, model.RoleAdmin, true},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.user.HasRole(tc.role) != tc.hasRole {
				t.Errorf("expected role %s to be %t", tc.role, tc.hasRole)
			}
		})
	}
}
