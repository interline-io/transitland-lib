package importer

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAgencyURLFilter_Filter(t *testing.T) {
	f := NewDefaultAgencyURLFilter("https://example.com/feeds/f-test")

	missing := &gtfs.Agency{AgencyID: tt.NewString("a1"), AgencyName: tt.NewString("A1"), AgencyTimezone: tt.NewTimezone("America/Los_Angeles")}
	if err := f.Filter(missing, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "https://example.com/feeds/f-test", missing.AgencyURL.Val, "missing agency_url should be filled")

	present := &gtfs.Agency{AgencyID: tt.NewString("a2"), AgencyName: tt.NewString("A2"), AgencyURL: tt.NewUrl("http://example.com"), AgencyTimezone: tt.NewTimezone("America/Los_Angeles")}
	if err := f.Filter(present, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "http://example.com", present.AgencyURL.Val, "existing agency_url should be left unchanged")
}

// TestDefaultAgencyURLFilter_NoCascade copies an agency missing agency_url with a
// route that references it. Without the filter the agency is dropped on the
// required-field error and the route cascade-drops as an orphan; with the filter
// both are kept.
func TestDefaultAgencyURLFilter_NoCascade(t *testing.T) {
	newReader := func() *direct.Reader {
		r := direct.NewReader()
		r.AgencyList = append(r.AgencyList, gtfs.Agency{
			AgencyID:       tt.NewString("a1"),
			AgencyName:     tt.NewString("Agency One"),
			AgencyTimezone: tt.NewTimezone("America/Los_Angeles"),
			// AgencyURL intentionally missing
		})
		r.RouteList = append(r.RouteList, gtfs.Route{
			RouteID:        tt.NewString("r1"),
			AgencyID:       tt.NewKey("a1"),
			RouteShortName: tt.NewString("1"),
			RouteType:      tt.NewInt(3),
		})
		return r
	}

	t.Run("without filter the agency and route are dropped", func(t *testing.T) {
		result, err := copier.CopyWithOptions(context.Background(), newReader(), direct.NewWriter(), copier.Options{})
		if err != nil {
			t.Fatal(err)
		}
		assert.Zero(t, result.EntityCount["agency.txt"], "agency should be dropped without the filter")
		assert.Zero(t, result.EntityCount["routes.txt"], "route should cascade-drop without the filter")
	})

	t.Run("with filter the agency and route are kept", func(t *testing.T) {
		opts := copier.Options{}
		opts.AddExtension(NewDefaultAgencyURLFilter("https://example.com/feeds/f-test"))
		result, err := copier.CopyWithOptions(context.Background(), newReader(), direct.NewWriter(), opts)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 1, result.EntityCount["agency.txt"], "agency should be kept with the filter")
		assert.Equal(t, 1, result.EntityCount["routes.txt"], "route should be kept with the filter")
		assert.Zero(t, result.SkipEntityErrorCount["agency.txt"], "agency should not be skipped for an entity error")
	})
}
