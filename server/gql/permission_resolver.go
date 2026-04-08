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
		// Check that user has permission to view each group
		childPerms, err := pm.ObjectPermissions(ctx, child)
		if err != nil {
			continue
		}
		groups = append(groups, &model.Group{ID: int(child.ID), Name: childPerms.Ref.Name})
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
	if len(ids) > 0 {
		tenants := make([]*model.Tenant, 0, len(ids))
		for _, id := range ids {
			ref := authz.ObjectRef{Type: authz.TenantType, ID: int64(id)}
			perms, err := pm.ObjectPermissions(ctx, ref)
			if err != nil {
				continue
			}
			tenants = append(tenants, &model.Tenant{ID: id, Name: perms.Ref.Name})
			if limit != nil && len(tenants) >= *limit {
				break
			}
		}
		return tenants, nil
	}
	refs, err := pm.ListObjects(ctx, authz.TenantType)
	if err != nil {
		return nil, err
	}
	tenants := make([]*model.Tenant, 0, len(refs))
	for _, ref := range refs {
		t := &model.Tenant{ID: int(ref.ID)}
		if perms, err := pm.ObjectPermissions(ctx, ref); err == nil {
			t.Name = perms.Ref.Name
		}
		tenants = append(tenants, t)
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
	if len(ids) > 0 {
		groups := make([]*model.Group, 0, len(ids))
		for _, id := range ids {
			ref := authz.ObjectRef{Type: authz.GroupType, ID: int64(id)}
			perms, err := pm.ObjectPermissions(ctx, ref)
			if err != nil {
				continue
			}
			groups = append(groups, &model.Group{ID: id, Name: perms.Ref.Name})
			if limit != nil && len(groups) >= *limit {
				break
			}
		}
		return groups, nil
	}
	refs, err := pm.ListObjects(ctx, authz.GroupType)
	if err != nil {
		return nil, err
	}
	groups := make([]*model.Group, 0, len(refs))
	for _, ref := range refs {
		g := &model.Group{ID: int(ref.ID)}
		if perms, err := pm.ObjectPermissions(ctx, ref); err == nil {
			g.Name = perms.Ref.Name
		}
		groups = append(groups, g)
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

// Mutation resolvers

func (r *mutationResolver) PermissionAdd(ctx context.Context, typeArg string, id int, input model.PermissionInput) (bool, error) {
	pm, ref, subject, rel, err := parsePermissionArgs(ctx, typeArg, id, input)
	if err != nil {
		return false, err
	}
	if err := pm.AddPermission(ctx, ref, subject, rel); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) PermissionRemove(ctx context.Context, typeArg string, id int, input model.PermissionInput) (bool, error) {
	pm, ref, subject, rel, err := parsePermissionArgs(ctx, typeArg, id, input)
	if err != nil {
		return false, err
	}
	if err := pm.RemovePermission(ctx, ref, subject, rel); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) PermissionSetParent(ctx context.Context, typeArg string, id int, input model.SetParentInput) (bool, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return false, errors.New("permission management not configured")
	}
	childType, err := authz.ObjectTypeString(typeArg)
	if err != nil {
		return false, err
	}
	parentType, err := authz.ObjectTypeString(input.ParentType)
	if err != nil {
		return false, err
	}
	child := authz.ObjectRef{Type: childType, ID: int64(id)}
	parent := authz.ObjectRef{Type: parentType, ID: int64(input.ParentID)}
	if err := pm.SetParent(ctx, child, parent); err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) TenantSave(ctx context.Context, id int, input model.TenantInput) (*model.Tenant, error) {
	am, err := getAdminManager(ctx)
	if err != nil {
		return nil, err
	}
	_, err = am.TenantSave(ctx, &authz.TenantSaveRequest{
		Tenant: &authz.Tenant{Id: int64(id), Name: input.Name},
	})
	if err != nil {
		return nil, err
	}
	return &model.Tenant{ID: id, Name: input.Name}, nil
}

func (r *mutationResolver) TenantCreateGroup(ctx context.Context, id int, input model.GroupInput) (*model.Group, error) {
	am, err := getAdminManager(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := am.TenantCreateGroup(ctx, &authz.TenantCreateGroupRequest{
		Id:    int64(id),
		Group: &authz.Group{Name: input.Name},
	})
	if err != nil {
		return nil, err
	}
	return &model.Group{ID: int(resp.Group.Id), Name: input.Name}, nil
}

func (r *mutationResolver) GroupSave(ctx context.Context, id int, input model.GroupInput) (*model.Group, error) {
	am, err := getAdminManager(ctx)
	if err != nil {
		return nil, err
	}
	_, err = am.GroupSave(ctx, &authz.GroupSaveRequest{
		Group: &authz.Group{Id: int64(id), Name: input.Name},
	})
	if err != nil {
		return nil, err
	}
	return &model.Group{ID: id, Name: input.Name}, nil
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

// parsePermissionArgs validates and converts the string arguments for
// permission add/remove mutations into typed authz values.
func parsePermissionArgs(ctx context.Context, typeArg string, id int, input model.PermissionInput) (authz.PermissionManager, authz.ObjectRef, authz.EntityKey, authz.Relation, error) {
	pm, err := getPermissionManager(ctx)
	if pm == nil || err != nil {
		return nil, authz.ObjectRef{}, authz.EntityKey{}, 0, errors.New("permission management not configured")
	}
	objType, err := authz.ObjectTypeString(typeArg)
	if err != nil {
		return nil, authz.ObjectRef{}, authz.EntityKey{}, 0, err
	}
	subjectType, err := authz.ObjectTypeString(input.SubjectType)
	if err != nil {
		return nil, authz.ObjectRef{}, authz.EntityKey{}, 0, err
	}
	rel, err := authz.RelationString(input.Relation)
	if err != nil {
		return nil, authz.ObjectRef{}, authz.EntityKey{}, 0, err
	}
	ref := authz.ObjectRef{Type: objType, ID: int64(id)}
	subject := authz.NewEntityKey(subjectType, input.SubjectID)
	return pm, ref, subject, rel, nil
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
	// Children - filter to only include children the user can view
	for _, child := range perms.Children {
		childPerms, err := pm.ObjectPermissions(ctx, child)
		if err != nil {
			continue
		}
		result.Children = append(result.Children, &model.PermissionRef{
			Type: displayTypeName(child.Type),
			ID:   int(child.ID),
			Name: childPerms.Ref.Name,
		})
	}
	return result, nil
}
