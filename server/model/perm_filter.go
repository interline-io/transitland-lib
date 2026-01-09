package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/server/auth/authz"
)

type PermFilter struct {
	GlobalAdmin         bool
	AllowedFeeds        []int
	AllowedFeedVersions []int
}

func (pf *PermFilter) IsGlobalAdmin() bool {
	if pf == nil {
		return false
	}
	return pf.GlobalAdmin
}

func (pf *PermFilter) GetAllowedFeeds() []int {
	if pf == nil {
		return nil
	}
	return pf.AllowedFeeds
}

func (pf *PermFilter) GetAllowedFeedVersions() []int {
	if pf == nil {
		return nil
	}
	return pf.AllowedFeedVersions
}

var pfCtxKey = &contextKey{"permFilter"}

func PermsForContext(ctx context.Context) *PermFilter {
	raw, ok := ctx.Value(pfCtxKey).(*PermFilter)
	// log.For(ctx).Trace().Msgf("PermsForContext: %#v", raw)
	if !ok {
		return &PermFilter{}
	}
	return raw
}

func WithPerms(ctx context.Context, checker Checker) context.Context {
	pf, err := checkActive(ctx, checker)
	if err != nil {
		panic(err)
	}
	//	log.For(ctx).Trace().Msgf("WithPerms: %#v", pf)
	r := context.WithValue(ctx, pfCtxKey, pf)
	return r
}

func AddPerms(checker Checker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			r = r.WithContext(WithPerms(ctx, checker))
			next.ServeHTTP(w, r)
		})
	}
}

type canCheckGlobalAdmin interface {
	CheckGlobalAdmin(context.Context) (bool, error)
}

func checkActive(ctx context.Context, checker Checker) (*PermFilter, error) {
	active := &PermFilter{}
	if checker == nil {
		// log.For(ctx).Trace().Msg("checkActive: no checker")
		return active, nil
	}

	// TODO: Make this part of actual checker interface
	if c, ok := checker.(canCheckGlobalAdmin); ok {
		if a, err := c.CheckGlobalAdmin(ctx); err != nil {
			return nil, err
		} else if a {
			active.GlobalAdmin = true
		}
	}

	okFeeds, err := checker.FeedList(ctx, &authz.FeedListRequest{})
	if err != nil {
		return nil, err
	}
	for _, feed := range okFeeds.Feeds {
		active.AllowedFeeds = append(active.AllowedFeeds, int(feed.Id))
	}
	okFvids, err := checker.FeedVersionList(ctx, &authz.FeedVersionListRequest{})
	if err != nil {
		return nil, err
	}
	for _, fv := range okFvids.FeedVersions {
		active.AllowedFeedVersions = append(active.AllowedFeedVersions, int(fv.Id))
	}
	// fmt.Println("active allowed feeds:", active.AllowedFeeds, "fvs:", active.AllowedFeedVersions)
	return active, nil
}
