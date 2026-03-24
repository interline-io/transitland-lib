package azchecker

import (
	"context"

	"github.com/interline-io/transitland-lib/server/auth/authz"
)

// Legacy response types that preserve backward compatibility with the
// proto-generated JSON shapes.  These are used only by the admin REST
// API handlers in server.go — the internal Checker / PermissionManager
// interfaces remain generic.

// meResponse matches the old MeResponse proto shape:
//
//	{"user":{...}, "groups":[...], "expanded_groups":[...], "roles":[...], "external_data":{...}}
type meResponse struct {
	User           *authz.User       `json:"user,omitempty"`
	Groups         []*authz.Group    `json:"groups,omitempty"`
	ExpandedGroups []*authz.Group    `json:"expanded_groups,omitempty"`
	Roles          []string          `json:"roles,omitempty"`
	ExternalData   map[string]string `json:"external_data,omitempty"`
}

func wrapMe(info *authz.UserInfo) *meResponse {
	if info == nil {
		return nil
	}
	resp := &meResponse{
		User:         &authz.User{Id: info.ID, Name: info.Name, Email: info.Email},
		Roles:        info.Roles,
		ExternalData: info.ExternalData,
	}
	for i := range info.Groups {
		resp.Groups = append(resp.Groups, &info.Groups[i])
	}
	for i := range info.ExpandedGroups {
		resp.ExpandedGroups = append(resp.ExpandedGroups, &info.ExpandedGroups[i])
	}
	return resp
}

// entityRelation matches the old proto EntityRelation JSON shape.
// Type and Relation serialize as integers (matching proto enum behavior).
type entityRelation struct {
	Type        int32  `json:"type,omitempty"`
	Id          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	RefRelation int32  `json:"ref_relation,omitempty"`
	Relation    int32  `json:"relation,omitempty"`
}

func subjectToEntityRelation(s authz.SubjectRef) entityRelation {
	return entityRelation{
		Type:     int32(s.Subject.Type),
		Id:       s.Subject.Name,
		Name:     s.Name,
		Relation: int32(s.Relation),
	}
}

// actionsMap converts ActionSet to a map[string]bool matching the old
// proto JSON behavior: keys are snake_case action names, only true values
// are present (proto3 omits false bools).
func actionsMap(as authz.ActionSet) map[string]bool {
	out := map[string]bool{}
	for k, v := range as {
		if v {
			out[k.String()] = true
		}
	}
	return out
}

// List response wrappers — match old {"tenants": [...]} etc.

type tenantListResponse struct {
	Tenants []*authz.Tenant `json:"tenants,omitempty"`
}

type groupListResponse struct {
	Groups []*authz.Group `json:"groups,omitempty"`
}

type feedListResponse struct {
	Feeds []*authz.Feed `json:"feeds,omitempty"`
}

type feedVersionListResponse struct {
	FeedVersions []*authz.FeedVersion `json:"feed_versions,omitempty"`
}

func refsToIDs(refs []authz.ObjectRef) []int64 {
	ids := make([]int64, len(refs))
	for i, r := range refs {
		ids[i] = r.ID
	}
	return ids
}

func wrapTenantList(ctx context.Context, c *Checker, refs []authz.ObjectRef) *tenantListResponse {
	ids := refsToIDs(refs)
	tenants, _ := c.getTenants(ctx, ids)
	return &tenantListResponse{Tenants: tenants}
}

func wrapGroupList(ctx context.Context, c *Checker, refs []authz.ObjectRef) *groupListResponse {
	ids := refsToIDs(refs)
	groups, _ := c.getGroups(ctx, ids)
	return &groupListResponse{Groups: groups}
}

func wrapFeedList(ctx context.Context, c *Checker, refs []authz.ObjectRef) *feedListResponse {
	ids := refsToIDs(refs)
	feeds, _ := c.getFeeds(ctx, ids)
	return &feedListResponse{Feeds: feeds}
}

func wrapFeedVersionList(ctx context.Context, c *Checker, refs []authz.ObjectRef) *feedVersionListResponse {
	ids := refsToIDs(refs)
	fvs, _ := c.getFeedVersions(ctx, ids)
	return &feedVersionListResponse{FeedVersions: fvs}
}

// Permissions response wrappers — per-type shapes matching old proto responses.

// tenantPermissionsResponse matches old TenantPermissionsResponse.
type tenantPermissionsResponse struct {
	Tenant  *authz.Tenant    `json:"tenant,omitempty"`
	Groups  []*authz.Group   `json:"groups,omitempty"`
	Actions map[string]bool  `json:"actions,omitempty"`
	Users   *tenantPermUsers `json:"users,omitempty"`
}

type tenantPermUsers struct {
	Admins  []entityRelation `json:"admins,omitempty"`
	Members []entityRelation `json:"members,omitempty"`
}

func wrapTenantPermissions(ctx context.Context, c *Checker, p *authz.ObjectPermissions) *tenantPermissionsResponse {
	if p == nil {
		return nil
	}
	resp := &tenantPermissionsResponse{
		Tenant:  &authz.Tenant{Id: p.Ref.ID, Name: p.Ref.Name},
		Actions: actionsMap(p.Actions),
		Users:   &tenantPermUsers{},
	}
	// Children (groups)
	if len(p.Children) > 0 {
		ids := refsToIDs(p.Children)
		resp.Groups, _ = c.getGroups(ctx, ids)
	}
	// Subjects grouped by role
	for _, s := range p.Subjects {
		er := subjectToEntityRelation(s)
		switch s.Relation {
		case authz.AdminRelation:
			resp.Users.Admins = append(resp.Users.Admins, er)
		case authz.MemberRelation:
			resp.Users.Members = append(resp.Users.Members, er)
		}
	}
	return resp
}

// groupPermissionsResponse matches old GroupPermissionsResponse.
type groupPermissionsResponse struct {
	Group   *authz.Group    `json:"group,omitempty"`
	Tenant  *authz.Tenant   `json:"tenant,omitempty"`
	Feeds   []*authz.Feed   `json:"feeds,omitempty"`
	Actions map[string]bool `json:"actions,omitempty"`
	Users   *groupPermUsers `json:"users,omitempty"`
}

type groupPermUsers struct {
	Managers []entityRelation `json:"managers,omitempty"`
	Editors  []entityRelation `json:"editors,omitempty"`
	Viewers  []entityRelation `json:"viewers,omitempty"`
}

func wrapGroupPermissions(ctx context.Context, c *Checker, p *authz.ObjectPermissions) *groupPermissionsResponse {
	if p == nil {
		return nil
	}
	resp := &groupPermissionsResponse{
		Group:   &authz.Group{Id: p.Ref.ID, Name: p.Ref.Name},
		Actions: actionsMap(p.Actions),
		Users:   &groupPermUsers{},
	}
	// Parent (tenant)
	if p.Parent != nil {
		resp.Tenant = &authz.Tenant{Id: p.Parent.ID, Name: p.Parent.Name}
	}
	// Children (feeds)
	if len(p.Children) > 0 {
		ids := refsToIDs(p.Children)
		resp.Feeds, _ = c.getFeeds(ctx, ids)
	}
	// Subjects grouped by role
	for _, s := range p.Subjects {
		er := subjectToEntityRelation(s)
		switch s.Relation {
		case authz.ManagerRelation:
			resp.Users.Managers = append(resp.Users.Managers, er)
		case authz.EditorRelation:
			resp.Users.Editors = append(resp.Users.Editors, er)
		case authz.ViewerRelation:
			resp.Users.Viewers = append(resp.Users.Viewers, er)
		}
	}
	return resp
}

// feedPermissionsResponse matches old FeedPermissionsResponse.
type feedPermissionsResponse struct {
	Feed    *authz.Feed     `json:"feed,omitempty"`
	Group   *authz.Group    `json:"group,omitempty"`
	Actions map[string]bool `json:"actions,omitempty"`
}

func wrapFeedPermissions(ctx context.Context, c *Checker, p *authz.ObjectPermissions) *feedPermissionsResponse {
	if p == nil {
		return nil
	}
	resp := &feedPermissionsResponse{
		Feed:    &authz.Feed{Id: p.Ref.ID, Name: p.Ref.Name},
		Actions: actionsMap(p.Actions),
	}
	// The old response populated feed.onestop_id — fetch full feed entity.
	if feeds, _ := c.getFeeds(ctx, []int64{p.Ref.ID}); len(feeds) > 0 && feeds[0] != nil {
		resp.Feed = feeds[0]
	}
	// Parent (group)
	if p.Parent != nil {
		resp.Group = &authz.Group{Id: p.Parent.ID, Name: p.Parent.Name}
	}
	return resp
}

// feedVersionPermissionsResponse matches old FeedVersionPermissionsResponse.
type feedVersionPermissionsResponse struct {
	FeedVersion *authz.FeedVersion `json:"feed_version,omitempty"`
	Feed        *authz.Feed        `json:"feed,omitempty"`
	Actions     map[string]bool    `json:"actions,omitempty"`
	Users       *fvPermUsers       `json:"users,omitempty"`
}

type fvPermUsers struct {
	Editors []entityRelation `json:"editors,omitempty"`
	Viewers []entityRelation `json:"viewers,omitempty"`
}

func wrapFeedVersionPermissions(ctx context.Context, c *Checker, p *authz.ObjectPermissions) *feedVersionPermissionsResponse {
	if p == nil {
		return nil
	}
	resp := &feedVersionPermissionsResponse{
		FeedVersion: &authz.FeedVersion{Id: p.Ref.ID, Name: p.Ref.Name},
		Actions:     actionsMap(p.Actions),
		Users:       &fvPermUsers{},
	}
	// The old response included the parent feed (with full entity) and
	// the feed version's feed_id + sha1.
	if fvs, _ := c.getFeedVersions(ctx, []int64{p.Ref.ID}); len(fvs) > 0 && fvs[0] != nil {
		resp.FeedVersion = fvs[0]
	}
	if p.Parent != nil {
		if feeds, _ := c.getFeeds(ctx, []int64{p.Parent.ID}); len(feeds) > 0 && feeds[0] != nil {
			resp.Feed = feeds[0]
		}
	}
	// Subjects grouped by role
	for _, s := range p.Subjects {
		er := subjectToEntityRelation(s)
		switch s.Relation {
		case authz.EditorRelation:
			resp.Users.Editors = append(resp.Users.Editors, er)
		case authz.ViewerRelation:
			resp.Users.Viewers = append(resp.Users.Viewers, er)
		}
	}
	return resp
}
