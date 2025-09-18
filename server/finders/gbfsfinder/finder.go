package gbfsfinder

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/caches/ecache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
)

type Finder struct {
	client           *redis.Client
	cache            *ecache.Cache[gbfs.GbfsFeed]
	ttlRecheck       time.Duration
	ttlExpire        time.Duration
	prefix           string
	bikeSearchKey    string
	stationSearchKey string
}

func NewFinder(client *redis.Client) *Finder {
	c := ecache.NewCache[gbfs.GbfsFeed](client, "gbfs")
	return &Finder{
		ttlRecheck:       5 * time.Minute,
		ttlExpire:        24 * time.Hour,
		cache:            c,
		client:           client,
		prefix:           "gbfs",
		bikeSearchKey:    fmt.Sprintf("%s:bike-bbox", "gbfs"),
		stationSearchKey: fmt.Sprintf("%s:station-bbox", "gbfs"),
	}
}

func (c *Finder) AddData(ctx context.Context, topic string, sf gbfs.GbfsFeed) error {
	// Save basic data
	if err := c.cache.SetTTL(ctx, topic, sf, c.ttlRecheck, c.ttlExpire); err != nil {
		return err
	}
	// Geosearch index bikes
	ts := time.Now().In(time.UTC).Unix()
	_ = ts
	if c.client != nil {
		bbox := geom.NewBounds(geom.XY)
		for _, ent := range sf.Bikes {
			bbox.Extend(geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{ent.Lon.Val, ent.Lat.Val}))
		}
		bc := fmt.Sprintf("%0.5f,%0.5f,%0.5f,%0.5f", bbox.Min(0), bbox.Min(1), bbox.Max(0), bbox.Max(1))
		if err := c.client.HSet(ctx, c.bikeSearchKey, topic, bc).Err(); err != nil {
			return err
		}
	}
	// Geosearch index docks
	if c.client != nil {
		bbox := geom.NewBounds(geom.XY)
		for _, ent := range sf.StationInformation {
			bbox.Extend(geom.NewPoint(geom.XY).MustSetCoords(geom.Coord{ent.Lon.Val, ent.Lat.Val}))
		}
		bc := fmt.Sprintf("%0.5f,%0.5f,%0.5f,%0.5f", bbox.Min(0), bbox.Min(1), bbox.Max(0), bbox.Max(1))
		if err := c.client.HSet(ctx, c.stationSearchKey, topic, bc).Err(); err != nil {
			return err
		}
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
	if c.client != nil {
		cmd := c.client.HGetAll(ctx, key)
		locs, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		for topicKey, loc := range locs {
			var coords []float64
			for _, c := range strings.Split(loc, ",") {
				cf, err := strconv.ParseFloat(c, 64)
				if err != nil {
					return nil, err
				}
				coords = append(coords, cf)
			}
			bbox := geom.NewBounds(geom.XY)
			bbox.Set(coords...)
			if bbox.OverlapsPoint(geom.XY, geom.Coord{pt.Lon, pt.Lat}) {
				// fmt.Println("in box", topicKey, pt.Lon, pt.Lat)
				topicKeys[topicKey] = true
			} else {
				// fmt.Println("not in box", topicKey, pt.Lon, pt.Lat)
			}

		}
	} else {
		// If not using redis, get local keys. This is not perfect.
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

func checkFloat(v *float64, min float64, max float64) float64 {
	if v == nil || *v < min {
		return min
	} else if *v > max {
		return max
	}
	return *v
}
