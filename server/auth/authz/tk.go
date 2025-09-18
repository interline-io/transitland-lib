package authz

import (
	"errors"
	"fmt"
	"strconv"
)

var ErrUnauthorized = errors.New("unauthorized")

// For convenience

var FeedType = ObjectType_feed
var UserType = ObjectType_user
var TenantType = ObjectType_tenant
var GroupType = ObjectType_org
var FeedVersionType = ObjectType_feed_version

var ViewerRelation = Relation_viewer
var MemberRelation = Relation_member
var AdminRelation = Relation_admin
var ManagerRelation = Relation_manager
var ParentRelation = Relation_parent
var EditorRelation = Relation_editor

var CanEdit = Action_can_edit
var CanView = Action_can_view
var CanCreateFeedVersion = Action_can_create_feed_version
var CanDeleteFeedVersion = Action_can_delete_feed_version
var CanCreateFeed = Action_can_create_feed
var CanDeleteFeed = Action_can_delete_feed
var CanSetGroup = Action_can_set_group
var CanCreateOrg = Action_can_create_org
var CanEditMembers = Action_can_edit_members
var CanDeleteOrg = Action_can_delete_org
var CanSetTenant = Action_can_set_tenant

func RelationString(v string) (Relation, error) {
	if a, ok := Relation_value[v]; ok {
		return Relation(a), nil
	}
	return Relation(0), errors.New("invalid relation")
}

func ActionString(v string) (Action, error) {
	if a, ok := Action_value[v]; ok {
		return Action(a), nil
	}
	return Action(0), errors.New("invalid action")
}

func ObjectTypeString(v string) (ObjectType, error) {
	if a, ok := ObjectType_value[v]; ok {
		return ObjectType(a), nil
	}
	return ObjectType(0), errors.New("invalid object type")
}

func IsRelation(v Relation) bool {
	_, ok := Relation_name[int32(v)]
	return ok && v > 0
}

func IsAction(v Action) bool {
	_, ok := Action_name[int32(v)]
	return ok && v > 0
}

func IsObjectType(v ObjectType) bool {
	_, ok := ObjectType_name[int32(v)]
	return ok && v > 0
}

func NewEntityRelation(ek EntityKey, rel Relation) *EntityRelation {
	ur := EntityRelation{
		Type:        ek.Type,
		Id:          ek.Name,
		RefRelation: ek.RefRel,
		Relation:    rel,
	}
	return &ur
}

func (er *EntityRelation) Int64() int64 {
	a, _ := strconv.Atoi(er.Id)
	return int64(a)
}

func (er *EntityRelation) WithObject(ek EntityKey) TupleKey {
	tk := NewTupleKey().
		WithSubject(er.GetType(), er.GetId()).
		WithObjectID(ek.Type, ek.ID()).
		WithRelation(er.GetRelation())
	if er.RefRelation > 0 {
		tk.Subject = tk.Subject.WithRefRel(er.RefRelation)
	}
	return tk

}

type EntityKey struct {
	Type   ObjectType `json:"type"`
	Name   string     `json:"name"`
	RefRel Relation   `json:"ref_rel"`
}

func NewEntityKey(t ObjectType, name string) EntityKey {
	return EntityKey{Type: t, Name: name}
}

func (ek EntityKey) Equals(other EntityKey) bool {
	return ek.Type == other.Type &&
		ek.Name == other.Name &&
		ek.RefRel == other.RefRel
}

func (ek EntityKey) WithRefRel(r Relation) EntityKey {
	ek.RefRel = r
	return ek
}

func (ek EntityKey) ID() int64 {
	v, _ := strconv.Atoi(ek.Name)
	return int64(v)
}

func (ek EntityKey) String() string {
	if ek.Name == "" {
		return ek.Type.String()
	}
	if ek.RefRel > 0 {
		return fmt.Sprintf("%s:%s#%s", ek.Type.String(), ek.Name, ek.RefRel.String())
	}
	return fmt.Sprintf("%s:%s", ek.Type.String(), ek.Name)
}

type TupleKey struct {
	Subject  EntityKey
	Object   EntityKey
	Action   Action   `json:"action"`
	Relation Relation `json:"relation"`
}

func NewTupleKey() TupleKey { return TupleKey{} }

func (tk TupleKey) Equals(other TupleKey) bool {
	return tk.Subject.Equals(other.Subject) &&
		tk.Object.Equals(other.Object) &&
		tk.Action == other.Action &&
		tk.Relation == other.Relation
}

func (tk TupleKey) String() string {
	r := ""
	if IsRelation(tk.Relation) {
		r = "|relation:" + tk.Relation.String()
	} else if IsAction(tk.Action) {
		r = "|action:" + tk.Action.String()
	}
	return fmt.Sprintf("%s|%s%s", tk.Subject.String(), tk.Object.String(), r)
}

func (tk TupleKey) IsValid() bool {
	return tk.Validate() == nil
}

func (tk TupleKey) Validate() error {
	if tk.Subject.Name != "" && !IsObjectType(tk.Subject.Type) {
		return errors.New("invalid user type")
	}
	if tk.Object.Name != "" && !IsObjectType(tk.Object.Type) {
		return errors.New("invalid object type")
	}
	if tk.Subject.Name == "" && tk.Object.Name == "" {
		return errors.New("user name or object name is required")
	}
	if tk.Subject.Name != "" && tk.Object.Name != "" {
		if tk.Action == 0 && !IsRelation(tk.Relation) {
			return errors.New("invalid relation")
		}
		if tk.Relation == 0 && !IsAction(tk.Action) {
			return errors.New("invalid action")
		}
	}
	return nil
}

func (tk TupleKey) ActionOrRelation() string {
	if IsAction(tk.Action) {
		return tk.Action.String()
	} else if IsRelation(tk.Relation) {
		return tk.Relation.String()
	}
	return ""
}

func (tk TupleKey) WithUser(user string) TupleKey {
	return TupleKey{
		Subject:  NewEntityKey(UserType, user),
		Object:   tk.Object,
		Relation: tk.Relation,
		Action:   tk.Action,
	}
}

func (tk TupleKey) WithSubject(userType ObjectType, userName string) TupleKey {
	return TupleKey{
		Subject:  NewEntityKey(userType, userName),
		Object:   tk.Object,
		Relation: tk.Relation,
		Action:   tk.Action,
	}
}

func (tk TupleKey) WithSubjectID(userType ObjectType, userId int64) TupleKey {
	return tk.WithSubject(userType, strconv.Itoa(int(userId)))
}

func (tk TupleKey) WithObject(objectType ObjectType, objectName string) TupleKey {
	return TupleKey{
		Subject:  tk.Subject,
		Object:   NewEntityKey(objectType, objectName),
		Relation: tk.Relation,
		Action:   tk.Action,
	}
}

func (tk TupleKey) WithObjectID(objectType ObjectType, objectId int64) TupleKey {
	return tk.WithObject(objectType, strconv.Itoa(int(objectId)))
}

func (tk TupleKey) WithRelation(relation Relation) TupleKey {
	return TupleKey{
		Subject:  tk.Subject,
		Object:   tk.Object,
		Relation: relation,
		Action:   tk.Action,
	}
}

func (tk TupleKey) WithAction(action Action) TupleKey {
	return TupleKey{
		Subject:  tk.Subject,
		Object:   tk.Object,
		Relation: tk.Relation,
		Action:   action,
	}
}
