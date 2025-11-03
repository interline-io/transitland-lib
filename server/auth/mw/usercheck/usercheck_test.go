package usercheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/mwtest"
)

const ADMIN_ROLE = "admin_role_test"

func newCtxUser(id string) authn.CtxUser {
	return authn.NewCtxUser(id, "", "")
}

func TestUserMiddleware(t *testing.T) {
	a := UseDefaultUserMiddleware("test")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mwtest.TestAuthMiddleware(t, req, a, 200, authn.NewCtxUser("test", "", ""))
}

func TestUserRoleMiddleware(t *testing.T) {
	a := UseDefaultUserMiddleware("test", ADMIN_ROLE)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mwtest.TestAuthMiddleware(t, req, a, 200, authn.NewCtxUser("test", "", "").WithRoles(ADMIN_ROLE))
}

// func TestNoMiddleware(t *testing.T) {
// 	a, err := GetUserMiddleware("", AuthConfig{}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req := httptest.NewRequest(http.MethodGet, "/", nil)
// 	testAuthMiddleware(t, req, a, 200, nil)
// }

func TestRoleRequired(t *testing.T) {
	testRole := "required_role"
	tcs := []struct {
		name string
		mwf  func(http.Handler) http.Handler
		code int
		user authn.User
	}{
		{"with user", func(next http.Handler) http.Handler {
			return UseDefaultUserMiddleware("test", ADMIN_ROLE)(RoleRequired(testRole)(next))
		}, 200, newCtxUser("test").WithRoles(testRole)},
		{"with user", func(next http.Handler) http.Handler {
			return UseDefaultUserMiddleware("test")(RoleRequired(testRole)(next))
		}, 200, newCtxUser("test")},
		{"no user", func(next http.Handler) http.Handler { return RoleRequired(testRole)(next) }, 401, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			mwtest.TestAuthMiddleware(t, req, tc.mwf, tc.code, tc.user)
		})
	}
}
