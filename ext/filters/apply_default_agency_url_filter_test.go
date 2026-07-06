package filters

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestApplyDefaultAgencyURLFilter_Filter(t *testing.T) {
	f := NewApplyDefaultAgencyURLFilter("https://example.com/feeds/f-test")

	missing := &gtfs.Agency{AgencyID: tt.NewString("a1")}
	if err := f.Filter(missing, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "https://example.com/feeds/f-test", missing.AgencyURL.Val, "missing agency_url should be filled")

	present := &gtfs.Agency{AgencyID: tt.NewString("a2"), AgencyURL: tt.NewUrl("http://example.com")}
	if err := f.Filter(present, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "http://example.com", present.AgencyURL.Val, "existing agency_url should be left unchanged")
}
