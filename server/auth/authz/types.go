package authz

import (
	"context"
	"fmt"
)

// Action represents a permission action that can be checked.
type Action int32

const (
	Action_empty_action            Action = 0
	Action_can_view                Action = 1
	Action_can_edit                Action = 2
	Action_can_edit_members        Action = 3
	Action_can_create_org          Action = 4
	Action_can_delete_org          Action = 5
	Action_can_create_feed_version Action = 6
	Action_can_delete_feed_version Action = 7
	Action_can_create_feed         Action = 8
	Action_can_delete_feed         Action = 9
	Action_can_set_group           Action = 10
	Action_can_set_tenant          Action = 11
)

var Action_name = map[int32]string{
	0:  "empty_action",
	1:  "can_view",
	2:  "can_edit",
	3:  "can_edit_members",
	4:  "can_create_org",
	5:  "can_delete_org",
	6:  "can_create_feed_version",
	7:  "can_delete_feed_version",
	8:  "can_create_feed",
	9:  "can_delete_feed",
	10: "can_set_group",
	11: "can_set_tenant",
}

var Action_value = map[string]int32{
	"empty_action":            0,
	"can_view":                1,
	"can_edit":                2,
	"can_edit_members":        3,
	"can_create_org":          4,
	"can_delete_org":          5,
	"can_create_feed_version": 6,
	"can_delete_feed_version": 7,
	"can_create_feed":         8,
	"can_delete_feed":         9,
	"can_set_group":           10,
	"can_set_tenant":          11,
}

func (a Action) String() string {
	if s, ok := Action_name[int32(a)]; ok {
		return s
	}
	return fmt.Sprintf("Action(%d)", a)
}

// MarshalText implements encoding.TextMarshaler so Action works as a JSON map key.
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (a *Action) UnmarshalText(text []byte) error {
	if v, ok := Action_value[string(text)]; ok {
		*a = Action(v)
		return nil
	}
	return fmt.Errorf("unknown action: %s", text)
}

// ObjectType represents the type of an entity in the authorization model.
type ObjectType int32

const (
	ObjectType_empty_object  ObjectType = 0
	ObjectType_tenant        ObjectType = 1
	ObjectType_org           ObjectType = 2
	ObjectType_feed          ObjectType = 3
	ObjectType_feed_version  ObjectType = 4
	ObjectType_user          ObjectType = 5
)

var ObjectType_name = map[int32]string{
	0: "empty_object",
	1: "tenant",
	2: "org",
	3: "feed",
	4: "feed_version",
	5: "user",
}

var ObjectType_value = map[string]int32{
	"empty_object": 0,
	"tenant":       1,
	"org":          2,
	"feed":         3,
	"feed_version": 4,
	"user":         5,
}

func (o ObjectType) String() string {
	if s, ok := ObjectType_name[int32(o)]; ok {
		return s
	}
	return fmt.Sprintf("ObjectType(%d)", o)
}

// Relation represents a relationship between entities.
type Relation int32

const (
	Relation_empty_relation Relation = 0
	Relation_admin          Relation = 1
	Relation_member         Relation = 2
	Relation_manager        Relation = 3
	Relation_viewer         Relation = 4
	Relation_editor         Relation = 5
	Relation_parent         Relation = 6
)

var Relation_name = map[int32]string{
	0: "empty_relation",
	1: "admin",
	2: "member",
	3: "manager",
	4: "viewer",
	5: "editor",
	6: "parent",
}

var Relation_value = map[string]int32{
	"empty_relation": 0,
	"admin":          1,
	"member":         2,
	"manager":        3,
	"viewer":         4,
	"editor":         5,
	"parent":         6,
}

func (r Relation) String() string {
	if s, ok := Relation_name[int32(r)]; ok {
		return s
	}
	return fmt.Sprintf("Relation(%d)", r)
}

//////
// Entity structs
//////

type User struct {
	Id    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

func (x *User) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *User) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *User) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

type Tenant struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (x *Tenant) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Tenant) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type Group struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (x *Group) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Group) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type Feed struct {
	Id        int64  `json:"id,omitempty"`
	OnestopId string `json:"onestop_id,omitempty"`
	Name      string `json:"name,omitempty"`
}

func (x *Feed) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Feed) GetOnestopId() string {
	if x != nil {
		return x.OnestopId
	}
	return ""
}

func (x *Feed) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type FeedVersion struct {
	Id     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Sha1   string `json:"sha1,omitempty"`
	FeedId int64  `json:"feed_id,omitempty"`
}

func (x *FeedVersion) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *FeedVersion) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FeedVersion) GetSha1() string {
	if x != nil {
		return x.Sha1
	}
	return ""
}

func (x *FeedVersion) GetFeedId() int64 {
	if x != nil {
		return x.FeedId
	}
	return 0
}

type EntityRelation struct {
	Type        ObjectType `json:"type,omitempty"`
	Id          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	RefRelation Relation   `json:"ref_relation,omitempty"`
	Relation    Relation   `json:"relation,omitempty"`
}

func (x *EntityRelation) GetType() ObjectType {
	if x != nil {
		return x.Type
	}
	return ObjectType_empty_object
}

func (x *EntityRelation) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *EntityRelation) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *EntityRelation) GetRefRelation() Relation {
	if x != nil {
		return x.RefRelation
	}
	return Relation_empty_relation
}

func (x *EntityRelation) GetRelation() Relation {
	if x != nil {
		return x.Relation
	}
	return Relation_empty_relation
}

//////
// Request/Response types used by azchecker
//////

// User requests/responses
type UserListRequest struct {
	Q string `json:"q,omitempty"`
}

func (x *UserListRequest) GetQ() string {
	if x != nil {
		return x.Q
	}
	return ""
}

type UserListResponse struct {
	Users []*User `json:"users,omitempty"`
}

type UserRequest struct {
	Id string `json:"id,omitempty"`
}

func (x *UserRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type UserResponse struct {
	User *User `json:"user,omitempty"`
}

type MeRequest struct{}

type MeResponse struct {
	User           *User             `json:"user,omitempty"`
	Groups         []*Group          `json:"groups,omitempty"`
	ExpandedGroups []*Group          `json:"expanded_groups,omitempty"`
	ExternalData   map[string]string `json:"external_data,omitempty"`
	Roles          []string          `json:"roles,omitempty"`
}

// Tenant requests/responses
type TenantRequest struct {
	Id int64 `json:"id,omitempty"`
}

func (x *TenantRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type TenantListRequest struct{}

type TenantListResponse struct {
	Tenants []*Tenant `json:"tenants,omitempty"`
}

type TenantResponse struct {
	Tenant *Tenant `json:"tenant,omitempty"`
}

type TenantPermissionsResponse struct {
	Tenant  *Tenant                            `json:"tenant,omitempty"`
	Groups  []*Group                           `json:"groups,omitempty"`
	Actions *TenantPermissionsResponse_Actions `json:"actions,omitempty"`
	Users   *TenantPermissionsResponse_Users   `json:"users,omitempty"`
}

type TenantPermissionsResponse_Actions struct {
	CanEditMembers bool `json:"can_edit_members,omitempty"`
	CanView        bool `json:"can_view,omitempty"`
	CanEdit        bool `json:"can_edit,omitempty"`
	CanCreateOrg   bool `json:"can_create_org,omitempty"`
	CanDeleteOrg   bool `json:"can_delete_org,omitempty"`
}

type TenantPermissionsResponse_Users struct {
	Admins  []*EntityRelation `json:"admins,omitempty"`
	Members []*EntityRelation `json:"members,omitempty"`
}

type TenantSaveRequest struct {
	Tenant *Tenant `json:"tenant,omitempty"`
}

func (x *TenantSaveRequest) GetTenant() *Tenant {
	if x != nil {
		return x.Tenant
	}
	return nil
}

type TenantSaveResponse struct{}

type TenantCreateRequest struct{}

type TenantCreateGroupRequest struct {
	Id    int64  `json:"id,omitempty"`
	Group *Group `json:"group,omitempty"`
}

func (x *TenantCreateGroupRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *TenantCreateGroupRequest) GetGroup() *Group {
	if x != nil {
		return x.Group
	}
	return nil
}

type TenantModifyPermissionRequest struct {
	Id             int64           `json:"id,omitempty"`
	EntityRelation *EntityRelation `json:"entity_relation,omitempty"`
}

func (x *TenantModifyPermissionRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *TenantModifyPermissionRequest) GetEntityRelation() *EntityRelation {
	if x != nil {
		return x.EntityRelation
	}
	return nil
}

// Group requests/responses
type GroupRequest struct {
	Id int64 `json:"id,omitempty"`
}

func (x *GroupRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type GroupListRequest struct{}

type GroupListResponse struct {
	Groups []*Group `json:"groups,omitempty"`
}

type GroupResponse struct {
	Group *Group `json:"group,omitempty"`
}

type GroupPermissionsResponse struct {
	Group   *Group                            `json:"group,omitempty"`
	Tenant  *Tenant                           `json:"tenant,omitempty"`
	Feeds   []*Feed                           `json:"feeds,omitempty"`
	Actions *GroupPermissionsResponse_Actions  `json:"actions,omitempty"`
	Users   *GroupPermissionsResponse_Users    `json:"users,omitempty"`
}

type GroupPermissionsResponse_Actions struct {
	CanView        bool `json:"can_view,omitempty"`
	CanEditMembers bool `json:"can_edit_members,omitempty"`
	CanCreateFeed  bool `json:"can_create_feed,omitempty"`
	CanDeleteFeed  bool `json:"can_delete_feed,omitempty"`
	CanEdit        bool `json:"can_edit,omitempty"`
	CanSetTenant   bool `json:"can_set_tenant,omitempty"`
}

type GroupPermissionsResponse_Users struct {
	Managers []*EntityRelation `json:"managers,omitempty"`
	Editors  []*EntityRelation `json:"editors,omitempty"`
	Viewers  []*EntityRelation `json:"viewers,omitempty"`
}

type GroupSaveRequest struct {
	Group *Group `json:"group,omitempty"`
}

func (x *GroupSaveRequest) GetGroup() *Group {
	if x != nil {
		return x.Group
	}
	return nil
}

type GroupSaveResponse struct {
	Group *Group `json:"group,omitempty"`
}

type GroupModifyPermissionRequest struct {
	Id             int64           `json:"id,omitempty"`
	EntityRelation *EntityRelation `json:"entity_relation,omitempty"`
}

func (x *GroupModifyPermissionRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *GroupModifyPermissionRequest) GetEntityRelation() *EntityRelation {
	if x != nil {
		return x.EntityRelation
	}
	return nil
}

type GroupSetTenantRequest struct {
	Id       int64 `json:"id,omitempty"`
	TenantId int64 `json:"tenant_id,omitempty"`
}

func (x *GroupSetTenantRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *GroupSetTenantRequest) GetTenantId() int64 {
	if x != nil {
		return x.TenantId
	}
	return 0
}

type GroupSetTenantResponse struct{}

// Feed requests/responses
type FeedRequest struct {
	Id int64 `json:"id,omitempty"`
}

func (x *FeedRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type FeedListRequest struct{}

type FeedListResponse struct {
	Feeds []*Feed `json:"feeds,omitempty"`
}

type FeedResponse struct {
	Feed *Feed `json:"feed,omitempty"`
}

type FeedPermissionsResponse struct {
	Feed    *Feed                            `json:"feed,omitempty"`
	Group   *Group                           `json:"group,omitempty"`
	Actions *FeedPermissionsResponse_Actions `json:"actions,omitempty"`
}

type FeedPermissionsResponse_Actions struct {
	CanView              bool `json:"can_view,omitempty"`
	CanEdit              bool `json:"can_edit,omitempty"`
	CanSetGroup          bool `json:"can_set_group,omitempty"`
	CanCreateFeedVersion bool `json:"can_create_feed_version,omitempty"`
	CanDeleteFeedVersion bool `json:"can_delete_feed_version,omitempty"`
}

type FeedSetGroupRequest struct {
	Id      int64 `json:"id,omitempty"`
	GroupId int64 `json:"group_id,omitempty"`
}

func (x *FeedSetGroupRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *FeedSetGroupRequest) GetGroupId() int64 {
	if x != nil {
		return x.GroupId
	}
	return 0
}

type FeedSaveResponse struct{}

// FeedVersion requests/responses
type FeedVersionRequest struct {
	Id int64 `json:"id,omitempty"`
}

func (x *FeedVersionRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type FeedVersionListRequest struct{}

type FeedVersionListResponse struct {
	FeedVersions []*FeedVersion `json:"feed_versions,omitempty"`
}

type FeedVersionResponse struct {
	FeedVersion *FeedVersion `json:"feed_version,omitempty"`
}

type FeedVersionPermissionsResponse struct {
	FeedVersion *FeedVersion                            `json:"feed_version,omitempty"`
	Feed        *Feed                                   `json:"feed,omitempty"`
	Group       *Group                                  `json:"group,omitempty"`
	Actions     *FeedVersionPermissionsResponse_Actions `json:"actions,omitempty"`
	Users       *FeedVersionPermissionsResponse_Users   `json:"users,omitempty"`
}

type FeedVersionPermissionsResponse_Actions struct {
	CanView        bool `json:"can_view,omitempty"`
	CanEditMembers bool `json:"can_edit_members,omitempty"`
	CanEdit        bool `json:"can_edit,omitempty"`
}

type FeedVersionPermissionsResponse_Users struct {
	Editors []*EntityRelation `json:"editors,omitempty"`
	Viewers []*EntityRelation `json:"viewers,omitempty"`
}

type FeedVersionModifyPermissionRequest struct {
	Id             int64           `json:"id,omitempty"`
	EntityRelation *EntityRelation `json:"entity_relation,omitempty"`
}

func (x *FeedVersionModifyPermissionRequest) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *FeedVersionModifyPermissionRequest) GetEntityRelation() *EntityRelation {
	if x != nil {
		return x.EntityRelation
	}
	return nil
}

type FeedVersionSaveResponse struct{}

//////
// CheckerServer interface — legacy, kept for azchecker internal use
// This will be removed in Phase 4 when server.go is refactored.
//////

type CheckerServer interface {
	UserList(context.Context, *UserListRequest) (*UserListResponse, error)
	User(context.Context, *UserRequest) (*UserResponse, error)
	LegacyMe(context.Context, *MeRequest) (*MeResponse, error)
	TenantList(context.Context, *TenantListRequest) (*TenantListResponse, error)
	Tenant(context.Context, *TenantRequest) (*TenantResponse, error)
	TenantPermissions(context.Context, *TenantRequest) (*TenantPermissionsResponse, error)
	TenantSave(context.Context, *TenantSaveRequest) (*TenantSaveResponse, error)
	TenantAddPermission(context.Context, *TenantModifyPermissionRequest) (*TenantSaveResponse, error)
	TenantRemovePermission(context.Context, *TenantModifyPermissionRequest) (*TenantSaveResponse, error)
	TenantCreate(context.Context, *TenantCreateRequest) (*TenantSaveResponse, error)
	TenantCreateGroup(context.Context, *TenantCreateGroupRequest) (*GroupSaveResponse, error)
	GroupList(context.Context, *GroupListRequest) (*GroupListResponse, error)
	Group(context.Context, *GroupRequest) (*GroupResponse, error)
	GroupPermissions(context.Context, *GroupRequest) (*GroupPermissionsResponse, error)
	GroupSave(context.Context, *GroupSaveRequest) (*GroupSaveResponse, error)
	GroupAddPermission(context.Context, *GroupModifyPermissionRequest) (*GroupSaveResponse, error)
	GroupRemovePermission(context.Context, *GroupModifyPermissionRequest) (*GroupSaveResponse, error)
	GroupSetTenant(context.Context, *GroupSetTenantRequest) (*GroupSetTenantResponse, error)
	FeedList(context.Context, *FeedListRequest) (*FeedListResponse, error)
	Feed(context.Context, *FeedRequest) (*FeedResponse, error)
	FeedPermissions(context.Context, *FeedRequest) (*FeedPermissionsResponse, error)
	FeedSetGroup(context.Context, *FeedSetGroupRequest) (*FeedSaveResponse, error)
	FeedVersionList(context.Context, *FeedVersionListRequest) (*FeedVersionListResponse, error)
	FeedVersion(context.Context, *FeedVersionRequest) (*FeedVersionResponse, error)
	FeedVersionPermissions(context.Context, *FeedVersionRequest) (*FeedVersionPermissionsResponse, error)
	FeedVersionAddPermission(context.Context, *FeedVersionModifyPermissionRequest) (*FeedVersionSaveResponse, error)
	FeedVersionRemovePermission(context.Context, *FeedVersionModifyPermissionRequest) (*FeedVersionSaveResponse, error)
}

// UnimplementedCheckerServer provides default no-op implementations of CheckerServer.
// Embed this in structs to implement the interface with only the methods you need.
type UnimplementedCheckerServer struct{}

func (UnimplementedCheckerServer) UserList(context.Context, *UserListRequest) (*UserListResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) User(context.Context, *UserRequest) (*UserResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) LegacyMe(context.Context, *MeRequest) (*MeResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantList(context.Context, *TenantListRequest) (*TenantListResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) Tenant(context.Context, *TenantRequest) (*TenantResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantPermissions(context.Context, *TenantRequest) (*TenantPermissionsResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantSave(context.Context, *TenantSaveRequest) (*TenantSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantAddPermission(context.Context, *TenantModifyPermissionRequest) (*TenantSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantRemovePermission(context.Context, *TenantModifyPermissionRequest) (*TenantSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantCreate(context.Context, *TenantCreateRequest) (*TenantSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) TenantCreateGroup(context.Context, *TenantCreateGroupRequest) (*GroupSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupList(context.Context, *GroupListRequest) (*GroupListResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) Group(context.Context, *GroupRequest) (*GroupResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupPermissions(context.Context, *GroupRequest) (*GroupPermissionsResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupSave(context.Context, *GroupSaveRequest) (*GroupSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupAddPermission(context.Context, *GroupModifyPermissionRequest) (*GroupSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupRemovePermission(context.Context, *GroupModifyPermissionRequest) (*GroupSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) GroupSetTenant(context.Context, *GroupSetTenantRequest) (*GroupSetTenantResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedList(context.Context, *FeedListRequest) (*FeedListResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) Feed(context.Context, *FeedRequest) (*FeedResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedPermissions(context.Context, *FeedRequest) (*FeedPermissionsResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedSetGroup(context.Context, *FeedSetGroupRequest) (*FeedSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedVersionList(context.Context, *FeedVersionListRequest) (*FeedVersionListResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedVersion(context.Context, *FeedVersionRequest) (*FeedVersionResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedVersionPermissions(context.Context, *FeedVersionRequest) (*FeedVersionPermissionsResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedVersionAddPermission(context.Context, *FeedVersionModifyPermissionRequest) (*FeedVersionSaveResponse, error) { return nil, nil }
func (UnimplementedCheckerServer) FeedVersionRemovePermission(context.Context, *FeedVersionModifyPermissionRequest) (*FeedVersionSaveResponse, error) { return nil, nil }

// UnsafeCheckerServer is kept for compile compatibility with azchecker.Checker struct.
// Will be removed in Phase 2.
type UnsafeCheckerServer interface {
	mustEmbedUnimplementedCheckerServer()
}
