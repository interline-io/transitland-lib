package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authz"
)

func TestAddConfig_PanicsOnNilChecker(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when Config.Checker is nil")
		}
	}()
	_ = AddConfig(Config{})
}

func TestAddConfig_NoPanicWithChecker(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	mw := AddConfig(Config{Checker: &authz.DenyAllChecker{}})
	// Exercise the returned middleware end-to-end so the test asserts
	// AddConfig produces a usable handler, not just that construction
	// doesn't panic.
	srv := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
