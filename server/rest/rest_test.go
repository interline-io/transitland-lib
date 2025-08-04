package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-mw/auth/authn"
	"github.com/interline-io/transitland-mw/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) {
	// Increase limit for test
	MAXLIMIT = 100_000
	gql.MAXLIMIT = MAXLIMIT
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
	return model.AddConfigAndPerms(cfg, graphqlHandler),
		model.AddConfigAndPerms(cfg, restHandler),
		cfg
}

func checkTestCase(t *testing.T, tc testCase) {
	graphqlHandler, _, _ := testHandlersWithOptions(t, testconfig.Options{
		WhenUtc: "2018-06-01T00:00:00Z",
		RTJsons: testconfig.DefaultRTJson(),
		Storage: testdata.Path("tmp"),
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
