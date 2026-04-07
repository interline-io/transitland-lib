package gql

import (
	"context"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func buildTestVehiclePositionData(t *testing.T) []byte {
	t.Helper()
	lat := float32(37.8044)
	lon := float32(-122.2712)
	bearing := float32(90.0)
	speed := float32(10.0)
	ts := uint64(time.Now().Unix())
	version := "2.0"
	incrementality := pb.FeedHeader_FULL_DATASET
	vehicleID := "test-vehicle-1"
	label := "Test Bus"
	entityID := "entity-1"
	tripID := "trip-1"
	routeID := "route-1"
	status := pb.VehiclePosition_IN_TRANSIT_TO

	msg := &pb.FeedMessage{
		Header: &pb.FeedHeader{
			GtfsRealtimeVersion: &version,
			Incrementality:      &incrementality,
			Timestamp:           &ts,
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
	data, err := proto.Marshal(msg)
	require.NoError(t, err)
	return data
}

// newSubscriptionTestClient creates a test client and returns the config so
// tests can push RT data into the cache via cfg.RTFinder.AddData.
func newSubscriptionTestClient(t *testing.T) (*client.Client, model.Config) {
	t.Helper()
	cfg := testconfig.Config(t, testconfig.Options{})
	srv, _ := NewServer()
	graphqlServer := model.AddConfigAndPerms(cfg, srv)
	srvMiddleware := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})
	return client.New(srvMiddleware(graphqlServer)), cfg
}

func TestSubscriptionVehiclePositions(t *testing.T) {
	c, cfg := newSubscriptionTestClient(t)
	ctx := context.Background()

	// Push VP data into cache before subscribing so the initial snapshot has data
	vpData := buildTestVehiclePositionData(t)
	err := cfg.RTFinder.AddData(ctx, "rtdata:test-feed:realtime_vehicle_positions", vpData)
	require.NoError(t, err)

	// Subscribe
	sub := c.Websocket(`subscription { vehicle_positions { feed_onestop_id bearing speed position vehicle { id label } trip { trip_id route_id } } }`)
	defer sub.Close()

	// Read initial snapshot
	var resp struct {
		VehiclePositions []struct {
			FeedOnestopID string   `json:"feed_onestop_id"`
			Bearing       *float64 `json:"bearing"`
			Speed         *float64 `json:"speed"`
			Position      any      `json:"position"`
			Vehicle       *struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"vehicle"`
			Trip *struct {
				TripID  string `json:"trip_id"`
				RouteID string `json:"route_id"`
			} `json:"trip"`
		} `json:"vehicle_positions"`
	}
	err = sub.Next(&resp)
	require.NoError(t, err)
	require.Len(t, resp.VehiclePositions, 1)

	vp := resp.VehiclePositions[0]
	assert.Equal(t, "test-feed", vp.FeedOnestopID)
	assert.NotNil(t, vp.Bearing)
	assert.InDelta(t, 90.0, *vp.Bearing, 0.1)
	assert.NotNil(t, vp.Speed)
	assert.InDelta(t, 10.0, *vp.Speed, 0.1)
	assert.NotNil(t, vp.Position)
	require.NotNil(t, vp.Vehicle)
	assert.Equal(t, "test-vehicle-1", vp.Vehicle.ID)
	assert.Equal(t, "Test Bus", vp.Vehicle.Label)
	require.NotNil(t, vp.Trip)
	assert.Equal(t, "trip-1", vp.Trip.TripID)
	assert.Equal(t, "route-1", vp.Trip.RouteID)
}

func TestSubscriptionVehiclePositions_FilterFeed(t *testing.T) {
	c, cfg := newSubscriptionTestClient(t)
	ctx := context.Background()

	// Push VP data for two feeds
	vpData := buildTestVehiclePositionData(t)
	err := cfg.RTFinder.AddData(ctx, "rtdata:feed-a:realtime_vehicle_positions", vpData)
	require.NoError(t, err)
	err = cfg.RTFinder.AddData(ctx, "rtdata:feed-b:realtime_vehicle_positions", vpData)
	require.NoError(t, err)

	// Subscribe with feed filter — only feed-a
	sub := c.Websocket(`subscription { vehicle_positions(where: {feed_onestop_ids: ["feed-a"]}) { feed_onestop_id } }`)
	defer sub.Close()

	var resp struct {
		VehiclePositions []struct {
			FeedOnestopID string `json:"feed_onestop_id"`
		} `json:"vehicle_positions"`
	}
	err = sub.Next(&resp)
	require.NoError(t, err)
	require.Len(t, resp.VehiclePositions, 1)
	assert.Equal(t, "feed-a", resp.VehiclePositions[0].FeedOnestopID)
}

func TestSubscriptionVehiclePositions_FilterBbox(t *testing.T) {
	c, cfg := newSubscriptionTestClient(t)
	ctx := context.Background()

	// Test data is at lat=37.8044, lon=-122.2712
	vpData := buildTestVehiclePositionData(t)
	err := cfg.RTFinder.AddData(ctx, "rtdata:test-feed:realtime_vehicle_positions", vpData)
	require.NoError(t, err)

	t.Run("matching bbox", func(t *testing.T) {
		sub := c.Websocket(`subscription { vehicle_positions(where: {bbox: {min_lon: -123, min_lat: 37, max_lon: -122, max_lat: 38}}) { feed_onestop_id } }`)
		defer sub.Close()

		var resp struct {
			VehiclePositions []struct {
				FeedOnestopID string `json:"feed_onestop_id"`
			} `json:"vehicle_positions"`
		}
		err := sub.Next(&resp)
		require.NoError(t, err)
		assert.Len(t, resp.VehiclePositions, 1)
	})

	t.Run("non-matching bbox", func(t *testing.T) {
		sub := c.Websocket(`subscription { vehicle_positions(where: {bbox: {min_lon: 0, min_lat: 0, max_lon: 1, max_lat: 1}}) { feed_onestop_id } }`)
		defer sub.Close()

		var resp struct {
			VehiclePositions []struct {
				FeedOnestopID string `json:"feed_onestop_id"`
			} `json:"vehicle_positions"`
		}
		err := sub.Next(&resp)
		require.NoError(t, err)
		assert.Len(t, resp.VehiclePositions, 0)
	})
}

func TestSubscriptionVehiclePositions_LiveUpdate(t *testing.T) {
	c, cfg := newSubscriptionTestClient(t)
	ctx := context.Background()

	// Subscribe first with no data — initial snapshot should be empty
	sub := c.Websocket(`subscription { vehicle_positions { feed_onestop_id vehicle { id } } }`)
	defer sub.Close()

	var resp struct {
		VehiclePositions []struct {
			FeedOnestopID string `json:"feed_onestop_id"`
			Vehicle       *struct {
				ID string `json:"id"`
			} `json:"vehicle"`
		} `json:"vehicle_positions"`
	}

	// Initial snapshot: empty
	err := sub.Next(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.VehiclePositions, 0)

	// Push data — should trigger a live update
	vpData := buildTestVehiclePositionData(t)
	err = cfg.RTFinder.AddData(ctx, "rtdata:test-feed:realtime_vehicle_positions", vpData)
	require.NoError(t, err)

	// Read the live update
	err = sub.Next(&resp)
	require.NoError(t, err)
	require.Len(t, resp.VehiclePositions, 1)
	assert.Equal(t, "test-feed", resp.VehiclePositions[0].FeedOnestopID)
	require.NotNil(t, resp.VehiclePositions[0].Vehicle)
	assert.Equal(t, "test-vehicle-1", resp.VehiclePositions[0].Vehicle.ID)
}
