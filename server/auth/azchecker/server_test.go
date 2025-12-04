package azchecker

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestServer(t *testing.T) {
	fgaUrl, a, ok := testutil.CheckEnv("TL_TEST_FGA_ENDPOINT")
	if !ok {
		t.Skip(a)
		return
	}
	dbx := testutil.MustOpenTestDB(t)
	serverTestData := []testCase{
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "BA-group"),
			Relation: ParentRelation,
			Notes:    "org:BA-group belongs to tenant:tl-tenant",
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "CT-group"),
			Relation: ParentRelation,
			Notes:    "org:CT-group belongs to tenant:tl-tenant",
		},
		{
			Subject:  newEntityKey(TenantType, "restricted-tenant"),
			Object:   newEntityKey(GroupType, "test-group"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(UserType, "tl-tenant-admin"),
			Object:   newEntityKey(TenantType, "tl-tenant"),
			Relation: AdminRelation,
		},

		{
			Subject:  newEntityKey(GroupType, "BA-group"),
			Object:   newEntityKey(FeedType, "BA"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(GroupType, "CT-group"),
			Object:   newEntityKey(FeedType, "CT"),
			Relation: ParentRelation,
		},

		{
			Subject:  newEntityKey(UserType, "ian"),
			Object:   newEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  newEntityKey(UserType, "drew"),
			Object:   newEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  newEntityKey(UserType, "ian"),
			Object:   newEntityKey(GroupType, "BA-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(UserType, "drew"),
			Object:   newEntityKey(GroupType, "CT-group"),
			Relation: EditorRelation,
		},
		{
			Subject:  newEntityKey(UserType, "drew"),
			Object:   newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ViewerRelation,
			Notes:    "assign drew permission to view this BA feed",
		},
	}

	// USERS
	t.Run("Me", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				CheckAsUser: "ian",
				ExpectKeys:  newEntityKeys(GroupType, "BA-group"),
			},
			{
				CheckAsUser: "drew",
				ExpectKeys:  newEntityKeys(GroupType, "CT-group"),
			},
			{
				CheckAsUser: "asdf",
				ExpectKeys:  newEntityKeys(GroupType),
				// ExpectError: true,
				// used to return an error, but is now returned without auth0 lookup
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", "/me", nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				assert.ElementsMatch(
					t,
					ekGetNames(tc.ExpectKeys),
					responseGetNames(t, rr.Body.Bytes(), "groups", "name"),
				)
			})
		}
	})

	// TENANTS
	t.Run("TenantList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant"),
			},
			{
				Subject:    newEntityKey(UserType, "ian"),
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant"),
			},
			{
				Subject:    newEntityKey(UserType, "drew"),
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant"),
			},
			{
				Subject:    newEntityKey(UserType, "unknown"),
				ExpectKeys: newEntityKeys(TenantType),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", "/tenants", nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				assert.ElementsMatch(
					t,
					ekGetNames(tc.ExpectKeys),
					responseGetNames(t, rr.Body.Bytes(), "tenants", "name"),
				)
			})
		}
	})

	t.Run("TenantPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateOrg, CanDeleteOrg},
			},
			{
				Subject:            newEntityKey(UserType, "tl-tenant-admin"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:            newEntityKey(UserType, "unknown"),
				Object:             newEntityKey(TenantType, "tl-tenant"),
				ExpectUnauthorized: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", fmt.Sprintf("/tenants/%s", ltk.Object.Name), nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				gotActions := responseGetActions(t, rr.Body.Bytes())
				assert.ElementsMatch(t, tc.ExpectActions, gotActions)
			})
		}
	})

	// GROUPS
	t.Run("GroupList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				ExpectKeys: newEntityKeys(GroupType, "BA-group", "CT-group"),
			},
			{
				Subject:    newEntityKey(UserType, "ian"),
				ExpectKeys: newEntityKeys(GroupType, "BA-group"),
			},
			{
				Subject:    newEntityKey(UserType, "drew"),
				ExpectKeys: newEntityKeys(GroupType, "CT-group"),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", "/groups", nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				assert.ElementsMatch(
					t,
					ekGetNames(tc.ExpectKeys),
					responseGetNames(t, rr.Body.Bytes(), "groups", "name"),
				)
			})
		}
	})

	t.Run("GroupPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, CanEdit},
			},
			{
				Subject:            newEntityKey(UserType, "unknown"),
				Object:             newEntityKey(GroupType, "CT-group"),
				ExpectUnauthorized: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", fmt.Sprintf("/groups/%s", ltk.Object.Name), nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				gotActions := responseGetActions(t, rr.Body.Bytes())
				assert.ElementsMatch(t, tc.ExpectActions, gotActions)
			})
		}
	})

	// FEEDS
	t.Run("FeedList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				ExpectKeys: newEntityKeys(TenantType, "BA", "CT"),
			},
			{
				Subject:    newEntityKey(UserType, "ian"),
				ExpectKeys: newEntityKeys(TenantType, "BA"),
			},
			{
				Subject:    newEntityKey(UserType, "drew"),
				ExpectKeys: newEntityKeys(TenantType, "CT"),
			},
			{
				Subject:    newEntityKey(UserType, "unknown"),
				ExpectKeys: newEntityKeys(TenantType),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", "/feeds", nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				assert.ElementsMatch(
					t,
					ekGetNames(tc.ExpectKeys),
					responseGetNames(t, rr.Body.Bytes(), "feeds", "onestop_id"),
				)
			})
		}
	})

	t.Run("FeedPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(FeedType, "BA"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion, CanSetGroup},
			},
			{
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedType, "BA"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(FeedType, "CT"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion},
			},
			{
				Subject:            newEntityKey(UserType, "unknown"),
				Object:             newEntityKey(FeedType, "CT"),
				ExpectUnauthorized: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", fmt.Sprintf("/feeds/%s", ltk.Object.Name), nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				gotActions := responseGetActions(t, rr.Body.Bytes())
				assert.ElementsMatch(t, tc.ExpectActions, gotActions)
			})
		}
	})

	// FEED VERSIONS
	t.Run("FeedVersionList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				ExpectKeys: newEntityKeys(FeedVersionType),
			},
			{
				Subject:    newEntityKey(UserType, "ian"),
				ExpectKeys: newEntityKeys(FeedVersionType),
			},
			{
				Subject:    newEntityKey(UserType, "drew"),
				ExpectKeys: newEntityKeys(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", "/feed_versions", nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				assert.ElementsMatch(
					t,
					ekGetNames(tc.ExpectKeys),
					responseGetNames(t, rr.Body.Bytes(), "feed_versions", "sha1"),
				)
			})
		}
	})

	t.Run("FeedVersionPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, serverTestData)
		checks := []testCase{
			{
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers},
			},
			{
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:            newEntityKey(UserType, "unknown"),
				Object:             newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectUnauthorized: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				srv := testServerWithUser(checker, tc)
				req, _ := http.NewRequest("GET", fmt.Sprintf("/feed_versions/%s", ltk.Object.Name), nil)
				rr := httptest.NewRecorder()
				srv.ServeHTTP(rr, req)
				checkHttpExpectError(t, tc, rr)
				gotActions := responseGetActions(t, rr.Body.Bytes())
				assert.ElementsMatch(t, tc.ExpectActions, gotActions)
			})
		}
	})

}

func testServerWithUser(c *Checker, tk testCase) http.Handler {
	srv, _ := NewServer(c)
	srv = usercheck.UseDefaultUserMiddleware(stringOr(tk.CheckAsUser, tk.Subject.Name))(srv)
	return srv
}

func checkHttpExpectError(t testing.TB, tk testCase, rr *httptest.ResponseRecorder) {
	status := rr.Code
	if tk.ExpectUnauthorized {
		if status != http.StatusUnauthorized {
			t.Errorf("got error code %d, expected %d", status, http.StatusUnauthorized)
		}
	} else if tk.ExpectError {
		if status == http.StatusOK {
			t.Errorf("got status %d, expected non-200", status)
		}
	} else if status != http.StatusOK {
		t.Errorf("got error code %d, expected 200", status)
	}

}

func responseGetNames(_ testing.TB, data []byte, path string, key string) []string {
	a := gjson.ParseBytes(data).Get(path)
	var ret []string
	for _, b := range a.Array() {
		ret = append(ret, b.Get(key).Str)
	}
	return ret
}

func ekGetNames(eks []EntityKey) []string {
	var ret []string
	for _, ek := range eks {
		ret = append(ret, ek.Name)
	}
	return ret
}

func responseGetActions(t testing.TB, data []byte) []Action {
	a := gjson.ParseBytes(data).Get("actions")
	var ret []Action
	for k, v := range a.Map() {
		if v.Bool() {
			a, err := authz.ActionString(k)
			if err != nil {
				t.Errorf("invalid action %s", k)
			}
			ret = append(ret, a)
		}
	}
	return ret
}
