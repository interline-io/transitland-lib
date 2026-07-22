package model

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestUnimplementedFinderNaming locks the two invariants of the self-naming
// stubs: the returned error still matches ErrNotImplemented (so errors.Is keeps
// working), and it names the calling Finder method (so an unimplemented loader
// identifies itself). The name comes from runtime.Caller frame math in
// finderUnimpl, which is sensitive to refactors/inlining — this guards it.
func TestUnimplementedFinderNaming(t *testing.T) {
	var f UnimplementedFinder
	ctx := context.Background()

	// Non-batched stub (notImplErr path).
	_, err := f.FindAgencies(ctx, nil, nil, nil, nil)
	assertNamed(t, err, "FindAgencies")

	// Batched stub (notImplBatch path): every per-key error is named.
	_, errs := f.AgenciesByIDs(ctx, []int{1, 2})
	if len(errs) != 2 {
		t.Fatalf("AgenciesByIDs returned %d errors, want 2", len(errs))
	}
	for _, e := range errs {
		assertNamed(t, e, "AgenciesByIDs")
	}

	// Mutator stub returning a bare error (the case PathwayDelete/LevelDelete
	// previously got wrong).
	assertNamed(t, f.PathwayDelete(ctx, 1), "PathwayDelete")
	assertNamed(t, f.LevelDelete(ctx, 1), "LevelDelete")
}

func assertNamed(t *testing.T, err error, method string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error", method)
	}
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("%s: errors.Is(err, ErrNotImplemented) = false (err=%v)", method, err)
	}
	if !strings.Contains(err.Error(), method) {
		t.Errorf("%s: error %q does not name the method", method, err.Error())
	}
}
