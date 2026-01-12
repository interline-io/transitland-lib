package model

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authz"
)

// mockChecker implements the Checker interface for testing
type mockChecker struct {
	authz.UnimplementedCheckerServer
	feeds        []int
	feedVersions []int
	isAdmin      bool
	shouldError  bool
}

func (m *mockChecker) FeedList(ctx context.Context, req *authz.FeedListRequest) (*authz.FeedListResponse, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	resp := &authz.FeedListResponse{}
	for _, id := range m.feeds {
		resp.Feeds = append(resp.Feeds, &authz.Feed{Id: int64(id)})
	}
	return resp, nil
}

func (m *mockChecker) FeedVersionList(ctx context.Context, req *authz.FeedVersionListRequest) (*authz.FeedVersionListResponse, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	resp := &authz.FeedVersionListResponse{}
	for _, id := range m.feedVersions {
		resp.FeedVersions = append(resp.FeedVersions, &authz.FeedVersion{Id: int64(id)})
	}
	return resp, nil
}

func (m *mockChecker) CheckGlobalAdmin(ctx context.Context) (bool, error) {
	return m.isAdmin, nil
}

func sortedInts(s []int) []int {
	result := make([]int, len(s))
	copy(result, s)
	sort.Ints(result)
	return result
}

func TestDedupeInts(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{"empty slice", []int{}, []int{}},
		{"nil slice", nil, nil},
		{"no duplicates", []int{1, 2, 3}, []int{1, 2, 3}},
		{"with duplicates", []int{1, 2, 2, 3, 1, 4}, []int{1, 2, 3, 4}},
		{"all duplicates", []int{1, 1, 1, 1}, []int{1}},
		{"preserves order", []int{3, 1, 2, 1, 3}, []int{3, 1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeInts(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("dedupeInts(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPermFilter_GetAllowedFeeds(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var pf *PermFilter
		if pf.GetAllowedFeeds() != nil {
			t.Error("expected nil for nil receiver")
		}
	})

	t.Run("non-nil receiver", func(t *testing.T) {
		pf := &PermFilter{AllowedFeeds: []int{1, 2, 3}}
		if !reflect.DeepEqual(pf.GetAllowedFeeds(), []int{1, 2, 3}) {
			t.Error("expected [1, 2, 3]")
		}
	})
}

func TestPermFilter_GetAllowedFeedVersions(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var pf *PermFilter
		if pf.GetAllowedFeedVersions() != nil {
			t.Error("expected nil for nil receiver")
		}
	})

	t.Run("non-nil receiver", func(t *testing.T) {
		pf := &PermFilter{AllowedFeedVersions: []int{1, 2, 3}}
		if !reflect.DeepEqual(pf.GetAllowedFeedVersions(), []int{1, 2, 3}) {
			t.Error("expected [1, 2, 3]")
		}
	})
}

func TestPermFilter_GetIsGlobalAdmin(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var pf *PermFilter
		if pf.GetIsGlobalAdmin() {
			t.Error("expected false for nil receiver")
		}
	})

	t.Run("non-admin", func(t *testing.T) {
		pf := &PermFilter{AllowedFeeds: []int{1, 2, 3}}
		if pf.GetIsGlobalAdmin() {
			t.Error("expected false")
		}
	})

	t.Run("admin", func(t *testing.T) {
		pf := &PermFilter{IsGlobalAdmin: true}
		if !pf.GetIsGlobalAdmin() {
			t.Error("expected true")
		}
	})
}

func TestPermsForContext(t *testing.T) {
	t.Run("no perm filter in context", func(t *testing.T) {
		ctx := context.Background()
		pf := PermsForContext(ctx)
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if len(pf.AllowedFeeds) != 0 || len(pf.AllowedFeedVersions) != 0 {
			t.Error("expected empty PermFilter")
		}
		if pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be false")
		}
	})

	t.Run("with perm filter in context", func(t *testing.T) {
		ctx := context.Background()
		expected := &PermFilter{AllowedFeeds: []int{1, 2}}
		ctx = WithPermFilter(ctx, expected)
		pf := PermsForContext(ctx)
		if pf != expected {
			t.Error("expected same PermFilter instance")
		}
	})

	t.Run("nil perm filter in context returns empty filter", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPermFilter(ctx, nil)
		pf := PermsForContext(ctx)
		// WithPermFilter converts nil to empty filter, PermsForContext always returns non-nil
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be false")
		}
	})

	t.Run("admin perm filter in context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPermFilter(ctx, &PermFilter{IsGlobalAdmin: true})
		pf := PermsForContext(ctx)
		if !pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be true")
		}
	})
}

func TestWithPermFilter(t *testing.T) {
	t.Run("sets perm filter", func(t *testing.T) {
		ctx := context.Background()
		pf := &PermFilter{AllowedFeeds: []int{1, 2, 3}}
		ctx = WithPermFilter(ctx, pf)

		result := PermsForContext(ctx)
		if result != pf {
			t.Error("expected same PermFilter instance")
		}
	})

	t.Run("nil becomes empty filter", func(t *testing.T) {
		ctx := context.Background()
		pf := &PermFilter{AllowedFeeds: []int{1, 2, 3}}
		ctx = WithPermFilter(ctx, pf)
		ctx = WithPermFilter(ctx, nil)

		// After setting nil, context should have empty PermFilter (not nil)
		result := PermsForContext(ctx)
		if result == nil {
			t.Error("expected non-nil PermFilter")
		}
		if len(result.AllowedFeeds) != 0 {
			t.Errorf("expected empty AllowedFeeds, got %v", result.AllowedFeeds)
		}
	})

	t.Run("does not mutate original", func(t *testing.T) {
		ctx := context.Background()
		original := &PermFilter{AllowedFeeds: []int{1, 2, 3}}
		ctx = WithPermFilter(ctx, original)

		// Simulate what WithPerms would do - it should create a new filter
		checker := &mockChecker{feeds: []int{4, 5}}
		ctx = WithPerms(ctx, checker)

		// Original should be unchanged
		if !reflect.DeepEqual(original.AllowedFeeds, []int{1, 2, 3}) {
			t.Errorf("original was mutated: %v", original.AllowedFeeds)
		}
	})

	t.Run("sets admin filter", func(t *testing.T) {
		ctx := context.Background()
		pf := &PermFilter{IsGlobalAdmin: true}
		ctx = WithPermFilter(ctx, pf)

		result := PermsForContext(ctx)
		if !result.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be true")
		}
	})
}

func TestWithPerms(t *testing.T) {
	t.Run("no existing filter - sets checker result", func(t *testing.T) {
		ctx := context.Background()
		checker := &mockChecker{feeds: []int{1, 2}, feedVersions: []int{10, 20}}
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2}) {
			t.Errorf("expected feeds [1, 2], got %v", pf.AllowedFeeds)
		}
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeedVersions), []int{10, 20}) {
			t.Errorf("expected feed versions [10, 20], got %v", pf.AllowedFeedVersions)
		}
	})

	t.Run("with existing filter - merges results", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{AllowedFeeds: []int{1, 2}, AllowedFeedVersions: []int{10}}
		ctx = WithPermFilter(ctx, existing)

		checker := &mockChecker{feeds: []int{3, 4}, feedVersions: []int{20, 30}}
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2, 3, 4}) {
			t.Errorf("expected feeds [1, 2, 3, 4], got %v", pf.AllowedFeeds)
		}
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeedVersions), []int{10, 20, 30}) {
			t.Errorf("expected feed versions [10, 20, 30], got %v", pf.AllowedFeedVersions)
		}
	})

	t.Run("with existing filter - deduplicates", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{AllowedFeeds: []int{1, 2, 3}, AllowedFeedVersions: []int{10, 20}}
		ctx = WithPermFilter(ctx, existing)

		// Checker returns overlapping IDs
		checker := &mockChecker{feeds: []int{2, 3, 4}, feedVersions: []int{20, 30}}
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2, 3, 4}) {
			t.Errorf("expected deduplicated feeds [1, 2, 3, 4], got %v", pf.AllowedFeeds)
		}
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeedVersions), []int{10, 20, 30}) {
			t.Errorf("expected deduplicated feed versions [10, 20, 30], got %v", pf.AllowedFeedVersions)
		}
	})

	t.Run("with existing filter - creates new instance", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{AllowedFeeds: []int{1, 2}, AllowedFeedVersions: []int{10}}
		ctx = WithPermFilter(ctx, existing)

		checker := &mockChecker{feeds: []int{3}, feedVersions: []int{20}}
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if pf == existing {
			t.Error("expected new PermFilter instance, got same pointer")
		}
	})

	t.Run("global admin - sets IsGlobalAdmin flag", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{AllowedFeeds: []int{1, 2}, AllowedFeedVersions: []int{10}}
		ctx = WithPermFilter(ctx, existing)

		checker := &mockChecker{isAdmin: true}
		ctx = WithPerms(ctx, checker)

		// For admin, the filter should have IsGlobalAdmin=true
		pf := PermsForContext(ctx)
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if !pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be true")
		}
	})

	t.Run("global admin - no existing filter", func(t *testing.T) {
		ctx := context.Background()
		checker := &mockChecker{isAdmin: true}
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if !pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be true")
		}
	})

	t.Run("nil checker - returns empty filter", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithPerms(ctx, nil)

		pf := PermsForContext(ctx)
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if len(pf.AllowedFeeds) != 0 || len(pf.AllowedFeedVersions) != 0 {
			t.Error("expected empty PermFilter")
		}
	})

	t.Run("nil checker with existing filter - merges empty", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{AllowedFeeds: []int{1, 2}, AllowedFeedVersions: []int{10}}
		ctx = WithPermFilter(ctx, existing)

		ctx = WithPerms(ctx, nil)

		pf := PermsForContext(ctx)
		// Should merge existing with empty checker result
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2}) {
			t.Errorf("expected feeds [1, 2], got %v", pf.AllowedFeeds)
		}
	})

	t.Run("existing admin + non-admin checker preserves admin", func(t *testing.T) {
		ctx := context.Background()
		existing := &PermFilter{IsGlobalAdmin: true}
		ctx = WithPermFilter(ctx, existing)

		checker := &mockChecker{feeds: []int{1, 2}} // non-admin checker
		ctx = WithPerms(ctx, checker)

		pf := PermsForContext(ctx)
		if !pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to remain true after merge with non-admin checker")
		}
		// Should also have the feeds from checker
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2}) {
			t.Errorf("expected feeds [1, 2], got %v", pf.AllowedFeeds)
		}
	})
}

func TestWithPerms_ThreadSafety(t *testing.T) {
	// Test that the original PermFilter is not mutated when merging
	// This is important for cases where the same PermFilter might be cached/reused
	t.Run("original not mutated on merge", func(t *testing.T) {
		original := &PermFilter{
			AllowedFeeds:        []int{1, 2, 3},
			AllowedFeedVersions: []int{10, 20},
		}
		originalFeedsCopy := make([]int, len(original.AllowedFeeds))
		copy(originalFeedsCopy, original.AllowedFeeds)

		ctx := context.Background()
		ctx = WithPermFilter(ctx, original)

		checker := &mockChecker{feeds: []int{4, 5}, feedVersions: []int{30}}
		_ = WithPerms(ctx, checker)

		// Verify original was not mutated
		if !reflect.DeepEqual(original.AllowedFeeds, originalFeedsCopy) {
			t.Errorf("original was mutated: got %v, want %v", original.AllowedFeeds, originalFeedsCopy)
		}
	})
}

func TestCheckActive(t *testing.T) {
	t.Run("nil checker returns empty filter", func(t *testing.T) {
		ctx := context.Background()
		pf, err := checkActive(ctx, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if len(pf.AllowedFeeds) != 0 {
			t.Error("expected empty AllowedFeeds")
		}
	})

	t.Run("checker returns feeds and versions", func(t *testing.T) {
		ctx := context.Background()
		checker := &mockChecker{feeds: []int{1, 2}, feedVersions: []int{10, 20}}
		pf, err := checkActive(ctx, checker)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeeds), []int{1, 2}) {
			t.Errorf("expected feeds [1, 2], got %v", pf.AllowedFeeds)
		}
		if !reflect.DeepEqual(sortedInts(pf.AllowedFeedVersions), []int{10, 20}) {
			t.Errorf("expected feed versions [10, 20], got %v", pf.AllowedFeedVersions)
		}
	})

	t.Run("global admin returns filter with IsGlobalAdmin", func(t *testing.T) {
		ctx := context.Background()
		checker := &mockChecker{isAdmin: true}
		pf, err := checkActive(ctx, checker)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if pf == nil {
			t.Error("expected non-nil PermFilter")
		}
		if !pf.IsGlobalAdmin {
			t.Error("expected IsGlobalAdmin to be true")
		}
	})
}
