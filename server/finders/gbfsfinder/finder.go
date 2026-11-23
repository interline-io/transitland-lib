package gbfsfinder

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/twpayne/go-geom"
)

const (
	// updatesChannel announces that a topic was written, so every finder
	// sharing the store can re-read the topics it holds.
	updatesChannel = "gbfs:updates"
	// reconnectDelay paces re-subscription attempts.
	reconnectDelay = 1 * time.Second
	// writeTimeout bounds storing and announcing one fetch.
	writeTimeout = 5 * time.Second
)

// Finder is a GbfsFinder over a kvcache.Store.
type Finder struct {
	cache            *kvcache.Cache[string, gbfs.GbfsFeed]
	hashes           kvcache.HashStore   // nil when the store has no hash index
	pubsub           kvcache.PubSubStore // nil when the store has no pub/sub
	ttlRecheck       time.Duration
	ttlExpire        time.Duration
	bikeSearchKey    string
	stationSearchKey string
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewFinder returns a GbfsFinder backed by store. Close it to stop following
// other processes' fetches.
//
// A HashStore holds the cross-process bounding-box index; without one,
// geosearch falls back to locally known topics. A PubSubStore carries fetches
// made by other processes into this one.
func NewFinder(store kvcache.Store) *Finder {
	ctx, cancel := context.WithCancel(context.Background())
	f := &Finder{
		ttlRecheck: 5 * time.Minute,
		ttlExpire:  24 * time.Hour,
		cache:      kvcache.NewCache[string, gbfs.GbfsFeed](store, "gbfs"),
		// Not the per-language hashes an earlier keying wrote: nothing prunes a
		// hash field, so those are left unread rather than returned twice.
		bikeSearchKey:    "gbfs:feed-bike-bbox",
		stationSearchKey: "gbfs:feed-station-bbox",
		ctx:              ctx,
		cancel:           cancel,
	}
	if hs, ok := store.(kvcache.HashStore); ok {
		f.hashes = hs
	}
	if ps, ok := store.(kvcache.PubSubStore); ok {
		f.pubsub = ps
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			f.subscribe()
		}()
	}
	return f
}

// Close stops following other processes' fetches.
func (c *Finder) Close() error {
	c.cancel()
	c.wg.Wait()
	return nil
}

// AddData stores a fetched system under topic, indexes it, and announces it to
// other finders. The caller's cancellation does not apply: a cancelled job
// still completes the write.
func (c *Finder) AddData(ctx context.Context, topic string, sf gbfs.GbfsFeed) error {
	// Data stored but never announced is not seen by other processes until the
	// next fetch, so this outlives the job that fetched it.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
	defer cancel()
	if err := c.cache.SetTTL(ctx, topic, sf, c.ttlRecheck, c.ttlExpire); err != nil {
		return err
	}
	if c.hashes != nil {
		// Index bike and dock bounding boxes for cross-process geosearch.
		bikeBox := bboxString(sf.Bikes, func(e *gbfs.FreeBikeStatus) (float64, float64) { return e.Lon.Val, e.Lat.Val })
		if err := c.hashes.HSet(ctx, c.bikeSearchKey, topic, bikeBox); err != nil {
			return err
		}
		stationBox := bboxString(sf.StationInformation, func(e *gbfs.StationInformation) (float64, float64) { return e.Lon.Val, e.Lat.Val })
		if err := c.hashes.HSet(ctx, c.stationSearchKey, topic, stationBox); err != nil {
			return err
		}
	}
	if c.pubsub != nil {
		// Last, so a failed announcement costs other processes freshness but
		// leaves the data stored and indexed. A finder re-reading on it finds
		// the new data.
		return c.pubsub.Publish(ctx, updatesChannel, []byte(topic))
	}
	return nil
}

// GetFeed returns the system stored under topic.
func (c *Finder) GetFeed(ctx context.Context, topic string) (*model.GbfsFeed, bool) {
	sf, ok := c.cache.Get(ctx, topic)
	if !ok {
		return nil, false
	}
	return &model.GbfsFeed{GbfsFeed: &sf}, true
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
			bbox, _ := tlxy.ParseBbox(loc)
			if bbox.Contains(tlxy.Point{Lon: pt.Lon, Lat: pt.Lat}) {
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

// subscribe re-reads each announced topic this process holds, until Close.
//
// Announcements are not replayed. One missed while the subscription is down —
// including across a reconnect the Redis client makes on its own, which this
// never sees — leaves that topic as it was until its next fetch.
func (c *Finder) subscribe() {
	for c.ctx.Err() == nil {
		sub, err := c.pubsub.Subscribe(c.ctx, updatesChannel)
		if err != nil {
			log.For(c.ctx).Error().Err(err).Msg("gbfsfinder: error subscribing to updates")
		} else {
			c.drain(sub)
			_ = sub.Close()
		}
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *Finder) drain(sub kvcache.Subscription) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-sub.Messages():
			if !ok {
				return
			}
			// Every fetch in the fleet is announced here. Only topics this
			// process has read are worth a store read; the rest load on demand.
			if topic := string(msg); c.cache.Contains(topic) {
				c.cache.Sync(c.ctx, topic)
			}
		}
	}
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
