package gql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// mockPermissionManager implements authz.PermissionManager for testing.
type mockPermissionManager struct {
	authz.GlobalAdminChecker
	objects     map[authz.ObjectType][]authz.ObjectRef
	permissions map[string]*authz.ObjectPermissions
	added       []mockPermCall
	removed     []mockPermCall
	parents     []mockParentCall
}

type mockPermCall struct {
	Ref      authz.ObjectRef
	Subject  authz.EntityKey
	Relation authz.Relation
}

type mockParentCall struct {
	Child  authz.ObjectRef
	Parent authz.ObjectRef
}

func newMockPermissionManager() *mockPermissionManager {
	return &mockPermissionManager{
		objects:     map[authz.ObjectType][]authz.ObjectRef{},
		permissions: map[string]*authz.ObjectPermissions{},
	}
}

func (m *mockPermissionManager) ListObjects(ctx context.Context, objType authz.ObjectType) ([]authz.ObjectRef, error) {
	return m.objects[objType], nil
}

func (m *mockPermissionManager) ObjectPermissions(ctx context.Context, obj authz.ObjectRef) (*authz.ObjectPermissions, error) {
	key := fmt.Sprintf("%s:%d", obj.Type.String(), obj.ID)
	if p, ok := m.permissions[key]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (m *mockPermissionManager) SetParent(ctx context.Context, child authz.ObjectRef, parent authz.ObjectRef) error {
	m.parents = append(m.parents, mockParentCall{Child: child, Parent: parent})
	return nil
}

func (m *mockPermissionManager) AddPermission(ctx context.Context, obj authz.ObjectRef, subject authz.EntityKey, relation authz.Relation) error {
	m.added = append(m.added, mockPermCall{Ref: obj, Subject: subject, Relation: relation})
	return nil
}

func (m *mockPermissionManager) RemovePermission(ctx context.Context, obj authz.ObjectRef, subject authz.EntityKey, relation authz.Relation) error {
	m.removed = append(m.removed, mockPermCall{Ref: obj, Subject: subject, Relation: relation})
	return nil
}

func (m *mockPermissionManager) addObject(objType authz.ObjectType, id int64) {
	m.objects[objType] = append(m.objects[objType], authz.ObjectRef{Type: objType, ID: id})
}

func (m *mockPermissionManager) addPermissions(objType authz.ObjectType, id int64, perms *authz.ObjectPermissions) {
	key := fmt.Sprintf("%s:%d", objType.String(), id)
	if perms.Ref.Type == 0 {
		perms.Ref = authz.ObjectRef{Type: objType, ID: id}
	}
	m.permissions[key] = perms
}

func newPermTestClient(t testing.TB, pm *mockPermissionManager) *client.Client {
	srv, _ := NewServer()
	cfg := model.Config{
		Checker:           pm,
		PermissionManager: pm,
	}
	handler := model.AddConfigAndPerms(cfg, srv)
	handler = usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "Test User", "test@example.com").WithRoles("admin")
	})(handler)
	return client.New(handler)
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

// Compile-time check
var _ authz.PermissionManager = (*mockPermissionManager)(nil)

// Tests

func TestPermissionResolver_Tenants(t *testing.T) {
	pm := newMockPermissionManager()
	pm.addObject(authz.TenantType, 1)
	pm.addPermissions(authz.TenantType, 1, &authz.ObjectPermissions{
		Ref:     authz.ObjectRef{Type: authz.TenantType, ID: 1, Name: "test-tenant"},
		Actions: authz.ActionSet{authz.CanView: true, authz.CanEdit: true},
		Children: []authz.ObjectRef{
			{Type: authz.GroupType, ID: 10, Name: "test-group"},
		},
		Subjects: []authz.SubjectRef{
			{Subject: authz.NewEntityKey(authz.UserType, "ian"), Relation: authz.AdminRelation, Name: "Ian"},
		},
	})

	c := newPermTestClient(t, pm)

	t.Run("list tenants", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { id name } }`, nil)
		tenants := gjson.Get(jj, "tenants").Array()
		assert.Equal(t, 1, len(tenants))
		assert.Equal(t, int64(1), tenants[0].Get("id").Int())
		assert.Equal(t, "test-tenant", tenants[0].Get("name").Str)
	})

	t.Run("tenant permissions", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { permissions { actions subjects { type id name relation } children { type id name } } } }`, nil)
		perms := gjson.Get(jj, "tenants.0.permissions")
		actions := perms.Get("actions").Array()
		var actionStrs []string
		for _, a := range actions {
			actionStrs = append(actionStrs, a.Str)
		}
		assert.ElementsMatch(t, []string{"can_view", "can_edit"}, actionStrs)

		subjects := perms.Get("subjects").Array()
		assert.Equal(t, 1, len(subjects))
		assert.Equal(t, "user", subjects[0].Get("type").Str)
		assert.Equal(t, "ian", subjects[0].Get("id").Str)
		assert.Equal(t, "admin", subjects[0].Get("relation").Str)

		children := perms.Get("children").Array()
		assert.Equal(t, 1, len(children))
		assert.Equal(t, "org", children[0].Get("type").Str)
	})

	t.Run("tenant groups", func(t *testing.T) {
		jj := postQuery(t, c, `{ tenants { groups { id name } } }`, nil)
		groups := gjson.Get(jj, "tenants.0.groups").Array()
		assert.Equal(t, 1, len(groups))
		assert.Equal(t, int64(10), groups[0].Get("id").Int())
		assert.Equal(t, "test-group", groups[0].Get("name").Str)
	})
}

func TestPermissionResolver_Groups(t *testing.T) {
	pm := newMockPermissionManager()
	pm.addObject(authz.GroupType, 10)
	pm.addPermissions(authz.GroupType, 10, &authz.ObjectPermissions{
		Ref: authz.ObjectRef{Type: authz.GroupType, ID: 10, Name: "test-group"},
		Parent: &authz.ObjectRef{
			Type: authz.TenantType, ID: 1, Name: "test-tenant",
		},
		Actions: authz.ActionSet{authz.CanView: true},
	})

	c := newPermTestClient(t, pm)

	t.Run("list groups", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { id name } }`, nil)
		groups := gjson.Get(jj, "groups").Array()
		assert.Equal(t, 1, len(groups))
		assert.Equal(t, "test-group", groups[0].Get("name").Str)
	})

	t.Run("group tenant", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { tenant { id name } } }`, nil)
		tenant := gjson.Get(jj, "groups.0.tenant")
		assert.Equal(t, int64(1), tenant.Get("id").Int())
		assert.Equal(t, "test-tenant", tenant.Get("name").Str)
	})

	t.Run("group permissions", func(t *testing.T) {
		jj := postQuery(t, c, `{ groups { permissions { actions parent { type id name } } } }`, nil)
		perms := gjson.Get(jj, "groups.0.permissions")
		actions := perms.Get("actions").Array()
		assert.Equal(t, 1, len(actions))
		assert.Equal(t, "can_view", actions[0].Str)
		assert.Equal(t, "tenant", perms.Get("parent.type").Str)
	})
}

func TestPermissionResolver_Mutations(t *testing.T) {
	t.Run("permission_add", func(t *testing.T) {
		pm := newMockPermissionManager()
		c := newPermTestClient(t, pm)
		jj := postQuery(t, c, `mutation {
			permission_add(type: "tenant", id: 1, input: {subject_type: "user", subject_id: "ian", relation: "admin"})
		}`, nil)
		assert.Equal(t, true, gjson.Get(jj, "permission_add").Bool())
		assert.Equal(t, 1, len(pm.added))
		assert.Equal(t, authz.TenantType, pm.added[0].Ref.Type)
		assert.Equal(t, int64(1), pm.added[0].Ref.ID)
		assert.Equal(t, "ian", pm.added[0].Subject.Name)
		assert.Equal(t, authz.AdminRelation, pm.added[0].Relation)
	})

	t.Run("permission_remove", func(t *testing.T) {
		pm := newMockPermissionManager()
		c := newPermTestClient(t, pm)
		jj := postQuery(t, c, `mutation {
			permission_remove(type: "org", id: 5, input: {subject_type: "user", subject_id: "drew", relation: "viewer"})
		}`, nil)
		assert.Equal(t, true, gjson.Get(jj, "permission_remove").Bool())
		assert.Equal(t, 1, len(pm.removed))
		assert.Equal(t, authz.GroupType, pm.removed[0].Ref.Type)
		assert.Equal(t, int64(5), pm.removed[0].Ref.ID)
	})

	t.Run("permission_set_parent", func(t *testing.T) {
		pm := newMockPermissionManager()
		c := newPermTestClient(t, pm)
		jj := postQuery(t, c, `mutation {
			permission_set_parent(type: "org", id: 10, input: {parent_type: "tenant", parent_id: 1})
		}`, nil)
		assert.Equal(t, true, gjson.Get(jj, "permission_set_parent").Bool())
		assert.Equal(t, 1, len(pm.parents))
		assert.Equal(t, authz.GroupType, pm.parents[0].Child.Type)
		assert.Equal(t, authz.TenantType, pm.parents[0].Parent.Type)
	})

	t.Run("invalid type", func(t *testing.T) {
		pm := newMockPermissionManager()
		c := newPermTestClient(t, pm)
		postQueryExpectError(t, c, `mutation {
			permission_add(type: "bogus", id: 1, input: {subject_type: "user", subject_id: "ian", relation: "admin"})
		}`)
	})

	t.Run("invalid relation", func(t *testing.T) {
		pm := newMockPermissionManager()
		c := newPermTestClient(t, pm)
		postQueryExpectError(t, c, `mutation {
			permission_add(type: "tenant", id: 1, input: {subject_type: "user", subject_id: "ian", relation: "bogus"})
		}`)
	})
}

func TestPermissionResolver_NilPermissionManager(t *testing.T) {
	// When PermissionManager is not configured, permissions field should return null
	srv, _ := NewServer()
	cfg := model.Config{}
	handler := model.AddConfigAndPerms(cfg, srv)
	handler = usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})(handler)
	c := client.New(handler.(http.Handler))

	t.Run("tenants returns error", func(t *testing.T) {
		postQueryExpectError(t, c, `{ tenants { id } }`)
	})

	t.Run("feed permissions returns null", func(t *testing.T) {
		// This test requires the DB test fixtures; skip if not available.
		t.Skip("requires test database")
	})
}
