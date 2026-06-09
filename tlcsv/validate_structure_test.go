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
