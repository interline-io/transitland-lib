package mwtest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T, req *http.Request, mwf func(http.Handler) http.Handler, expectCode int, expectUser authn.User) {
	var user authn.User
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		user = authn.ForContext(r.Context())
	}
	router := http.NewServeMux()
	router.HandleFunc("/", testHandler)
	//
	a := mwf(router)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	//
	assert.Equal(t, expectCode, w.Result().StatusCode)
	if expectUser != nil && user != nil {
		assert.Equal(t, expectUser.ID(), user.ID())
		for _, checkRole := range expectUser.Roles() {
			assert.Equalf(t, true, user.HasRole(checkRole), "checking role '%s'", checkRole)
		}
	} else if expectUser == nil && user != nil {
		t.Errorf("got user, expected none")
	} else if expectUser != nil && user == nil {
		t.Errorf("got no user, expected user")
	}
}
