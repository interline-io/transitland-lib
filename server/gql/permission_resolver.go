package gql

import (
	"context"
	"errors"
	"sort"

	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/model"
)

// objectTypeDisplayName maps internal FGA type names to user-facing GraphQL names.
// The FGA model uses "org" but the GraphQL API exposes "group".
var objectTypeDisplayName = map[string]string{
	"org": "group",
}

func displayTypeName(t authz.ObjectType) string {
	s := t.String()
	if mapped, ok := objectTypeDisplayName[s]; ok {
		return mapped
	}
	return s
}

// Tenant resolver

type tenantResolver struct{ *Resolver }

func (r *Resolver) Tenant() gqlout.TenantResolver { return &tenantResolver{r} }

func (r *tenantResolver) Groups(ctx context.Context, obj *model.Tenant, limit *int) ([]*model.Group, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return []*model.Group{}, err
	}
	ref := authz.ObjectRef{Type: authz.TenantType, ID: int64(obj.ID)}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	groups := []*model.Group{}
	for _, child := range perms.Children {
		if child.Type != authz.GroupType {
			continue
		}
		groups = append(groups, &model.Group{ID: int(child.ID), Name: child.Name})
		if limit != nil && len(groups) >= *limit {
			break
		}
	}
	return groups, nil
}

func (r *tenantResolver) Permissions(ctx context.Context, obj *model.Tenant) (*model.Permissions, error) {
	return resolvePermissions(ctx, authz.TenantType, int64(obj.ID))
}

// Group resolver

type groupResolver struct{ *Resolver }

func (r *Resolver) Group() gqlout.GroupResolver { return &groupResolver{r} }

func (r *groupResolver) Tenant(ctx context.Context, obj *model.Group) (*model.Tenant, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return nil, err
	}
	ref := authz.ObjectRef{Type: authz.GroupType, ID: int64(obj.ID)}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	if perms.Parent != nil && perms.Parent.Type == authz.TenantType {
		return &model.Tenant{ID: int(perms.Parent.ID), Name: perms.Parent.Name}, nil
	}
	return nil, nil
}

func (r *groupResolver) Feeds(ctx context.Context, obj *model.Group, limit *int) ([]*model.Feed, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return []*model.Feed{}, err
	}
	ref := authz.ObjectRef{Type: authz.GroupType, ID: int64(obj.ID)}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	// No per-child ObjectPermissions check needed here: feed IDs are passed
	// to FindFeeds which applies PermFilter at the SQL layer, so unauthorized
	// feeds are filtered out before results are returned.
	var ids []int
	for _, child := range perms.Children {
		if child.Type == authz.FeedType {
			ids = append(ids, int(child.ID))
		}
	}
	if len(ids) == 0 {
		return []*model.Feed{}, nil
	}
	cfg := model.ForContext(ctx)
	if cfg.Finder == nil {
		return []*model.Feed{}, nil
	}
	return cfg.Finder.FindFeeds(ctx, limit, nil, ids, nil)
}

func (r *groupResolver) Permissions(ctx context.Context, obj *model.Group) (*model.Permissions, error) {
	return resolvePermissions(ctx, authz.GroupType, int64(obj.ID))
}

// Feed permissions resolver (extends existing feedResolver)

func (r *feedResolver) Permissions(ctx context.Context, obj *model.Feed) (*model.Permissions, error) {
	return resolvePermissions(ctx, authz.FeedType, int64(obj.ID))
}

// FeedVersion permissions resolver (extends existing feedVersionResolver)

func (r *feedVersionResolver) Permissions(ctx context.Context, obj *model.FeedVersion) (*model.Permissions, error) {
	return resolvePermissions(ctx, authz.FeedVersionType, int64(obj.ID))
}

// Query resolvers for tenants and groups

func (r *queryResolver) Tenants(ctx context.Context, limit *int, ids []int) ([]*model.Tenant, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return nil, nil
	}
	var refs []authz.ObjectRef
	if len(ids) > 0 {
		for _, id := range ids {
			ref := authz.ObjectRef{Type: authz.TenantType, ID: int64(id)}
			perms, err := pm.ObjectPermissions(ctx, ref)
			if err != nil {
				continue
			}
			refs = append(refs, perms.Ref)
			if limit != nil && len(refs) >= *limit {
				break
			}
		}
	} else {
		refs, err = pm.ListObjects(ctx, authz.TenantType)
		if err != nil {
			return nil, err
		}
	}
	tenants := make([]*model.Tenant, 0, len(refs))
	for _, ref := range refs {
		tenants = append(tenants, &model.Tenant{ID: int(ref.ID), Name: ref.Name})
		if limit != nil && len(tenants) >= *limit {
			break
		}
	}
	return tenants, nil
}

func (r *queryResolver) Groups(ctx context.Context, limit *int, ids []int) ([]*model.Group, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return nil, nil
	}
	var refs []authz.ObjectRef
	if len(ids) > 0 {
		for _, id := range ids {
			ref := authz.ObjectRef{Type: authz.GroupType, ID: int64(id)}
			perms, err := pm.ObjectPermissions(ctx, ref)
			if err != nil {
				continue
			}
			refs = append(refs, perms.Ref)
			if limit != nil && len(refs) >= *limit {
				break
			}
		}
	} else {
		refs, err = pm.ListObjects(ctx, authz.GroupType)
		if err != nil {
			return nil, err
		}
	}
	groups := make([]*model.Group, 0, len(refs))
	for _, ref := range refs {
		groups = append(groups, &model.Group{ID: int(ref.ID), Name: ref.Name})
		if limit != nil && len(groups) >= *limit {
			break
		}
	}
	return groups, nil
}

func (r *queryResolver) Users(ctx context.Context, limit *int, where *model.UserFilter) ([]*model.User, error) {
	am, err := getAdminManager(ctx)
	if err != nil {
		return []*model.User{}, nil
	}
	// Single user lookup by ID
	if where != nil && where.ID != nil {
		resp, err := am.User(ctx, &authz.UserRequest{Id: *where.ID})
		if err != nil || resp.User == nil {
			return []*model.User{}, nil
		}
		return []*model.User{{
			ID:    resp.User.Id,
			Name:  resp.User.Name,
			Email: resp.User.Email,
		}}, nil
	}
	// Search/list users
	searchQ := ""
	if where != nil && where.Q != nil {
		searchQ = *where.Q
	}
	resp, err := am.UserList(ctx, &authz.UserListRequest{Q: searchQ})
	if err != nil {
		return nil, err
	}
	users := make([]*model.User, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, &model.User{
			ID:    u.Id,
			Name:  u.Name,
			Email: u.Email,
		})
		if limit != nil && len(users) >= *limit {
			break
		}
	}
	return users, nil
}

// Helpers

func getPermissionManager(ctx context.Context) (authz.PermissionManager, error) {
	cfg := model.ForContext(ctx)
	if pm, ok := cfg.Checker.(authz.PermissionManager); ok {
		return pm, nil
	}
	return nil, nil
}

func getAdminManager(ctx context.Context) (authz.AdminManager, error) {
	cfg := model.ForContext(ctx)
	if am, ok := cfg.Checker.(authz.AdminManager); ok {
		return am, nil
	}
	return nil, errors.New("admin operations not configured")
}

func resolvePermissions(ctx context.Context, objType authz.ObjectType, id int64) (*model.Permissions, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return nil, err
	}
	ref := authz.ObjectRef{Type: objType, ID: id}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	result := &model.Permissions{
		Actions:  []string{},
		Subjects: []*model.PermissionSubject{},
		Children: []*model.PermissionRef{},
	}
	// Actions
	for action, granted := range perms.Actions {
		if granted {
			result.Actions = append(result.Actions, action.String())
		}
	}
	sort.Strings(result.Actions)
	// Subjects
	for _, s := range perms.Subjects {
		result.Subjects = append(result.Subjects, &model.PermissionSubject{
			Type:     displayTypeName(s.Subject.Type),
			ID:       s.Subject.Name,
			Name:     s.Name,
			Relation: s.Relation.String(),
		})
	}
	// Parent
	if perms.Parent != nil {
		result.Parent = &model.PermissionRef{
			Type: displayTypeName(perms.Parent.Type),
			ID:   int(perms.Parent.ID),
			Name: perms.Parent.Name,
		}
	}
	// Children (already filtered to viewable by ObjectPermissions)
	for _, child := range perms.Children {
		result.Children = append(result.Children, &model.PermissionRef{
			Type: displayTypeName(child.Type),
			ID:   int(child.ID),
			Name: child.Name,
		})
	}
	return result, nil
}
