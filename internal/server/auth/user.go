package auth

import (
	"context"

	"github.com/interline-io/transitland-lib/internal/server/model"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// User defines role access methods.
type User struct {
	Name    string
	IsAnon  bool
	IsUser  bool
	IsAdmin bool
}

// HasRole checks if a User is allowed to use a defined role.
func (user *User) HasRole(role model.Role) bool {
	switch role {
	case model.RoleAnon:
		return user.IsAnon || user.IsUser || user.IsAdmin
	case model.RoleUser:
		return user.IsUser || user.IsAdmin
	case model.RoleAdmin:
		return user.IsAdmin
	}
	return false
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *User {
	raw, _ := ctx.Value(userCtxKey).(*User)
	return raw
}
