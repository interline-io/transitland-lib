package azchecker

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-mw/auth/auth0"
	"github.com/interline-io/transitland-mw/auth/authn"
	"github.com/interline-io/transitland-mw/auth/authz"
	"github.com/interline-io/transitland-mw/auth/fga"
	"github.com/interline-io/transitland-mw/dbutil"
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
	authz.UnsafeCheckerServer
}

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

func (c *Checker) Me(ctx context.Context, req *authz.MeRequest) (*authz.MeResponse, error) {
	user := authn.ForContext(ctx)
	if user == nil || user.ID() == "" {
		return nil, ErrUnauthorized
	}

	// TODO: consider an explicit check to authn provider .GetUser,
	// however this requires a authn provider to be configured and not just the default.

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
	expandedGroupIds, err := c.listCtxObjectRelations(
		ctx,
		GroupType,
		ViewerRelation,
	)
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

	// Return
	ret := &authz.MeResponse{
		User:           newAzpbUser(user),
		Groups:         directGroups,
		Roles:          user.Roles(),
		ExpandedGroups: expandedGroups,
		ExternalData:   extData,
	}
	return ret, nil
}

func (c *Checker) CheckGlobalAdmin(ctx context.Context) (bool, error) {
	return c.checkGlobalAdmin(authn.ForContext(ctx)), nil
}

func (c *Checker) hydrateEntityRels(ctx context.Context, ers []*authz.EntityRelation) ([]*authz.EntityRelation, error) {
	// This is awful :( :(
	for i, v := range ers {
		if v.Type == TenantType {
			if t, _ := c.getTenants(ctx, []int64{v.Int64()}); len(t) > 0 && t[0] != nil {
				ers[i].Name = t[0].Name
			}
		} else if v.Type == GroupType {
			if t, _ := c.getGroups(ctx, []int64{v.Int64()}); len(t) > 0 && t[0] != nil {
				ers[i].Name = t[0].Name
			}
		} else if v.Type == UserType {
			if t, err := c.User(ctx, &authz.UserRequest{Id: v.Id}); err == nil && t != nil && t.User != nil {
				ers[i].Name = t.User.Name
			}
		}
	}
	return ers, nil
}

// ///////////////////
// TENANTS
// ///////////////////

func (c *Checker) getTenants(ctx context.Context, ids []int64) ([]*authz.Tenant, error) {
	return getEntities[*authz.Tenant](ctx, c.db, ids, "tl_tenants", "id", "coalesce(tenant_name,'') as name")
}

func (c *Checker) TenantList(ctx context.Context, req *authz.TenantListRequest) (*authz.TenantListResponse, error) {
	ids, err := c.listCtxObjects(ctx, TenantType, CanView)
	if err != nil {
		return nil, err
	}
	t, err := c.getTenants(ctx, ids)
	return &authz.TenantListResponse{Tenants: t}, err
}

func (c *Checker) Tenant(ctx context.Context, req *authz.TenantRequest) (*authz.TenantResponse, error) {
	tenantId := req.GetId()
	if err := c.checkActionOrError(ctx, CanView, newEntityID(TenantType, tenantId)); err != nil {
		return nil, err
	}
	t, err := c.getTenants(ctx, []int64{tenantId})
	return &authz.TenantResponse{Tenant: first(t)}, err
}

func (c *Checker) TenantPermissions(ctx context.Context, req *authz.TenantRequest) (*authz.TenantPermissionsResponse, error) {
	ent, err := c.Tenant(ctx, req)
	if err != nil {
		return nil, err
	}
	ret := &authz.TenantPermissionsResponse{
		Tenant:  ent.Tenant,
		Actions: &authz.TenantPermissionsResponse_Actions{},
		Users:   &authz.TenantPermissionsResponse_Users{},
	}

	// Actions
	entKey := newEntityID(TenantType, req.GetId())
	groupIds, _ := c.listSubjectRelations(ctx, entKey, GroupType, ParentRelation)
	ret.Groups, _ = c.getGroups(ctx, groupIds)
	ret.Actions.CanView, _ = c.checkAction(ctx, CanView, entKey)
	ret.Actions.CanEditMembers, _ = c.checkAction(ctx, CanEditMembers, entKey)
	ret.Actions.CanEdit, _ = c.checkAction(ctx, CanEdit, entKey)
	ret.Actions.CanCreateOrg, _ = c.checkAction(ctx, CanCreateOrg, entKey)
	ret.Actions.CanDeleteOrg, _ = c.checkAction(ctx, CanDeleteOrg, entKey)

	// Get tenant metadata
	tps, err := c.getObjectTuples(ctx, entKey)
	if err != nil {
		return nil, err
	}
	for _, tk := range tps {
		if tk.Relation == AdminRelation {
			ret.Users.Admins = append(ret.Users.Admins, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
		if tk.Relation == MemberRelation {
			ret.Users.Members = append(ret.Users.Members, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
	}
	ret.Users.Admins, _ = c.hydrateEntityRels(ctx, ret.Users.Admins)
	ret.Users.Members, _ = c.hydrateEntityRels(ctx, ret.Users.Members)
	return ret, nil
}

func (c *Checker) TenantSave(ctx context.Context, req *authz.TenantSaveRequest) (*authz.TenantSaveResponse, error) {
	t := req.GetTenant()
	tenantId := t.GetId()
	if check, err := c.TenantPermissions(ctx, &authz.TenantRequest{Id: tenantId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEdit {
		return nil, ErrUnauthorized
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

func (c *Checker) TenantAddPermission(ctx context.Context, req *authz.TenantModifyPermissionRequest) (*authz.TenantSaveResponse, error) {
	tenantId := req.GetId()
	if check, err := c.TenantPermissions(ctx, &authz.TenantRequest{Id: tenantId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(TenantType, tenantId))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", tenantId).Msg("TenantAddPermission")
	return &authz.TenantSaveResponse{}, c.fgaClient.SetExclusiveSubjectRelation(ctx, tk, MemberRelation, AdminRelation)
}

func (c *Checker) TenantRemovePermission(ctx context.Context, req *authz.TenantModifyPermissionRequest) (*authz.TenantSaveResponse, error) {
	tenantId := req.GetId()
	if check, err := c.TenantPermissions(ctx, &authz.TenantRequest{Id: tenantId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(TenantType, tenantId))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", tenantId).Msg("TenantRemovePermission")
	return &authz.TenantSaveResponse{}, c.fgaClient.DeleteTuple(ctx, tk)
}

func (c *Checker) TenantCreate(ctx context.Context, req *authz.TenantCreateRequest) (*authz.TenantSaveResponse, error) {
	return &authz.TenantSaveResponse{}, nil
}

func (c *Checker) TenantCreateGroup(ctx context.Context, req *authz.TenantCreateGroupRequest) (*authz.GroupSaveResponse, error) {
	tenantId := req.GetId()
	groupName := req.GetGroup().GetName()
	if check, err := c.TenantPermissions(ctx, &authz.TenantRequest{Id: tenantId}); err != nil {
		return nil, err
	} else if !check.Actions.CanCreateOrg {
		return nil, ErrUnauthorized
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

// ///////////////////
// GROUPS
// ///////////////////

func (c *Checker) getGroups(ctx context.Context, ids []int64) ([]*authz.Group, error) {
	return getEntities[*authz.Group](ctx, c.db, ids, "tl_groups", "id", "coalesce(group_name,'') as name")
}

func (c *Checker) GroupList(ctx context.Context, req *authz.GroupListRequest) (*authz.GroupListResponse, error) {
	ids, err := c.listCtxObjects(ctx, GroupType, CanView)
	if err != nil {
		return nil, err
	}
	t, err := c.getGroups(ctx, ids)
	return &authz.GroupListResponse{Groups: t}, err
}

func (c *Checker) Group(ctx context.Context, req *authz.GroupRequest) (*authz.GroupResponse, error) {
	groupId := req.GetId()
	if err := c.checkActionOrError(ctx, CanView, newEntityID(GroupType, groupId)); err != nil {
		return nil, err
	}
	t, err := c.getGroups(ctx, []int64{groupId})
	return &authz.GroupResponse{Group: first(t)}, err
}

func (c *Checker) GroupPermissions(ctx context.Context, req *authz.GroupRequest) (*authz.GroupPermissionsResponse, error) {
	groupId := req.GetId()
	ent, err := c.Group(ctx, req)
	if err != nil {
		return nil, err
	}
	ret := &authz.GroupPermissionsResponse{
		Group:   ent.Group,
		Users:   &authz.GroupPermissionsResponse_Users{},
		Actions: &authz.GroupPermissionsResponse_Actions{},
	}

	// Actions
	entKey := newEntityID(GroupType, groupId)
	ret.Actions.CanView, _ = c.checkAction(ctx, CanView, entKey)
	ret.Actions.CanEditMembers, _ = c.checkAction(ctx, CanEditMembers, entKey)
	ret.Actions.CanEdit, _ = c.checkAction(ctx, CanEdit, entKey)
	ret.Actions.CanCreateFeed, _ = c.checkAction(ctx, CanCreateFeed, entKey)
	ret.Actions.CanDeleteFeed, _ = c.checkAction(ctx, CanDeleteFeed, entKey)
	ret.Actions.CanSetTenant = c.ctxIsGlobalAdmin(ctx)

	// Get feeds
	feedIds, _ := c.listSubjectRelations(ctx, entKey, FeedType, ParentRelation)
	ret.Feeds, _ = c.getFeeds(ctx, feedIds)

	// Get group metadata
	tps, err := c.getObjectTuples(ctx, entKey)
	if err != nil {
		return nil, err
	}
	for _, tk := range tps {
		if tk.Relation == ParentRelation {
			ct, _ := c.Tenant(ctx, &authz.TenantRequest{Id: tk.Subject.ID()})
			ret.Tenant = ct.Tenant
		}
		if tk.Relation == ManagerRelation {
			ret.Users.Managers = append(ret.Users.Managers, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
		if tk.Relation == EditorRelation {
			ret.Users.Editors = append(ret.Users.Editors, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
		if tk.Relation == ViewerRelation {
			ret.Users.Viewers = append(ret.Users.Viewers, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
	}
	ret.Users.Managers, _ = c.hydrateEntityRels(ctx, ret.Users.Managers)
	ret.Users.Editors, _ = c.hydrateEntityRels(ctx, ret.Users.Editors)
	ret.Users.Viewers, _ = c.hydrateEntityRels(ctx, ret.Users.Viewers)
	return ret, nil
}

func (c *Checker) GroupSave(ctx context.Context, req *authz.GroupSaveRequest) (*authz.GroupSaveResponse, error) {
	group := req.GetGroup()
	groupId := group.GetId()
	newName := group.GetName()
	if check, err := c.GroupPermissions(ctx, &authz.GroupRequest{Id: groupId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEdit {
		return nil, ErrUnauthorized
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

func (c *Checker) GroupAddPermission(ctx context.Context, req *authz.GroupModifyPermissionRequest) (*authz.GroupSaveResponse, error) {
	groupId := req.GetId()
	if check, err := c.GroupPermissions(ctx, &authz.GroupRequest{Id: groupId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(GroupType, groupId))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", groupId).Msg("GroupAddPermission")
	return &authz.GroupSaveResponse{}, c.fgaClient.SetExclusiveSubjectRelation(ctx, tk, ViewerRelation, EditorRelation, ManagerRelation)
}

func (c *Checker) GroupRemovePermission(ctx context.Context, req *authz.GroupModifyPermissionRequest) (*authz.GroupSaveResponse, error) {
	groupId := req.GetId()
	if check, err := c.GroupPermissions(ctx, &authz.GroupRequest{Id: groupId}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(GroupType, groupId))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", groupId).Msg("GroupRemovePermission")
	return &authz.GroupSaveResponse{}, c.fgaClient.DeleteTuple(ctx, tk)
}

func (c *Checker) GroupSetTenant(ctx context.Context, req *authz.GroupSetTenantRequest) (*authz.GroupSetTenantResponse, error) {
	groupId := req.GetId()
	newTenantId := req.GetTenantId()
	if check, err := c.GroupPermissions(ctx, &authz.GroupRequest{Id: groupId}); err != nil {
		return nil, err
	} else if !check.Actions.CanSetTenant {
		return nil, ErrUnauthorized
	}
	tk := authz.NewTupleKey().WithSubjectID(TenantType, newTenantId).WithObjectID(GroupType, groupId).WithRelation(ParentRelation)
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", groupId).Msg("GroupSetTenant")
	return &authz.GroupSetTenantResponse{}, c.fgaClient.SetExclusiveRelation(ctx, tk)
}

// ///////////////////
// FEEDS
// ///////////////////

func (c *Checker) getFeeds(ctx context.Context, ids []int64) ([]*authz.Feed, error) {
	return getEntities[*authz.Feed](ctx, c.db, ids, "current_feeds", "id", "onestop_id", "coalesce(name,'') as name")
}

func (c *Checker) FeedList(ctx context.Context, req *authz.FeedListRequest) (*authz.FeedListResponse, error) {
	feedIds, err := c.listCtxObjects(ctx, FeedType, CanView)
	if err != nil {
		return nil, err
	}
	t, err := c.getFeeds(ctx, feedIds)
	return &authz.FeedListResponse{Feeds: t}, err
}

func (c *Checker) Feed(ctx context.Context, req *authz.FeedRequest) (*authz.FeedResponse, error) {
	feedId := req.GetId()
	if err := c.checkActionOrError(ctx, CanView, newEntityID(FeedType, feedId)); err != nil {
		return nil, err
	}
	t, err := c.getFeeds(ctx, []int64{feedId})
	return &authz.FeedResponse{Feed: first(t)}, err
}

func (c *Checker) FeedPermissions(ctx context.Context, req *authz.FeedRequest) (*authz.FeedPermissionsResponse, error) {
	ent, err := c.Feed(ctx, req)
	if err != nil {
		return nil, err
	}
	ret := &authz.FeedPermissionsResponse{
		Feed:    ent.Feed,
		Actions: &authz.FeedPermissionsResponse_Actions{},
	}

	// Actions
	entKey := newEntityID(FeedType, req.GetId())
	ret.Actions.CanView, _ = c.checkAction(ctx, CanView, entKey)
	ret.Actions.CanEdit, _ = c.checkAction(ctx, CanEdit, entKey)
	ret.Actions.CanSetGroup, _ = c.checkAction(ctx, CanSetGroup, entKey)
	ret.Actions.CanCreateFeedVersion, _ = c.checkAction(ctx, CanCreateFeedVersion, entKey)
	ret.Actions.CanDeleteFeedVersion, _ = c.checkAction(ctx, CanDeleteFeedVersion, entKey)

	// Get feed metadata
	tps, err := c.getObjectTuples(ctx, entKey)
	if err != nil {
		return nil, err
	}
	for _, tk := range tps {
		if tk.Relation == ParentRelation {
			ct, _ := c.Group(ctx, &authz.GroupRequest{Id: tk.Subject.ID()})
			ret.Group = ct.Group
		}
	}
	return ret, nil
}

func (c *Checker) FeedSetGroup(ctx context.Context, req *authz.FeedSetGroupRequest) (*authz.FeedSaveResponse, error) {
	feedId := req.GetId()
	newGroup := req.GetGroupId()
	if check, err := c.FeedPermissions(ctx, &authz.FeedRequest{Id: feedId}); err != nil {
		return nil, err
	} else if !check.Actions.CanSetGroup {
		return nil, ErrUnauthorized
	}
	tk := authz.NewTupleKey().WithSubjectID(GroupType, newGroup).WithObjectID(FeedType, feedId).WithRelation(ParentRelation)
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", feedId).Msg("FeedSetGroup")
	return &authz.FeedSaveResponse{}, c.fgaClient.SetExclusiveRelation(ctx, tk)
}

/////////////////////
// FEED VERSIONS
/////////////////////

func (c *Checker) getFeedVersions(ctx context.Context, ids []int64) ([]*authz.FeedVersion, error) {
	return getEntities[*authz.FeedVersion](ctx, c.db, ids, "feed_versions", "id", "feed_id", "sha1", "coalesce(name,'') as name")
}

func (c *Checker) FeedVersionList(ctx context.Context, req *authz.FeedVersionListRequest) (*authz.FeedVersionListResponse, error) {
	fvids, err := c.listCtxObjects(ctx, FeedVersionType, CanView)
	if err != nil {
		return nil, err
	}
	t, err := c.getFeedVersions(ctx, fvids)
	return &authz.FeedVersionListResponse{FeedVersions: t}, err
}

func (c *Checker) FeedVersion(ctx context.Context, req *authz.FeedVersionRequest) (*authz.FeedVersionResponse, error) {
	fvid := req.GetId()
	feedId := int64(0)
	// We need to get feed id before any other checks
	// If there is a "not found" error here, save it for after the global admin check
	// This is for consistency with other permission checks
	t, fvErr := c.getFeedVersions(ctx, []int64{fvid})
	fv := first(t)
	if fv != nil {
		feedId = fv.FeedId
	}
	ctxTk := authz.NewTupleKey().WithObjectID(FeedVersionType, fvid).WithSubjectID(FeedType, feedId).WithRelation(ParentRelation)
	if err := c.checkActionOrError(ctx, CanView, newEntityID(FeedVersionType, fvid), ctxTk); err != nil {
		return nil, err
	}
	// Now return deferred fvErr
	if fvErr != nil {
		return nil, fvErr
	}
	return &authz.FeedVersionResponse{FeedVersion: fv}, nil
}

func (c *Checker) FeedVersionPermissions(ctx context.Context, req *authz.FeedVersionRequest) (*authz.FeedVersionPermissionsResponse, error) {
	ent, err := c.FeedVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	ret := &authz.FeedVersionPermissionsResponse{
		FeedVersion: ent.FeedVersion,
		Users:       &authz.FeedVersionPermissionsResponse_Users{},
		Actions:     &authz.FeedVersionPermissionsResponse_Actions{},
	}

	// Actions
	ctxTk := authz.NewTupleKey().WithObjectID(FeedVersionType, ent.FeedVersion.Id).WithSubjectID(FeedType, ent.FeedVersion.FeedId).WithRelation(ParentRelation)
	entKey := newEntityID(FeedVersionType, req.GetId())
	ret.Actions.CanView, _ = c.checkAction(ctx, CanView, entKey, ctxTk)
	ret.Actions.CanEditMembers, _ = c.checkAction(ctx, CanEditMembers, entKey, ctxTk)
	ret.Actions.CanEdit, _ = c.checkAction(ctx, CanEdit, entKey, ctxTk)

	// Get fv metadata
	tps, err := c.getObjectTuples(ctx, entKey, ctxTk)
	if err != nil {
		return nil, err
	}
	for _, tk := range tps {
		if tk.Relation == EditorRelation {
			ret.Users.Editors = append(ret.Users.Editors, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
		if tk.Relation == ViewerRelation {
			ret.Users.Viewers = append(ret.Users.Viewers, authz.NewEntityRelation(tk.Subject, tk.Relation))
		}
	}
	ret.Users.Editors, _ = c.hydrateEntityRels(ctx, ret.Users.Editors)
	ret.Users.Viewers, _ = c.hydrateEntityRels(ctx, ret.Users.Viewers)
	return ret, nil
}

func (c *Checker) FeedVersionAddPermission(ctx context.Context, req *authz.FeedVersionModifyPermissionRequest) (*authz.FeedVersionSaveResponse, error) {
	fvid := req.GetId()
	if check, err := c.FeedVersionPermissions(ctx, &authz.FeedVersionRequest{Id: fvid}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(FeedVersionType, fvid))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", fvid).Msg("FeedVersionAddPermission")
	return &authz.FeedVersionSaveResponse{}, c.fgaClient.SetExclusiveSubjectRelation(ctx, tk, ViewerRelation, EditorRelation, ManagerRelation)
}

func (c *Checker) FeedVersionRemovePermission(ctx context.Context, req *authz.FeedVersionModifyPermissionRequest) (*authz.FeedVersionSaveResponse, error) {
	fvid := req.GetId()
	if check, err := c.FeedVersionPermissions(ctx, &authz.FeedVersionRequest{Id: fvid}); err != nil {
		return nil, err
	} else if !check.Actions.CanEditMembers {
		return nil, ErrUnauthorized
	}
	tk := req.GetEntityRelation().WithObject(newEntityID(FeedVersionType, fvid))
	log.For(ctx).Trace().Str("tk", tk.String()).Int64("id", fvid).Msg("FeedVersionRemovePermission")
	return &authz.FeedVersionSaveResponse{}, c.fgaClient.DeleteTuple(ctx, tk)
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

func (c *Checker) getSubjectTuples(ctx context.Context, obj EntityKey, ctxtk ...TupleKey) ([]TupleKey, error) {
	return c.fgaClient.GetObjectTuples(ctx, authz.NewTupleKey().WithSubject(obj.Type, obj.Name))
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
