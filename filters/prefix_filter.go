package filters

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type PrefixFilter struct {
	PrefixAll   bool
	prefixes    map[int]string
	prefixFiles map[string]bool
}

func NewPrefixFilter() (*PrefixFilter, error) {
	return &PrefixFilter{
		prefixes:    map[int]string{},
		prefixFiles: map[string]bool{},
	}, nil
}

func newPrefixFilterFromJson(args string) (*PrefixFilter, error) {
	type prefixFilterOptions struct {
		PrefixAll   bool
		Prefixes    map[string]string
		PrefixFiles []string
	}
	pfx, _ := NewPrefixFilter()
	opts := &prefixFilterOptions{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	pfx.PrefixAll = opts.PrefixAll
	for _, fn := range opts.PrefixFiles {
		pfx.prefixFiles[fn] = true
	}
	// for k, v := range opts.Prefixes {
	// 	pfx.SetPrefix(k, v)
	// }
	return pfx, nil
}

func (filter *PrefixFilter) SetPrefix(fvid int, prefix string) {
	filter.prefixes[fvid] = prefix
}

func (filter *PrefixFilter) PrefixFile(fn string) {
	filter.prefixFiles[fn] = true
}

func (filter *PrefixFilter) Filter(ent tl.Entity, emap *tt.EntityMap) error {
	if _, ok := filter.prefixFiles[ent.Filename()]; !(ok || filter.PrefixAll) {
		return nil
	}
	switch v := ent.(type) {
	case *tl.Stop:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.StopID = fmt.Sprintf("%s%s", prefix, v.StopID)
			if v.ZoneID != "" {
				v.ZoneID = fmt.Sprintf("%s%s", prefix, v.ZoneID)
			}
		}
	case *tl.Agency:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.AgencyID = fmt.Sprintf("%s%s", prefix, v.AgencyID)
		}
	case *tl.Trip:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.TripID = fmt.Sprintf("%s%s", prefix, v.TripID)
			if v.BlockID != "" {
				v.BlockID = fmt.Sprintf("%s%s", prefix, v.BlockID)
			}
		}
	case *tl.Route:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.RouteID = fmt.Sprintf("%s%s", prefix, v.RouteID)
		}
	case *tlutil.Service:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ServiceID = fmt.Sprintf("%s%s", prefix, v.ServiceID)
		}
	case *tl.Calendar:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ServiceID = fmt.Sprintf("%s%s", prefix, v.ServiceID)
		}
	case *tl.Shape:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ShapeID = fmt.Sprintf("%s%s", prefix, v.ShapeID)
		}
	case *tl.FareAttribute:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.FareID = fmt.Sprintf("%s%s", prefix, v.FareID)
		}
	case *tl.FareRule:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			if v.OriginID != "" {
				v.OriginID = fmt.Sprintf("%s%s", prefix, v.OriginID)
			}
			if v.DestinationID != "" {
				v.DestinationID = fmt.Sprintf("%s%s", prefix, v.DestinationID)
			}
			if v.ContainsID != "" {
				v.ContainsID = fmt.Sprintf("%s%s", prefix, v.ContainsID)
			}
		}
	case *tl.Level:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.LevelID = fmt.Sprintf("%s%s", prefix, v.LevelID)
		}
	case *tl.Pathway:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.PathwayID = fmt.Sprintf("%s%s", prefix, v.PathwayID)
		}
	default:
	}
	return nil
}

func (filter *PrefixFilter) getprefix(fvid int) (string, bool) {
	prefix, ok := filter.prefixes[fvid]
	return prefix, ok
}
