package authn

import "context"

// User provides access to key user metadata and roles.
type User interface {
	ID() string
	Name() string
	Email() string
	Roles() []string
	HasRole(string) bool
	GetExternalData(string) (string, bool)
}

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var ctxUserKey = &contextKey{"user"}

type contextKey struct {
	name string
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) User {
	raw, ok := ctx.Value(ctxUserKey).(User)
	if !ok {
		return nil
	}
	return raw
}

func WithUser(ctx context.Context, user User) context.Context {
	r := context.WithValue(ctx, ctxUserKey, user)
	return r
}
