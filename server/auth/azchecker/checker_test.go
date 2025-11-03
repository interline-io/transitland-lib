package azchecker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Ensure Checker implements CheckerServer
	var _ authz.CheckerServer = &Checker{}
}

type testCase struct {
	Subject            authz.EntityKey
	Object             authz.EntityKey
	Action             authz.Action
	Relation           authz.Relation
	Expect             string
	Notes              string
	ExpectError        bool
	ExpectUnauthorized bool
	CheckAsUser        string
	ExpectActions      []authz.Action
	ExpectKeys         []authz.EntityKey
}

func (tk *testCase) TupleKey() TupleKey {
	return TupleKey{Subject: tk.Subject, Object: tk.Object, Relation: tk.Relation, Action: tk.Action}
}

func (tk *testCase) String() string {
	if tk.Notes != "" {
		return tk.Notes
	}
	a := tk.TupleKey().String()
	if tk.CheckAsUser != "" {
		a = a + "|checkuser:" + tk.CheckAsUser
	}
	return a
}

func TestMain(m *testing.M) {
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

func TestChecker(t *testing.T) {
	fgaUrl, a, ok := testutil.CheckEnv("TL_TEST_FGA_ENDPOINT")
	if !ok {
		t.Skip(a)
		return
	}
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
		return
	}
	dbx := testutil.MustOpenTestDB(t)
	checkerTestData := []testCase{
		// Assign users to tenants
		{
			Notes:    "all users can access all-users-tenant",
			Subject:  newEntityKey(UserType, "*"),
			Object:   newEntityKey(TenantType, "all-users-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  newEntityKey(UserType, "tl-tenant-admin"),
			Object:   newEntityKey(TenantType, "tl-tenant"),
			Relation: AdminRelation,
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
			Subject:  newEntityKey(UserType, "test2"),
			Object:   newEntityKey(TenantType, "restricted-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  newEntityKey(UserType, "tl-tenant-member"),
			Object:   newEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		// Assign groups to tenants
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "CT-group"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "BA-group"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "HA-group"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant"),
			Object:   newEntityKey(GroupType, "EX-group"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant").WithRefRel(MemberRelation),
			Object:   newEntityKey(GroupType, "HA-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "restricted-tenant"),
			Object:   newEntityKey(GroupType, "test-group"),
			Relation: ParentRelation,
		},
		// Assign users to groups
		{
			Subject:  newEntityKey(UserType, "ian"),
			Object:   newEntityKey(GroupType, "CT-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(UserType, "ian"),
			Object:   newEntityKey(GroupType, "BA-group"),
			Relation: EditorRelation,
		},
		{
			Subject:  newEntityKey(UserType, "drew"),
			Object:   newEntityKey(GroupType, "CT-group"),
			Relation: ManagerRelation,
		},
		{
			Subject:  newEntityKey(UserType, "test-group-viewer"),
			Object:   newEntityKey(GroupType, "test-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(UserType, "test-group-editor"),
			Object:   newEntityKey(GroupType, "test-group"),
			Relation: EditorRelation,
		},
		// Assign feeds to groups
		{
			Subject:  newEntityKey(GroupType, "CT-group"),
			Object:   newEntityKey(FeedType, "CT"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(GroupType, "BA-group"),
			Object:   newEntityKey(FeedType, "BA"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(GroupType, "HA-group"),
			Object:   newEntityKey(FeedType, "HA"),
			Relation: ParentRelation,
		},
		{
			Subject:  newEntityKey(GroupType, "EX-group"),
			Object:   newEntityKey(FeedType, "EX"),
			Relation: ParentRelation,
		},
		// Assign feed versions
		{
			Subject:  newEntityKey(UserType, "tl-tenant-member"),
			Object:   newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(GroupType, "test-group").WithRefRel(ViewerRelation),
			Object:   newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ViewerRelation,
		},
		{
			Subject:  newEntityKey(TenantType, "tl-tenant").WithRefRel(MemberRelation),
			Object:   newEntityKey(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			Relation: ViewerRelation,
		},
	}

	// Users

	t.Run("UserList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		tcs := []struct {
			Notes       string
			CheckAsUser string
			ExpectUsers []string
			ExpectError bool
			Query       string
		}{
			{
				Notes:       "user ian can see all users",
				CheckAsUser: "ian",
				ExpectUsers: []string{"ian", "drew", "tl-tenant-member", "new-user"},
			},
			{
				Notes:       "user ian can filter with query=drew",
				CheckAsUser: "ian",
				Query:       "drew",
				ExpectUsers: []string{"drew"},
			},
			// TODO: user filtering
			// {
			// 	CheckAsUser: "no-one",
			// 	ExpectUsers: []string{},
			// 	ExpectError: true,
			// },
		}
		for _, tc := range tcs {
			t.Run(tc.Notes, func(t *testing.T) {
				ents, err := checker.UserList(newUserCtx(tc.CheckAsUser), &authz.UserListRequest{Q: tc.Query})
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				var entNames []string
				for _, ent := range ents.Users {
					entNames = append(entNames, ent.Id)
				}
				assert.ElementsMatch(t, tc.ExpectUsers, entNames)
			})
		}
	})

	t.Run("User", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		tcs := []struct {
			Notes        string
			CheckAsUser  string
			ExpectUserId string
			ExpectError  bool
		}{
			{
				Notes:        "ok",
				CheckAsUser:  "ian",
				ExpectUserId: "drew",
			},
			{
				Notes:        "not found",
				CheckAsUser:  "ian",
				ExpectUserId: "not found",
				ExpectError:  true,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.Notes, func(t *testing.T) {
				ent, err := checker.User(
					newUserCtx(tc.CheckAsUser),
					&authz.UserRequest{Id: tc.ExpectUserId},
				)
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				if ent == nil {
					t.Fatal("got no result")
				}
				assert.Equal(t, tc.ExpectUserId, ent.User.Id)
			})
		}
	})

	t.Run("Me", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		tcs := []struct {
			Name                     string
			CheckAsUser              string
			ExpectUserId             string
			ExpectDirectGroupNames   []string
			ExpectExpandedGroupNames []string
			ExpectError              bool
		}{
			{
				Name:                     "ian",
				CheckAsUser:              "ian",
				ExpectUserId:             "ian",
				ExpectDirectGroupNames:   []string{"CT-group", "BA-group"},
				ExpectExpandedGroupNames: []string{"CT-group", "HA-group", "BA-group"},
			},
			{
				Name:                     "drew",
				CheckAsUser:              "drew",
				ExpectUserId:             "drew",
				ExpectDirectGroupNames:   []string{"CT-group"},
				ExpectExpandedGroupNames: []string{"CT-group", "HA-group"},
			},
			{
				Name:         "no one",
				CheckAsUser:  "no one",
				ExpectUserId: "",
				ExpectError:  true,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.Name, func(t *testing.T) {
				ent, err := checker.Me(newUserCtx(tc.CheckAsUser), &authz.MeRequest{})
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				if ent == nil {
					t.Fatal("got no result")
				}
				assert.Equal(t, tc.ExpectUserId, ent.User.Id)
				var directGroupNames []string
				for _, g := range ent.Groups {
					directGroupNames = append(directGroupNames, g.Name)
				}
				assert.ElementsMatch(t, tc.ExpectDirectGroupNames, directGroupNames, "group names")

				var expandedGroupNames []string
				for _, g := range ent.ExpandedGroups {
					expandedGroupNames = append(expandedGroupNames, g.Name)
				}
				assert.ElementsMatch(t, tc.ExpectExpandedGroupNames, expandedGroupNames, "group names")
			})
		}
	})

	// TENANTS
	t.Run("TenantList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			{
				Notes:      "user tl-tenant-admin is admin of tl-tenant and user:* on all-users-tenant",
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				Object:     newEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "user ian is member of tl-tenant and user:* on all-users-tenant",
				Subject:    newEntityKey(UserType, "ian"),
				Object:     newEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "user tl-tenant-member is member of tl-tenant and user:* on all-users-tenant",
				Subject:    newEntityKey(UserType, "tl-tenant-member"),
				Object:     newEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "user new-user is user:* on all-users-tenant",
				Subject:    newEntityKey(UserType, "new-user"),
				Object:     newEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "all-users-tenant"),
			},
		}

		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ret, err := checker.TenantList(
					newUserCtx(tc.CheckAsUser, tc.Subject.Name),
					&authz.TenantListRequest{},
				)
				if err != nil {
					t.Fatal(err)
				}
				var gotNames []string
				for _, v := range ret.Tenants {
					gotNames = append(gotNames, v.Name)
				}
				var expectNames []string
				for _, v := range tc.ExpectKeys {
					expectNames = append(expectNames, v.Name)
				}
				assert.ElementsMatch(t, expectNames, gotNames, "tenant names")
			})
		}
	})

	t.Run("TenantPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			// User checks
			{
				Notes:         "user tl-tenant-admin is an admin of tl-tenant",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateOrg, CanDeleteOrg},
			},
			{
				Notes:              "user tl-tenant-admin is unauthorized for restricted-tenant",
				Subject:            newEntityKey(UserType, "tl-tenant-admin"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user ian is viewer of tl-tenant",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Notes:              "user ian is unauthorized for restricted-tenant",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user drew is a viewer of tl-tenant",
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Notes:              "user drew is unauthorized for restricted-tenant",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user tl-tenant-member is a viewer of tl-tenant",
				Subject:       newEntityKey(UserType, "tl-tenant-member"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Notes:              "user tl-tenant-member is unauthorized for restricted-tenant",
				Subject:            newEntityKey(UserType, "tl-tenant-member"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user tl-tenant-member is a viewer of tl-tenant",
				Subject:       newEntityKey(UserType, "tl-tenant-member"),
				Object:        newEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Notes:              "user tl-tenant-member is unauthorized for restricted-tenant",
				Subject:            newEntityKey(UserType, "tl-tenant-member"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user tl-tenant-member expects unauthorized error for non-existing tenant",
				Subject:            newEntityKey(UserType, "tl-tenant-member"),
				Object:             newEntityKey(TenantType, "not found"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user ian is viewer of all-users-tenant through user:*",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(TenantType, "all-users-tenant"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Notes:         "user new-user is viewer of all-users-tenant through user:*",
				Subject:       newEntityKey(UserType, "new-user"),
				Object:        newEntityKey(TenantType, "all-users-tenant"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			// General checks
			{
				Notes:         "global admins are admins of all tenants",
				Subject:       newEntityKey(UserType, "global_admin"),
				Object:        newEntityKey(TenantType, "all-users-tenant"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateOrg, CanDeleteOrg},
			},
			{
				Notes:       "global admins get not found on not found tenant",
				Subject:     newEntityKey(UserType, "global_admin"),
				Object:      newEntityKey(TenantType, "not found"),
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				ret, err := checker.TenantPermissions(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.TenantRequest{Id: ltk.Object.ID()},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
				if err != nil {
					return
				}
				checkActionSubset(t, ret.Actions, tc.ExpectActions)
			})
		}
	})

	t.Run("TenantAddPermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:       "user tl-tenant-admin is an admin of tl-tenant and can add a user",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "user tl-tenant-admin is an admin of tl-tenant and can add user:*",
				Subject:     newEntityKey(UserType, "*"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is a vewier of tl-tenant is not authorized to add a user",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(TenantType, "tl-tenant"),
				Relation:           MemberRelation,
				CheckAsUser:        "ian",
				ExpectUnauthorized: true,
			},
			// General checks
			{
				Notes:       "error for invalid relation",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    ParentRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "error for disallowed relation",
				Subject:     newEntityKey(UserType, "*"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    AdminRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "replaces relation if it already exists",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "users get unauthorized when attempting to add user to not found tenant",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(TenantType, "not found"),
				Relation:           MemberRelation,
				CheckAsUser:        "tl-tenant-admin",
				ExpectUnauthorized: true,
			},
			{
				Notes:       "global admins can add users to all tenants",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(TenantType, "restricted-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins get not found when adding user to a not found tenant",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(TenantType, "not found"),
				Relation:    MemberRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
			// TODO
			// {
			// 	Notes:       "user tl-tenant-admin gets an error when attempting to add a user that does not exist",
			// 	Subject:     newEntityKey(UserType, "not found"),
			// 	Object:      newEntityKey(TenantType, "tl-tenant"),
			// 	Relation:    MemberRelation,
			// 	CheckAsUser: "tl-tenant-admin",
			// 	ExpectError: true,
			// },
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.TenantAddPermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.TenantModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

	t.Run("TenantRemovePermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:       "tl-tenant-admin is a admin of tl-tenant and can remove a user",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is a viewer of tl-tenant and is not authorized to remove a user",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(TenantType, "tl-tenant"),
				Relation:           MemberRelation,
				CheckAsUser:        "ian",
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user tl-tenant-admin is not a member of restricted-tenant and is not authorized to remove a user",
				Subject:            newEntityKey(UserType, "test2"),
				Object:             newEntityKey(TenantType, "restricted-tenant"),
				Relation:           MemberRelation,
				CheckAsUser:        "tl-tenant-admin",
				ExpectUnauthorized: true,
			},
			// General checks
			{
				Notes:       "error if relation does not exist",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "tl-tenant-admin",
				ExpectError: true,
			}, {
				Notes:              "users get unauthorized when attemping to add user to not found tenant",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(TenantType, "not found"),
				Relation:           MemberRelation,
				CheckAsUser:        "tl-tenant-admin",
				ExpectUnauthorized: true,
			},
			{
				Notes:       "global admins can remove users from all tenants",
				Subject:     newEntityKey(UserType, "test2"),
				Object:      newEntityKey(TenantType, "restricted-tenant"),
				Relation:    MemberRelation,
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins get error when removing user from not found tenant",
				Subject:     newEntityKey(UserType, "test2"),
				Object:      newEntityKey(TenantType, "not found"),
				Relation:    MemberRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
			// TODO
			// {
			// 	Notes:       "removing a non-existing user causes an error",
			// 	Subject:     newEntityKey(UserType, "asd123"),
			// 	Object:      newEntityKey(TenantType, "tl-tenant"),
			// 	Relation:    MemberRelation,
			// 	CheckAsUser: "tl-tenant-admin",
			// 	ExpectError: true,
			// },
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.TenantRemovePermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.TenantModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

	t.Run("TenantSave", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:              "user ian is a viewer of tl-tenant and is not authorized to edit",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(TenantType, "tl-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user new-user is not a viewer of tl-tenant",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(TenantType, "tl-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user new-user is a viewer of all-users-tenant through user:* but not admin",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(TenantType, "all-users-tenant"),
				ExpectUnauthorized: true,
			},
			{
				Notes:   "user tl-tenant-admin is admin of tl-tenant and can edit",
				Subject: newEntityKey(UserType, "tl-tenant-admin"),
				Object:  newEntityKey(TenantType, "tl-tenant"),
			},
			// General checks
			{
				Notes:              "users get unauthorized for tenant that does not exist",
				Subject:            newEntityKey(UserType, "tl-tenant-admin"),
				Object:             newEntityKey(TenantType, "not found"),
				ExpectUnauthorized: true,
			},
			{
				Notes:       "global admins get error for not found tenant",
				Subject:     newEntityKey(UserType, "tl-tenant-admin"),
				Object:      newEntityKey(TenantType, "new tenant"),
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.TenantSave(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.TenantSaveRequest{
						Tenant: &authz.Tenant{
							Id:   ltk.Object.ID(),
							Name: tc.Object.Name,
						},
					},
				)
				if checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized) {
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	t.Run("TenantCreateGroup", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:              "user ian is viewer of tl-tenant and not authorized to create groups",
				Subject:            newEntityKey(TenantType, "tl-tenant"),
				Object:             newEntityKey(GroupType, "new-group"),
				ExpectUnauthorized: true,
				CheckAsUser:        "ian",
			},
			{
				Notes:              "user new-user is viewer of all-users-tenant and not authorized to create groups",
				Subject:            newEntityKey(TenantType, "all-users-tenant"),
				Object:             newEntityKey(GroupType, "new-group"),
				ExpectUnauthorized: true,
				CheckAsUser:        "new-user",
			},
			{
				Notes:       "user tl-tenant-admin is admin of tl-tenant and can create groups",
				Subject:     newEntityKey(TenantType, "tl-tenant"),
				Object:      newEntityKey(GroupType, fmt.Sprintf("new-group2-%d", time.Now().UnixNano())),
				CheckAsUser: "tl-tenant-admin",
			},
			// General checks
			{
				Notes:       "global admins can create groups in all tenants",
				Subject:     newEntityKey(TenantType, "tl-tenant"),
				Object:      newEntityKey(GroupType, fmt.Sprintf("new-group3-%d", time.Now().UnixNano())),
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins can create groups in all tenants",
				Subject:     newEntityKey(TenantType, "restricted-tenant"),
				Object:      newEntityKey(GroupType, fmt.Sprintf("new-group4-%d", time.Now().UnixNano())),
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins get not found for tenant that does not exist",
				Subject:     newEntityKey(TenantType, "not found"),
				Object:      newEntityKey(GroupType, fmt.Sprintf("new-group5-%d", time.Now().UnixNano())),
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.TenantCreateGroup(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.TenantCreateGroupRequest{
						Id:    ltk.Subject.ID(),
						Group: &authz.Group{Name: tc.Object.Name},
					},
				)
				if checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized) {
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				// TODO: DELETE GROUP
			})
		}
	})

	// GROUPS
	t.Run("GroupList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			{
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				Object:     newEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "BA-group", "HA-group", "EX-group"),
			},
			{
				Subject:    newEntityKey(UserType, "ian"),
				Object:     newEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "BA-group", "HA-group"),
			},
			{
				Subject:    newEntityKey(UserType, "drew"),
				Object:     newEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "HA-group"),
			},
			{
				Subject:    newEntityKey(UserType, "tl-tenant-member"),
				Object:     newEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "HA-group"),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ret, err := checker.GroupList(
					newUserCtx(tc.CheckAsUser, tc.Subject.Name),
					&authz.GroupListRequest{},
				)
				if err != nil {
					t.Fatal(err)
				}
				var gotNames []string
				for _, v := range ret.Groups {
					gotNames = append(gotNames, v.Name)
				}
				var expectNames []string
				for _, v := range tc.ExpectKeys {
					expectNames = append(expectNames, v.Name)
				}
				assert.ElementsMatch(t, expectNames, gotNames, "group names")
			})
		}
	})

	t.Run("GroupPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			// User checks
			{
				Notes:         "user tl-tenant-admin is admin of parent tenant to CT-group",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Notes:         "user tl-tenant-admin is admin of parent tenant to BA-group",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Notes:         "user tl-tenant-admin is admin of parent tenant to BA-group",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Notes:         "user tl-tenant-admin is admin of parent tenant to BA-group",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Notes:         "user ian is a viewer of CT-group",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Notes:         "user ian is a editor of CT-group",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView, CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Notes:         "user ian is a viewer of HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Notes:              "user ian is not authorized for EX-group",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(GroupType, "EX-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user ian is not authorized for test-group",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(GroupType, "test-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user drew is a manager of CT-group",
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Notes:              "user drew is not authrozied for BA-group",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(GroupType, "BA-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user drew is a viewer of HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Notes:              "user drew is not authorized for EX-group",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(GroupType, "EX-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user drew is not authorized for group test-group",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(GroupType, "test-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user tl-tenant-member is a viewer of HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "tl-tenant-member"),
				Object:        newEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Notes:              "tl-tenant-member is not authorized to access EX-group",
				Subject:            newEntityKey(UserType, "tl-tenant-member"),
				Object:             newEntityKey(GroupType, "EX-group"),
				ExpectUnauthorized: true,
			},
			// General checks
			{
				Notes:              "users get unauthorized for groups that are not found",
				Subject:            newEntityKey(UserType, "tl-tenant-admin"),
				Object:             newEntityKey(GroupType, "test-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "global admins are managers of all groups",
				Subject:       newEntityKey(UserType, "global_admin"),
				Object:        newEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed, CanSetTenant},
			},
			{
				Notes:       "global admins get not found for not found groups",
				Subject:     newEntityKey(UserType, "global_admin"),
				Object:      newEntityKey(GroupType, "not found"),
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				ret, err := checker.GroupPermissions(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.GroupRequest{Id: ltk.Object.ID()},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
				if err != nil {
					return
				}
				checkActionSubset(t, ret.Actions, tc.ExpectActions)
			})
		}
	})

	t.Run("GroupAddPermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			// TODO
			// {
			// 	Notes:       "user tl-tenant-admin gets error when adding user that does not exist",
			// 	Subject:     newEntityKey(UserType, "test100"),
			// 	Object:      newEntityKey(GroupType, "HA-group"),
			// 	Relation:    ViewerRelation,
			// 	ExpectError: true,
			// 	CheckAsUser: "tl-tenant-admin",
			// },
			{
				Notes:       "tl-tenant-admin is manager of CT-group through tl-tenant and can add user",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "tl-tenant-admin is manager of CT-group through tl-tenant and can add tenant#member as viewer",
				Subject:     newEntityKey(TenantType, "tl-tenant").WithRefRel(MemberRelation),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "tl-tenant-admin is manager of CT-group through tl-tenant and can add tenant#member as editor",
				Subject:     newEntityKey(TenantType, "tl-tenant").WithRefRel(MemberRelation),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    EditorRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is viewer of CT-group and not authorized to add users",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(GroupType, "CT-group"),
				Relation:           ViewerRelation,
				ExpectUnauthorized: true,
				CheckAsUser:        "ian",
			},
			{
				Notes:       "user drew is a manager of CT-group and can add users",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "drew",
			},
			// General checks
			{
				Notes:              "users get unauthorized for groups that do not exist",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(GroupType, "not found"),
				Relation:           ViewerRelation,
				CheckAsUser:        "ian",
				ExpectUnauthorized: true,
			},
			{
				Notes:       "error for invalid relation",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ParentRelation,
				CheckAsUser: "tl-tenant-admin",
				ExpectError: true,
			},
			{
				Notes:       "error for invalid relation",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(GroupType, "HA-group"),
				Relation:    ParentRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "error for disallowed relation",
				Subject:     newEntityKey(GroupType, "BA-group").WithRefRel(MemberRelation),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
				ExpectError: true,
			},
			{
				Notes:       "error for disallowed relation",
				Subject:     newEntityKey(TenantType, "tl-tenant#admin"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    EditorRelation,
				CheckAsUser: "tl-tenant-admin",
				ExpectError: true,
			},
			{
				Notes:       "global admin can add users to any group",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admin gets not found for groups that do not exist",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "not found"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.GroupAddPermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.GroupModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

	t.Run("GroupRemovePermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:       "user tl-tenant-admin is manager of CT-group through tl-tenant",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is viewer of BA-group and is not authorized to add users",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(GroupType, "BA-group"),
				Relation:           ViewerRelation,
				ExpectUnauthorized: true,
				CheckAsUser:        "ian",
			},
			// General checks
			{
				Notes:              "users get authorized for groups that do not exist",
				Subject:            newEntityKey(UserType, "new-user"),
				Object:             newEntityKey(GroupType, "not found"),
				Relation:           ViewerRelation,
				ExpectUnauthorized: true,
				CheckAsUser:        "ian",
			},
			{
				Notes:       "users get error for removing tuple that does not exist",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "BA-group"),
				Relation:    ViewerRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "global admins can remove users from any group",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins get error for removing tuples that do not exist",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(GroupType, "CT-group"),
				Relation:    EditorRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
			{
				Notes:       "global admins get not found for groups that do not exist",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(GroupType, "not found"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.GroupRemovePermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.GroupModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

	t.Run("GroupSave", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:              "user ian is a viewer of CT-group and cannot edit",
				CheckAsUser:        "ian",
				Object:             newEntityKey(GroupType, "CT-group"),
				ExpectUnauthorized: true,
			},
			{
				Notes:       "user tl-tenant-admin is a manager of BA-group through tl-tenant and can edit",
				Object:      newEntityKey(GroupType, "BA-group"),
				CheckAsUser: "tl-tenant-admin",
			},
			// General checks
			{
				Notes:              "users get unauthorized for groups that are not found",
				Object:             newEntityKey(GroupType, "not found"),
				CheckAsUser:        "tl-tenant-admin",
				ExpectUnauthorized: true,
			},
			{
				Notes:       "global admins can edit any group",
				Object:      newEntityKey(GroupType, "BA-group"),
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins get not found for groups that are not found",
				Object:      newEntityKey(GroupType, "not found"),
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.GroupSave(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.GroupSaveRequest{
						Group: &authz.Group{
							Id:   ltk.Object.ID(),
							Name: tc.Object.Name,
						},
					},
				)
				if checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized) {
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	// FEEDS
	t.Run("FeedList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			{
				Notes:      "user tl-tenant-admin can see all feeds with groups that are in tl-tenant",
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				Object:     newEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "BA", "HA", "EX"),
			},
			{
				Notes:      "user ian is viewer in CT-group, BA-group, and also HA-group through tl-tenant#member",
				Subject:    newEntityKey(UserType, "ian"),
				Object:     newEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "BA", "HA"),
			},
			{
				Notes:      "user drew is editor of CT-group and can see feed CT and also feed HA through tl-tenant#member on HA-group",
				Subject:    newEntityKey(UserType, "drew"),
				Object:     newEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "HA"),
			},
			{
				Notes:      "user tl-tenant-member can see feed HA through tl-tenant#member on HA-group",
				Subject:    newEntityKey(UserType, "tl-tenant-member"),
				Object:     newEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "HA"),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ret, err := checker.FeedList(
					newUserCtx(tc.CheckAsUser, tc.Subject.Name),
					&authz.FeedListRequest{},
				)
				if err != nil {
					t.Fatal(err)
				}
				var gotNames []string
				for _, v := range ret.Feeds {
					gotNames = append(gotNames, v.OnestopId)
				}
				var expectNames []string
				for _, v := range tc.ExpectKeys {
					expectNames = append(expectNames, v.Name)
				}
				assert.ElementsMatch(t, expectNames, gotNames, "feed names")
			})
		}
	})

	t.Run("FeedPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			// User checks
			{
				Notes:         "user ian is a viewer of CT through CT-group",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedType, "CT"),
				ExpectActions: []Action{CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Notes:         "user ian is a editor of BA through BA-group",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedType, "BA"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion},
			},
			{
				Notes:         "user ian is a viewer of HA through HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Notes:              "user ian is unauthorized for feed EX",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(FeedType, "EX"),
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user ian is unauthorized for feed test",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(FeedType, "test"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user drew is manager of feed CT through CT-group",
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(FeedType, "CT"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion, CanSetGroup},
			},
			{
				Notes:              "user drew is unauthorized for feed BA",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(FeedType, "BA"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user drew is viewer of feed HA through HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "drew"),
				Object:        newEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Notes:              "user tl-tenant-member is unauthorized for feed BA",
				Subject:            newEntityKey(UserType, "tl-tenant-member"),
				Object:             newEntityKey(FeedType, "BA"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "user tl-tenant-member is viewer for feed HA through HA-group through tl-tenant#member",
				Subject:       newEntityKey(UserType, "tl-tenant-member"),
				Object:        newEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			// General checks
			{
				Notes:              "users get unauthorized for a feed that is not found",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(FeedType, "not found"),
				ExpectUnauthorized: true,
			},
			{
				Notes:         "global admins are manager for all feeds",
				Subject:       newEntityKey(UserType, "global_admin"),
				Object:        newEntityKey(FeedType, "BA"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion, CanSetGroup},
			},
			{
				Notes:       "global admins get not found for feed that does not exist",
				Subject:     newEntityKey(UserType, "global_admin"),
				Object:      newEntityKey(FeedType, "not found"),
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				ret, err := checker.FeedPermissions(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedRequest{Id: ltk.Object.ID()},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
				if err != nil {
					return
				}
				checkActionSubset(t, ret.Actions, tc.ExpectActions)
			})
		}
	})

	t.Run("FeedSetGroup", func(t *testing.T) {
		tcs := []testCase{
			// User checks
			// TODO!!
			// {
			// 	Notes:       "user drew is a manager of feed CT and can assign it to a different group",
			// 	Subject:     newEntityKey(FeedType, "CT"),
			// 	Object:      newEntityKey(GroupType, "test-group"),
			// 	CheckAsUser: "drew",
			// },
			{
				Notes:              "user ian is an editor of group BA and is not authorized to assign to a different group",
				Subject:            newEntityKey(FeedType, "BA"),
				Object:             newEntityKey(GroupType, "test-group"),
				CheckAsUser:        "ian",
				ExpectUnauthorized: true,
			},
			{
				Notes:              "user drew is not authorized to assign feed EX to a group",
				Subject:            newEntityKey(FeedType, "EX"),
				Object:             newEntityKey(GroupType, "EX-group"),
				CheckAsUser:        "drew",
				ExpectUnauthorized: true,
			},
			// General checks
			{
				Notes:       "user global_admin is a global admin and can assign a feed to a group",
				Subject:     newEntityKey(FeedType, "BA"),
				Object:      newEntityKey(GroupType, "CT-group"),
				CheckAsUser: "global_admin",
			},
		}
		for _, tc := range tcs {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.FeedSetGroup(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedSetGroupRequest{Id: ltk.Subject.ID(), GroupId: ltk.Object.ID()},
				)
				if checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized) {
					return
				}
				// Verify write
				fr, err := checker.FeedPermissions(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedRequest{Id: ltk.Subject.ID()},
				)
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.Object.Name, fr.Group.Name)
			})
		}
	})

	// FEED VERSIONS
	t.Run("FeedVersionList", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		// Only user:tl-tenant-member has permissions explicitly defined
		checks := []testCase{
			// User checks
			{
				Notes:      "tl-tenant-admin has no explicit feed versions but can access d281 through tenant#member",
				Subject:    newEntityKey(UserType, "tl-tenant-admin"),
				Object:     newEntityKey(FeedVersionType, ""),
				ExpectKeys: newEntityKeys(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "ian has no explicit feed versions but can access d281 through tenant#member",
				Subject:    newEntityKey(UserType, "ian"),
				Object:     newEntityKey(FeedVersionType, ""),
				ExpectKeys: newEntityKeys(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "drew has no explicit feed versions but can access d281 through tenant#member",
				Subject:    newEntityKey(UserType, "drew"),
				Object:     newEntityKey(FeedVersionType, ""),
				ExpectKeys: newEntityKeys(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "tl-tenant-admin has explicit access to e535 and can access d281 through tenant#member",
				Subject:    newEntityKey(UserType, "tl-tenant-member"),
				Object:     newEntityKey(FeedVersionType, ""),
				ExpectKeys: newEntityKeys(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			// General checks
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ret, err := checker.FeedVersionList(
					newUserCtx(tc.CheckAsUser, tc.Subject.Name),
					&authz.FeedVersionListRequest{},
				)
				if err != nil {
					t.Fatal(err)
				}
				var gotNames []string
				for _, v := range ret.FeedVersions {
					gotNames = append(gotNames, v.Sha1)
				}
				var expectNames []string
				for _, v := range tc.ExpectKeys {
					expectNames = append(expectNames, v.Name)
				}
				assert.ElementsMatch(t, expectNames, gotNames, "feed version names")
			})
		}
	})

	t.Run("FeedVersionPermissions", func(t *testing.T) {
		checker := newTestChecker(t, fgaUrl, checkerTestData)
		checks := []testCase{
			// User checks
			{
				Notes:         "tl-tenant-admin is a editor of e535 through tenant",
				Subject:       newEntityKey(UserType, "tl-tenant-admin"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers},
			},
			{
				Notes:         "user ian is an editor of e535 through feed BA through group BA-group",
				Subject:       newEntityKey(UserType, "ian"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, CanEdit},
			},
			{
				Notes:              "drew is not authorized to read e535",
				Subject:            newEntityKey(UserType, "drew"),
				Object:             newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions:      []Action{-CanView},
				ExpectUnauthorized: true,
			},
			{
				Notes:         "tl-tenant-member is directly granted viewer on e535",
				Subject:       newEntityKey(UserType, "tl-tenant-member"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Notes:         "user test-group-viewer is viewer on e535 through grant to test-group#viewer",
				Subject:       newEntityKey(UserType, "test-group-viewer"),
				Object:        newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers},
			},
			// General checks
			{
				Notes:              "users get unauthorized for feed versions that do not exist",
				Subject:            newEntityKey(UserType, "test-group-viewer"),
				Object:             newEntityKey(FeedVersionType, "not found"),
				ExpectUnauthorized: true,
			},
			{
				Notes:       "global admins get error for feed versions that do not exist",
				Subject:     newEntityKey(UserType, "global_admin"),
				Object:      newEntityKey(FeedVersionType, "not found"),
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				ret, err := checker.FeedVersionPermissions(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedVersionRequest{Id: ltk.Object.ID()},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
				if err != nil {
					return
				}
				checkActionSubset(t, ret.Actions, tc.ExpectActions)
			})
		}
	})

	t.Run("FeedVersionAddPermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:       "user tl-tenant-admin is a manager of e535 through tenant#admin and can edit users",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is editor of e535 through BA and BA-group and can not edit users",
				Subject:            newEntityKey(UserType, "test3"),
				Object:             newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:           ViewerRelation,
				CheckAsUser:        "ian",
				ExpectUnauthorized: true,
			},
			// General checks
			{
				Notes:       "existing tuple will still remove other subject matched tuples",
				Subject:     newEntityKey(UserType, "tl-tenant-member"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "invalid relation returns error",
				Subject:     newEntityKey(UserType, "tl-tenant-member"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ParentRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "disallowed relation returns error",
				Subject:     newEntityKey(GroupType, "BA-group#editor"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				ExpectError: true,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:       "global admins get error when editing feed version that does not exist",
				Subject:     newEntityKey(UserType, "test3"),
				Object:      newEntityKey(FeedVersionType, "not found"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
				ExpectError: true,
			},
			{
				Notes:       "global admins can edit users of any feed version",
				Subject:     newEntityKey(UserType, "new-user"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.FeedVersionAddPermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedVersionModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

	t.Run("FeedVersionRemovePermission", func(t *testing.T) {
		checks := []testCase{
			// User checks
			{
				Notes:       "user tl-tenant-admin is a manager of feed version e535 through tenant#admin and can edit permissions",
				Subject:     newEntityKey(UserType, "tl-tenant-member"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				CheckAsUser: "tl-tenant-admin",
			},
			{
				Notes:              "user ian is not a manager of feed version e535",
				Subject:            newEntityKey(UserType, "ian"),
				Object:             newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:           ViewerRelation,
				ExpectUnauthorized: true,
				CheckAsUser:        "ian",
			},
			// General checks
			{
				Notes:       "global admins get not found when editing permissions of a feed version that does not exist",
				Subject:     newEntityKey(UserType, "ian"),
				Object:      newEntityKey(FeedVersionType, "not found"),
				Relation:    ViewerRelation,
				ExpectError: true,
				CheckAsUser: "global_admin",
			},
			{
				Notes:       "global admins can edit permissions of any feed version",
				Subject:     newEntityKey(UserType, "tl-tenant-member"),
				Object:      newEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				CheckAsUser: "global_admin",
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test - initialize for each test
				checker := newTestChecker(t, fgaUrl, checkerTestData)
				ltk := dbTupleLookup(t, dbx, tc.TupleKey())
				_, err := checker.FeedVersionRemovePermission(
					newUserCtx(tc.CheckAsUser, ltk.Subject.Name),
					&authz.FeedVersionModifyPermissionRequest{
						Id:             ltk.Object.ID(),
						EntityRelation: authz.NewEntityRelation(ltk.Subject, ltk.Relation),
					},
				)
				checkErrUnauthorized(t, err, tc.ExpectError, tc.ExpectUnauthorized)
			})
		}
	})

}

func stringOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func checkActionSubset(t testing.TB, actions any, checks []Action) {
	checkA, err := actionsToMap(actions)
	if err != nil {
		t.Error(err)
	}
	checkActions := checkActionsToMap(checks)
	checkMapSubset(t, checkA, checkActions)
}

func checkMapSubset(t testing.TB, got map[string]bool, expect map[string]bool) {
	var keys = map[string]bool{}
	for k := range got {
		keys[k] = true
	}
	for k := range expect {
		keys[k] = true
	}
	for k := range keys {
		if got[k] != expect[k] {
			t.Errorf("key %s mismatch, got %t expect %t", k, got[k], expect[k])
		}
	}
}

func actionsToMap(actions any) (map[string]bool, error) {
	jj, err := json.Marshal(actions)
	if err != nil {
		return nil, err
	}
	ret := map[string]bool{}
	if err := json.Unmarshal(jj, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func checkActionsToMap(v []Action) map[string]bool {
	ret := map[string]bool{}
	for _, checkAction := range v {
		expect := true
		if checkAction < 0 {
			expect = false
			checkAction *= -1
		}
		ret[checkAction.String()] = expect
	}
	return ret
}

func newTestChecker(t testing.TB, url string, testData []testCase) *Checker {
	ctx := context.Background()
	dbx := testutil.MustOpenTestDB(t)
	cfg := CheckerConfig{
		FGAEndpoint:      url,
		FGALoadModelFile: testdata.Path("server/authz/tls.json"),
	}

	checker, err := NewCheckerFromConfig(ctx, cfg, dbx)
	if err != nil {
		t.Fatal(err)
	}

	// Add test data
	for _, tc := range testData {
		if err := checker.fgaClient.WriteTuple(ctx, dbTupleLookup(t, dbx, tc.TupleKey())); err != nil {
			t.Fatal(err)
		}
	}

	// Override UserProvider
	userClient := NewMockUserProvider()
	userClient.AddUser("ian", authn.NewCtxUser("ian", "Ian", "ian@example.com"))
	userClient.AddUser("drew", authn.NewCtxUser("drew", "Drew", "drew@example.com"))
	userClient.AddUser("tl-tenant-member", authn.NewCtxUser("tl-tenant-member", "Tenant Member", "tl-tenant-member@example.com"))
	userClient.AddUser("new-user", authn.NewCtxUser("new-user", "Unassigned Member", "new-user@example.com"))
	checker.userClient = userClient
	return checker
}

func checkErrUnauthorized(t testing.TB, err error, expectError bool, expectUnauthorized bool) bool {
	// return true if there was an error
	// log unexpected errors
	if err == nil {
		if expectUnauthorized {
			t.Errorf("expected unauthorized, got no error")
		} else if expectError {
			t.Errorf("expected error, got no error")
		}
	} else {
		if expectUnauthorized && err != ErrUnauthorized {
			t.Errorf("expected unauthorized, got error '%s'", err.Error())
		}
		if !(expectUnauthorized || expectError) {
			t.Errorf("got error '%s', expected no error", err.Error())
		}
	}
	return err != nil
}

func checkExpectError(t testing.TB, err error, expect bool) bool {
	if err != nil && !expect {
		t.Errorf("got error '%s', did not expect error", err.Error())
		return false
	}
	if err == nil && expect {
		t.Errorf("got no error, expected error")
		return false
	}
	if err != nil {
		return false
	}
	return true
}

func newEntityKey(t ObjectType, name string) EntityKey {
	return authz.NewEntityKey(t, name)
}

func newEntityKeys(t ObjectType, keys ...string) []EntityKey {
	var ret []EntityKey
	for _, k := range keys {
		ret = append(ret, newEntityKey(t, k))
	}
	return ret
}

func newUserCtx(first ...string) context.Context {
	for _, f := range first {
		if f != "" {
			return authn.WithUser(context.Background(), authn.NewCtxUser(f, f, f))
		}
	}
	return context.Background()
}
