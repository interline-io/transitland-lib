package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/testdata"
)

func Test_onestopIdEntityRedirectHandler(t *testing.T) {
	// Test cases
	tests := []struct {
		name       string
		onestopID  string
		wantStatus int
		wantLoc    string
	}{
		{"Feed", "f-123", http.StatusFound, "/feeds/f-123"},
		{"Operator", "o-123", http.StatusFound, "/operators/o-123"},
		{"Stop", "s-123", http.StatusFound, "/stops/s-123"},
		{"Route", "r-123", http.StatusFound, "/routes/r-123"},
		{"Unknown", "x-123", http.StatusNotFound, ""},
	}

	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new request
			req := httptest.NewRequest("GET", fmt.Sprintf("/onestop_id/%s", tt.onestopID), nil)
			w := httptest.NewRecorder()

			// Call the handler
			restSrv.ServeHTTP(w, req)

			// Check the status code
			if got := w.Result().StatusCode; got != tt.wantStatus {
				t.Errorf("got status %d, want %d", got, tt.wantStatus)
			}

			// Check the Location header
			if got := w.Header().Get("Location"); got != tt.wantLoc {
				t.Errorf("got Location %q, want %q", got, tt.wantLoc)
			}
		})
	}
}
