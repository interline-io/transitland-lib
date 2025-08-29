package headercheck

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

func TestKongMiddleware(t *testing.T) {
	tcs := []struct {
		name       string
		consumerId string
		code       int
		user       authn.User
	}{
		{"test", "test@transitland", 200, newCtxUser("test@transitland")},
		{"no user", "", 200, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			mf, err := UserHeaderMiddleware("x-consumer-username")
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.consumerId != "" {
				req.Header.Add("x-consumer-username", tc.consumerId)
			}
			mwtest.TestAuthMiddleware(t, req, mf, tc.code, tc.user)
		})
	}
}
