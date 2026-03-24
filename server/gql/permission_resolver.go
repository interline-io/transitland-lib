package gql

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/model"
)

// Tenant resolver

type tenantResolver struct{ *Resolver }

func (r *Resolver) Tenant() gqlout.TenantResolver { return &tenantResolver{r} }

func (r *tenantResolver) Groups(ctx context.Context, obj *model.Tenant) ([]*model.Group, error) {
	pm, err := getPermissionManager(ctx)
	if err != nil {
		return nil, err
	}
	ref := authz.ObjectRef{Type: authz.TenantType, ID: int64(obj.ID)}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	var groups []*model.Group
	for _, child := range perms.Children {
		if child.Type == authz.GroupType {
			groups = append(groups, &model.Group{ID: int(child.ID), Name: child.Name})
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
	if err != nil {
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

func (r *groupResolver) Feeds(ctx context.Context, obj *model.Group) ([]*model.Feed, error) {
	pm, err := getPermissionManager(ctx)
	if err != nil {
		return nil, err
	}
	ref := authz.ObjectRef{Type: authz.GroupType, ID: int64(obj.ID)}
	perms, err := pm.ObjectPermissions(ctx, ref)
	if err != nil {
		return nil, err
	}
	if len(perms.Children) == 0 {
		return nil, nil
	}
	var ids []int
	for _, child := range perms.Children {
		if child.Type == authz.FeedType {
			ids = append(ids, int(child.ID))
		}
	}
	cfg := model.ForContext(ctx)
	return cfg.Finder.FindFeeds(ctx, nil, nil, ids, nil)
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

func (r *queryResolver) Tenants(ctx context.Context) ([]*model.Tenant, error) {
	checker := model.ForContext(ctx).Checker
	if checker == nil {
		return nil, errors.New("permissions not configured")
	}
	refs, err := checker.ListObjects(ctx, authz.TenantType)
	if err != nil {
		return nil, err
	}
	// Hydrate names via ObjectPermissions
	pm, _ := getPermissionManager(ctx)
	var tenants []*model.Tenant
	for _, ref := range refs {
		t := &model.Tenant{ID: int(ref.ID)}
		if pm != nil {
			if perms, err := pm.ObjectPermissions(ctx, ref); err == nil {
				t.Name = perms.Ref.Name
			}
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (r *queryResolver) Groups(ctx context.Context) ([]*model.Group, error) {
	checker := model.ForContext(ctx).Checker
	if checker == nil {
		return nil, errors.New("permissions not configured")
	}
	refs, err := checker.ListObjects(ctx, authz.GroupType)
	if err != nil {
		return nil, err
	}
	pm, _ := getPermissionManager(ctx)
	var groups []*model.Group
	for _, ref := range refs {
		g := &model.Group{ID: int(ref.ID)}
		if pm != nil {
			if perms, err := pm.ObjectPermissions(ctx, ref); err == nil {
				g.Name = perms.Ref.Name
			}
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// Mutation resolvers

func (r *mutationResolver) PermissionAdd(ctx context.Context, typeArg string, id int, input model.PermissionInput) (bool, error) {
	pm, ref, subject, rel, err := parsePermissionArgs(ctx, typeArg, id, input)
	if err != nil {
		return false, err
	}
	return true, pm.AddPermission(ctx, ref, subject, rel)
}

func (r *mutationResolver) PermissionRemove(ctx context.Context, typeArg string, id int, input model.PermissionInput) (bool, error) {
	pm, ref, subject, rel, err := parsePermissionArgs(ctx, typeArg, id, input)
	if err != nil {
		return false, err
	}
	return true, pm.RemovePermission(ctx, ref, subject, rel)
}

func (r *mutationResolver) PermissionSetParent(ctx context.Context, typeArg string, id int, input model.SetParentInput) (bool, error) {
	pm, err := getPermissionManager(ctx)
	if err != nil {
		return false, err
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
	return true, pm.SetParent(ctx, child, parent)
}

func (r *mutationResolver) TenantSave(ctx context.Context, id int, input model.TenantInput) (*model.Tenant, error) {
	pm, err := getPermissionManagerConcrete(ctx)
	if err != nil {
		return nil, err
	}
	_, err = pm.TenantSave(ctx, &authz.TenantSaveRequest{
		Tenant: &authz.Tenant{Id: int64(id), Name: input.Name},
	})
	if err != nil {
		return nil, err
	}
	return &model.Tenant{ID: id, Name: input.Name}, nil
}

func (r *mutationResolver) TenantCreateGroup(ctx context.Context, id int, input model.GroupInput) (*model.Group, error) {
	pm, err := getPermissionManagerConcrete(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := pm.TenantCreateGroup(ctx, &authz.TenantCreateGroupRequest{
		Id:    int64(id),
		Group: &authz.Group{Name: input.Name},
	})
	if err != nil {
		return nil, err
	}
	return &model.Group{ID: int(resp.Group.Id), Name: input.Name}, nil
}

func (r *mutationResolver) GroupSave(ctx context.Context, id int, input model.GroupInput) (*model.Group, error) {
	pm, err := getPermissionManagerConcrete(ctx)
	if err != nil {
		return nil, err
	}
	_, err = pm.GroupSave(ctx, &authz.GroupSaveRequest{
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
	if cfg.PermissionManager != nil {
		return cfg.PermissionManager, nil
	}
	return nil, nil
}

// getPermissionManagerConcrete returns the azchecker.Checker for admin-specific
// methods (TenantSave, TenantCreateGroup, GroupSave) that are not on the
// PermissionManager interface. These require the concrete type.
func getPermissionManagerConcrete(ctx context.Context) (concretePermissionManager, error) {
	cfg := model.ForContext(ctx)
	if pm, ok := cfg.PermissionManager.(concretePermissionManager); ok {
		return pm, nil
	}
	return nil, errors.New("admin operations not configured")
}

// concretePermissionManager extends PermissionManager with admin-specific
// DB write methods that only the concrete azchecker.Checker provides.
type concretePermissionManager interface {
	authz.PermissionManager
	TenantSave(ctx context.Context, req *authz.TenantSaveRequest) (*authz.TenantSaveResponse, error)
	TenantCreateGroup(ctx context.Context, req *authz.TenantCreateGroupRequest) (*authz.GroupSaveResponse, error)
	GroupSave(ctx context.Context, req *authz.GroupSaveRequest) (*authz.GroupSaveResponse, error)
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
	result := &model.Permissions{}
	// Actions
	for action, granted := range perms.Actions {
		if granted {
			result.Actions = append(result.Actions, action.String())
		}
	}
	// Subjects
	for _, s := range perms.Subjects {
		result.Subjects = append(result.Subjects, &model.PermissionSubject{
			Type:     s.Subject.Type.String(),
			ID:       s.Subject.Name,
			Name:     s.Name,
			Relation: s.Relation.String(),
		})
	}
	// Parent
	if perms.Parent != nil {
		result.Parent = &model.PermissionRef{
			Type: perms.Parent.Type.String(),
			ID:   int(perms.Parent.ID),
			Name: perms.Parent.Name,
		}
	}
	// Children
	for _, child := range perms.Children {
		result.Children = append(result.Children, &model.PermissionRef{
			Type: child.Type.String(),
			ID:   int(child.ID),
			Name: child.Name,
		})
	}
	return result, nil
}
