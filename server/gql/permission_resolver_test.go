package gql

import (
	"net/http"
	"testing"

	"fmt"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/azchecker"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// testTuples defines the FGA authorization tuples loaded for permission resolver tests.
// These reference entities that exist in the test database fixtures (test_supplement.pgsql).
var testTuples = []authz.TupleKey{
	// Assign users to tenant
	{Subject: authz.NewEntityKey(authz.UserType, "tl-tenant-admin"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.AdminRelation},
	{Subject: authz.NewEntityKey(authz.UserType, "ian"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.MemberRelation},
	// Assign groups to tenant
	{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "CT-group"), Relation: authz.ParentRelation},
	{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "BA-group"), Relation: authz.ParentRelation},
	// Assign users to groups
	{Subject: authz.NewEntityKey(authz.UserType, "ian"), Object: authz.NewEntityKey(authz.GroupType, "CT-group"), Relation: authz.ViewerRelation},
	{Subject: authz.NewEntityKey(authz.UserType, "ian"), Object: authz.NewEntityKey(authz.GroupType, "BA-group"), Relation: authz.EditorRelation},
	// Assign feeds to groups
	{Subject: authz.NewEntityKey(authz.GroupType, "CT-group"), Object: authz.NewEntityKey(authz.FeedType, "CT"), Relation: authz.ParentRelation},
	{Subject: authz.NewEntityKey(authz.GroupType, "BA-group"), Object: authz.NewEntityKey(authz.FeedType, "BA"), Relation: authz.ParentRelation},
}

func fgaTestOpts(t testing.TB) testconfig.Options {
	t.Helper()
	return testconfig.Options{
		FGAEndpoint:    testutil.FGAServer(t),
		FGAModelFile:   testdata.Path("server/authz/tls.json"),
		FGAModelTuples: testTuples,
	}
}

// newPermTestClientFromConfig creates a GraphQL test client from an existing config.
func newPermTestClientFromConfig(cfg model.Config, user string, roles ...string) *client.Client {
	srv, _ := NewServer()
	handler := model.AddConfigAndPerms(cfg, srv)
	handler = usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser(user, user, user+"@example.com").WithRoles(roles...)
	})(handler)
	return client.New(handler)
}

// newPermTestClient creates a GraphQL test client with a real azchecker backed by
// in-memory OpenFGA and the test database. The user gets the "admin" role by
// default, matching the existing test convention.
func newPermTestClient(t testing.TB, user string) *client.Client {
	t.Helper()
	cfg := testconfig.Config(t, fgaTestOpts(t))
	return newPermTestClientFromConfig(cfg, user, "admin")
}

func postQuery(t testing.TB, c *client.Client, query string, vars map[string]interface{}) string {
	t.Helper()
	var resp map[string]interface{}
	opts := []client.Option{}
	for k, v := range vars {
		opts = append(opts, client.Var(k, v))
	}
	err := c.Post(query, &resp, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return toJson(resp)
}

func postQueryExpectError(t testing.TB, c *client.Client, query string) {
	t.Helper()
	var resp map[string]interface{}
	err := c.Post(query, &resp)
	assert.Error(t, err)
}

// Tests

func TestPermissionResolver_Tenants(t *testing.T) {
	c := newPermTestClient(t, "tl-tenant-admin")

	t.Run("list tenants", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { id name } }`, nil)
		tenants := gjson.Get(jj, "tenants").Array()
		assert.GreaterOrEqual(t, len(tenants), 1)
		var found bool
		for _, tenant := range tenants {
			if tenant.Get("name").Str == "tl-tenant" {
				found = true
				assert.Greater(t, tenant.Get("id").Int(), int64(0))
			}
		}
		assert.True(t, found, "expected to find tl-tenant")
	})

	t.Run("get tenant by ids", func(t *testing.T) {
		// First get the tenant ID
		jj := postQuery(t, c, `{ tenants { id name } }`, nil)
		var tenantID int64
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str == "tl-tenant" {
				tenantID = tenant.Get("id").Int()
			}
		}
		assert.Greater(t, tenantID, int64(0))

		// Look up by IDs
		jj = postQuery(t, c, `query($ids: [Int!]) { tenants(ids: $ids) { id name } }`, map[string]interface{}{"ids": []int64{tenantID}})
		tenants := gjson.Get(jj, "tenants").Array()
		assert.Equal(t, 1, len(tenants))
		assert.Equal(t, tenantID, tenants[0].Get("id").Int())
		assert.Equal(t, "tl-tenant", tenants[0].Get("name").Str)
	})

	t.Run("get tenant by ids not found", func(t *testing.T) {
		jj := postQuery(t, c, `query($ids: [Int!]) { tenants(ids: $ids) { id name } }`, map[string]interface{}{"ids": []int{999999}})
		tenants := gjson.Get(jj, "tenants").Array()
		assert.Equal(t, 0, len(tenants))
	})

	t.Run("tenant permissions", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { name permissions { actions subjects { type id name relation } children { type id name } } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			perms := tenant.Get("permissions")

			var actionStrs []string
			for _, a := range perms.Get("actions").Array() {
				actionStrs = append(actionStrs, a.Str)
			}
			assert.Contains(t, actionStrs, "can_view")
			assert.Contains(t, actionStrs, "can_edit")

			subjects := perms.Get("subjects").Array()
			assert.GreaterOrEqual(t, len(subjects), 1)
			var foundAdmin bool
			for _, s := range subjects {
				if s.Get("id").Str == "tl-tenant-admin" && s.Get("relation").Str == "admin" {
					foundAdmin = true
					assert.Equal(t, "user", s.Get("type").Str)
				}
			}
			assert.True(t, foundAdmin, "expected tl-tenant-admin as admin subject")

			children := perms.Get("children").Array()
			assert.GreaterOrEqual(t, len(children), 2)
			for _, child := range children {
				assert.Equal(t, "group", child.Get("type").Str)
			}
			return
		}
		t.Fatal("tl-tenant not found in response")
	})

	t.Run("tenant groups", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { name groups { id name } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			groups := tenant.Get("groups").Array()
			assert.GreaterOrEqual(t, len(groups), 2)
			var groupNames []string
			for _, g := range groups {
				groupNames = append(groupNames, g.Get("name").Str)
			}
			assert.Contains(t, groupNames, "CT-group")
			assert.Contains(t, groupNames, "BA-group")
			return
		}
		t.Fatal("tl-tenant not found in response")
	})
}

func TestPermissionResolver_Groups(t *testing.T) {
	c := newPermTestClient(t, "ian")

	t.Run("list groups", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { id name } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.GreaterOrEqual(t, len(groups), 2)
		var groupNames []string
		for _, g := range groups {
			groupNames = append(groupNames, g.Get("name").Str)
		}
		assert.Contains(t, groupNames, "CT-group")
		assert.Contains(t, groupNames, "BA-group")
	})

	t.Run("get group by ids", func(t *testing.T) {
		// First get the group ID
		jj := postQuery(t, c, `{ groups { id name } }`, nil)
		var groupID int64
		for _, g := range gjson.Get(jj, "groups").Array() {
			if g.Get("name").Str == "CT-group" {
				groupID = g.Get("id").Int()
			}
		}
		assert.Greater(t, groupID, int64(0))

		// Look up by IDs
		jj = postQuery(t, c, `query($ids: [Int!]) { groups(ids: $ids) { id name } }`, map[string]interface{}{"ids": []int64{groupID}})
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 1, len(groups))
		assert.Equal(t, groupID, groups[0].Get("id").Int())
		assert.Equal(t, "CT-group", groups[0].Get("name").Str)
	})

	t.Run("get group by ids not found", func(t *testing.T) {
		jj := postQuery(t, c, `query($ids: [Int!]) { groups(ids: $ids) { id name } }`, map[string]interface{}{"ids": []int{999999}})
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 0, len(groups))
	})

	t.Run("group tenant", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { name tenant { id name } } }`, nil)
		for _, group := range gjson.Get(jj, "groups").Array() {
			if group.Get("name").Str != "CT-group" {
				continue
			}
			tenant := group.Get("tenant")
			assert.Equal(t, "tl-tenant", tenant.Get("name").Str)
			assert.Greater(t, tenant.Get("id").Int(), int64(0))
			return
		}
		t.Fatal("CT-group not found in response")
	})

	t.Run("group permissions", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { name permissions { actions parent { type id name } } } }`, nil)
		for _, group := range gjson.Get(jj, "groups").Array() {
			if group.Get("name").Str != "CT-group" {
				continue
			}
			perms := group.Get("permissions")
			actions := perms.Get("actions").Array()
			assert.GreaterOrEqual(t, len(actions), 1)
			assert.Equal(t, "tenant", perms.Get("parent.type").Str)
			assert.Equal(t, "tl-tenant", perms.Get("parent.name").Str)
			return
		}
		t.Fatal("CT-group not found in response")
	})

	t.Run("group feeds", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { name feeds { id onestop_id } } }`, nil)
		for _, group := range gjson.Get(jj, "groups").Array() {
			if group.Get("name").Str != "CT-group" {
				continue
			}
			feeds := group.Get("feeds").Array()
			assert.Equal(t, 1, len(feeds))
			assert.Equal(t, "CT", feeds[0].Get("onestop_id").Str)
			return
		}
		t.Fatal("CT-group not found in response")
	})
}

func TestPermissionResolver_FeedPermissions(t *testing.T) {
	c := newPermTestClient(t, "ian")

	t.Run("feed with permissions", func(t *testing.T) {
		jj := postQuery(t, c, `{ feeds(where:{onestop_id:"CT"}) { onestop_id permissions { actions parent { type name } } } }`, nil)
		feeds := gjson.Get(jj, "feeds").Array()
		assert.Equal(t, 1, len(feeds))
		perms := feeds[0].Get("permissions")
		assert.True(t, perms.Exists(), "expected permissions to be present")
		actions := perms.Get("actions").Array()
		assert.GreaterOrEqual(t, len(actions), 1)
		assert.Equal(t, "group", perms.Get("parent.type").Str)
		assert.Equal(t, "CT-group", perms.Get("parent.name").Str)
	})
}

func TestPermissionResolver_FeedVersionPermissions(t *testing.T) {
	c := newPermTestClient(t, "ian")

	t.Run("feed version with permissions", func(t *testing.T) {
		// Query a feed version belonging to CT (which has a parent group).
		// The checker injects a contextual feed→feed_version parent tuple
		// for action checks, so the feed version inherits actions from the feed.
		jj := postQuery(t, c, `{ feed_versions(where:{feed_onestop_id:"CT"}) { sha1 permissions { actions } } }`, nil)
		fvs := gjson.Get(jj, "feed_versions").Array()
		assert.GreaterOrEqual(t, len(fvs), 1)
		perms := fvs[0].Get("permissions")
		assert.True(t, perms.Exists(), "expected permissions to be present")
		actions := perms.Get("actions").Array()
		assert.GreaterOrEqual(t, len(actions), 1)
		var actionStrs []string
		for _, a := range actions {
			actionStrs = append(actionStrs, a.Str)
		}
		assert.Contains(t, actionStrs, "can_view")
	})

	t.Run("feed version without permissions returns null", func(t *testing.T) {
		// Without a PermissionManager, permissions should be null
		cfg := testconfig.Config(t, testconfig.Options{})
		noAuthClient := newPermTestClientFromConfig(cfg, "testuser", "testrole")
		jj := postQuery(t, noAuthClient, `{ feed_versions(where:{feed_onestop_id:"CT"}) { sha1 permissions { actions } } }`, nil)
		fvs := gjson.Get(jj, "feed_versions").Array()
		assert.GreaterOrEqual(t, len(fvs), 1)
		p := fvs[0].Get("permissions")
		assert.True(t, p.Type == gjson.Null || !p.Exists(), "expected permissions to be null")
	})
}

func TestPermissionResolver_Mutations(t *testing.T) {
	c := newPermTestClient(t, "tl-tenant-admin")

	// Look up the tl-tenant ID for use in mutations
	jj := postQuery(t, c, `{ tenants { id name } }`, nil)
	var tenantID int64
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str == "tl-tenant" {
			tenantID = tenant.Get("id").Int()
		}
	}
	assert.Greater(t, tenantID, int64(0), "expected to find tl-tenant ID")

	t.Run("permission_add", func(t *testing.T) {
		jj := postQuery(t, c, `mutation($id: Int!) {
			permission_add(type: "tenant", id: $id, input: {subject_type: "user", subject_id: "newuser", relation: "member"})
		}`, map[string]interface{}{"id": tenantID})
		assert.Equal(t, true, gjson.Get(jj, "permission_add").Bool())
	})

	t.Run("permission_remove", func(t *testing.T) {
		postQuery(t, c, `mutation($id: Int!) {
			permission_add(type: "tenant", id: $id, input: {subject_type: "user", subject_id: "tempuser", relation: "member"})
		}`, map[string]interface{}{"id": tenantID})

		jj := postQuery(t, c, `mutation($id: Int!) {
			permission_remove(type: "tenant", id: $id, input: {subject_type: "user", subject_id: "tempuser", relation: "member"})
		}`, map[string]interface{}{"id": tenantID})
		assert.Equal(t, true, gjson.Get(jj, "permission_remove").Bool())
	})

	t.Run("permission_set_parent", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { id name } }`, nil)
		var groupID int64
		for _, g := range gjson.Get(jj, "groups").Array() {
			if g.Get("name").Str == "CT-group" {
				groupID = g.Get("id").Int()
			}
		}
		assert.Greater(t, groupID, int64(0))

		// Use "group" (the display alias) instead of "org" to verify the alias works
		jj = postQuery(t, c, `mutation($groupId: Int!, $tenantId: Int!) {
			permission_set_parent(type: "group", id: $groupId, input: {parent_type: "tenant", parent_id: $tenantId})
		}`, map[string]interface{}{"groupId": groupID, "tenantId": tenantID})
		assert.Equal(t, true, gjson.Get(jj, "permission_set_parent").Bool())
	})

	t.Run("invalid type", func(t *testing.T) {
		postQueryExpectError(t, c, `mutation {
			permission_add(type: "bogus", id: 1, input: {subject_type: "user", subject_id: "ian", relation: "admin"})
		}`)
	})

	t.Run("invalid relation", func(t *testing.T) {
		postQueryExpectError(t, c, `mutation {
			permission_add(type: "tenant", id: 1, input: {subject_type: "user", subject_id: "ian", relation: "bogus"})
		}`)
	})
}

func TestPermissionResolver_AdminMutations(t *testing.T) {
	// Admin mutations modify DB rows, so run inside a rollback transaction
	testconfig.ConfigTxRollback(t, fgaTestOpts(t), func(cfg model.Config) {
		c := newPermTestClientFromConfig(cfg, "tl-tenant-admin")

		// Look up the tl-tenant ID
		jj := postQuery(t, c, `{ tenants { id name } }`, nil)
		var tenantID int64
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str == "tl-tenant" {
				tenantID = tenant.Get("id").Int()
			}
		}
		assert.Greater(t, tenantID, int64(0))

		t.Run("tenant_save", func(t *testing.T) {
			jj := postQuery(t, c, `mutation($id: Int!) {
				tenant_save(id: $id, input: {name: "tl-tenant-renamed"}) { id name }
			}`, map[string]interface{}{"id": tenantID})
			assert.Equal(t, tenantID, gjson.Get(jj, "tenant_save.id").Int())
			assert.Equal(t, "tl-tenant-renamed", gjson.Get(jj, "tenant_save.name").Str)
		})

		t.Run("tenant_create_group", func(t *testing.T) {
			jj := postQuery(t, c, `mutation($id: Int!) {
				tenant_create_group(id: $id, input: {name: "new-test-group"}) { id name }
			}`, map[string]interface{}{"id": tenantID})
			assert.Greater(t, gjson.Get(jj, "tenant_create_group.id").Int(), int64(0))
			assert.Equal(t, "new-test-group", gjson.Get(jj, "tenant_create_group.name").Str)
		})

		t.Run("group_save", func(t *testing.T) {
			jj := postQuery(t, c, `{ groups { id name } }`, nil)
			var groupID int64
			for _, g := range gjson.Get(jj, "groups").Array() {
				if g.Get("name").Str == "CT-group" {
					groupID = g.Get("id").Int()
				}
			}
			assert.Greater(t, groupID, int64(0))

			jj = postQuery(t, c, `mutation($id: Int!) {
				group_save(id: $id, input: {name: "CT-group-renamed"}) { id name }
			}`, map[string]interface{}{"id": groupID})
			assert.Equal(t, groupID, gjson.Get(jj, "group_save.id").Int())
			assert.Equal(t, "CT-group-renamed", gjson.Get(jj, "group_save.name").Str)
		})
	})
}

func TestPermissionResolver_Users(t *testing.T) {
	// Build a checker with seeded users
	mockUsers := azchecker.NewMockUserProvider()
	mockUsers.AddUser("alice", authn.NewCtxUser("alice", "Alice Smith", "alice@example.com"))
	mockUsers.AddUser("bob", authn.NewCtxUser("bob", "Bob Jones", "bob@example.com"))
	mockUsers.AddUser("charlie", authn.NewCtxUser("charlie", "Charlie Brown", "charlie@example.com"))
	checker := azchecker.NewChecker(mockUsers, azchecker.NewMockFGAClient(), nil)
	cfg := testconfig.Config(t, testconfig.Options{})
	cfg.Checker = checker
	c := newPermTestClientFromConfig(cfg, "tl-tenant-admin", "admin")

	t.Run("list all users", func(t *testing.T) {
		jj := postQuery(t, c, `{ users { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 3, len(users))
	})

	t.Run("search users with q", func(t *testing.T) {
		jj := postQuery(t, c, `{ users(where:{q:"alice"}) { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 1, len(users))
		assert.Equal(t, "alice", users[0].Get("id").Str)
		assert.Equal(t, "Alice Smith", users[0].Get("name").Str)
		assert.Equal(t, "alice@example.com", users[0].Get("email").Str)
	})

	t.Run("search users with q no match", func(t *testing.T) {
		jj := postQuery(t, c, `{ users(where:{q:"nobody"}) { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 0, len(users))
	})

	t.Run("list users with limit", func(t *testing.T) {
		jj := postQuery(t, c, `{ users(limit: 2) { id } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 2, len(users))
	})

	t.Run("get user by id", func(t *testing.T) {
		jj := postQuery(t, c, `{ users(where:{id:"bob"}) { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 1, len(users))
		assert.Equal(t, "bob", users[0].Get("id").Str)
		assert.Equal(t, "Bob Jones", users[0].Get("name").Str)
		assert.Equal(t, "bob@example.com", users[0].Get("email").Str)
	})

	t.Run("get user by id not found", func(t *testing.T) {
		jj := postQuery(t, c, `{ users(where:{id:"nonexistent"}) { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 0, len(users))
	})

	t.Run("users returns empty without admin manager", func(t *testing.T) {
		noAuthCfg := testconfig.Config(t, testconfig.Options{})
		noAuthClient := newPermTestClientFromConfig(noAuthCfg, "testuser", "testrole")
		jj := postQuery(t, noAuthClient, `{ users { id name email } }`, nil)
		users := gjson.Get(jj, "users").Array()
		assert.Equal(t, 0, len(users))
	})
}

// TestPermissionResolver_Filtering verifies that traversing relationships
// (tenant→groups, group→tenant→groups, permissions→children, group→feeds)
// never widens the result set beyond what the querying user is authorized to see.
func TestPermissionResolver_Filtering(t *testing.T) {
	// Hierarchy:
	//   tl-tenant
	//     ├── CT-group  → feed CT (public)
	//     ├── BA-group  → feed BA (public)
	//     └── HA-group  → feed HA (public), feed EG (non-public)
	//   restricted-tenant
	//     └── EX-group  (no feeds)
	//
	// Users:
	//   "full-user"    : admin of tl-tenant → sees all groups/feeds under tl-tenant
	//   "partial-user" : viewer of CT-group only → sees only CT-group
	//   "multi-user"   : viewer of CT-group + editor of EX-group → spans two tenants
	//   "nobody"       : no tuples → sees nothing
	filterTuples := []authz.TupleKey{
		// tl-tenant groups
		{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "CT-group"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "BA-group"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Object: authz.NewEntityKey(authz.GroupType, "HA-group"), Relation: authz.ParentRelation},
		// restricted-tenant groups
		{Subject: authz.NewEntityKey(authz.TenantType, "restricted-tenant"), Object: authz.NewEntityKey(authz.GroupType, "EX-group"), Relation: authz.ParentRelation},
		// Feed assignments
		{Subject: authz.NewEntityKey(authz.GroupType, "CT-group"), Object: authz.NewEntityKey(authz.FeedType, "CT"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.GroupType, "BA-group"), Object: authz.NewEntityKey(authz.FeedType, "BA"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.GroupType, "HA-group"), Object: authz.NewEntityKey(authz.FeedType, "HA"), Relation: authz.ParentRelation},
		{Subject: authz.NewEntityKey(authz.GroupType, "HA-group"), Object: authz.NewEntityKey(authz.FeedType, "EG"), Relation: authz.ParentRelation},
		// full-user: admin of tl-tenant (inherits access to all children)
		{Subject: authz.NewEntityKey(authz.UserType, "full-user"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.AdminRelation},
		// partial-user: viewer of CT-group only, member of tl-tenant
		{Subject: authz.NewEntityKey(authz.UserType, "partial-user"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.MemberRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "partial-user"), Object: authz.NewEntityKey(authz.GroupType, "CT-group"), Relation: authz.ViewerRelation},
		// multi-user: viewer of CT-group + editor of EX-group (spans two tenants)
		{Subject: authz.NewEntityKey(authz.UserType, "multi-user"), Object: authz.NewEntityKey(authz.TenantType, "tl-tenant"), Relation: authz.MemberRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "multi-user"), Object: authz.NewEntityKey(authz.GroupType, "CT-group"), Relation: authz.ViewerRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "multi-user"), Object: authz.NewEntityKey(authz.TenantType, "restricted-tenant"), Relation: authz.MemberRelation},
		{Subject: authz.NewEntityKey(authz.UserType, "multi-user"), Object: authz.NewEntityKey(authz.GroupType, "EX-group"), Relation: authz.EditorRelation},
	}
	opts := testconfig.Options{
		FGAEndpoint:    testutil.FGAServer(t),
		FGAModelFile:   testdata.Path("server/authz/tls.json"),
		FGAModelTuples: filterTuples,
	}
	cfg := testconfig.Config(t, opts)

	// Helper to collect names from a gjson array
	names := func(arr []gjson.Result, key string) []string {
		var out []string
		for _, r := range arr {
			out = append(out, r.Get(key).Str)
		}
		return out
	}

	t.Run("partial-user tenant groups filtered", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ tenants { name groups { name } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			groupNames := names(tenant.Get("groups").Array(), "name")
			assert.Contains(t, groupNames, "CT-group")
			assert.NotContains(t, groupNames, "BA-group", "partial-user should not see BA-group")
			assert.NotContains(t, groupNames, "HA-group", "partial-user should not see HA-group")
			return
		}
		t.Fatal("tl-tenant not found")
	})

	t.Run("partial-user group tenant groups no leak", func(t *testing.T) {
		// Traversing group → tenant → groups must not widen results
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ groups { name tenant { name groups { name } } } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 1, len(groups), "partial-user should see exactly 1 group")
		assert.Equal(t, "CT-group", groups[0].Get("name").Str)
		tenantGroups := names(groups[0].Get("tenant.groups").Array(), "name")
		assert.Contains(t, tenantGroups, "CT-group")
		assert.NotContains(t, tenantGroups, "BA-group", "traversal should not widen to BA-group")
		assert.NotContains(t, tenantGroups, "HA-group", "traversal should not widen to HA-group")
	})

	t.Run("partial-user permissions children filtered", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ tenants { name permissions { children { type name } } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			childNames := names(tenant.Get("permissions.children").Array(), "name")
			assert.Contains(t, childNames, "CT-group")
			assert.NotContains(t, childNames, "BA-group", "permissions children should not include BA-group")
			assert.NotContains(t, childNames, "HA-group", "permissions children should not include HA-group")
			return
		}
		t.Fatal("tl-tenant not found")
	})

	t.Run("partial-user group feeds", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ groups { name feeds { onestop_id } } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 1, len(groups))
		feedIDs := names(groups[0].Get("feeds").Array(), "onestop_id")
		assert.Contains(t, feedIDs, "CT")
		assert.NotContains(t, feedIDs, "BA")
	})

	t.Run("full-user sees all tenant groups", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "full-user")
		jj := postQuery(t, c, `{ tenants { name groups { name } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			groupNames := names(tenant.Get("groups").Array(), "name")
			assert.Contains(t, groupNames, "CT-group")
			assert.Contains(t, groupNames, "BA-group")
			assert.Contains(t, groupNames, "HA-group")
			return
		}
		t.Fatal("tl-tenant not found")
	})

	t.Run("full-user sees non-public feed via group", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "full-user")
		jj := postQuery(t, c, `{ groups { name feeds { onestop_id } } }`, nil)
		for _, group := range gjson.Get(jj, "groups").Array() {
			if group.Get("name").Str != "HA-group" {
				continue
			}
			feedIDs := names(group.Get("feeds").Array(), "onestop_id")
			assert.Contains(t, feedIDs, "HA")
			assert.Contains(t, feedIDs, "EG", "admin should see non-public feed EG via HA-group")
			return
		}
		t.Fatal("HA-group not found")
	})

	t.Run("multi-user cross-tenant isolation", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "multi-user")
		jj := postQuery(t, c, `{ tenants { name groups { name } } }`, nil)
		tenants := gjson.Get(jj, "tenants").Array()
		assert.GreaterOrEqual(t, len(tenants), 2, "multi-user should see two tenants")
		for _, tenant := range tenants {
			groupNames := names(tenant.Get("groups").Array(), "name")
			switch tenant.Get("name").Str {
			case "tl-tenant":
				assert.Contains(t, groupNames, "CT-group")
				assert.NotContains(t, groupNames, "BA-group", "multi-user should not see BA-group")
				assert.NotContains(t, groupNames, "HA-group", "multi-user should not see HA-group")
			case "restricted-tenant":
				assert.Contains(t, groupNames, "EX-group")
			}
		}
	})

	t.Run("multi-user top-level groups", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "multi-user")
		jj := postQuery(t, c, `{ groups { name } }`, nil)
		groupNames := names(gjson.Get(jj, "groups").Array(), "name")
		assert.Contains(t, groupNames, "CT-group")
		assert.Contains(t, groupNames, "EX-group")
		assert.NotContains(t, groupNames, "BA-group")
		assert.NotContains(t, groupNames, "HA-group")
	})

	t.Run("partial-user group permissions children filtered", func(t *testing.T) {
		// Verify resolvePermissions filters children on a non-tenant type (group).
		// CT-group has one feed child (CT); the permissions.children list should
		// contain exactly that feed and nothing from other groups.
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ groups { name permissions { children { type id } } } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 1, len(groups), "partial-user should see exactly 1 group")
		assert.Equal(t, "CT-group", groups[0].Get("name").Str)
		children := groups[0].Get("permissions.children").Array()
		assert.Equal(t, 1, len(children), "CT-group should have exactly 1 visible child feed")
		assert.Equal(t, "feed", children[0].Get("type").Str)
	})

	t.Run("partial-user tenant groups with limit", func(t *testing.T) {
		// Verify that limit is applied after filtering, not before.
		// tl-tenant has 3 groups but partial-user can only see CT-group.
		// With limit=1 the user should still get CT-group (not an empty
		// result from limiting before filtering).
		c := newPermTestClientFromConfig(cfg, "partial-user")
		jj := postQuery(t, c, `{ tenants { name groups(limit: 1) { name } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			groupNames := names(tenant.Get("groups").Array(), "name")
			assert.Equal(t, 1, len(groupNames), "should return exactly 1 group")
			assert.Contains(t, groupNames, "CT-group")
			return
		}
		t.Fatal("tl-tenant not found")
	})

	t.Run("full-user tenant groups with limit", func(t *testing.T) {
		// full-user can see all 3 groups; limit=2 should cap at 2
		c := newPermTestClientFromConfig(cfg, "full-user")
		jj := postQuery(t, c, `{ tenants { name groups(limit: 2) { name } } }`, nil)
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str != "tl-tenant" {
				continue
			}
			groupNames := names(tenant.Get("groups").Array(), "name")
			assert.Equal(t, 2, len(groupNames), "should return exactly 2 groups")
			return
		}
		t.Fatal("tl-tenant not found")
	})

	t.Run("nobody sees nothing", func(t *testing.T) {
		c := newPermTestClientFromConfig(cfg, "nobody")
		jj := postQuery(t, c, `{ tenants { id } }`, nil)
		assert.Equal(t, 0, len(gjson.Get(jj, "tenants").Array()))
		jj = postQuery(t, c, `{ groups { id } }`, nil)
		assert.Equal(t, 0, len(gjson.Get(jj, "groups").Array()))
	})
}

func TestPermissionResolver_NilPermissionManager(t *testing.T) {
	srv, _ := NewServer()
	cfg := testconfig.Config(t, testconfig.Options{})
	handler := model.AddConfigAndPerms(cfg, srv)
	handler = usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})(handler)
	c := client.New(handler.(http.Handler))

	t.Run("tenants returns empty list", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { id } }`, nil)
		tenants := gjson.Get(jj, "tenants").Array()
		assert.Equal(t, 0, len(tenants))
	})

	t.Run("feed permissions returns null", func(t *testing.T) {
		jj := postQuery(t, c, `{ feeds(where:{onestop_id:"CT"}) { onestop_id permissions { actions } } }`, nil)
		feeds := gjson.Get(jj, "feeds").Array()
		assert.GreaterOrEqual(t, len(feeds), 1)
		// permissions should be null when no PermissionManager is configured
		p := feeds[0].Get("permissions")
		assert.True(t, p.Type == gjson.Null || !p.Exists(), "expected permissions to be null")
	})
}

func TestPermissionResolver_UnauthorizedUser(t *testing.T) {
	// Both clients share the same config (same FGA store and DB) so authorization
	// tuples are visible across users. Only the authn user identity differs.
	cfg := testconfig.Config(t, fgaTestOpts(t))
	adminClient := newPermTestClientFromConfig(cfg, "tl-tenant-admin")
	nobodyClient := newPermTestClientFromConfig(cfg, "nobody", "testrole")

	t.Run("tenants returns empty", func(t *testing.T) {
		jj := postQuery(t, nobodyClient, `{ tenants { id name } }`, nil)
		tenants := gjson.Get(jj, "tenants").Array()
		assert.Equal(t, 0, len(tenants))
	})

	t.Run("groups returns empty", func(t *testing.T) {
		jj := postQuery(t, nobodyClient, `{ groups { id name } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 0, len(groups))
	})

	// Look up real IDs via the admin client for use in mutation tests
	jj := postQuery(t, adminClient, `{ tenants { id name } }`, nil)
	var tenantID int64
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str == "tl-tenant" {
			tenantID = tenant.Get("id").Int()
		}
	}
	assert.Greater(t, tenantID, int64(0))

	jj = postQuery(t, adminClient, `{ groups { id name } }`, nil)
	var groupID int64
	for _, g := range gjson.Get(jj, "groups").Array() {
		if g.Get("name").Str == "CT-group" {
			groupID = g.Get("id").Int()
		}
	}
	assert.Greater(t, groupID, int64(0))

	t.Run("permission_add unauthorized", func(t *testing.T) {
		postQueryExpectError(t, nobodyClient, fmt.Sprintf(`mutation {
			permission_add(type: "tenant", id: %d, input: {subject_type: "user", subject_id: "someone", relation: "member"})
		}`, tenantID))
	})

	t.Run("tenant_save unauthorized", func(t *testing.T) {
		postQueryExpectError(t, nobodyClient, fmt.Sprintf(`mutation {
			tenant_save(id: %d, input: {name: "hacked"}) { id }
		}`, tenantID))
	})

	t.Run("tenant_create_group unauthorized", func(t *testing.T) {
		postQueryExpectError(t, nobodyClient, fmt.Sprintf(`mutation {
			tenant_create_group(id: %d, input: {name: "hacked-group"}) { id }
		}`, tenantID))
	})

	t.Run("group_save unauthorized", func(t *testing.T) {
		postQueryExpectError(t, nobodyClient, fmt.Sprintf(`mutation {
			group_save(id: %d, input: {name: "hacked-group"}) { id }
		}`, groupID))
	})
}

func TestPermissionResolver_MutationRoundTrip(t *testing.T) {
	// Verify that permission_add actually creates a visible permission,
	// and permission_remove actually deletes it.
	cfg := testconfig.Config(t, fgaTestOpts(t))
	c := newPermTestClientFromConfig(cfg, "tl-tenant-admin", "admin")

	// Look up tenant ID
	jj := postQuery(t, c, `{ tenants { id name } }`, nil)
	var tenantID int64
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str == "tl-tenant" {
			tenantID = tenant.Get("id").Int()
		}
	}
	assert.Greater(t, tenantID, int64(0))

	// Verify "roundtrip-user" is not a subject before we add them
	jj = postQuery(t, c, `{ tenants { id name permissions { subjects { id relation } } } }`, nil)
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str != "tl-tenant" {
			continue
		}
		for _, s := range tenant.Get("permissions.subjects").Array() {
			assert.NotEqual(t, "roundtrip-user", s.Get("id").Str, "roundtrip-user should not exist before add")
		}
	}

	// Add the permission
	postQuery(t, c, fmt.Sprintf(`mutation {
		permission_add(type: "tenant", id: %d, input: {subject_type: "user", subject_id: "roundtrip-user", relation: "member"})
	}`, tenantID), nil)

	// Verify "roundtrip-user" now appears as a subject
	jj = postQuery(t, c, `{ tenants { name permissions { subjects { id relation } } } }`, nil)
	var foundAfterAdd bool
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str != "tl-tenant" {
			continue
		}
		for _, s := range tenant.Get("permissions.subjects").Array() {
			if s.Get("id").Str == "roundtrip-user" && s.Get("relation").Str == "member" {
				foundAfterAdd = true
			}
		}
	}
	assert.True(t, foundAfterAdd, "roundtrip-user should appear as member after permission_add")

	// Remove the permission
	postQuery(t, c, fmt.Sprintf(`mutation {
		permission_remove(type: "tenant", id: %d, input: {subject_type: "user", subject_id: "roundtrip-user", relation: "member"})
	}`, tenantID), nil)

	// Verify "roundtrip-user" is gone
	jj = postQuery(t, c, `{ tenants { name permissions { subjects { id relation } } } }`, nil)
	var foundAfterRemove bool
	for _, tenant := range gjson.Get(jj, "tenants").Array() {
		if tenant.Get("name").Str != "tl-tenant" {
			continue
		}
		for _, s := range tenant.Get("permissions.subjects").Array() {
			if s.Get("id").Str == "roundtrip-user" {
				foundAfterRemove = true
			}
		}
	}
	assert.False(t, foundAfterRemove, "roundtrip-user should be gone after permission_remove")
}
