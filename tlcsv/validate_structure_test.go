package tlcsv

import (
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/testpath"
)

// TestValidateStructure_Stops covers the conditional requirement of stops.txt.
// Per the GTFS spec stops.txt is conditionally required: it may be empty
// (header only) or omitted entirely when the feed provides location-based
// service via locations.geojson or location_groups.txt (GTFS-Flex).
func TestValidateStructure_Stops(t *testing.T) {
	stopsFileRequiredErrors := func(errs []error) int {
		n := 0
		for _, err := range errs {
			var fre *causes.FileRequiredError
			if errors.As(err, &fre) && fre.Filename == "stops.txt" {
				n++
			}
		}
		return n
	}

	cases := []struct {
		name              string
		dir               string
		wantStopsRequired bool
	}{
		{
			// Header-only stops.txt alongside locations.geojson (the WSDOT
			// MetroAccess flex case). An empty optional file must not error.
			name:              "header-only stops with locations.geojson",
			dir:               "testdata/flex/stops-header-only",
			wantStopsRequired: false,
		},
		{
			// stops.txt present with data rows but no recognized columns and no
			// flex alternative: the column-presence check must still fire. Guards
			// against the empty-file early return weakening column validation.
			name:              "stops present with unrecognized columns",
			dir:               "testdata/flex/stops-bad-columns",
			wantStopsRequired: true,
		},
		{
			// stops.txt omitted entirely, but location_groups.txt /
			// locations.geojson cover service: allowed for flex feeds.
			name:              "absent stops with flex alternative",
			dir:               "testdata/flex/stops-absent-flex",
			wantStopsRequired: false,
		},
		{
			// stops.txt omitted with location_groups.txt as the ONLY flex
			// alternative (no locations.geojson). Isolates the location_groups
			// branch of the alternative-present check.
			name:              "absent stops with location_groups only",
			dir:               "testdata/flex/stops-absent-locationgroups-only",
			wantStopsRequired: false,
		},
		{
			// stops.txt omitted and no flex alternative present: still an error.
			name:              "absent stops without flex alternative",
			dir:               "testdata/flex/stops-absent-no-flex",
			wantStopsRequired: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &Reader{Adapter: NewDirAdapter(testpath.RelPath(tc.dir))}
			if err := reader.Open(); err != nil {
				t.Fatalf("failed to open reader: %v", err)
			}
			defer reader.Close()
			errs := reader.ValidateStructure()
			got := stopsFileRequiredErrors(errs)
			if tc.wantStopsRequired && got == 0 {
				t.Errorf("expected a stops.txt FileRequiredError, got none (all errs: %v)", errs)
			}
			if !tc.wantStopsRequired && got > 0 {
				t.Errorf("expected no stops.txt FileRequiredError, got %d (all errs: %v)", got, errs)
			}
		})
	}
}

// TestValidateStructure_RequiresDataRow pins the header-vs-data-row distinction:
// the early-out at the first row must still tell a header-only file (treated as
// empty) apart from a file that has at least one data row. A required file errors
// in the first case and validates in the second.
func TestValidateStructure_RequiresDataRow(t *testing.T) {
	header := []string{"route_id", "route_short_name", "route_long_name", "route_type"}
	dataRow := []string{"r1", "R", "Route One", "3"}
	routesRequired := func(errs []error) bool {
		for _, err := range errs {
			var fre *causes.FileRequiredError
			if errors.As(err, &fre) && fre.Filename == "routes.txt" {
				return true
			}
		}
		return false
	}
	writeRoutes := func(t *testing.T, rows [][]string) *Reader {
		t.Helper()
		dir := t.TempDir()
		w := NewDirAdapter(dir)
		if err := w.WriteRows("routes.txt", rows); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		reader := &Reader{Adapter: NewDirAdapter(dir)}
		if err := reader.Open(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { reader.Close() })
		return reader
	}

	t.Run("header only is treated as empty", func(t *testing.T) {
		if !routesRequired(writeRoutes(t, [][]string{header}).ValidateStructure()) {
			t.Error("header-only routes.txt should report a FileRequiredError")
		}
	})
	t.Run("header plus a data row is not empty", func(t *testing.T) {
		if routesRequired(writeRoutes(t, [][]string{header, dataRow}).ValidateStructure()) {
			t.Error("routes.txt with a data row should not be treated as empty")
		}
	})
}

// TestValidateStructure_RequiredHeaderOnly ensures the empty-file handling does
// not weaken required files: a required file with only a header (no data rows)
// must still report a FileRequiredError.
func TestValidateStructure_RequiredHeaderOnly(t *testing.T) {
	reader := &Reader{Adapter: NewDirAdapter(testpath.RelPath("testdata/flex/required-header-only"))}
	if err := reader.Open(); err != nil {
		t.Fatalf("failed to open reader: %v", err)
	}
	defer reader.Close()
	found := false
	for _, err := range reader.ValidateStructure() {
		var fre *causes.FileRequiredError
		if errors.As(err, &fre) && fre.Filename == "routes.txt" {
			found = true
		}
	}
	if !found {
		t.Error("expected a routes.txt FileRequiredError for a header-only required file, got none")
	}
}
