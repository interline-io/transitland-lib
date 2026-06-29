package importer

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// DefaultAgencyURLFilter fills a missing agency_url with a default value.
//
// It is a copier Filter, so it runs before entity validation. agency_url is
// required by the GTFS spec, and without a value the agency is dropped on
// import; its routes then reference an unknown agency_id and are dropped, and
// their trips cascade-drop with them. Supplying a value keeps the agency (and
// everything referencing it) without changing the validator's spec-compliant
// behavior or the stored feed file: only the imported value is synthesized.
type DefaultAgencyURLFilter struct {
	URL string
}

// NewDefaultAgencyURLFilter returns a filter that fills missing agency_url values with url.
func NewDefaultAgencyURLFilter(url string) *DefaultAgencyURLFilter {
	return &DefaultAgencyURLFilter{URL: url}
}

// Filter fills a missing agency_url; entities are never skipped, so it always returns nil.
func (f *DefaultAgencyURLFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	if agency, ok := ent.(*gtfs.Agency); ok && agency.AgencyURL.Val == "" {
		agency.AgencyURL.Set(f.URL)
	}
	return nil
}
