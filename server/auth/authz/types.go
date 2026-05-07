package authz

import "fmt"

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
	ObjectType_empty_object ObjectType = 0
	ObjectType_tenant       ObjectType = 1
	ObjectType_org          ObjectType = 2
	ObjectType_feed         ObjectType = 3
	ObjectType_feed_version ObjectType = 4
	ObjectType_user         ObjectType = 5
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
	"group":        2, // alias for "org" — the GraphQL API exposes "group"
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

// MarshalText implements encoding.TextMarshaler so ObjectType works as a JSON map key.
func (o ObjectType) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (o *ObjectType) UnmarshalText(text []byte) error {
	if v, ok := ObjectType_value[string(text)]; ok {
		*o = ObjectType(v)
		return nil
	}
	return fmt.Errorf("unknown object type: %s", text)
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

// MarshalText implements encoding.TextMarshaler so Relation works as a JSON map key.
func (r Relation) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (r *Relation) UnmarshalText(text []byte) error {
	if v, ok := Relation_value[string(text)]; ok {
		*r = Relation(v)
		return nil
	}
	return fmt.Errorf("unknown relation: %s", text)
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
// Request/Response types used by admin-specific methods on azchecker.Checker
//////

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
