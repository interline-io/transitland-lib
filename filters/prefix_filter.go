package filters

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

type PrefixFilter struct {
	prefixes          map[int]string
	prefixFiles       map[string]bool
	prefixFileDefault bool
}

func NewPrefixFilter() (*PrefixFilter, error) {
	return &PrefixFilter{
		prefixes:          map[int]string{},
		prefixFiles:       map[string]bool{},
		prefixFileDefault: true,
	}, nil
}

func newPrefixFilterFromJson(args string) (*PrefixFilter, error) {
	type prefixFilterOptions struct {
		Prefixes      map[string]string
		PrefixFiles   []string // backwards compat
		PrefixInclude []string
		PrefixExclude []string
	}
	pfx, _ := NewPrefixFilter()
	opts := &prefixFilterOptions{}
	if err := json.Unmarshal([]byte(args), opts); err != nil {
		return nil, err
	}
	for _, fn := range opts.PrefixFiles {
		pfx.setPrefixFile(fn, true)
	}
	for _, fn := range opts.PrefixInclude {
		pfx.setPrefixFile(fn, true)
	}
	for _, fn := range opts.PrefixExclude {
		pfx.setPrefixFile(fn, false)
	}
	return pfx, nil
}

func (filter *PrefixFilter) SetPrefix(fvid int, prefix string) {
	filter.prefixes[fvid] = prefix
}

func (filter *PrefixFilter) setPrefixFile(fn string, state bool) {
	filter.prefixFiles[fn] = state
}

func (filter *PrefixFilter) PrefixFile(fn string) {
	filter.prefixFileDefault = false
	filter.setPrefixFile(fn, true)
}

func (filter *PrefixFilter) UnprefixFile(fn string) {
	filter.prefixFileDefault = true
	filter.setPrefixFile(fn, false)
}

func (filter *PrefixFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	ok := filter.prefixFileDefault
	if prefixFile, prefixFileOk := filter.prefixFiles[ent.Filename()]; prefixFileOk {
		ok = prefixFile
	}
	if !ok {
		return nil
	}

	switch v := ent.(type) {
	case *gtfs.Stop:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.StopID.Set(fmt.Sprintf("%s%s", prefix, v.StopID.Val))
			if v.ZoneID.Valid {
				v.ZoneID.Set(fmt.Sprintf("%s%s", prefix, v.ZoneID.Val))
			}
		}
	case *gtfs.Agency:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.AgencyID.Set(fmt.Sprintf("%s%s", prefix, v.AgencyID.Val))
		}
	case *gtfs.Trip:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.TripID.Set(fmt.Sprintf("%s%s", prefix, v.TripID.Val))
			if v.BlockID.Valid {
				v.BlockID.Set(fmt.Sprintf("%s%s", prefix, v.BlockID.Val))
			}
		}
	case *gtfs.Route:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.RouteID.Set(fmt.Sprintf("%s%s", prefix, v.RouteID.Val))
		}
	case *gtfs.Calendar:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ServiceID.Set(fmt.Sprintf("%s%s", prefix, v.ServiceID.Val))
		}
	case *gtfs.Shape:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ShapeID.Set(fmt.Sprintf("%s%s", prefix, v.ShapeID.Val))
		}
	case *service.ShapeLine:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.ShapeID.Set(fmt.Sprintf("%s%s", prefix, v.ShapeID.Val))
		}
	case *gtfs.FareAttribute:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.FareID.Set(fmt.Sprintf("%s%s", prefix, v.FareID.Val))
		}
	case *gtfs.FareRule:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			if v.OriginID.Valid {
				v.OriginID.Set(fmt.Sprintf("%s%s", prefix, v.OriginID.Val))
			}
			if v.DestinationID.Valid {
				v.DestinationID.Set(fmt.Sprintf("%s%s", prefix, v.DestinationID))
			}
			if v.ContainsID.Valid {
				v.ContainsID.Set(fmt.Sprintf("%s%s", prefix, v.ContainsID))
			}
		}
	case *gtfs.Level:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.LevelID.Set(fmt.Sprintf("%s%s", prefix, v.LevelID.Val))
		}
	case *gtfs.Pathway:
		if prefix, ok := filter.getprefix(v.FeedVersionID); ok {
			v.PathwayID.Set(fmt.Sprintf("%s%s", prefix, v.PathwayID.Val))
		}
	default:
		return nil
	}
	// fmt.Println("prefixed:", ent.Filename(), ent.EntityID())
	return nil
}

func (filter *PrefixFilter) getprefix(fvid int) (string, bool) {
	prefix, ok := filter.prefixes[fvid]
	return prefix, ok
}
