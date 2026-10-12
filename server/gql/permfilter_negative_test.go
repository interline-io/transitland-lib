package gql

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

// Flipping feed_states.public = false strips the only access path the default
// empty PermFilter has, so every loader under test should go from returning rows
// to returning none. If pfJoinCheckFv is missing from any loader, that loader
// keeps returning rows and its subtest fails.
func TestPermFilter_Negative(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		f := cfg.Finder

		// Look up every id while the feeds are still public, since internal ids
		// depend on the order the fixtures load in.
		fvID := func(sha1 string) int {
			t.Helper()
			fvs, err := f.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
			if err != nil {
				t.Fatal(err)
			}
			if len(fvs) != 1 {
				t.Fatalf("found %d feed versions with sha1 %s, expected 1", len(fvs), sha1)
			}
			return fvs[0].ID
		}
		ctFv := fvID("d2813c293bcfd7a97dde599527ae6c62c98e66c6")
		flexFv := fvID("e8bc76c3c8602cad745f41a49ed5c5627ad6904c")
		baFv := fvID("e535eb2b3b9ac3ef15d82c56575e914575e732e0")
		haFv := fvID("c969427f56d3a645195dd8365cde6d7feae7e99b")
		wmataFv := fvID("148d00724546e1526d5d84d1f9d095df24f6517c")
		exFv := fvID("43e2278aa272879c79460582152b04e7487f0493")

		agencies, err := f.AgenciesByFeedVersionIDs(ctx, nil, nil, []int{wmataFv})
		if err != nil {
			t.Fatal(err)
		}
		if len(agencies[0]) == 0 {
			t.Fatal("found no WMATA agency")
		}
		wmataAgency := agencies[0][0].ID

		segments, err := f.SegmentsByFeedVersionIDs(ctx, nil, nil, []int{haFv})
		if err != nil {
			t.Fatal(err)
		}
		if len(segments[0]) < 2 {
			t.Fatalf("found %d HA segments, need 2", len(segments[0]))
		}
		segmentIDs := []int{segments[0][0].ID, segments[0][1].ID}
		patterns, err := f.SegmentPatternsBySegmentIDs(ctx, nil, nil, segmentIDs)
		if err != nil {
			t.Fatal(err)
		}
		var routeIDs []int
		for _, group := range patterns {
			for _, p := range group {
				if !slices.Contains(routeIDs, p.RouteID) {
					routeIDs = append(routeIDs, p.RouteID)
				}
			}
		}
		if len(routeIDs) == 0 {
			t.Fatal("found no routes for the HA segments")
		}

		checks := []struct {
			name  string
			count func() (int, error)
		}{
			// The permission join sits inside the lateral, below the agency
			// membership subquery, so it is worth checking it is still reached.
			{"StopsByAgencyIDs", func() (int, error) { return countGroups(f.StopsByAgencyIDs(ctx, nil, nil, nil, []int{wmataAgency})) }},
			{"SegmentsByFeedVersionIDs", func() (int, error) { return countGroups(f.SegmentsByFeedVersionIDs(ctx, nil, nil, []int{haFv})) }},
			{"SegmentsByIDs", func() (int, error) { return countFound(f.SegmentsByIDs(ctx, segmentIDs)) }},
			{"SegmentsByRouteIDs", func() (int, error) { return countGroups(f.SegmentsByRouteIDs(ctx, nil, nil, routeIDs)) }},
			{"SegmentPatternsByRouteIDs", func() (int, error) { return countGroups(f.SegmentPatternsByRouteIDs(ctx, nil, nil, routeIDs)) }},
			{"SegmentPatternsBySegmentIDs", func() (int, error) { return countGroups(f.SegmentPatternsBySegmentIDs(ctx, nil, nil, segmentIDs)) }},
			{"ValidationReportsByFeedVersionIDs", func() (int, error) {
				return countGroups(f.ValidationReportsByFeedVersionIDs(ctx, nil, nil, []int{haFv}))
			}},
			{"FeedVersionServiceLevelsByFeedVersionIDs", func() (int, error) {
				return countGroups(f.FeedVersionServiceLevelsByFeedVersionIDs(ctx, nil, nil, []int{haFv}))
			}},
			{"PathwaysByFeedVersionIDs", func() (int, error) {
				return countGroups(f.PathwaysByFeedVersionIDs(ctx, nil, nil, nil, []int{wmataFv}))
			}},
			{"LevelsByFeedVersionIDs", func() (int, error) { return countGroups(f.LevelsByFeedVersionIDs(ctx, nil, nil, nil, []int{wmataFv})) }},
			{"FrequenciesByFeedVersionIDs", func() (int, error) {
				return countGroups(f.FrequenciesByFeedVersionIDs(ctx, nil, nil, nil, []int{exFv}))
			}},
			{"TransfersByFeedVersionIDs", func() (int, error) { return countGroups(f.TransfersByFeedVersionIDs(ctx, nil, nil, nil, []int{baFv})) }},
			{"LocationGroupStopsByFeedVersionIDs", func() (int, error) {
				return countGroups(f.LocationGroupStopsByFeedVersionIDs(ctx, nil, nil, nil, []int{flexFv}))
			}},
			{"StopAreasByFeedVersionIDs", func() (int, error) { return countGroups(f.StopAreasByFeedVersionIDs(ctx, nil, nil, nil, []int{ctFv})) }},
			{"TranslationsByFeedVersionIDs", func() (int, error) {
				return countGroups(f.TranslationsByFeedVersionIDs(ctx, nil, nil, nil, []int{ctFv}))
			}},
			{"AttributionsByFeedVersionIDs", func() (int, error) {
				return countGroups(f.AttributionsByFeedVersionIDs(ctx, nil, nil, nil, []int{ctFv}))
			}},
		}

		// Each loader has to return rows first, or its check below would pass
		// without exercising the permission filter.
		for _, c := range checks {
			n, err := c.count()
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if n == 0 {
				t.Fatalf("%s: returned no rows while the feeds are public", c.name)
			}
		}

		if _, err := cfg.Adapter.DBX().ExecContext(ctx, "update feed_states set public = false"); err != nil {
			t.Fatal(err)
		}
		for _, c := range checks {
			t.Run(c.name, func(t *testing.T) {
				n, err := c.count()
				assert.NoError(t, err)
				assert.Zero(t, n, "expected no rows once every feed is private")
			})
		}
	})
}

// countGroups counts the records that a grouped finder returned across its keys.
func countGroups[T any](groups [][]T, err error) (int, error) {
	n := 0
	for _, group := range groups {
		n += len(group)
	}
	return n, err
}

// countFound counts the records that a by-id finder returned; it returns nil
// for each record it cannot see.
func countFound[T any](ents []*T, errs []error) (int, error) {
	n := 0
	for _, ent := range ents {
		if ent != nil {
			n++
		}
	}
	return n, errors.Join(errs...)
}
