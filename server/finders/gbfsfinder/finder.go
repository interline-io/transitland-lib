package gbfsfinder

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
)

type Finder struct {
	cache            *kvcache.Cache[string, gbfs.GbfsFeed]
	hashes           kvcache.HashStore // nil when the store has no hash index
	ttlRecheck       time.Duration
	ttlExpire        time.Duration
	bikeSearchKey    string
	stationSearchKey string
}

// NewFinder returns a GbfsFinder backed by store. When store supports the
// HashStore capability it holds the cross-process bounding-box index;
// otherwise geosearch falls back to locally known topics.
func NewFinder(store kvcache.Store) *Finder {
	f := &Finder{
		ttlRecheck:       5 * time.Minute,
		ttlExpire:        24 * time.Hour,
		cache:            kvcache.NewCache[string, gbfs.GbfsFeed](store, "gbfs"),
		bikeSearchKey:    "gbfs:bike-bbox",
		stationSearchKey: "gbfs:station-bbox",
	}
	if hs, ok := store.(kvcache.HashStore); ok {
		f.hashes = hs
	}
	return f
}

func (c *Finder) AddData(ctx context.Context, topic string, sf gbfs.GbfsFeed) error {
	// Save basic data
	if err := c.cache.SetTTL(ctx, topic, sf, c.ttlRecheck, c.ttlExpire); err != nil {
		return err
	}
	if c.hashes == nil {
		return nil
	}
	// Index bike and dock bounding boxes for cross-process geosearch.
	bikeBox := bboxString(sf.Bikes, func(e *gbfs.FreeBikeStatus) (float64, float64) { return e.Lon.Val, e.Lat.Val })
	if err := c.hashes.HSet(ctx, c.bikeSearchKey, topic, []byte(bikeBox)); err != nil {
		return err
	}
	stationBox := bboxString(sf.StationInformation, func(e *gbfs.StationInformation) (float64, float64) { return e.Lon.Val, e.Lat.Val })
	if err := c.hashes.HSet(ctx, c.stationSearchKey, topic, []byte(stationBox)); err != nil {
		return err
	}
	return nil
}

func (c *Finder) FindBikes(ctx context.Context, limit *int, where *model.GbfsBikeRequest) ([]*model.GbfsFreeBikeStatus, error) {
	if where == nil || where.Near == nil {
		return nil, nil
	}
	where.Near.Radius = checkFloat(&where.Near.Radius, 0, 1_000_000)
	pt := *where.Near
	ptxy := tlxy.Point{Lon: pt.Lon, Lat: pt.Lat}
	topicKeys, err := c.geosearch(ctx, c.bikeSearchKey, pt)
	if err != nil {
		return nil, err
	}
	var ret []*model.GbfsFreeBikeStatus
	for _, topicKey := range topicKeys {
		sf, ok := c.cache.Get(ctx, topicKey)
		if !ok {
			continue
		}
		for _, ent := range sf.Bikes {
			if d := tlxy.DistanceHaversine(ptxy, tlxy.Point{Lon: ent.Lon.Val, Lat: ent.Lat.Val}); d > pt.Radius {
				continue
			}
			b := model.GbfsFreeBikeStatus{
				FreeBikeStatus: ent,
				Feed:           &model.GbfsFeed{GbfsFeed: &sf},
			}
			ret = append(ret, &b)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].BikeID.Val < ret[j].BikeID.Val
	})
	if limit != nil && len(ret) > *limit {
		ret = ret[0:*limit]
	}
	return ret, nil
}

func (c *Finder) FindDocks(ctx context.Context, limit *int, where *model.GbfsDockRequest) ([]*model.GbfsStationInformation, error) {
	if where == nil || where.Near == nil {
		return nil, nil
	}
	where.Near.Radius = checkFloat(&where.Near.Radius, 0, 1_000_000)
	pt := *where.Near
	ptxy := tlxy.Point{Lon: pt.Lon, Lat: pt.Lat}
	topicKeys, err := c.geosearch(ctx, c.stationSearchKey, pt)
	if err != nil {
		return nil, err
	}
	var ret []*model.GbfsStationInformation
	for _, topicKey := range topicKeys {
		sf, ok := c.cache.Get(ctx, topicKey)
		if !ok {
			continue
		}
		for _, ent := range sf.StationInformation {
			if d := tlxy.DistanceHaversine(ptxy, tlxy.Point{Lon: ent.Lon.Val, Lat: ent.Lat.Val}); d > pt.Radius {
				continue
			}
			b := model.GbfsStationInformation{
				StationInformation: ent,
				Feed:               &model.GbfsFeed{GbfsFeed: &sf},
			}
			ret = append(ret, &b)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].StationID.Val < ret[j].StationID.Val
	})
	if limit != nil && len(ret) > *limit {
		ret = ret[0:*limit]
	}
	return ret, nil
}

func (c *Finder) geosearch(ctx context.Context, key string, pt model.PointRadius) ([]string, error) {
	topicKeys := map[string]bool{}
	if c.hashes != nil {
		locs, err := c.hashes.HGetAll(ctx, key)
		if err != nil {
			return nil, err
		}
		for topicKey, loc := range locs {
			var coords []float64
			for _, c := range strings.Split(string(loc), ",") {
				cf, err := strconv.ParseFloat(c, 64)
				if err != nil {
					return nil, err
				}
				coords = append(coords, cf)
			}
			bbox := geom.NewBounds(geom.XY)
			bbox.Set(coords...)
			if bbox.OverlapsPoint(geom.XY, geom.Coord{pt.Lon, pt.Lat}) {
				topicKeys[topicKey] = true
			}
		}
	} else {
		// No shared bbox index: fall back to locally known topics.
		for _, k := range c.cache.LocalKeys() {
			topicKeys[k] = true
		}
	}
	var ret []string
	for k := range topicKeys {
		ret = append(ret, k)
	}
	return ret, nil
}

// bboxString returns the "minX,minY,maxX,maxY" bounding box of ents, whose
// lon/lat are read by coord.
func bboxString[T any](ents []T, coord func(T) (lon float64, lat float64)) string {
	bbox := geom.NewBounds(geom.XY)
	for _, ent := range ents {
		lon, lat := coord(ent)
		bbox.Extend(geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{lon, lat}))
	}
	return fmt.Sprintf("%0.5f,%0.5f,%0.5f,%0.5f", bbox.Min(0), bbox.Min(1), bbox.Max(0), bbox.Max(1))
}

func checkFloat(v *float64, min float64, max float64) float64 {
	if v == nil || *v < min {
		return min
	} else if *v > max {
		return max
	}
	return *v
}
