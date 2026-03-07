package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	ilog "github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/finders/dbfinder"
	"github.com/interline-io/transitland-lib/server/finders/rtfinder"
	"github.com/interline-io/transitland-lib/server/model"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	_ "github.com/interline-io/transitland-lib/tldb/postgres"
)

// Oakland center
const (
	centerLat = 37.8044
	centerLon = -122.2712
	radiusDeg = 0.02 // ~2km
)

func main() {
	port := flag.String("port", "8888", "HTTP port for test RT server")
	dbURL := flag.String("dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	redisURL := flag.String("redisurl", "", "Redis URL (default: $TL_REDIS_URL)")
	fetchInterval := flag.Int("fetch-interval", 10, "Fetch RT feeds every N seconds")
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("TL_DATABASE_URL")
	}
	if *redisURL == "" {
		*redisURL = os.Getenv("TL_REDIS_URL")
	}

	ctx := context.Background()

	// Start RT fetch loop if DB and Redis are configured
	if *dbURL != "" && *redisURL != "" {
		db, err := dbutil.OpenDB(*dbURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening database: %s\n", err)
			os.Exit(1)
		}
		redisClient, err := dbutil.OpenRedis(*redisURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening redis: %s\n", err)
			os.Exit(1)
		}
		finder := dbfinder.NewFinder(db)
		cache := rtfinder.NewRedisCache(redisClient)
		interval := time.Duration(*fetchInterval) * time.Second
		go rtFetchLoop(ctx, finder, cache, interval)
	} else {
		fmt.Println("Note: --dburl and --redisurl not set, fetch loop disabled")
	}

	// Serve test RT data
	http.HandleFunc("/vehicle_positions.pb", handleVehiclePositions)
	fmt.Printf("RT test server listening on :%s\n", *port)
	fmt.Println("  GET /vehicle_positions.pb")
	fmt.Println("  Optional query param: ?time=<unix_timestamp>")
	fmt.Println("  Set Accept: application/json for JSON output")
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		fmt.Printf("error: %s\n", err)
	}
}

// RT fetch loop — queries DB for RT feeds and pushes data to Redis

func rtFetchLoop(ctx context.Context, finder *dbfinder.Finder, cache *rtfinder.RedisCache, interval time.Duration) {
	ilog.For(ctx).Info().Msgf("RT fetch: starting background fetcher every %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	rtFetchAll(ctx, finder, cache)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rtFetchAll(ctx, finder, cache)
		}
	}
}

func rtFetchAll(ctx context.Context, finder *dbfinder.Finder, cache *rtfinder.RedisCache) {
	spec := model.FeedSpecTypesGtfsRt
	feeds, err := finder.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{Spec: []model.FeedSpecTypes{spec}})
	if err != nil {
		ilog.For(ctx).Error().Err(err).Msg("RT fetch: error finding feeds")
		return
	}
	type rtFeed struct {
		feedID  string
		url     string
		urlType string
	}
	var toFetch []rtFeed
	for _, feed := range feeds {
		if u := feed.URLs.RealtimeVehiclePositions; u != "" {
			toFetch = append(toFetch, rtFeed{feed.FeedID, u, "realtime_vehicle_positions"})
		}
		if u := feed.URLs.RealtimeTripUpdates; u != "" {
			toFetch = append(toFetch, rtFeed{feed.FeedID, u, "realtime_trip_updates"})
		}
		if u := feed.URLs.RealtimeAlerts; u != "" {
			toFetch = append(toFetch, rtFeed{feed.FeedID, u, "realtime_alerts"})
		}
	}
	ilog.For(ctx).Info().Int("feeds", len(feeds)).Int("urls", len(toFetch)).Msg("RT fetch: fetching")
	for _, f := range toFetch {
		if err := rtFetchOne(ctx, cache, f.feedID, f.url, f.urlType); err != nil {
			ilog.For(ctx).Error().Err(err).Str("feed", f.feedID).Str("url_type", f.urlType).Msg("RT fetch: error")
		}
	}
}

func rtFetchOne(ctx context.Context, cache *rtfinder.RedisCache, feedID string, url string, urlType string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("rtdata:%s:%s", feedID, urlType)
	ilog.For(ctx).Info().Str("feed", feedID).Str("url_type", urlType).Int("bytes", len(data)).Msg("RT fetch: ok")
	return cache.AddData(ctx, key, data)
}

// Test RT data server

func buildMessage(now time.Time) *pb.FeedMessage {
	// Position within the hour: 0.0 to 1.0
	secondsIntoHour := float64(now.Minute()*60 + now.Second())
	fraction := secondsIntoHour / 3600.0
	angle := fraction * 2 * math.Pi

	lat := float32(centerLat + radiusDeg*math.Sin(angle))
	lon := float32(centerLon + radiusDeg*math.Cos(angle))
	bearing := float32(math.Mod(math.Mod(90-angle*180/math.Pi, 360)+360, 360))
	speed := float32(2 * math.Pi * radiusDeg * 111000 / 3600) // approx m/s

	ts := uint64(now.Unix())
	headerTs := uint64(now.Unix())
	version := "2.0"
	incrementality := pb.FeedHeader_FULL_DATASET

	vehicleID := "test-vehicle-1"
	tripID := "test-trip-1"
	routeID := "test-route-1"
	label := "Oakland Circle Bus"
	entityID := "vehicle-1"
	status := pb.VehiclePosition_IN_TRANSIT_TO

	return &pb.FeedMessage{
		Header: &pb.FeedHeader{
			GtfsRealtimeVersion: &version,
			Incrementality:      &incrementality,
			Timestamp:           &headerTs,
		},
		Entity: []*pb.FeedEntity{
			{
				Id: &entityID,
				Vehicle: &pb.VehiclePosition{
					Vehicle: &pb.VehicleDescriptor{
						Id:    &vehicleID,
						Label: &label,
					},
					Trip: &pb.TripDescriptor{
						TripId:  &tripID,
						RouteId: &routeID,
					},
					Position: &pb.Position{
						Latitude:  &lat,
						Longitude: &lon,
						Bearing:   &bearing,
						Speed:     &speed,
					},
					Timestamp:     &ts,
					CurrentStatus: &status,
				},
			},
		},
	}
}

func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json") || strings.Contains(accept, "text/json")
}

func handleVehiclePositions(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	if tStr := r.URL.Query().Get("time"); tStr != "" {
		if unix, err := strconv.ParseInt(tStr, 10, 64); err == nil {
			now = time.Unix(unix, 0).UTC()
		}
	}

	msg := buildMessage(now)
	pos := msg.Entity[0].Vehicle.Position

	if wantsJSON(r) {
		data, err := protojson.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	} else {
		data, err := proto.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(data)
	}

	fmt.Printf("[%s] Served vehicle at (%.4f, %.4f) bearing=%.1f\n", now.Format("15:04:05"), pos.GetLatitude(), pos.GetLongitude(), pos.GetBearing())
}

