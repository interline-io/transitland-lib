package usercheck

import (
	"encoding/json"
	"net/http"

	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// UseDefaultUserMiddleware uses a default "user" context.
func UseDefaultUserMiddleware(defaultName string, roles ...string) func(http.Handler) http.Handler {
	return newInjectUserMiddleware(func() authn.User { return authn.NewCtxUser(defaultName, "", "").WithRoles(roles...) })
}

// NewInjectUserMiddleware uses a default "user" context.
func newInjectUserMiddleware(cb func() authn.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := cb()
			r = r.WithContext(authn.WithUser(r.Context(), user))
			next.ServeHTTP(w, r)
		})
	}
}

func RoleRequired(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := authn.ForContext(ctx)
			if user == nil || !user.HasRole(role) {
				writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func AnyRoleRequired(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := authn.ForContext(ctx)
			foundRole := false
			if user != nil {
				for _, role := range roles {
					if user.HasRole(role) {
						foundRole = true
					}
				}
			}
			if !foundRole {
				writeJsonError(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJsonError(w http.ResponseWriter, msg string, statusCode int) {
	a := map[string]string{
		"error": msg,
	}
	jj, _ := json.Marshal(&a)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jj)
}
