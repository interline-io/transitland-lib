package gql

import (
	"net/http"
	"testing"

	"fmt"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
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

func TestPermissionResolver_NilPermissionManager(t *testing.T) {
	srv, _ := NewServer()
	cfg := testconfig.Config(t, testconfig.Options{})
	handler := model.AddConfigAndPerms(cfg, srv)
	handler = usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})(handler)
	c := client.New(handler.(http.Handler))

	t.Run("tenants returns error", func(t *testing.T) {
		postQueryExpectError(t, c, `{ tenants { id } }`)
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

	t.Run("mutation permission_add unauthorized", func(t *testing.T) {
		// Look up a real tenant ID via the admin client
		jj := postQuery(t, adminClient, `{ tenants { id name } }`, nil)
		var tenantID int64
		for _, tenant := range gjson.Get(jj, "tenants").Array() {
			if tenant.Get("name").Str == "tl-tenant" {
				tenantID = tenant.Get("id").Int()
			}
		}
		assert.Greater(t, tenantID, int64(0))

		// Unauthorized user should be denied
		postQueryExpectError(t, nobodyClient, fmt.Sprintf(`mutation {
			permission_add(type: "tenant", id: %d, input: {subject_type: "user", subject_id: "someone", relation: "member"})
		}`, tenantID))
	})
}
