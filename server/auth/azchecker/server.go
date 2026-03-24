package azchecker

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

func NewServer(checker *Checker) (http.Handler, error) {
	router := chi.NewRouter()

	/////////////////
	// USERS
	/////////////////

	router.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.UserList(r.Context(), &authz.UserListRequest{Q: r.URL.Query().Get("q")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/users/{user_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.User(r.Context(), &authz.UserRequest{Id: chi.URLParam(r, "user_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.Me(r.Context())
		handleJson(r.Context(), w, ret, err)
	})

	/////////////////
	// TENANTS
	/////////////////

	router.Get("/tenants", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ListObjects(r.Context(), TenantType)
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/tenants/{tenant_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ObjectPermissions(r.Context(), authz.ObjectRef{Type: TenantType, ID: checkId(r, "tenant_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Post("/tenants/{tenant_id}", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Tenant{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		check.Id = checkId(r, "tenant_id")
		_, err := checker.TenantSave(r.Context(), &authz.TenantSaveRequest{Tenant: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Post("/tenants/{tenant_id}/groups", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Group{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		_, err := checker.TenantCreateGroup(r.Context(), &authz.TenantCreateGroupRequest{Id: checkId(r, "tenant_id"), Group: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Post("/tenants/{tenant_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: TenantType, ID: checkId(r, "tenant_id")}
		err := checker.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/tenants/{tenant_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: TenantType, ID: checkId(r, "tenant_id")}
		err := checker.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// GROUPS
	/////////////////

	router.Get("/groups", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ListObjects(r.Context(), GroupType)
		handleJson(r.Context(), w, ret, err)
	})
	router.Post("/groups/{group_id}", func(w http.ResponseWriter, r *http.Request) {
		check := authz.Group{}
		if err := parseJson(r.Body, &check); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		check.Id = checkId(r, "group_id")
		_, err := checker.GroupSave(r.Context(), &authz.GroupSaveRequest{Group: &check})
		handleJson(r.Context(), w, nil, err)
	})
	router.Get("/groups/{group_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ObjectPermissions(r.Context(), authz.ObjectRef{Type: GroupType, ID: checkId(r, "group_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Post("/groups/{group_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: GroupType, ID: checkId(r, "group_id")}
		err := checker.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/groups/{group_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: GroupType, ID: checkId(r, "group_id")}
		err := checker.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
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
		child := authz.ObjectRef{Type: GroupType, ID: checkId(r, "group_id")}
		parent := authz.ObjectRef{Type: TenantType, ID: body.TenantId}
		err := checker.SetParent(r.Context(), child, parent)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// FEEDS
	/////////////////

	router.Get("/feeds", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ListObjects(r.Context(), FeedType)
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/feeds/{feed_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ObjectPermissions(r.Context(), authz.ObjectRef{Type: FeedType, ID: checkId(r, "feed_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Post("/feeds/{feed_id}/group", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			GroupId int64 `json:"group_id"`
		}
		if err := parseJson(r.Body, &body); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		child := authz.ObjectRef{Type: FeedType, ID: checkId(r, "feed_id")}
		parent := authz.ObjectRef{Type: GroupType, ID: body.GroupId}
		err := checker.SetParent(r.Context(), child, parent)
		handleJson(r.Context(), w, nil, err)
	})

	/////////////////
	// FEED VERSIONS
	/////////////////

	router.Get("/feed_versions", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ListObjects(r.Context(), FeedVersionType)
		handleJson(r.Context(), w, ret, err)
	})
	router.Get("/feed_versions/{feed_version_id}", func(w http.ResponseWriter, r *http.Request) {
		ret, err := checker.ObjectPermissions(r.Context(), authz.ObjectRef{Type: FeedVersionType, ID: checkId(r, "feed_version_id")})
		handleJson(r.Context(), w, ret, err)
	})
	router.Post("/feed_versions/{feed_version_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: FeedVersionType, ID: checkId(r, "feed_version_id")}
		err := checker.AddPermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
		handleJson(r.Context(), w, nil, err)
	})
	router.Delete("/feed_versions/{feed_version_id}/permissions", func(w http.ResponseWriter, r *http.Request) {
		er := &authz.EntityRelation{}
		if err := parseJson(r.Body, er); err != nil {
			handleJson(r.Context(), w, nil, err)
			return
		}
		ref := authz.ObjectRef{Type: FeedVersionType, ID: checkId(r, "feed_version_id")}
		err := checker.RemovePermission(r.Context(), ref, authz.NewEntityKey(er.Type, er.Id), er.Relation)
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

func handleJson(ctx context.Context, w http.ResponseWriter, ret any, err error) {
	if err == ErrUnauthorized {
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
