package auth

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
)

// NoAuthMiddleware stores the user context, but always as admin
func NoAuthMiddleware(db sqlx.Ext) (func(http.Handler) http.Handler, error) {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := &User{
				Name:    "",
				IsAnon:  true,
				IsUser:  true,
				IsAdmin: true,
			}
			ctx := context.WithValue(r.Context(), userCtxKey, user)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}, nil
}
