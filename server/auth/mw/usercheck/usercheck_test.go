package usercheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/mwtest"
)

func newCtxUser(id string) authn.CtxUser {
	return authn.NewCtxUser(id, "", "")
}

func TestUserMiddleware(t *testing.T) {
	a := UserDefaultMiddleware("test")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mwtest.TestAuthMiddleware(t, req, a, 200, authn.NewCtxUser("test", "", ""))
}

func TestAdminMiddleware(t *testing.T) {
	a := AdminDefaultMiddleware("test")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mwtest.TestAuthMiddleware(t, req, a, 200, authn.NewCtxUser("test", "", "").WithRoles("admin"))
}

// func TestNoMiddleware(t *testing.T) {
// 	a, err := GetUserMiddleware("", AuthConfig{}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req := httptest.NewRequest(http.MethodGet, "/", nil)
// 	testAuthMiddleware(t, req, a, 200, nil)
// }

func TestUserRequired(t *testing.T) {
	tcs := []struct {
		name string
		mwf  func(http.Handler) http.Handler
		code int
		user authn.User
	}{
		{"with user", func(next http.Handler) http.Handler {
			return AdminDefaultMiddleware("test")(UserRequired(next))
		}, 200, newCtxUser("test").WithRoles("admin")},
		{"with user", func(next http.Handler) http.Handler {
			return UserDefaultMiddleware("test")(UserRequired(next))
		}, 200, newCtxUser("test")},
		{"no user", func(next http.Handler) http.Handler { return UserRequired(next) }, 401, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			mwtest.TestAuthMiddleware(t, req, tc.mwf, tc.code, tc.user)
		})
	}
}

func TestAdminRequired(t *testing.T) {
	tcs := []struct {
		name string
		mwf  func(http.Handler) http.Handler
		code int
		user authn.User
	}{
		{"with admin", func(next http.Handler) http.Handler {
			return AdminDefaultMiddleware("test")(AdminRequired(next))
		}, 200, newCtxUser("test").WithRoles("admin")},
		{"with user", func(next http.Handler) http.Handler {
			return UserDefaultMiddleware("test")(AdminRequired(next))
		}, 401, nil}, // mw kills request before handler
		{"no user", func(next http.Handler) http.Handler { return AdminRequired(next) }, 401, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			mwtest.TestAuthMiddleware(t, req, tc.mwf, tc.code, tc.user)
		})
	}
}
