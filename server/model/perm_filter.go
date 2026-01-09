package model

import (
	"context"
	"net/http"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/caches/ecache"
)

type PermFilter struct {
	AllowedFeeds        []int
	AllowedFeedVersions []int
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

// PermFilterCache is a TTL cache for PermFilter objects, keyed by user ID.
// Pass nil to NewPermFilterCache to disable caching.
type PermFilterCache = ecache.Cache[*PermFilter]

// NewPermFilterCache creates a new PermFilter cache with the given TTL.
// Pass nil for redisClient to use in-memory only caching.
func NewPermFilterCache(redisClient interface{}, ttl time.Duration) *PermFilterCache {
	// ecache expects *redis.Client, but we accept interface{} to avoid
	// requiring redis import at call sites. Pass nil for in-memory only.
	return ecache.NewCache[*PermFilter](nil, "permfilter")
}

func PermsForContext(ctx context.Context) *PermFilter {
	raw, ok := ctx.Value(pfCtxKey).(*PermFilter)
	// log.For(ctx).Trace().Msgf("PermsForContext: %#v", raw)
	if !ok {
		return &PermFilter{}
	}
	return raw
}

func WithPerms(ctx context.Context, checker Checker, cache *PermFilterCache, cacheTTL time.Duration) context.Context {
	// Check if user is authenticated
	user := authn.ForContext(ctx)
	if user == nil || user.ID() == "" {
		// Anonymous user - skip cache, return empty PermFilter
		return context.WithValue(ctx, pfCtxKey, &PermFilter{})
	}

	userKey := user.ID()

	// Check cache first (if cache is enabled)
	if cache != nil {
		if pf, ok := cache.Get(ctx, userKey); ok {
			return context.WithValue(ctx, pfCtxKey, pf)
		}
	}

	// Cache miss or no cache - fetch from checker
	pf, err := checkActive(ctx, checker)
	if err != nil {
		panic(err)
	}

	// Store in cache (if cache is enabled)
	if cache != nil && pf != nil {
		cache.SetTTL(ctx, userKey, pf, cacheTTL, cacheTTL)
	}

	return context.WithValue(ctx, pfCtxKey, pf)
}

// DefaultPermFilterCacheTTL is the default TTL for cached PermFilter entries.
const DefaultPermFilterCacheTTL = 1 * time.Minute

func AddPerms(checker Checker, cache *PermFilterCache) func(http.Handler) http.Handler {
	return AddPermsWithTTL(checker, cache, DefaultPermFilterCacheTTL)
}

func AddPermsWithTTL(checker Checker, cache *PermFilterCache, cacheTTL time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			r = r.WithContext(WithPerms(ctx, checker, cache, cacheTTL))
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
			return nil, nil
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
