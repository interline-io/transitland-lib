package authz

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ctxWithUser(id string, roles ...string) context.Context {
	u := authn.NewCtxUser(id, "", "").WithRoles(roles...)
	return authn.WithUser(context.Background(), u)
}

func TestAdminRoleChecker_AdminRoleAllows(t *testing.T) {
	c := &AdminRoleChecker{}
	ctx := ctxWithUser("alice", "admin")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.True(t, ok)

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.True(t, isGA)
}

func TestAdminRoleChecker_NonAdminDenied(t *testing.T) {
	c := &AdminRoleChecker{}
	ctx := ctxWithUser("bob", "viewer")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok)

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.False(t, isGA)
}

func TestAdminRoleChecker_AllowlistedUserIDAllows(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"service-account-1", "service-account-2"}}
	ctx := ctxWithUser("service-account-1")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.True(t, ok)

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.True(t, isGA)
}

func TestAdminRoleChecker_AllowlistDoesNotMatchOtherUser(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"service-account-1"}}
	ctx := ctxWithUser("eve")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAdminRoleChecker_NilUserDenied(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"someone"}}
	ctx := context.Background()

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok)

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.False(t, isGA)
}

func TestAdminRoleChecker_Me(t *testing.T) {
	c := &AdminRoleChecker{}

	t.Run("nil user returns ErrUnauthorized", func(t *testing.T) {
		_, err := c.Me(context.Background())
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("present user projects to UserInfo", func(t *testing.T) {
		ctx := ctxWithUser("alice", "admin")
		info, err := c.Me(ctx)
		require.NoError(t, err)
		assert.Equal(t, "alice", info.ID)
		assert.Equal(t, []string{"admin"}, info.Roles)
	})
}

func TestAdminRoleChecker_ListObjectsAlwaysEmpty(t *testing.T) {
	// Admins take the IsGlobalAdmin shortcut; non-admins get no per-object
	// allowlist. Nil keeps model.WithPerms happy.
	c := &AdminRoleChecker{}
	for _, ctx := range []context.Context{
		context.Background(),
		ctxWithUser("alice", "admin"),
		ctxWithUser("bob"),
	} {
		refs, err := c.ListObjects(ctx, FeedType)
		require.NoError(t, err)
		assert.Nil(t, refs)
	}
}
