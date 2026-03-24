package authz

import "context"

// ObjectRef identifies an entity in the authorization system.
type ObjectRef struct {
	Type ObjectType `json:"type"`
	ID   int64      `json:"id"`
	Name string     `json:"name,omitempty"`
}

// ActionSet is the result of checking what a user can do on an object.
type ActionSet = map[Action]bool

// SubjectRef describes who has a relationship to an object.
type SubjectRef struct {
	Subject  EntityKey `json:"subject"`
	Relation Relation  `json:"relation"`
	Name     string    `json:"name"`
}

// ObjectPermissions is the generic return from a permissions query.
type ObjectPermissions struct {
	Ref      ObjectRef    `json:"ref"`
	Name     string       `json:"name"`
	Actions  ActionSet    `json:"actions"`
	Subjects []SubjectRef `json:"subjects"`
	Parent   *ObjectRef   `json:"parent,omitempty"`
	Children []ObjectRef  `json:"children,omitempty"`
}

// UserInfo is the return from Checker.Me().
type UserInfo struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Email          string            `json:"email"`
	Roles          []string          `json:"roles"`
	Groups         []Group           `json:"groups"`
	ExpandedGroups []Group           `json:"expanded_groups"`
	ExternalData   map[string]string `json:"external_data"`
}

// Checker is the read-only query interface used by the data path
// (perm_filter, actions, dbfinder mutations, GraphQL resolvers).
type Checker interface {
	Me(ctx context.Context) (*UserInfo, error)
	IsGlobalAdmin(ctx context.Context) (bool, error)
	ListObjects(ctx context.Context, objType ObjectType) ([]ObjectRef, error)
	Check(ctx context.Context, obj ObjectRef, action Action) (bool, error)
}

// PermissionManager extends Checker with write operations for managing
// permissions, parents, and viewing detailed permission info.
// Used by the admin REST API only.
type PermissionManager interface {
	Checker
	ObjectPermissions(ctx context.Context, obj ObjectRef) (*ObjectPermissions, error)
	SetParent(ctx context.Context, child ObjectRef, parent ObjectRef) error
	AddPermission(ctx context.Context, obj ObjectRef, subject EntityKey, relation Relation) error
	RemovePermission(ctx context.Context, obj ObjectRef, subject EntityKey, relation Relation) error
}

// GlobalAdminChecker implements Checker and always grants access.
// Used when auth is disabled (e.g., --disable-auth flag).
type GlobalAdminChecker struct{}

func (c *GlobalAdminChecker) Me(ctx context.Context) (*UserInfo, error) {
	return &UserInfo{}, nil
}

func (c *GlobalAdminChecker) IsGlobalAdmin(ctx context.Context) (bool, error) {
	return true, nil
}

func (c *GlobalAdminChecker) ListObjects(ctx context.Context, objType ObjectType) ([]ObjectRef, error) {
	return nil, nil
}

func (c *GlobalAdminChecker) Check(ctx context.Context, obj ObjectRef, action Action) (bool, error) {
	return true, nil
}
