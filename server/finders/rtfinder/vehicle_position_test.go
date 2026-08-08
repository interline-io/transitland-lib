package rtfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMakeVehiclePosition(t *testing.T) {
	t.Run("full message", func(t *testing.T) {
		ent := VehiclePositionEntity{
			ID: "ent-1",
			Position: &pb.VehiclePosition{
				Trip:     &pb.TripDescriptor{TripId: proto.String("t1"), RouteId: proto.String("r1")},
				Vehicle:  &pb.VehicleDescriptor{Id: proto.String("v1"), Label: proto.String("Train")},
				Position: &pb.Position{Latitude: proto.Float32(37.8), Longitude: proto.Float32(-122.3), Bearing: proto.Float32(90), Speed: proto.Float32(10)},
				StopId:   proto.String("s1"),
				// GTFS-RT timestamps are seconds since the epoch, in UTC.
				Timestamp:           proto.Uint64(1661990340),
				CurrentStopSequence: proto.Uint32(3),
				OccupancyPercentage: proto.Uint32(50),
			},
		}
		vp := makeVehiclePosition(ent, "f-rt", 7, true)
		assert.Equal(t, "ent-1", vp.ID)
		assert.Equal(t, "f-rt", vp.RtFeedOnestopID)
		assert.Equal(t, 7, vp.FeedVersionID)
		assert.Equal(t, "t1", *vp.TripDescriptor.TripID)
		assert.Equal(t, "v1", *vp.Vehicle.ID)
		assert.InDelta(t, 90.0, *vp.Bearing, 0.001)
		assert.InDelta(t, 10.0, *vp.Speed, 0.001)
		assert.Equal(t, "s1", *vp.StopID)
		assert.Equal(t, 3, *vp.CurrentStopSequence)
		assert.Equal(t, 50, *vp.OccupancyPercentage)
		assert.Equal(t, "2022-08-31T23:59:00Z", vp.Timestamp.Format("2006-01-02T15:04:05Z"))
	})

	// Everything but the position is optional, and an absent field must not
	// arrive as a zero value a client cannot tell from a real reading.
	t.Run("bare message", func(t *testing.T) {
		ent := VehiclePositionEntity{
			ID:       "ent-2",
			Position: &pb.VehiclePosition{Position: &pb.Position{Latitude: proto.Float32(37.8), Longitude: proto.Float32(-122.3)}},
		}
		vp := makeVehiclePosition(ent, "f-rt", 7, true)
		assert.Nil(t, vp.Bearing)
		assert.Nil(t, vp.Speed)
		assert.Nil(t, vp.Vehicle)
		assert.Nil(t, vp.TripDescriptor)
		assert.Nil(t, vp.StopID)
		assert.Nil(t, vp.CurrentStopSequence)
		assert.Nil(t, vp.CurrentStatus)
		assert.Nil(t, vp.CongestionLevel)
		assert.Nil(t, vp.OccupancyStatus)
		assert.Nil(t, vp.OccupancyPercentage)
		assert.Nil(t, vp.Timestamp)
	})
}

// Times and dates in a trip descriptor are free-form strings, so a value that
// does not parse must not take the whole descriptor with it.
func TestMakeTripDescriptor_UnparseableValues(t *testing.T) {
	td := makeTripDescriptor(&pb.TripDescriptor{
		TripId:    proto.String("t1"),
		StartTime: proto.String("not a time"),
		StartDate: proto.String("20221301"),
	})
	assert.Equal(t, "t1", *td.TripID)
	assert.Nil(t, td.StartTime)
	assert.Nil(t, td.StartDate)
}

func TestSourceProcessMessage_VehicleTimestamp(t *testing.T) {
	ctx := context.Background()
	const headerTime = uint64(1661990400)
	newMsg := func(vehicleTimestamp uint64) *pb.FeedMessage {
		vp := &pb.VehiclePosition{Trip: &pb.TripDescriptor{TripId: proto.String("t1")}}
		if vehicleTimestamp > 0 {
			vp.Timestamp = proto.Uint64(vehicleTimestamp)
		}
		return &pb.FeedMessage{
			Header: &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0"), Timestamp: proto.Uint64(headerTime)},
			Entity: []*pb.FeedEntity{{
				Id:         proto.String("ent-1"),
				Vehicle:    vp,
				TripUpdate: &pb.TripUpdate{Trip: &pb.TripDescriptor{TripId: proto.String("t1")}},
			}},
		}
	}
	process := func(t *testing.T, msg *pb.FeedMessage) *Source {
		src, err := NewSource("f-rt")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, msg); err != nil {
			t.Fatal(err)
		}
		return src
	}

	t.Run("a reported timestamp is kept", func(t *testing.T) {
		vps := process(t, newMsg(1661990340)).GetVehiclePositions()
		assert.Len(t, vps, 1)
		assert.Equal(t, uint64(1661990340), vps[0].Position.GetTimestamp())
	})

	// The header carries the moment the dataset was generated, which is newer
	// than every reading in it. Defaulting a vehicle to it would rank a vehicle
	// that reported nothing ahead of every vehicle that reported a real time.
	t.Run("the header timestamp does not fill in", func(t *testing.T) {
		src := process(t, newMsg(0))
		vps := src.GetVehiclePositions()
		assert.Len(t, vps, 1)
		assert.Nil(t, vps[0].Position.Timestamp)
		assert.Nil(t, makeVehiclePosition(vps[0], "f-rt", 1, true).Timestamp)

		// Trip updates keep the header default; their timestamp is not used to
		// rank anything.
		tu, ok := src.GetTrip("t1")
		assert.True(t, ok)
		assert.Equal(t, headerTime, tu.GetTimestamp())
	})

	t.Run("entity id is carried", func(t *testing.T) {
		assert.Equal(t, "ent-1", process(t, newMsg(0)).GetVehiclePositions()[0].ID)
	})
}
