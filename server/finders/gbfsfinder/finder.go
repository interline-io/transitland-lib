package gbfsfinder

import (
	"context"
	"encoding/json"
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
	// lastTTL bounds how long a system's last fetch survives in the store, and
	// with it how long a held copy is served after the last successful fetch.
	lastTTL = 24 * time.Hour
	// missingTTL is how long a topic the store could not answer for is
	// remembered as absent.
	missingTTL = 1 * time.Minute
	// reconnectDelay paces re-subscription attempts.
	reconnectDelay = 1 * time.Second
	// updatesChannel carries topic pointers to the notify-then-read listeners.
	updatesChannel = "gbfs:updates"
)

// Finder is a GbfsFinder over a kvcache.Store.
type Finder struct {
	store            kvcache.Store
	hashes           kvcache.HashStore   // nil when the store has no hash index
	pubsub           kvcache.PubSubStore // nil when the store has no pub/sub
	feeds            *kvcache.Cache[string, *gbfs.GbfsFeed]
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
		store:            store,
		bikeSearchKey:    "gbfs:bike-bbox",
		stationSearchKey: "gbfs:station-bbox",
		ctx:              ctx,
		cancel:           cancel,
	}
	f.feeds = kvcache.NewRefreshCache[string, *gbfs.GbfsFeed](nil, "gbfsfeed", f.readTopic)
	// A held system and the payload behind it both live lastTTL. Recheck is
	// pinned to the same bound so it never comes due first: nothing refreshes
	// in the background, updates arrive by announcement.
	f.feeds.Expires = lastTTL
	f.feeds.Recheck = lastTTL
	f.feeds.NegativeTTL = missingTTL
	// Expiry stops an entry being served but does not release it, and a large
	// system is megabytes. Scanning prunes them; with nothing ever due, that is
	// all this does.
	f.feeds.Start(missingTTL)
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
	c.feeds.Stop()
	return nil
}

// AddData stores a fetched system under topic, indexes it, and announces it to
// other finders. A cancelled caller does not interrupt it: data stored but never
// announced is not seen by other processes until the next fetch.
func (c *Finder) AddData(ctx context.Context, topic string, sf gbfs.GbfsFeed) error {
	ctx = context.WithoutCancel(ctx)
	data, err := json.Marshal(sf)
	if err != nil {
		return err
	}
	if err := c.store.Set(ctx, lastKey(topic), data, lastTTL); err != nil {
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
		// leaves the data stored and indexed. Every listener, this process
		// included, re-reads the payload if it holds the topic.
		return c.pubsub.Publish(ctx, updatesChannel, []byte(topic))
	}
	// No shared distribution: decode straight into the local tier, replacing
	// any record of absence. Decoded from the payload so this process holds
	// what any other reader would.
	held, err := decode(data)
	if err != nil {
		return err
	}
	return c.feeds.Set(ctx, topic, held)
}

// GetFeed returns the system stored under topic, loading it on first use. The
// system is shared with other readers and must not be modified.
func (c *Finder) GetFeed(ctx context.Context, topic string) (*model.GbfsFeed, bool) {
	// What is already held is served whatever state the caller is in. Only a
	// load is shed: it is detached from its caller by design, so starting one
	// for a caller that has gone away spends a store read nobody waits for.
	sf, ok := c.feeds.Peek(topic)
	if !ok && ctx.Err() == nil {
		sf, ok = c.feeds.Get(ctx, topic)
	}
	if !ok {
		return nil, false
	}
	return &model.GbfsFeed{GbfsFeed: sf}, true
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
		feed, ok := c.GetFeed(ctx, topicKey)
		if !ok {
			continue
		}
		for _, ent := range feed.Bikes {
			if d := tlxy.DistanceHaversine(ptxy, tlxy.Point{Lon: ent.Lon.Val, Lat: ent.Lat.Val}); d > pt.Radius {
				continue
			}
			b := model.GbfsFreeBikeStatus{
				FreeBikeStatus: ent,
				Feed:           feed,
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
		feed, ok := c.GetFeed(ctx, topicKey)
		if !ok {
			continue
		}
		for _, ent := range feed.GbfsFeed.StationInformation {
			if d := tlxy.DistanceHaversine(ptxy, tlxy.Point{Lon: ent.Lon.Val, Lat: ent.Lat.Val}); d > pt.Radius {
				continue
			}
			b := model.GbfsStationInformation{
				StationInformation: ent,
				Feed:               feed,
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
		for _, k := range c.feeds.LocalKeys() {
			topicKeys[k] = true
		}
	}
	var ret []string
	for k := range topicKeys {
		ret = append(ret, k)
	}
	return ret, nil
}

// readTopic reads and decodes a topic's last payload. It is the cache's
// refresh function, and reports every failure as kvcache.ErrNotFound so the
// topic is remembered as absent for missingTTL.
func (c *Finder) readTopic(ctx context.Context, topic string) (*gbfs.GbfsFeed, error) {
	// Logged apart: absent is a feed that has not fetched, failed is the store
	// not answering.
	data, ok, err := c.store.Get(ctx, lastKey(topic))
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Dur("retry_after", missingTTL).Msg("gbfsfinder: topic read failed, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	if !ok || len(data) == 0 {
		log.For(ctx).Trace().Str("topic", topic).Dur("retry_after", missingTTL).Msg("gbfsfinder: topic absent from store, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	sf, err := decode(data)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Dur("retry_after", missingTTL).Msg("gbfsfinder: topic decode failed, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	return sf, nil
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
			c.handleUpdate(string(msg))
		}
	}
}

// handleUpdate re-reads an announced topic, replacing this process's copy.
func (c *Finder) handleUpdate(topic string) {
	// Every fetch in the fleet is announced here, so a process takes only the
	// topics it holds; the rest load on first read. A writer holds nothing it
	// has not read, so its own announcements pass through.
	if !c.feeds.Contains(topic) {
		return
	}
	// Reload rather than Refresh: an update this process could not read is a
	// failure to observe a change, not evidence the system went away, and must
	// not replace a good copy with a record of absence.
	if _, err := c.feeds.Reload(c.ctx, topic); err == nil {
		log.For(c.ctx).Trace().Str("topic", topic).Msg("gbfsfinder: processed update")
	}
}

func decode(data []byte) (*gbfs.GbfsFeed, error) {
	var sf gbfs.GbfsFeed
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return &sf, nil
}

func lastKey(topic string) string {
	return "gbfs:last:" + topic
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
