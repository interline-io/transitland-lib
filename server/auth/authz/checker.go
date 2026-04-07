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
// Implementations must enforce authorization checks internally — callers
// (e.g., GraphQL resolvers) delegate all access control to these methods.
type PermissionManager interface {
	Checker
	ObjectPermissions(ctx context.Context, obj ObjectRef) (*ObjectPermissions, error)
	SetParent(ctx context.Context, child ObjectRef, parent ObjectRef) error
	AddPermission(ctx context.Context, obj ObjectRef, subject EntityKey, relation Relation) error
	RemovePermission(ctx context.Context, obj ObjectRef, subject EntityKey, relation Relation) error
}

// AdminManager extends PermissionManager with admin-specific DB write
// operations for managing tenants and groups. These are not expressible
// through the generic permission interface because they create/update
// database entities, not just authorization tuples.
//
// Implementations that expose user search (e.g., for assigning users to
// tenants/groups) must handle visibility scoping in the UserProvider layer.
// The GraphQL resolvers gate access via can_edit_members but do not filter
// results — the UserProvider is responsible for limiting which users are
// returned based on deployment-specific rules (e.g., Auth0 organization
// boundaries, tenant membership, etc.).
type AdminManager interface {
	PermissionManager
	TenantSave(ctx context.Context, req *TenantSaveRequest) (*TenantSaveResponse, error)
	TenantCreateGroup(ctx context.Context, req *TenantCreateGroupRequest) (*GroupSaveResponse, error)
	GroupSave(ctx context.Context, req *GroupSaveRequest) (*GroupSaveResponse, error)
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
