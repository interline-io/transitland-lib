package rtfinder

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
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
		vp := makeVehiclePosition(ent, "f-rt", 7)
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
		vp := makeVehiclePosition(ent, "f-rt", 7)
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

func TestSourceProcessMessage_DefaultTimestamp(t *testing.T) {
	ctx := context.Background()
	newMsg := func(headerTimestamp uint64) *pb.FeedMessage {
		msg := &pb.FeedMessage{
			Header: &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
			Entity: []*pb.FeedEntity{{Id: proto.String("ent-1"), Vehicle: &pb.VehiclePosition{}}},
		}
		if headerTimestamp > 0 {
			msg.Header.Timestamp = proto.Uint64(headerTimestamp)
		}
		return msg
	}

	t.Run("header timestamp fills in", func(t *testing.T) {
		src, err := NewSource("f-rt")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, newMsg(1661990400)); err != nil {
			t.Fatal(err)
		}
		vps := src.GetVehiclePositions()
		assert.Len(t, vps, 1)
		assert.Equal(t, uint64(1661990400), vps[0].Position.GetTimestamp())
	})

	// The header timestamp is itself optional. Filling in a zero would serve
	// every vehicle in the feed as 1970-01-01.
	t.Run("no header timestamp leaves it unset", func(t *testing.T) {
		src, err := NewSource("f-rt")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, newMsg(0)); err != nil {
			t.Fatal(err)
		}
		vps := src.GetVehiclePositions()
		assert.Len(t, vps, 1)
		assert.Nil(t, vps[0].Position.Timestamp)
		assert.Nil(t, makeVehiclePosition(vps[0], "f-rt", 1).Timestamp)
	})

	t.Run("entity id is carried", func(t *testing.T) {
		src, err := NewSource("f-rt")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, newMsg(1661990400)); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "ent-1", src.GetVehiclePositions()[0].ID)
	})
}

func TestSortVehiclePositions(t *testing.T) {
	at := func(secs int) *time.Time {
		v := time.Unix(int64(secs), 0).In(time.UTC)
		return &v
	}
	keys := func(ents []*model.VehiclePosition) []string {
		var got []string
		for _, ent := range ents {
			got = append(got, ent.RtFeedOnestopID+":"+ent.ID)
		}
		return got
	}

	t.Run("most recent first", func(t *testing.T) {
		ents := []*model.VehiclePosition{
			{RtFeedOnestopID: "f-a", ID: "old", Timestamp: at(100)},
			{RtFeedOnestopID: "f-a", ID: "new", Timestamp: at(300)},
			{RtFeedOnestopID: "f-a", ID: "mid", Timestamp: at(200)},
		}
		sortVehiclePositions(ents)
		assert.Equal(t, []string{"f-a:new", "f-a:mid", "f-a:old"}, keys(ents))
	})

	t.Run("a limit keeps the freshest", func(t *testing.T) {
		ents := []*model.VehiclePosition{
			{RtFeedOnestopID: "f-a", ID: "old", Timestamp: at(100)},
			{RtFeedOnestopID: "f-a", ID: "new", Timestamp: at(300)},
			{RtFeedOnestopID: "f-a", ID: "mid", Timestamp: at(200)},
		}
		limit := 2
		assert.Equal(t, []string{"f-a:new", "f-a:mid"}, keys(limitVehiclePositions(ents, &limit)))
	})

	t.Run("missing timestamps sort last", func(t *testing.T) {
		ents := []*model.VehiclePosition{
			{RtFeedOnestopID: "f-b", ID: "none"},
			{RtFeedOnestopID: "f-a", ID: "none"},
			{RtFeedOnestopID: "f-a", ID: "dated", Timestamp: at(1)},
		}
		sortVehiclePositions(ents)
		assert.Equal(t, []string{"f-a:dated", "f-a:none", "f-b:none"}, keys(ents))
	})

	t.Run("equal timestamps break on feed then id", func(t *testing.T) {
		ents := []*model.VehiclePosition{
			{RtFeedOnestopID: "f-b", ID: "1", Timestamp: at(100)},
			{RtFeedOnestopID: "f-a", ID: "2", Timestamp: at(100)},
			{RtFeedOnestopID: "f-a", ID: "1", Timestamp: at(100)},
		}
		sortVehiclePositions(ents)
		assert.Equal(t, []string{"f-a:1", "f-a:2", "f-b:1"}, keys(ents))
	})
}
