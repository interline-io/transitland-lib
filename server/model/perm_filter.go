package model

import (
	"context"
	"net/http"

	"github.com/interline-io/transitland-lib/server/auth/authz"
)

// PermFilter holds permission-based filtering criteria for feeds and feed versions.
// When IsGlobalAdmin is true, no filtering is applied (unrestricted access).
// Otherwise, access is restricted to the specified AllowedFeeds and AllowedFeedVersions IDs.
type PermFilter struct {
	AllowedFeeds        []int
	AllowedFeedVersions []int
	IsGlobalAdmin       bool
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

// GetIsGlobalAdmin returns true if the user has global admin privileges (unrestricted access).
func (pf *PermFilter) GetIsGlobalAdmin() bool {
	if pf == nil {
		return false
	}
	return pf.IsGlobalAdmin
}

// dedupeInts returns a new slice with duplicate values removed, preserving order.
func dedupeInts(vals []int) []int {
	if len(vals) == 0 {
		return vals
	}
	seen := make(map[int]struct{}, len(vals))
	result := make([]int, 0, len(vals))
	for _, v := range vals {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

var pfCtxKey = &contextKey{"permFilter"}

// PermsForContext retrieves the PermFilter from context.
// Always returns a non-nil PermFilter. If none is set, returns an empty filter.
func PermsForContext(ctx context.Context) *PermFilter {
	raw, ok := ctx.Value(pfCtxKey).(*PermFilter)
	// log.For(ctx).Trace().Msgf("PermsForContext: %#v", raw)
	if !ok || raw == nil {
		return &PermFilter{}
	}
	return raw
}

// WithPerms populates permission filters in the context using the provided Checker.
// If an existing PermFilter is already set in context (e.g., via WithPermFilter),
// the checker's results are merged into a new PermFilter (the original is not mutated).
//
// Merge behavior:
//   - No existing filter: checker results are used directly
//   - Existing filter + checker results: creates new filter with merged, deduplicated IDs
//   - Either filter has IsGlobalAdmin=true: resulting filter has IsGlobalAdmin=true
func WithPerms(ctx context.Context, checker Checker) context.Context {
	checkerPf, err := checkActive(ctx, checker)
	if err != nil {
		panic(err)
	}

	existing, hasExisting := ctx.Value(pfCtxKey).(*PermFilter)

	// If there's an existing filter, merge with checker results into a new PermFilter
	// We create a new instance to avoid mutating the original (thread safety)
	if hasExisting && existing != nil {
		merged := &PermFilter{
			AllowedFeeds:        dedupeInts(append(existing.AllowedFeeds, checkerPf.AllowedFeeds...)),
			AllowedFeedVersions: dedupeInts(append(existing.AllowedFeedVersions, checkerPf.AllowedFeedVersions...)),
			IsGlobalAdmin:       existing.IsGlobalAdmin || checkerPf.IsGlobalAdmin,
		}
		return context.WithValue(ctx, pfCtxKey, merged)
	}

	// No existing PermFilter, set the checker's result
	return context.WithValue(ctx, pfCtxKey, checkerPf)
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
			return &PermFilter{IsGlobalAdmin: true}, nil
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

// WithPermFilter stores a PermFilter directly in context.
// Use this when you need to set permissions without going through a Checker,
// such as when populating AllowedFeeds from an external source like gatekeeper.
//
// Note: The provided PermFilter will NOT be mutated by subsequent calls to WithPerms.
// WithPerms creates a new merged PermFilter if it needs to combine permissions.
//
// To grant unrestricted access, pass a PermFilter with IsGlobalAdmin=true.
func WithPermFilter(ctx context.Context, pf *PermFilter) context.Context {
	if pf == nil {
		pf = &PermFilter{}
	}
	return context.WithValue(ctx, pfCtxKey, pf)
}
