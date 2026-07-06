package filters

import (
	"encoding/json"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ApplyDefaultAgencyURLFilter fills a missing agency_url with a default value.
//
// agency_url is required, so an agency missing one is dropped on import, cascading
// to its routes and trips. Filling a value before validation keeps them. The
// validator is unchanged and still reports the missing value; only the imported
// value is synthesized.
type ApplyDefaultAgencyURLFilter struct {
	url string
}

// NewApplyDefaultAgencyURLFilter returns a filter that fills missing agency_url values with url.
func NewApplyDefaultAgencyURLFilter(url string) *ApplyDefaultAgencyURLFilter {
	return &ApplyDefaultAgencyURLFilter{url: url}
}

func newApplyDefaultAgencyURLFilterFromJson(args string) (*ApplyDefaultAgencyURLFilter, error) {
	opts := &struct{ URL string }{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	return NewApplyDefaultAgencyURLFilter(opts.URL), nil
}

// Filter fills a missing agency_url; entities are never skipped, so it always returns nil.
func (f *ApplyDefaultAgencyURLFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	if agency, ok := ent.(*gtfs.Agency); ok && agency.AgencyURL.Val == "" {
		agency.AgencyURL.Set(f.url)
	}
	return nil
}
