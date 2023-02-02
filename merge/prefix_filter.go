package merge

import (
	"fmt"

	"github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/tl"
)

type PrefixFilter struct {
	feedVersionPrefixes map[int]string
}

func (filter *PrefixFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Stop:
		prefix := filter.getprefix(v.FeedVersionID)
		v.StopID = fmt.Sprintf("%s:%s", prefix, v.StopID)
		if v.ZoneID != "" {
			v.ZoneID = fmt.Sprintf("%s:%s", prefix, v.ZoneID)
		}
	case *tl.Agency:
		prefix := filter.getprefix(v.FeedVersionID)
		v.AgencyID = fmt.Sprintf("%s:%s", prefix, v.AgencyID)
	case *tl.Trip:
		prefix := filter.getprefix(v.FeedVersionID)
		v.TripID = fmt.Sprintf("%s:%s", prefix, v.TripID)
		if v.BlockID != "" {
			v.BlockID = fmt.Sprintf("%s:%s", prefix, v.BlockID)
		}
	case *tl.Route:
		v.RouteID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.RouteID)
	case *tl.Service:
		v.ServiceID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.ServiceID)
	case *tl.Shape:
		v.ShapeID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.ShapeID)
	case *tl.FareAttribute:
		prefix := filter.getprefix(v.FeedVersionID)
		v.FareID = fmt.Sprintf("%s:%s", prefix, v.FareID)
	case *tl.FareRule:
		prefix := filter.getprefix(v.FeedVersionID)
		if v.OriginID != "" {
			v.OriginID = fmt.Sprintf("%s:%s", prefix, v.OriginID)
		}
		if v.DestinationID != "" {
			v.DestinationID = fmt.Sprintf("%s:%s", prefix, v.DestinationID)
		}
		if v.ContainsID != "" {
			v.ContainsID = fmt.Sprintf("%s:%s", prefix, v.ContainsID)
		}
	case *plus.FarezoneAttribute:
		v.ZoneID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.ZoneID)
	case *tl.Level:
		v.LevelID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.LevelID)
	case *tl.Pathway:
		v.PathwayID = fmt.Sprintf("%s:%s", filter.getprefix(v.FeedVersionID), v.PathwayID)
	default:
	}
	return nil
}

func (filter *PrefixFilter) getprefix(fvid int) string {
	prefix, ok := filter.feedVersionPrefixes[fvid]
	if !ok {
		panic("no prefix")
	}
	return prefix
}
