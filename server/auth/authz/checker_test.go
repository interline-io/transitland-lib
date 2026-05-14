package authz

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check — AdminRoleChecker satisfies the basic Checker contract.
var _ Checker = (*AdminRoleChecker)(nil)

func ctxWithUser(id string, roles ...string) context.Context {
	u := authn.NewCtxUser(id, "", "").WithRoles(roles...)
	return authn.WithUser(context.Background(), u)
}

func TestAdminRoleChecker_AdminRoleAllows(t *testing.T) {
	c := &AdminRoleChecker{}
	ctx := ctxWithUser("alice", "admin")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.True(t, ok, "user with admin role should pass Check")

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.True(t, isGA, "user with admin role should be global admin")
}

func TestAdminRoleChecker_NonAdminDenied(t *testing.T) {
	c := &AdminRoleChecker{}
	ctx := ctxWithUser("bob", "viewer")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok, "user without admin role should be denied")

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.False(t, isGA)
}

func TestAdminRoleChecker_AllowlistedUserIDAllows(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"service-account-1", "service-account-2"}}
	ctx := ctxWithUser("service-account-1") // no role, ID matches allowlist

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.True(t, ok, "allowlisted user ID should pass Check")

	isGA, err := c.IsGlobalAdmin(ctx)
	require.NoError(t, err)
	assert.True(t, isGA)
}

func TestAdminRoleChecker_AllowlistDoesNotMatchOtherUser(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"service-account-1"}}
	ctx := ctxWithUser("eve")

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok, "user not in allowlist and without admin role should be denied")
}

func TestAdminRoleChecker_NilUserDenied(t *testing.T) {
	c := &AdminRoleChecker{GlobalAdminUserIDs: []string{"someone"}}
	ctx := context.Background() // no authn user

	ok, err := c.Check(ctx, ObjectRef{Type: FeedType, ID: 1}, CanEdit)
	require.NoError(t, err)
	assert.False(t, ok, "missing authn user should deny without panic")

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
	// PermFilter populated from this Checker should have empty AllowedFeeds
	// — admins get the IsGlobalAdmin shortcut, non-admins get no per-object
	// allowlist. Returning nil keeps the existing model.WithPerms logic happy.
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
