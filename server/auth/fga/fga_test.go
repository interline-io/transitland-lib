package fga

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
)

// For less typing

type Action = authz.Action
type ObjectType = authz.ObjectType
type Relation = authz.Relation

var FeedType = authz.FeedType
var UserType = authz.UserType
var TenantType = authz.TenantType
var GroupType = authz.GroupType
var FeedVersionType = authz.FeedVersionType

var ViewerRelation = authz.ViewerRelation
var MemberRelation = authz.MemberRelation
var AdminRelation = authz.AdminRelation
var ManagerRelation = authz.ManagerRelation
var ParentRelation = authz.ParentRelation
var EditorRelation = authz.EditorRelation

var CanEdit = authz.CanEdit
var CanView = authz.CanView
var CanCreateFeedVersion = authz.CanCreateFeedVersion
var CanDeleteFeedVersion = authz.CanDeleteFeedVersion
var CanCreateFeed = authz.CanCreateFeed
var CanDeleteFeed = authz.CanDeleteFeed
var CanSetGroup = authz.CanSetGroup
var CanCreateOrg = authz.CanCreateOrg
var CanEditMembers = authz.CanEditMembers
var CanDeleteOrg = authz.CanDeleteOrg
var CanSetTenant = authz.CanSetTenant

// Tests

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

func (tk *testCase) TupleKey() authz.TupleKey {
	return authz.TupleKey{Subject: tk.Subject, Object: tk.Object, Relation: tk.Relation, Action: tk.Action}
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

func TestFGAClient(t *testing.T) {
	fgaUrl, a, ok := testutil.CheckEnv("TL_TEST_FGA_ENDPOINT")
	if !ok {
		t.Skip(a)
		return
	}

	testData := []testCase{
		// Assign users to tenants
		{
			Notes:    "All users can access all-users-tenant",
			Subject:  authz.NewEntityKey(UserType, "*"),
			Object:   authz.NewEntityKey(TenantType, "all-users-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "tl-tenant-admin"),
			Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
			Relation: AdminRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "ian"),
			Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "drew"),
			Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "tl-tenant-member"),
			Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
			Relation: MemberRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "test2"),
			Object:   authz.NewEntityKey(TenantType, "restricted-tenant"),
			Relation: MemberRelation,
		},
		// Assign groups to tenants
		{
			Notes:    "org:CT-group belongs to tenant:tl-tenant",
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant"),
			Object:   authz.NewEntityKey(GroupType, "CT-group"),
			Relation: ParentRelation,
		},
		{
			Notes:    "org:BA-group belongs to tenant:tl-tenant",
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant"),
			Object:   authz.NewEntityKey(GroupType, "BA-group"),
			Relation: ParentRelation,
		},
		{
			Notes:    "org:HA-group belongs to tenant:tl-tenant",
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant"),
			Object:   authz.NewEntityKey(GroupType, "HA-group"),
			Relation: ParentRelation,
		},
		{
			Notes:    "org:EX-group will be for admins only",
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant"),
			Object:   authz.NewEntityKey(GroupType, "EX-group"),
			Relation: ParentRelation,
		},
		{
			Notes:    "all tl-tenant members can view HA-group",
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant#member"),
			Object:   authz.NewEntityKey(GroupType, "HA-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  authz.NewEntityKey(TenantType, "restricted-tenant"),
			Object:   authz.NewEntityKey(GroupType, "test-group"),
			Relation: ParentRelation,
		},
		// Assign users to groups
		{
			Subject:  authz.NewEntityKey(UserType, "ian"),
			Object:   authz.NewEntityKey(GroupType, "CT-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "ian"),
			Object:   authz.NewEntityKey(GroupType, "BA-group"),
			Relation: EditorRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "drew"),
			Object:   authz.NewEntityKey(GroupType, "CT-group"),
			Relation: EditorRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "test-group-viewer"),
			Object:   authz.NewEntityKey(GroupType, "test-group"),
			Relation: ViewerRelation,
		},
		{
			Subject:  authz.NewEntityKey(UserType, "test-group-editor"),
			Object:   authz.NewEntityKey(GroupType, "test-group"),
			Relation: EditorRelation,
		},
		// Assign feeds to groups
		{
			Subject:  authz.NewEntityKey(GroupType, "CT-group"),
			Object:   authz.NewEntityKey(FeedType, "CT"),
			Relation: ParentRelation,
			Notes:    "feed:CT should be viewable to members of org:CT-group (ian drew) and editable by org:CT-group editors (drew)",
		},
		{
			Subject:  authz.NewEntityKey(GroupType, "BA-group"),
			Object:   authz.NewEntityKey(FeedType, "BA"),
			Relation: ParentRelation,
			Notes:    "feed:BA should be viewable to members of org:BA-group () and editable by org:BA-group editors (ian)",
		},
		{
			Subject:  authz.NewEntityKey(GroupType, "HA-group"),
			Object:   authz.NewEntityKey(FeedType, "HA"),
			Relation: ParentRelation,
			Notes:    "feed:HA should be viewable to all members of tenant:tl-tenant",
		},
		{
			Subject:  authz.NewEntityKey(GroupType, "EX-group"),
			Object:   authz.NewEntityKey(FeedType, "EX"),
			Relation: ParentRelation,
			Notes:    "feed:EX should only be viewable to admins of tenant:tl-tenant",
		},
		// Assign feed version specific permissions
		// NOTE: This assignment is necessary for FGA tests
		// This relation is implicit in full Checker tests
		{
			Subject:  authz.NewEntityKey(FeedType, "BA"),
			Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ParentRelation,
		},
		// Assign users to feed versions
		{
			Subject:  authz.NewEntityKey(UserType, "tl-tenant-member"),
			Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ViewerRelation,
		},
		{
			Subject:  authz.NewEntityKey(GroupType, "test-group").WithRefRel(ViewerRelation),
			Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
			Relation: ViewerRelation,
		},
		{
			Subject:  authz.NewEntityKey(TenantType, "tl-tenant").WithRefRel(MemberRelation),
			Object:   authz.NewEntityKey(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			Relation: ViewerRelation,
		},
	}

	t.Run("GetObjectTuples", func(t *testing.T) {
		fgac := newTestFGAClient(t, fgaUrl, testData)
		checks := []testCase{
			{
				Object: authz.NewEntityKey(TenantType, "tl-tenant"),
				Expect: "user:tl-tenant-admin:admin user:ian:member user:drew:member user:tl-tenant-member:member",
			},
			{
				Object: authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Expect: "feed:BA:parent user:tl-tenant-member:viewer org:test-group#viewer:viewer",
			},
			{
				Object: authz.NewEntityKey(FeedType, "CT"),
				Expect: "org:CT-group:parent",
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				tks, err := fgac.GetObjectTuples(context.Background(), tc.TupleKey())
				if err != nil {
					t.Error(err)
				}
				expect := strings.Split(tc.Expect, " ")
				var got []string
				for _, vtk := range tks {
					got = append(got, fmt.Sprintf("%s:%s", vtk.Subject.String(), vtk.Relation))
				}
				assert.ElementsMatch(t, expect, got, "usertype:username:relation does not match")
			})
		}
	})

	t.Run("Check", func(t *testing.T) {
		fgac := newTestFGAClient(t, fgaUrl, testData)
		checks := []testCase{
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(GroupType, "test-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, CanEdit, CanEditMembers, CanCreateOrg, CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:        authz.NewEntityKey(TenantType, "restricted-tenant"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedType, "CT"),
				ExpectActions: []Action{CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedType, "BA"),
				ExpectActions: []Action{CanView, CanEdit, CanCreateFeedVersion, CanDeleteFeedVersion},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedType, "EX"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(FeedType, "test"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanCreateFeedVersion, -CanDeleteFeedVersion},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{CanView, CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(GroupType, "test-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(TenantType, "restricted-tenant"),
				ExpectActions: []Action{-CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "ian"),
				Object:        authz.NewEntityKey(TenantType, "all-users-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(TenantType, "all-users-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{-CanView},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedType, "CT"),
				ExpectActions: []Action{CanView, CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedType, "BA"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedType, "EX"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(FeedType, "test"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(GroupType, "CT-group"),
				ExpectActions: []Action{CanView, CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(GroupType, "BA-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(GroupType, "test-group"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateFeed, -CanDeleteFeed},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "drew"),
				Object:        authz.NewEntityKey(TenantType, "restricted-tenant"),
				ExpectActions: []Action{-CanView, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(TenantType, "restricted-tenant"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedType, "CT"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedType, "BA"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedType, "HA"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedType, "EX"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(FeedType, "test"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(GroupType, "HA-group"),
				ExpectActions: []Action{CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(GroupType, "EX-group"),
				ExpectActions: []Action{-CanView, -CanEdit},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(TenantType, "tl-tenant"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:        authz.NewEntityKey(TenantType, "restricted-tenant"),
				ExpectActions: []Action{-CanView, -CanEdit, -CanEditMembers, -CanCreateOrg, -CanDeleteOrg},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "test-group-viewer"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers},
			},
			{
				Subject:       authz.NewEntityKey(UserType, "test-group-editor"),
				Object:        authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				ExpectActions: []Action{CanView, -CanEdit, -CanEditMembers},
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				for _, checkAction := range tc.ExpectActions {
					expect := true
					if checkAction < 0 {
						expect = false
						checkAction = checkAction * -1
					}
					var err error
					ltk := tc.TupleKey()
					ltk.Action = checkAction
					ok, err := fgac.Check(context.Background(), ltk)
					if err != nil {
						t.Fatal(err)
					}
					if ok && !expect {
						t.Errorf("for %s got %t, expected %t", checkAction.String(), ok, expect)
					}
					if !ok && expect {
						t.Errorf("got %s %t, expected %t", checkAction.String(), ok, expect)
					}
				}
			})
		}
	})

	t.Run("ListObjects", func(t *testing.T) {
		fgac := newTestFGAClient(t, fgaUrl, testData)
		checks := []testCase{
			{
				Notes:      "tl-tenant-admin can access all feeds in tl-tenant",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "BA", "HA", "EX"),
			},
			{
				Notes:      "tl-tenant-admin can edit all feeds in tl-tenant",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanEdit,
				ExpectKeys: newEntityKeys(FeedType, "CT", "BA", "HA", "EX"),
			},
			{
				Notes:      "tl-tenant-admin can view all groups in tl-tenant",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "BA-group", "HA-group", "EX-group"),
			},
			{
				Notes:      "tl-tenant-admin can edit all groups in tl-tenant",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(GroupType, ""),
				Action:     CanEdit,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "BA-group", "HA-group", "EX-group"),
			},
			{
				Notes:      "tl-tenant-admin can view tenants tl-tenant and all-users-tenant",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "tl-tenant-admin can view a feed version that belongs to a feed or group in tl-tenant or d281 which viewable to all tenant members",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-admin"),
				Object:     authz.NewEntityKey(FeedVersionType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "ian can edit feed BA in tl-tenant",
				Subject:    authz.NewEntityKey(UserType, "ian"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanEdit,
				ExpectKeys: newEntityKeys(FeedType, "BA"),
			},
			{
				Notes:      "ian can view feeds CT, BA, HA",
				Subject:    authz.NewEntityKey(UserType, "ian"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "BA", "HA"),
			},
			{
				Notes:      "ian can view groups CT-group BA-group HA-group",
				Subject:    authz.NewEntityKey(UserType, "ian"),
				Object:     authz.NewEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "BA-group", "HA-group"),
			},
			{
				Notes:      "ian can view tenants tl-tenant (member explicitly) and all-users-tenant (user:*)",
				Subject:    authz.NewEntityKey(UserType, "ian"),
				Object:     authz.NewEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "ian can view feed version e535 because of access to feed BA, group BA-group or d281 which is viewable to all tenant members",
				Subject:    authz.NewEntityKey(UserType, "ian"),
				Object:     authz.NewEntityKey(FeedVersionType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "drew can edit feed CT because editor of CT-group",
				Subject:    authz.NewEntityKey(UserType, "drew"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanEdit,
				ExpectKeys: newEntityKeys(FeedType, "CT"),
			},
			{
				Notes:      "drew can view feed CT because editor of CT-group and HA because HA has all tenant members",
				Subject:    authz.NewEntityKey(UserType, "drew"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "CT", "HA"),
			},
			{
				Notes:      "drew can access tl-tenant because member and all-users-tenant because user:*",
				Subject:    authz.NewEntityKey(UserType, "drew"),
				Object:     authz.NewEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
			{
				Notes:      "drew can access group CT-group because member and HA-group through tenant#member",
				Subject:    authz.NewEntityKey(UserType, "drew"),
				Object:     authz.NewEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "CT-group", "HA-group"),
			},
			{
				Notes:      "drew is not explicitly assigned any feed versions but can access d281 because it is viewable to all tenant members",
				Subject:    authz.NewEntityKey(UserType, "drew"),
				Object:     authz.NewEntityKey(FeedVersionType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedVersionType, "d2813c293bcfd7a97dde599527ae6c62c98e66c6"),
			},
			{
				Notes:      "tl-tenant-member can access HA-group through HA-group#viewer:tl-tenant#member",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:     authz.NewEntityKey(GroupType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(GroupType, "HA-group"),
			},
			{
				Notes:      "tl-tenant-member can access feed HA through group:HA-group",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:     authz.NewEntityKey(FeedType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(FeedType, "HA"),
			},
			{
				Notes:      "tl-tenant-member can view tl-tenant through member and all-users-tenant through user:*",
				Subject:    authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:     authz.NewEntityKey(TenantType, ""),
				Action:     CanView,
				ExpectKeys: newEntityKeys(TenantType, "tl-tenant", "all-users-tenant"),
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				ltk := tc.TupleKey()
				ltk.Action = tc.Action
				objs, err := fgac.ListObjects(context.Background(), ltk)
				if err != nil {
					t.Fatal(err)
				}
				var gotNames []string
				for _, v := range objs {
					gotNames = append(gotNames, v.Object.Name)
				}
				var expectNames []string
				for _, ek := range tc.ExpectKeys {
					expectNames = append(expectNames, ek.Name)
				}
				assert.ElementsMatch(t, expectNames, gotNames, "object ids")
			})
		}
	})

	t.Run("WriteTuple", func(t *testing.T) {
		checks := []testCase{
			{
				Notes:    "user:* can be a member of a tenant",
				Subject:  authz.NewEntityKey(UserType, "*"),
				Object:   authz.NewEntityKey(TenantType, "tl-tenants"),
				Relation: MemberRelation,
			},
			{
				Notes:       "user:* cannot be an admin of a tenant",
				Subject:     authz.NewEntityKey(UserType, "*"),
				Object:      authz.NewEntityKey(TenantType, "tl-tenants"),
				Relation:    AdminRelation,
				ExpectError: true,
			},
			{
				Notes:    "a tenant#member can be a viewer of a group",
				Subject:  authz.NewEntityKey(TenantType, "tl-tenant#member"),
				Object:   authz.NewEntityKey(GroupType, "BA-group"),
				Relation: ViewerRelation,
			},
			{
				Notes:       "a tenant#admin cannot be a viewer of a group",
				Subject:     authz.NewEntityKey(TenantType, "tl-tenant#admin"),
				Object:      authz.NewEntityKey(GroupType, "BA-group"),
				Relation:    ViewerRelation,
				ExpectError: true,
			},
			{
				Notes:    "a tenant#member can be an editor of a group",
				Subject:  authz.NewEntityKey(TenantType, "tl-tenant#member"),
				Object:   authz.NewEntityKey(GroupType, "BA-group"),
				Relation: EditorRelation,
				// Formerly disallowed, now OK
				// ExpectError: true,
			},
			{
				Notes:    "user can be a member of a tenant",
				Subject:  authz.NewEntityKey(UserType, "test100"),
				Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
				Relation: MemberRelation,
			},
			{
				Notes:    "user can be an admin of a tenant",
				Subject:  authz.NewEntityKey(UserType, "test100"),
				Object:   authz.NewEntityKey(TenantType, "tl-tenant"),
				Relation: AdminRelation,
			},
			{
				Notes:       "already exists",
				Subject:     authz.NewEntityKey(UserType, "ian"),
				Object:      authz.NewEntityKey(TenantType, "tl-tenant"),
				Relation:    MemberRelation,
				ExpectError: true,
			},
			{
				Notes:    "a user can be a viewer of a group",
				Subject:  authz.NewEntityKey(UserType, "test100"),
				Object:   authz.NewEntityKey(GroupType, "HA-group"),
				Relation: ViewerRelation,
			},
			{
				Notes:    "a user can be an editor of a group",
				Subject:  authz.NewEntityKey(UserType, "test100"),
				Object:   authz.NewEntityKey(GroupType, "HA-group"),
				Relation: EditorRelation,
			},
			{
				Notes:    "a user can be a manager of a group",
				Subject:  authz.NewEntityKey(UserType, "test100"),
				Object:   authz.NewEntityKey(GroupType, "HA-group"),
				Relation: ManagerRelation,
			},
			{
				Notes:       "invalid relation",
				Subject:     authz.NewEntityKey(UserType, "ian"),
				Object:      authz.NewEntityKey(GroupType, "HA-group"),
				Relation:    ParentRelation,
				ExpectError: true,
			},
			{
				Notes:    "a user can be a viewer of a group",
				Subject:  authz.NewEntityKey(UserType, "test102"),
				Object:   authz.NewEntityKey(GroupType, "100"),
				Relation: ViewerRelation,
			},
			{
				Notes:    "a user can be a viewer of a feed version",
				Subject:  authz.NewEntityKey(UserType, "ian"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: ViewerRelation,
			},
			{
				Notes:    "a user can be a editor of a feed version",
				Subject:  authz.NewEntityKey(UserType, "ian"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: EditorRelation,
			},
			{
				Notes:       "already exists",
				Subject:     authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:      authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				ExpectError: true,
			},
			{
				Notes:    "a tenant#member can be a viewer of a feed version",
				Subject:  authz.NewEntityKey(TenantType, "tl-tenant#member"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: ViewerRelation,
			},
			{
				Notes:    "a tenant#member can be an editor of a feed version",
				Subject:  authz.NewEntityKey(TenantType, "tl-tenant#member"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: EditorRelation,
				// Formerly disallowed, now OK
				// ExpectError: true,
			},
			{
				Notes:       "a tenant#admin can be a viewer of a feed version",
				Subject:     authz.NewEntityKey(TenantType, "tl-tenant#admin"),
				Object:      authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				ExpectError: true,
			},
			{
				Notes:    "a group#member can be a viewer of a feed version",
				Subject:  authz.NewEntityKey(TenantType, "HA-group#member"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: ViewerRelation,
			},
			{
				Notes:       "a group#editor cannot be a viewer of a feed version",
				Subject:     authz.NewEntityKey(GroupType, "HA-group#editor"),
				Object:      authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    ViewerRelation,
				ExpectError: true,
			},
			{
				Notes:    "a group#viewer can be an editor of a feed version",
				Subject:  authz.NewEntityKey(GroupType, "HA-group").WithRefRel(ViewerRelation),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: EditorRelation,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating, so create fresh each test
				fgac := newTestFGAClient(t, fgaUrl, testData)
				// Write tuple and check if error was expected
				ltk := tc.TupleKey()
				err := fgac.WriteTuple(context.Background(), ltk)
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				// Check was written
				tks, err := fgac.GetObjectTuples(context.Background(), ltk)
				if err != nil {
					t.Error(err)
				}
				var gotTks []string
				for _, v := range tks {
					gotTks = append(gotTks, fmt.Sprintf("%s:%s", v.Subject.String(), v.Relation))
				}
				checkTk := fmt.Sprintf("%s:%s", ltk.Subject.String(), ltk.Relation)
				assert.Contains(t, gotTks, checkTk, "written tuple not found in updated object tuples")
			})
		}
	})

	t.Run("DeleteTuple", func(t *testing.T) {
		checks := []testCase{
			{
				Subject:  authz.NewEntityKey(UserType, "ian"),
				Object:   authz.NewEntityKey(GroupType, "CT-group"),
				Relation: 4,
			},
			{
				Subject:     authz.NewEntityKey(UserType, "test102"),
				Object:      authz.NewEntityKey(GroupType, "100"),
				Relation:    4,
				Notes:       "unauthorized",
				ExpectError: true,
			},
			{
				Subject:  authz.NewEntityKey(UserType, "tl-tenant-member"),
				Object:   authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation: 4,
			},
			{
				Subject:     authz.NewEntityKey(UserType, "ian"),
				Object:      authz.NewEntityKey(FeedVersionType, "e535eb2b3b9ac3ef15d82c56575e914575e732e0"),
				Relation:    4,
				Notes:       "unauthorized",
				ExpectError: true,
			},
			{
				Subject:  authz.NewEntityKey(UserType, "test2"),
				Object:   authz.NewEntityKey(TenantType, "restricted-tenant"),
				Relation: 2,
			},
			{
				Subject:     authz.NewEntityKey(UserType, "test101"),
				Object:      authz.NewEntityKey(GroupType, "BA-group"),
				Relation:    4,
				Notes:       "does not exist",
				ExpectError: true,
			},
			{
				Subject:     authz.NewEntityKey(UserType, "test101"),
				Object:      authz.NewEntityKey(GroupType, "BA-group"),
				Relation:    4,
				Notes:       "unauthorized",
				ExpectError: true,
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test
				fgac := newTestFGAClient(t, fgaUrl, testData)
				ltk := tc.TupleKey()
				err := fgac.DeleteTuple(context.Background(), ltk)
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
			})
		}
	})

	t.Run("SetExclusiveSubjectRelation", func(t *testing.T) {
		checks := []testCase{
			{
				Notes:    "changes ian permissions from Viewer to Manager",
				Subject:  authz.NewEntityKey(UserType, "ian"),
				Object:   authz.NewEntityKey(GroupType, "CT-group"),
				Relation: ManagerRelation,
				Expect:   "user:ian:manager user:drew:editor",
			},
			{
				Notes:    "changes drew permissions from Editor to Viewer",
				Subject:  authz.NewEntityKey(UserType, "drew"),
				Object:   authz.NewEntityKey(GroupType, "CT-group"),
				Relation: ViewerRelation,
				Expect:   "user:drew:viewer user:ian:viewer",
			},
			{
				Notes:    "assigns ian permissions as Manager, nothing to delete",
				Subject:  authz.NewEntityKey(UserType, "ian"),
				Object:   authz.NewEntityKey(GroupType, "HA-group"),
				Relation: ManagerRelation,
				Expect:   "user:ian:manager tenant:tl-tenant#member:viewer",
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test
				fgac := newTestFGAClient(t, fgaUrl, testData)
				ltk := tc.TupleKey()
				checkRelTypes := []Relation{ViewerRelation, EditorRelation, ManagerRelation}
				err := fgac.SetExclusiveSubjectRelation(context.Background(), ltk, checkRelTypes...)
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				newTks, err := fgac.GetObjectTuples(context.Background(), authz.NewTupleKey().WithObject(ltk.Object.Type, ltk.Object.Name))
				if err != nil {
					t.Error(err)
				}
				expect := strings.Split(tc.Expect, " ")
				var got []string
				for _, vtk := range newTks {
					ok := false
					for _, checkRel := range checkRelTypes {
						if vtk.Relation == checkRel {
							ok = true
						}
					}
					if !ok {
						continue
					}
					got = append(got, fmt.Sprintf("%s:%s", vtk.Subject.String(), vtk.Relation))
				}
				assert.ElementsMatch(t, expect, got, "usertype:username:relation does not match")
			})
		}
	})

	t.Run("SetExclusiveRelation", func(t *testing.T) {
		checks := []testCase{
			{
				Notes:    "changes feed parent",
				Object:   authz.NewEntityKey(FeedType, "CT"),
				Subject:  authz.NewEntityKey(GroupType, "BA-group"),
				Relation: ParentRelation,
				Expect:   "org:BA-group:parent",
			},
			{
				Notes:    "changes group tenant",
				Object:   authz.NewEntityKey(GroupType, "CT-group"),
				Subject:  authz.NewEntityKey(TenantType, "all-users-tenant"),
				Relation: ParentRelation,
				Expect:   "tenant:all-users-tenant:parent",
			},
			{
				Notes:    "assigns group to tenant",
				Object:   authz.NewEntityKey(GroupType, "new-group"),
				Subject:  authz.NewEntityKey(TenantType, "all-users-tenant"),
				Relation: ParentRelation,
				Expect:   "tenant:all-users-tenant:parent",
			},
		}
		for _, tc := range checks {
			t.Run(tc.String(), func(t *testing.T) {
				// Mutating test
				fgac := newTestFGAClient(t, fgaUrl, testData)
				ltk := tc.TupleKey()
				checkRelTypes := []Relation{ParentRelation}
				err := fgac.SetExclusiveRelation(context.Background(), ltk)
				if !checkExpectError(t, err, tc.ExpectError) {
					return
				}
				newTks, err := fgac.GetObjectTuples(context.Background(), authz.NewTupleKey().WithObject(ltk.Object.Type, ltk.Object.Name))
				if err != nil {
					t.Error(err)
				}
				expect := strings.Split(tc.Expect, " ")
				var got []string
				for _, vtk := range newTks {
					ok := false
					for _, checkRel := range checkRelTypes {
						if vtk.Relation == checkRel {
							ok = true
						}
					}
					if !ok {
						continue
					}
					got = append(got, fmt.Sprintf("%s:%s", vtk.Subject.String(), vtk.Relation))
				}
				assert.ElementsMatch(t, expect, got, "usertype:username:relation does not match")
			})
		}
	})

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

func newEntityKeys(t authz.ObjectType, keys ...string) []authz.EntityKey {
	var ret []authz.EntityKey
	for _, k := range keys {
		ret = append(ret, authz.NewEntityKey(t, k))
	}
	return ret
}

func newTestFGAClient(t testing.TB, url string, testTuples []testCase) *FGAClient {
	fgac, err := NewFGAClient(url, "", "")
	if err != nil {
		t.Fatal(err)
		return nil
	}
	if _, err := fgac.CreateStore(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := fgac.CreateModel(context.Background(), testdata.Path("server/authz/tls.json")); err != nil {
		t.Fatal(err)
	}
	for _, tk := range testTuples {
		if err := fgac.WriteTuple(context.Background(), tk.TupleKey()); err != nil {
			t.Fatal(err)
		}
	}
	return fgac
}
