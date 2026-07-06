package filters

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ApplyDefaultAgencyURLFilter fills a missing agency_url before validation, keeping
// the agency (and the routes and trips that would cascade-drop with it) from being
// discarded on import for the missing required field. It loosens no validation: a
// standalone validate run does not apply this filter and still reports the missing value.
type ApplyDefaultAgencyURLFilter struct {
	url string
}

// NewApplyDefaultAgencyURLFilter returns a filter that fills a missing agency_url with url, which must be a non-empty valid URL.
func NewApplyDefaultAgencyURLFilter(url string) (*ApplyDefaultAgencyURLFilter, error) {
	if url == "" {
		return nil, fmt.Errorf("a url must be specified")
	}
	if !tt.IsValidURL(url) {
		return nil, fmt.Errorf("invalid url '%s'", url)
	}
	return &ApplyDefaultAgencyURLFilter{url: url}, nil
}

func newApplyDefaultAgencyURLFilterFromJson(args string) (*ApplyDefaultAgencyURLFilter, error) {
	opts := &struct{ URL string }{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	return NewApplyDefaultAgencyURLFilter(opts.URL)
}

// Filter fills a missing agency_url; entities are never skipped, so it always returns nil.
func (f *ApplyDefaultAgencyURLFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	if agency, ok := ent.(*gtfs.Agency); ok && agency.AgencyURL.Val == "" {
		agency.AgencyURL.Set(f.url)
	}
	return nil
}
