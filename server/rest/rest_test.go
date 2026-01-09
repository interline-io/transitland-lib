package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) {
	// Increase limit for test
	MAXLIMIT = 100_000
	gql.RESOLVER_MAXLIMIT = MAXLIMIT
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

type testCase struct {
	name         string
	h            apiHandler
	format       string
	selector     string
	expectSelect []string
	expectLength int
	expectError  bool
	f            func(*testing.T, string)
	user         string
	userRoles    []string
}

func testHandlersWithOptions(t testing.TB, opts testconfig.Options) (http.Handler, http.Handler, model.Config) {
	cfg := testconfig.Config(t, opts)
	graphqlHandler, err := gql.NewServer()
	if err != nil {
		t.Fatal(err)
	}
	restHandler, err := NewServer(graphqlHandler)
	if err != nil {
		t.Fatal(err)
	}
	return model.AddConfigAndPerms(cfg, nil, graphqlHandler),
		model.AddConfigAndPerms(cfg, nil, restHandler),
		cfg
}

func checkTestCase(t *testing.T, tc testCase) {
	graphqlHandler, _, _ := testHandlersWithOptions(t, testconfig.Options{
		WhenUtc: "2018-06-01T00:00:00Z",
		RTJsons: testconfig.DefaultRTJson(),
		Storage: testdata.Path("server", "tmp"),
	})
	tested := false

	// Inject user
	// This is not the best place to inject the user
	// But we are calling the handler directly, not through middleware.
	// TODO: Clean this up
	ctx := context.Background()
	if tc.user != "" {
		user := authn.NewCtxUser(tc.user, "", "").WithRoles(tc.userRoles...)
		ctx = authn.WithUser(ctx, user)
	}

	data, err := makeRequest(ctx, graphqlHandler, tc.h, tc.format, nil)
	if err != nil {
		if tc.expectError {
			tested = true
		} else {
			t.Error(err)
			return
		}
	} else if tc.expectError {
		t.Error("expected error")
		return
	}
	jj := string(data)
	if tc.f != nil {
		tested = true
		tc.f(t, jj)
	}
	if tc.selector != "" {
		tested = true
		a := []string{}
		for _, v := range gjson.Get(jj, tc.selector).Array() {
			a = append(a, v.String())
		}
		if len(tc.expectSelect) > 0 {
			if len(a) == 0 {
				t.Errorf("selector '%s' returned zero elements", tc.selector)
			} else {
				if !assert.ElementsMatch(t, a, tc.expectSelect) {
					t.Errorf("got %#v -- expect %#v\n\n", a, tc.expectSelect)
				}
			}
		} else {
			if len(a) != tc.expectLength {
				t.Errorf("got %d elements, expected %d", len(a), tc.expectLength)
			}
		}
	}
	if !tested {
		t.Errorf("no test performed, check test case")
	}
}

func toJson(m map[string]interface{}) string {
	rr, _ := json.Marshal(&m)
	return string(rr)
}

func TestRootRedirect(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("tmp"),
	})

	t.Run("root redirect to openapi.json", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)

		if sc := rr.Result().StatusCode; sc != http.StatusMovedPermanently {
			t.Errorf("got status code %d, expected %d", sc, http.StatusMovedPermanently)
		}

		location := rr.Header().Get("Location")
		if location != "/openapi.json" {
			t.Errorf("got location %s, expected /openapi.json", location)
		}
	})

	t.Run("openapi.json endpoint returns json", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/openapi.json", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)

		if sc := rr.Result().StatusCode; sc != http.StatusOK {
			t.Errorf("got status code %d, expected %d", sc, http.StatusOK)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("got content type %s, expected application/json", contentType)
		}

		// Verify it's valid JSON
		var schema map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &schema); err != nil {
			t.Errorf("response is not valid JSON: %v", err)
		}
	})
}

func TestSingleEntityNotFound(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		// Single entity requests that don't exist should return 404
		{"feed not found", "/feeds/f-nonexistent", http.StatusNotFound},
		{"route not found", "/routes/r-nonexistent", http.StatusNotFound},
		{"stop not found", "/stops/s-nonexistent", http.StatusNotFound},
		{"agency not found", "/agencies/o-nonexistent", http.StatusNotFound},
		{"operator not found", "/operators/o-nonexistent", http.StatusNotFound},
		{"feed_version not found", "/feed_versions/nonexistent-sha1", http.StatusNotFound},
		{"trip not found", "/routes/r-9q9-bu~1u-e535e4/trips/99999999", http.StatusNotFound},

		// Single entity requests that exist should return 200
		{"feed exists", "/feeds/BA", http.StatusOK},
		{"feed exists by id", "/feeds/1", http.StatusOK},

		// Collection requests with no results should return 200 with empty array
		{"feeds collection empty filter", "/feeds?onestop_id=nonexistent", http.StatusOK},
		{"routes collection empty filter", "/routes?onestop_id=nonexistent", http.StatusOK},
		{"trips collection for route", "/routes/r-9q9-bu~1u-e535e4/trips", http.StatusOK},

		// GeoJSON format should also return 404 for not found
		{"feed not found geojson", "/feeds/f-nonexistent.geojson", http.StatusNotFound},
		{"feed exists geojson", "/feeds/BA.geojson", http.StatusOK},

		// PNG format should also return 404 for not found
		{"route not found png", "/routes/r-nonexistent.png", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			restSrv.ServeHTTP(rr, req)

			if sc := rr.Result().StatusCode; sc != tt.wantStatus {
				t.Errorf("got status %d, want %d for path %s", sc, tt.wantStatus, tt.path)
			}
		})
	}
}
