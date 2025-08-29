package headercheck

import (
	"net/http"

	"github.com/interline-io/transitland-lib/server/auth/authn"
)

// UserHeaderMiddleware checks and pulls user ID from specified headers.
func UserHeaderMiddleware(header string) (func(http.Handler) http.Handler, error) {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v := r.Header.Get(header); v != "" {
				user := authn.NewCtxUser(v, "", "")
				r = r.WithContext(authn.WithUser(r.Context(), user))
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}
