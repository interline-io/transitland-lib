package azchecker

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"testing"

	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/auth0"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/auth/fga"
	"github.com/interline-io/transitland-lib/server/dbutil"
)

// For less typing

type Action = authz.Action
type ObjectType = authz.ObjectType
type Relation = authz.Relation

var FeedType = authz.FeedType
var UserType = authz.UserType
var TenantType = authz.TenantType
var GroupType = authz.GroupType
var FeedVersionType = authz.FeedVersionType

var ViewerRelation = authz.ViewerRelation
var MemberRelation = authz.MemberRelation
var AdminRelation = authz.AdminRelation
var ManagerRelation = authz.ManagerRelation
var ParentRelation = authz.ParentRelation
var EditorRelation = authz.EditorRelation

var CanEdit = authz.CanEdit
var CanView = authz.CanView
var CanCreateFeedVersion = authz.CanCreateFeedVersion
var CanDeleteFeedVersion = authz.CanDeleteFeedVersion
var CanCreateFeed = authz.CanCreateFeed
var CanDeleteFeed = authz.CanDeleteFeed
var CanSetGroup = authz.CanSetGroup
var CanCreateOrg = authz.CanCreateOrg
var CanEditMembers = authz.CanEditMembers
var CanDeleteOrg = authz.CanDeleteOrg
var CanSetTenant = authz.CanSetTenant

type EntityKey = authz.EntityKey
type TupleKey = authz.TupleKey

var ErrUnauthorized = authz.ErrUnauthorized

type UserProvider interface {
	Users(context.Context, string) ([]authn.User, error)
	UserByID(context.Context, string) (authn.User, error)
}

type FGAProvider interface {
	Check(context.Context, TupleKey, ...TupleKey) (bool, error)
	ListObjects(context.Context, TupleKey) ([]TupleKey, error)
	GetObjectTuples(context.Context, TupleKey) ([]TupleKey, error)
	WriteTuple(context.Context, TupleKey) error
	SetExclusiveSubjectRelation(context.Context, TupleKey, ...Relation) error
	SetExclusiveRelation(context.Context, TupleKey) error
	DeleteTuple(context.Context, TupleKey) error
}

type CheckerConfig struct {
	Auth0Domain       string
	Auth0ClientID     string
	Auth0ClientSecret string
	Auth0Connection   string
	FGAStoreID        string
	FGAModelID        string
	FGAEndpoint       string
	FGALoadModelFile  string
	FGALoadTestData   []TupleKey
	GlobalAdmin       string
}

type Checker struct {
	userClient   UserProvider
	fgaClient    FGAProvider
	db           sqlx.Ext
	globalAdmins []string
}

// Compile-time check that Checker implements authz.PermissionManager
var _ authz.PermissionManager = (*Checker)(nil)

func NewCheckerFromConfig(ctx context.Context, cfg CheckerConfig, db sqlx.Ext) (*Checker, error) {
	var userClient UserProvider
	userClient = NewMockUserProvider()
	var fgaClient FGAProvider
	fgaClient = NewMockFGAClient()

	// Use Auth0 if configured
	if cfg.Auth0Domain != "" {
		auth0Client, err := auth0.NewAuth0Client(cfg.Auth0Domain, cfg.Auth0ClientID, cfg.Auth0ClientSecret)
		auth0Client.Connection = cfg.Auth0Connection
		if err != nil {
			return nil, err
		}
		userClient = auth0Client
	}

	// Use FGA if configured
	if cfg.FGAEndpoint != "" {
		fgac, err := fga.NewFGAClient(cfg.FGAEndpoint, cfg.FGAStoreID, cfg.FGAModelID)
		if err != nil {
			return nil, err
		}
		fgaClient = fgac
		// Create test FGA environment
		if cfg.FGALoadModelFile != "" {
			if cfg.FGAStoreID == "" {
				if _, err := fgac.CreateStore(ctx, "test"); err != nil {
					return nil, err
				}
			}
			if _, err := fgac.CreateModel(ctx, cfg.FGALoadModelFile); err != nil {
				return nil, err
			}
		}
		// Add test data
		for _, tk := range cfg.FGALoadTestData {
			ltk, found, err := ekLookup(db, tk)
			if !found {
				log.For(ctx).Info().Msgf("warning, tuple entities not found in database: %s", tk.String())
			}
			if err != nil {
				return nil, err
			}
			if err := fgaClient.WriteTuple(ctx, ltk); err != nil {
				return nil, err
			}
		}
	}

	checker := NewChecker(userClient, fgaClient, db)
	if cfg.GlobalAdmin != "" {
		checker.globalAdmins = append(checker.globalAdmins, cfg.GlobalAdmin)
	}
	return checker, nil
}

func NewChecker(n UserProvider, p FGAProvider, db sqlx.Ext) *Checker {
	return &Checker{
		userClient: n,
		fgaClient:  p,
		db:         db,
	}
}

// ///////////////////
// USERS
// ///////////////////

func (c *Checker) UserList(ctx context.Context, req *authz.UserListRequest) (*authz.UserListResponse, error) {
	// TODO: filter users
	users, err := c.userClient.Users(ctx, req.GetQ())
	if err != nil {
		return nil, err
	}
	var ret []*authz.User
	for _, user := range users {
		ret = append(ret, newAzpbUser(user))
	}
	return &authz.UserListResponse{Users: ret}, nil
}

func (c *Checker) User(ctx context.Context, req *authz.UserRequest) (*authz.UserResponse, error) {
	// Special case "*"
	if req.Id == "*" {
		user := &authz.User{Id: "*", Name: "All users"}
		return &authz.UserResponse{User: user}, nil
	}
	// TODO: filter users
	user, err := c.userClient.UserByID(ctx, req.GetId())
	if user == nil || err != nil {
		return nil, ErrUnauthorized
	}
	return &authz.UserResponse{User: newAzpbUser(user)}, err
}

// ///////////////////
// DB helpers
// ///////////////////

func (c *Checker) getTenants(ctx context.Context, ids []int64) ([]*authz.Tenant, error) {
	return getEntities[*authz.Tenant](ctx, c.db, ids, "tl_tenants", "id", "coalesce(tenant_name,'') as name")
}

func (c *Checker) getGroups(ctx context.Context, ids []int64) ([]*authz.Group, error) {
	return getEntities[*authz.Group](ctx, c.db, ids, "tl_groups", "id", "coalesce(group_name,'') as name")
}

// ///////////////////
// Admin-specific methods (DB writes, not expressible through generic interface)
// ///////////////////

func (c *Checker) TenantSave(ctx context.Context, req *authz.TenantSaveRequest) (*authz.TenantSaveResponse, error) {
	t := req.GetTenant()
	tenantId := t.GetId()
	ref := authz.ObjectRef{Type: TenantType, ID: tenantId}
	if ok, err := c.Check(ctx, ref, CanEdit); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUnauthorized
	}
	if exists, _ := c.entityExists(ctx, ref); !exists {
		return nil, errors.New("not found")
	}
	newName := t.GetName()
	log.For(ctx).Trace().Str("tenantName", newName).Int64("id", tenantId).Msg("TenantSave")
	_, err := sq.StatementBuilder.
		RunWith(c.db).
		PlaceholderFormat(sq.Dollar).
		Update("tl_tenants").
		SetMap(map[string]any{
			"tenant_name": newName,
		}).
		Where("id = ?", tenantId).Exec()
	return &authz.TenantSaveResponse{}, err
}

func (c *Checker) TenantCreateGroup(ctx context.Context, req *authz.TenantCreateGroupRequest) (*authz.GroupSaveResponse, error) {
	tenantId := req.GetId()
	groupName := req.GetGroup().GetName()
	ref := authz.ObjectRef{Type: TenantType, ID: tenantId}
	if ok, err := c.Check(ctx, ref, CanCreateOrg); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUnauthorized
	}
	if exists, _ := c.entityExists(ctx, ref); !exists {
		return nil, errors.New("not found")
	}
	log.For(ctx).Trace().Str("groupName", groupName).Int64("id", tenantId).Msg("TenantCreateGroup")
	groupId := int64(0)
	err := sq.StatementBuilder.
		RunWith(c.db).
		PlaceholderFormat(sq.Dollar).
		Insert("tl_groups").
		Columns("group_name").
		Values(groupName).
		Suffix(`RETURNING "id"`).
		QueryRow().
		Scan(&groupId)
	if err != nil {
		return nil, err
	}
	addTk := authz.NewTupleKey().WithSubjectID(TenantType, tenantId).WithObjectID(GroupType, groupId).WithRelation(ParentRelation)
	if err := c.fgaClient.WriteTuple(ctx, addTk); err != nil {
		return nil, err
	}
	return &authz.GroupSaveResponse{Group: &authz.Group{Id: groupId}}, err
}

func (c *Checker) GroupSave(ctx context.Context, req *authz.GroupSaveRequest) (*authz.GroupSaveResponse, error) {
	group := req.GetGroup()
	groupId := group.GetId()
	newName := group.GetName()
	ref := authz.ObjectRef{Type: GroupType, ID: groupId}
	if ok, err := c.Check(ctx, ref, CanEdit); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUnauthorized
	}
	if exists, _ := c.entityExists(ctx, ref); !exists {
		return nil, errors.New("not found")
	}
	log.For(ctx).Trace().Str("groupName", newName).Int64("id", groupId).Msg("GroupSave")
	_, err := sq.StatementBuilder.
		RunWith(c.db).
		PlaceholderFormat(sq.Dollar).
		Update("tl_groups").
		SetMap(map[string]any{
			"group_name": newName,
		}).
		Where("id = ?", groupId).Exec()
	return &authz.GroupSaveResponse{}, err
}

// ///////////////////
// New generic Checker / PermissionManager interface
// ///////////////////

// actionsForType returns the actions to check for each object type.
func actionsForType(t ObjectType) []Action {
	switch t {
	case TenantType:
		return []Action{CanView, CanEdit, CanEditMembers, CanCreateOrg, CanDeleteOrg}
	case GroupType:
		return []Action{CanView, CanEdit, CanEditMembers, CanCreateFeed, CanDeleteFeed}
	case FeedType:
		return []Action{CanView, CanEdit, CanSetGroup, CanCreateFeedVersion, CanDeleteFeedVersion}
	case FeedVersionType:
		return []Action{CanView, CanEdit, CanEditMembers}
	}
	return nil
}

// exclusiveRelationsForType returns the mutually exclusive relations for AddPermission.
func exclusiveRelationsForType(t ObjectType) []Relation {
	switch t {
	case TenantType:
		return []Relation{MemberRelation, AdminRelation}
	case GroupType, FeedVersionType:
		return []Relation{ViewerRelation, EditorRelation, ManagerRelation}
	}
	return nil
}

// setParentActionForType returns the action required to set a parent on this type.
func setParentActionForType(t ObjectType) Action {
	switch t {
	case GroupType:
		return CanSetTenant
	case FeedType:
		return CanSetGroup
	}
	return Action(0)
}

func (c *Checker) Me(ctx context.Context) (*authz.UserInfo, error) {
	user := authn.ForContext(ctx)
	if user == nil || user.ID() == "" {
		return nil, ErrUnauthorized
	}

	// Direct groups
	var directGroupIds []int64
	checkTk := authz.NewTupleKey().
		WithSubject(authz.UserType, user.ID()).
		WithObject(authz.GroupType, "")
	groupTuples, err := c.fgaClient.GetObjectTuples(ctx, checkTk)
	if err != nil {
		return nil, err
	}
	for _, groupTuple := range groupTuples {
		directGroupIds = append(directGroupIds, groupTuple.Object.ID())
	}
	directGroups, err := c.getGroups(ctx, directGroupIds)
	if err != nil {
		return nil, err
	}

	// Expanded groups
	expandedGroupIds, err := c.listCtxObjectRelations(ctx, GroupType, ViewerRelation)
	if err != nil {
		return nil, err
	}
	expandedGroups, err := c.getGroups(ctx, expandedGroupIds)
	if err != nil {
		return nil, err
	}

	extData := map[string]string{}
	if gkData, ok := user.GetExternalData("gatekeeper"); ok {
		extData["gatekeeper"] = gkData
	}

	// Convert groups
	var dg []authz.Group
	for _, g := range directGroups {
		if g != nil {
			dg = append(dg, *g)
		}
	}
	var eg []authz.Group
	for _, g := range expandedGroups {
		if g != nil {
			eg = append(eg, *g)
		}
	}

	return &authz.UserInfo{
		ID:             user.ID(),
		Name:           user.Name(),
		Email:          user.Email(),
		Roles:          user.Roles(),
		Groups:         dg,
		ExpandedGroups: eg,
		ExternalData:   extData,
	}, nil
}

func (c *Checker) IsGlobalAdmin(ctx context.Context) (bool, error) {
	return c.checkGlobalAdmin(authn.ForContext(ctx)), nil
}

func (c *Checker) ListObjects(ctx context.Context, objType ObjectType) ([]authz.ObjectRef, error) {
	ids, err := c.listCtxObjects(ctx, objType, CanView)
	if err != nil {
		return nil, err
	}
	refs := make([]authz.ObjectRef, len(ids))
	for i, id := range ids {
		refs[i] = authz.ObjectRef{Type: objType, ID: id}
	}
	return refs, nil
}

func (c *Checker) Check(ctx context.Context, obj authz.ObjectRef, action Action) (bool, error) {
	ctxTuples := c.contextualTuples(ctx, obj)
	return c.checkAction(ctx, action, newEntityID(obj.Type, obj.ID), ctxTuples...)
}

func (c *Checker) ObjectPermissions(ctx context.Context, obj authz.ObjectRef) (*authz.ObjectPermissions, error) {
	entKey := newEntityID(obj.Type, obj.ID)
	ctxTuples := c.contextualTuples(ctx, obj)

	// Check view access
	if err := c.checkActionOrError(ctx, CanView, entKey, ctxTuples...); err != nil {
		return nil, err
	}

	// Verify entity exists in DB (global admins pass the view check above
	// but should still get a not-found error for non-existent entities)
	if exists, err := c.entityExists(ctx, obj); err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.New("not found")
	}

	ret := &authz.ObjectPermissions{
		Ref:     obj,
		Actions: authz.ActionSet{},
	}

	// Check all actions relevant to this type
	for _, action := range actionsForType(obj.Type) {
		ret.Actions[action], _ = c.checkAction(ctx, action, entKey, ctxTuples...)
	}
	// Special case: CanSetTenant on groups is global-admin only
	if obj.Type == GroupType {
		ret.Actions[CanSetTenant] = c.ctxIsGlobalAdmin(ctx)
	}

	// Get tuples — subjects + parent
	tuples, _ := c.getObjectTuples(ctx, entKey, ctxTuples...)
	for _, tk := range tuples {
		if tk.Relation == ParentRelation {
			ret.Parent = &authz.ObjectRef{Type: tk.Subject.Type, ID: tk.Subject.ID()}
		} else {
			ret.Subjects = append(ret.Subjects, authz.SubjectRef{
				Subject:  tk.Subject,
				Relation: tk.Relation,
			})
		}
	}
	c.hydrateSubjectRefs(ctx, ret.Subjects)

	// Children
	if childType, ok := childTypeForType(obj.Type); ok {
		childIds, _ := c.listSubjectRelations(ctx, entKey, childType, ParentRelation)
		for _, id := range childIds {
			ret.Children = append(ret.Children, authz.ObjectRef{Type: childType, ID: id})
		}
	}

	// Batch hydrate names: self + parent + children
	toHydrate := []authz.ObjectRef{obj}
	if ret.Parent != nil {
		toHydrate = append(toHydrate, *ret.Parent)
	}
	toHydrate = append(toHydrate, ret.Children...)
	c.hydrateObjectRefs(ctx, toHydrate)
	// Apply back
	ret.Ref.Name = toHydrate[0].Name
	ret.Name = toHydrate[0].Name
	idx := 1
	if ret.Parent != nil {
		ret.Parent.Name = toHydrate[idx].Name
		idx++
	}
	for i := range ret.Children {
		ret.Children[i].Name = toHydrate[idx].Name
		idx++
	}

	return ret, nil
}

func (c *Checker) SetParent(ctx context.Context, child authz.ObjectRef, parent authz.ObjectRef) error {
	action := setParentActionForType(child.Type)
	if action == Action(0) {
		return errors.New("set parent not supported for this type")
	}
	// Verify view access before checking specific action
	if ok, err := c.Check(ctx, child, CanView); err != nil {
		return err
	} else if !ok {
		return ErrUnauthorized
	}
	ok, err := c.Check(ctx, child, action)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnauthorized
	}
	tk := authz.NewTupleKey().WithSubjectID(parent.Type, parent.ID).WithObjectID(child.Type, child.ID).WithRelation(ParentRelation)
	return c.fgaClient.SetExclusiveRelation(ctx, tk)
}

func (c *Checker) AddPermission(ctx context.Context, obj authz.ObjectRef, subject EntityKey, relation Relation) error {
	if ok, err := c.Check(ctx, obj, CanView); err != nil {
		return err
	} else if !ok {
		return ErrUnauthorized
	}
	if exists, _ := c.entityExists(ctx, obj); !exists {
		return errors.New("not found")
	}
	ok, err := c.Check(ctx, obj, CanEditMembers)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnauthorized
	}
	tk := TupleKey{Subject: subject, Object: authz.NewEntityKey(obj.Type, strconv.FormatInt(obj.ID, 10)), Relation: relation}
	exclusiveRels := exclusiveRelationsForType(obj.Type)
	if len(exclusiveRels) > 0 {
		return c.fgaClient.SetExclusiveSubjectRelation(ctx, tk, exclusiveRels...)
	}
	return c.fgaClient.WriteTuple(ctx, tk)
}

func (c *Checker) RemovePermission(ctx context.Context, obj authz.ObjectRef, subject EntityKey, relation Relation) error {
	if ok, err := c.Check(ctx, obj, CanView); err != nil {
		return err
	} else if !ok {
		return ErrUnauthorized
	}
	if exists, _ := c.entityExists(ctx, obj); !exists {
		return errors.New("not found")
	}
	ok, err := c.Check(ctx, obj, CanEditMembers)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnauthorized
	}
	tk := TupleKey{Subject: subject, Object: authz.NewEntityKey(obj.Type, strconv.FormatInt(obj.ID, 10)), Relation: relation}
	return c.fgaClient.DeleteTuple(ctx, tk)
}

// contextualTuples builds the contextual tuples needed for FGA checks.
// Feed versions need a parent tuple pointing to their feed.
func (c *Checker) contextualTuples(ctx context.Context, obj authz.ObjectRef) []TupleKey {
	if obj.Type != FeedVersionType {
		return nil
	}
	feedId := c.lookupFeedIDForVersion(obj.ID)
	if feedId == 0 {
		return nil
	}
	return []TupleKey{
		authz.NewTupleKey().WithObjectID(FeedVersionType, obj.ID).WithSubjectID(FeedType, feedId).WithRelation(ParentRelation),
	}
}

// lookupFeedIDForVersion gets the feed_id for a feed_version from the DB.
func (c *Checker) lookupFeedIDForVersion(fvid int64) int64 {
	var feedId int64
	if err := sqlx.Get(c.db, &feedId, "SELECT feed_id FROM feed_versions WHERE id = $1", fvid); err != nil {
		return 0
	}
	return feedId
}

// entityExists checks if an entity exists in the DB.
func (c *Checker) entityExists(ctx context.Context, obj authz.ObjectRef) (bool, error) {
	var table string
	switch obj.Type {
	case TenantType:
		table = "tl_tenants"
	case GroupType:
		table = "tl_groups"
	case FeedType:
		table = "current_feeds"
	case FeedVersionType:
		table = "feed_versions"
	default:
		return false, nil
	}
	var exists bool
	q := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Select("true").From(table).Where(sq.Eq{"id": obj.ID})
	err := dbutil.Get(ctx, c.db, q, &exists)
	if err != nil {
		return false, nil // not found
	}
	return exists, nil
}

// hydrateObjectRefs populates Name on a slice of ObjectRef, batched by type.
func (c *Checker) hydrateObjectRefs(ctx context.Context, refs []authz.ObjectRef) {
	// Group refs by type
	byType := map[ObjectType][]int{}
	for _, ref := range refs {
		byType[ref.Type] = append(byType[ref.Type], int(ref.ID))
	}
	// Batch lookup names per type
	names := map[ObjectType]map[int64]string{}
	for objType, ids := range byType {
		names[objType] = c.lookupNames(ctx, objType, ids)
	}
	// Apply
	for i := range refs {
		if m, ok := names[refs[i].Type]; ok {
			refs[i].Name = m[refs[i].ID]
		}
	}
}

// lookupNames fetches display names for a batch of IDs of the same type.
func (c *Checker) lookupNames(ctx context.Context, objType ObjectType, ids []int) map[int64]string {
	type row struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	if len(ids) == 0 {
		return nil
	}
	var table, nameCol string
	switch objType {
	case TenantType:
		table, nameCol = "tl_tenants", "coalesce(tenant_name,'') as name"
	case GroupType:
		table, nameCol = "tl_groups", "coalesce(group_name,'') as name"
	case FeedType:
		table, nameCol = "current_feeds", "coalesce(name,'') as name"
	case FeedVersionType:
		table, nameCol = "feed_versions", "coalesce(name,'') as name"
	default:
		return nil
	}
	q := sq.StatementBuilder.Select("id", nameCol).From(table).Where(sq.Eq{"id": ids})
	var rows []row
	if err := dbutil.Select(ctx, c.db, q, &rows); err != nil {
		return nil
	}
	result := make(map[int64]string, len(rows))
	for _, r := range rows {
		result[r.ID] = r.Name
	}
	return result
}

// childTypeForType returns the child type for listing children.
func childTypeForType(t ObjectType) (ObjectType, bool) {
	switch t {
	case TenantType:
		return GroupType, true
	case GroupType:
		return FeedType, true
	}
	return 0, false
}

// hydrateSubjectRefs populates display names on subject refs.
func (c *Checker) hydrateSubjectRefs(ctx context.Context, refs []authz.SubjectRef) {
	for i, v := range refs {
		switch v.Subject.Type {
		case TenantType:
			if t, _ := c.getTenants(ctx, []int64{v.Subject.ID()}); len(t) > 0 && t[0] != nil {
				refs[i].Name = t[0].Name
			}
		case GroupType:
			if t, _ := c.getGroups(ctx, []int64{v.Subject.ID()}); len(t) > 0 && t[0] != nil {
				refs[i].Name = t[0].Name
			}
		case UserType:
			if t, err := c.User(ctx, &authz.UserRequest{Id: v.Subject.Name}); err == nil && t != nil && t.User != nil {
				refs[i].Name = t.User.Name
			}
		}
	}
}

// ///////////////////
// internal
// ///////////////////

func (c *Checker) listCtxObjects(ctx context.Context, objectType ObjectType, action Action) ([]int64, error) {
	checkUser := authn.ForContext(ctx)
	if checkUser == nil {
		return nil, nil
	}
	tk := authz.NewTupleKey().WithUser(checkUser.ID()).WithObject(objectType, "").WithAction(action)
	objTks, err := c.fgaClient.ListObjects(ctx, tk)
	if err != nil {
		return nil, err
	}
	var ret []int64
	for _, tk := range objTks {
		ret = append(ret, tk.Object.ID())
	}
	return ret, nil
}

func (c *Checker) listCtxObjectRelations(ctx context.Context, objectType ObjectType, rel Relation) ([]int64, error) {
	checkUser := authn.ForContext(ctx)
	if checkUser == nil {
		return nil, nil
	}
	tk := authz.NewTupleKey().WithUser(checkUser.ID()).WithObject(objectType, "").WithRelation(rel)
	objTks, err := c.fgaClient.ListObjects(ctx, tk)
	if err != nil {
		return nil, err
	}
	var ret []int64
	for _, tk := range objTks {
		ret = append(ret, tk.Object.ID())
	}
	return ret, nil
}

func (c *Checker) listSubjectRelations(ctx context.Context, sub EntityKey, objectType ObjectType, relation Relation) ([]int64, error) {
	tk := authz.NewTupleKey().WithSubject(sub.Type, sub.Name).WithObject(objectType, "").WithRelation(relation)
	rels, err := c.fgaClient.ListObjects(ctx, tk)
	if err != nil {
		return nil, err
	}
	var ret []int64
	for _, v := range rels {
		ret = append(ret, v.Object.ID())
	}
	return ret, nil
}

func (c *Checker) getObjectTuples(ctx context.Context, obj EntityKey, ctxtk ...TupleKey) ([]TupleKey, error) {
	return c.fgaClient.GetObjectTuples(ctx, authz.NewTupleKey().WithObject(obj.Type, obj.Name))
}

func (c *Checker) checkActionOrError(ctx context.Context, checkAction Action, obj EntityKey, ctxtk ...TupleKey) error {
	ok, err := c.checkAction(ctx, checkAction, obj, ctxtk...)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUnauthorized
	}
	return nil
}

func (c *Checker) checkAction(ctx context.Context, checkAction Action, obj EntityKey, ctxtk ...TupleKey) (bool, error) {
	checkUser := authn.ForContext(ctx)
	if checkUser == nil {
		return false, nil
	}
	userName := checkUser.ID()
	if c.checkGlobalAdmin(checkUser) {
		log.For(ctx).Debug().Str("check_user", userName).Str("obj", obj.String()).Str("check_action", checkAction.String()).Msg("global admin action")
		return true, nil
	}
	checkTk := authz.NewTupleKey().WithUser(userName).WithObject(obj.Type, obj.Name).WithAction(checkAction)
	ret, err := c.fgaClient.Check(ctx, checkTk, ctxtk...)
	log.For(ctx).Trace().Str("tk", checkTk.String()).Bool("result", ret).Err(err).Msg("checkAction")
	return ret, err
}

func (c *Checker) ctxIsGlobalAdmin(ctx context.Context) bool {
	checkUser := authn.ForContext(ctx)
	if checkUser == nil {
		return false
	}
	return c.checkGlobalAdmin(checkUser)
}

func (c *Checker) checkGlobalAdmin(checkUser authn.User) bool {
	if c == nil {
		return false
	}
	if checkUser == nil {
		return false
	}
	if checkUser.HasRole("admin") {
		return true
	}
	userName := checkUser.ID()
	for _, v := range c.globalAdmins {
		if v == userName {
			return true
		}
	}
	return false
}

// Helpers

type hasId interface {
	GetId() int64
}

func checkIds[T hasId](ents []T, ids []int64) error {
	if len(ents) != len(ids) {
		return errors.New("not found")
	}
	check := map[int64]bool{}
	for _, ent := range ents {
		check[ent.GetId()] = true
	}
	for _, id := range ids {
		if _, ok := check[id]; !ok {
			return errors.New("not found")
		}
	}
	return nil
}

func getEntities[T hasId](ctx context.Context, db sqlx.Ext, ids []int64, table string, cols ...string) ([]T, error) {
	var t []T
	q := sq.StatementBuilder.Select(cols...).From(table).Where(sq.Eq{"id": ids})
	if err := dbutil.Select(ctx, db, q, &t); err != nil {
		log.For(ctx).Trace().Err(err)
		return nil, err
	}
	if err := checkIds(t, ids); err != nil {
		return nil, err
	}
	return t, nil
}

func first[T any](v []T) T {
	var xt T
	if len(v) > 0 {
		return v[0]
	}
	return xt
}

// todo: rename to dbTestTupleLookup and make arg TestTuple
func dbTupleLookup(t testing.TB, dbx sqlx.Ext, tk TupleKey) TupleKey {
	var err error
	var found bool
	tk.Subject, found, err = dbNameToEntityKey(dbx, tk.Subject)
	if !found && t != nil {
		t.Logf("lookup warning: %s not found", tk.Subject.String())
	}
	if err != nil {
		t.Log(err)
	}
	tk.Object, found, err = dbNameToEntityKey(dbx, tk.Object)
	if !found && t != nil {
		t.Logf("lookup warning: %s not found", tk.Object.String())
	}
	if err != nil {
		t.Log(err)
	}
	return tk
}

func ekLookup(dbx sqlx.Ext, tk TupleKey) (TupleKey, bool, error) {
	var err error
	var found1 bool
	var found2 bool
	tk.Subject, found1, err = dbNameToEntityKey(dbx, tk.Subject)
	if err != nil {
		return tk, false, err
	}
	tk.Object, found2, err = dbNameToEntityKey(dbx, tk.Object)
	if err != nil {
		return tk, false, err
	}
	return tk, found1 && found2, nil
}

func dbNameToEntityKey(dbx sqlx.Ext, ek EntityKey) (EntityKey, bool, error) {
	if ek.Name == "" {
		return ek, false, nil
	}
	nsplit := strings.Split(ek.Name, "#")
	oname := nsplit[0]
	nname := ek.Name
	var err error
	switch ek.Type {
	case TenantType:
		err = sqlx.Get(dbx, &nname, "select id from tl_tenants where tenant_name = $1", oname)
	case GroupType:
		err = sqlx.Get(dbx, &nname, "select id from tl_groups where group_name = $1", oname)
	case FeedType:
		err = sqlx.Get(dbx, &nname, "select id from current_feeds where onestop_id = $1", oname)
	case FeedVersionType:
		err = sqlx.Get(dbx, &nname, "select id from feed_versions where sha1 = $1", oname)
	case UserType:
	}
	found := false
	if err == sql.ErrNoRows {
		err = nil
	} else {
		found = true
	}
	if err != nil {
		return ek, found, err
	}
	nsplit[0] = nname
	ek.Name = strings.Join(nsplit, "#")
	return ek, found, nil
}

func newEntityID(t ObjectType, id int64) EntityKey {
	return authz.NewEntityKey(t, strconv.Itoa(int(id)))
}

func newAzpbUser(u authn.User) *authz.User {
	return &authz.User{Id: u.ID(), Name: u.Name(), Email: u.Email()}
}
