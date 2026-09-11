package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
)

// EG is the only feed with feed_states.public = false in the test fixtures, and
// this is its one feed version.
const (
	privateFeedOnestopID = "EG"
	privateFeedVersion   = "793c9c759eb54007e57ce2dbb2c927700c5c34d4"
)

// These endpoints are the only GTFS export open to ordinary users, and the
// permission filter is the only thing between a caller and a feed they were
// never granted. Nothing else in this package exercises a non-public feed: the
// other download tests all use public fixtures, which pass under the default
// deny-all checker because public wins regardless of grants.
func TestFeedVersionDownloadPermissions(t *testing.T) {
	// granted-user reaches EG through a group; nobody-user holds no tuples.
	tuples := []authz.TupleKey{
		{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "EG-group"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.GroupType, "EG-group"), Object: authz.NewEntityKey(authz.FeedType, privateFeedOnestopID), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "granted-user"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.MemberRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "granted-user"), Object: authz.NewEntityKey(authz.GroupType, "EG-group"), Relation: authz.ViewerRelation},
	}
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		FGAEndpoint:    testutil.FGAServer(t),
		FGAModelFile:   testdata.Path("server/authz/tls.json"),
		FGAModelTuples: tuples,
		Storage:        testdata.Path("server", "tmp"),
		// The private feed needs realtime data for the realtime case to mean
		// anything: the RT store is an unfiltered cache, so without a message
		// present that endpoint answers "not found" whatever the permissions say.
		RTJsons: []testconfig.RTJsonFile{
			{Feed: privateFeedOnestopID, Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
		},
	})

	get := func(user, path string) *httptest.ResponseRecorder {
		h := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser(user, user, user+"@example.com")
		})(restSrv)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		return rr
	}

	fvPath := "/feed_versions/" + privateFeedVersion + "/download"
	latestPath := "/feeds/" + privateFeedOnestopID + "/download_latest_feed_version"
	rtPath := "/feeds/" + privateFeedOnestopID + "/download_latest_rt/alerts.json"

	// The negative cases are the point. The granted-user cases are the control:
	// without them a 404 could just mean the fixture is missing.
	t.Run("no grants cannot download by feed version", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get("nobody-user", fvPath).Result().StatusCode)
	})
	t.Run("no grants cannot download latest", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get("nobody-user", latestPath).Result().StatusCode)
	})
	t.Run("no grants cannot read realtime", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get("nobody-user", rtPath).Result().StatusCode)
	})

	t.Run("granted user downloads the private feed version", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get("granted-user", fvPath).Result().StatusCode)
	})
	t.Run("granted user downloads the private feed", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get("granted-user", latestPath).Result().StatusCode)
	})
	t.Run("granted user reads the private feed realtime", func(t *testing.T) {
		// Without this the denial above would pass even if the fixture stopped
		// loading, since a missing message is a 404 whatever the permissions say.
		assert.Equal(t, http.StatusOK, get("granted-user", rtPath).Result().StatusCode)
	})
}
