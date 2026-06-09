// Package adminapi exposes the authz admin HTTP API behind a generic
// Server interface so any AdminManager+EntityProvider can back it.
package adminapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authz"
)

type Server interface {
	authz.AdminManager
	authz.EntityProvider
}

func NewServer(srv Server) (http.Handler, error) {
	router := chi.NewRouter()

	/////////////////
	// USERS
	/////////////////

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		ret, err := srv.UserList(r.Context(), &authz.UserListRequest{Q: r.URL.Query().Get("q")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := srv.User(r.Context(), &authz.UserRequest{Id: chi.URLParam(r, "user_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		info, err := srv.Me(r.Context())
		handleJsonOr(r.Context(), w, err, func() any { return wrapMe(info) })
	})

	/////////////////
	// TENANTS
	/////////////////

	router.Get("/tenants", func(w http.ResponseWriter, r *http.Request) {
		refs, err := srv.ListObjects(r.Context(), authz.TenantType)
		handleJsonOr(r.Context(), w, err, func() any { return wrapTenantList(r.Context(), srv, refs) })
	})
	router.Get("/tenants/{tenant_id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := srv.ObjectPermissions(r.Context(), authz.ObjectRef{Type: authz.TenantType, ID: checkId(r, "tenant_id")})
		handleJsonOr(r.Context(), w, err, func() any { return wrapTenantPermissions(r.Context(), srv, p) })
	})
	router.Post("/tenants/{tenant_id}", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Tenant{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		check.Id = checkId(r, "tenant_id")
		_, err := srv.TenantSave(r.Context(), &authz.TenantSaveRequest{Tenant: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Post("/tenants/{tenant_id}/groups", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Group{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		_, err := srv.TenantCreateGroup(r.Context(), &authz.TenantCreateGroupRequest{Id: checkId(r, "tenant_id"), Group: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Post("/tenants/{tenant_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.TenantType, ID: checkId(r, "tenant_id")}
		err := srv.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/tenants/{tenant_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.TenantType, ID: checkId(r, "tenant_id")}
		err := srv.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// GROUPS
	/////////////////

	router.Get("/groups", func(w http.ResponseWriter, r *http.Request) {
		refs, err := srv.ListObjects(r.Context(), authz.GroupType)
		handleJsonOr(r.Context(), w, err, func() any { return wrapGroupList(r.Context(), srv, refs) })
	})
	router.Post("/groups/{group_id}", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Group{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		check.Id = checkId(r, "group_id")
		_, err := srv.GroupSave(r.Context(), &authz.GroupSaveRequest{Group: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Get("/groups/{group_id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := srv.ObjectPermissions(r.Context(), authz.ObjectRef{Type: authz.GroupType, ID: checkId(r, "group_id")})
		handleJsonOr(r.Context(), w, err, func() any { return wrapGroupPermissions(r.Context(), srv, p) })
	})
	router.Post("/groups/{group_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.GroupType, ID: checkId(r, "group_id")}
		err := srv.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/groups/{group_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.GroupType, ID: checkId(r, "group_id")}
		err := srv.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Post("/groups/{group_id}/tenant", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TenantId int64 `json:"tenant_id"`
		}
		if err := parseJson(r.Body, &body); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		child := authz.ObjectRef{Type: authz.GroupType, ID: checkId(r, "group_id")}
		parent := authz.ObjectRef{Type: authz.TenantType, ID: body.TenantId}
		err := srv.SetParent(r.Context(), child, parent)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// FEEDS
	/////////////////

	router.Get("/feeds", func(w http.ResponseWriter, r *http.Request) {
		refs, err := srv.ListObjects(r.Context(), authz.FeedType)
		handleJsonOr(r.Context(), w, err, func() any { return wrapFeedList(r.Context(), srv, refs) })
	})
	router.Get("/feeds/{feed_id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := srv.ObjectPermissions(r.Context(), authz.ObjectRef{Type: authz.FeedType, ID: checkId(r, "feed_id")})
		handleJsonOr(r.Context(), w, err, func() any { return wrapFeedPermissions(r.Context(), srv, p) })
	})
	router.Post("/feeds/{feed_id}/group", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			GroupId int64 `json:"group_id"`
		}
		if err := parseJson(r.Body, &body); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		child := authz.ObjectRef{Type: authz.FeedType, ID: checkId(r, "feed_id")}
		parent := authz.ObjectRef{Type: authz.GroupType, ID: body.GroupId}
		err := srv.SetParent(r.Context(), child, parent)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// FEED VERSIONS
	/////////////////

	router.Get("/feed_versions", func(w http.ResponseWriter, r *http.Request) {
		refs, err := srv.ListObjects(r.Context(), authz.FeedVersionType)
		handleJsonOr(r.Context(), w, err, func() any { return wrapFeedVersionList(r.Context(), srv, refs) })
	})
	router.Get("/feed_versions/{feed_version_id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := srv.ObjectPermissions(r.Context(), authz.ObjectRef{Type: authz.FeedVersionType, ID: checkId(r, "feed_version_id")})
		handleJsonOr(r.Context(), w, err, func() any { return wrapFeedVersionPermissions(r.Context(), srv, p) })
	})
	router.Post("/feed_versions/{feed_version_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.FeedVersionType, ID: checkId(r, "feed_version_id")}
		err := srv.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/feed_versions/{feed_version_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: authz.FeedVersionType, ID: checkId(r, "feed_version_id")}
		err := srv.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id).WithRefRel(er.RefRelation), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})

	return router, nil
}

func makeJsonError(msg string) string {
	a := map[string]string{
		"error": msg,
	}
	jj, _ := json.Marshal(&a)
	return string(jj)
}

// handleJsonOr writes an error response if err is non-nil, otherwise
// calls fn() to build the response value. This avoids running expensive
// wrapper/hydration logic on error paths.
func handleJsonOr(ctx context.Context, w http.ResponseWriter, err error, fn func() any) {
	if err != nil {
		handleJson(ctx, w, nil, err)
		return
	}
	handleJson(ctx, w, fn(), nil)
}

func handleJson(ctx context.Context, w http.ResponseWriter, ret any, err error) {
	if err == authz.ErrUnauthorized {
		log.For(ctx).Error().Err(err).Msg("unauthorized")
		http.Error(w, makeJsonError(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
		return
	} else if err != nil {
		log.For(ctx).Error().Err(err).Msg("admin api error")
		http.Error(w, makeJsonError(http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError)
		return
	}
	if ret == nil {
		ret = map[string]bool{"success": true}
	}
	jj, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	w.Write(jj)
}

func checkId(r *http.Request, key string) int64 {
	v, _ := strconv.Atoi(chi.URLParam(r, key))
	return int64(v)
}

func parseJson(r io.Reader, v any) error {
	data, err := io.ReadAll(io.LimitReader(r, 1_000_000))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
