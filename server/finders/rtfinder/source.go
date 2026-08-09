package rtfinder

import (
	"context"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/proto"
)

type Source struct {
	feed             string
	msg              *pb.FeedMessage
	entityByTrip     map[string]*pb.TripUpdate
	alerts           []*pb.Alert
	vehiclePositions []VehiclePositionEntity
}

// VehiclePositionEntity pairs a vehicle position with the id of the FeedEntity
// that carried it. Every id on the message itself is optional, so the entity id
// is the only key a consumer can rely on to follow a vehicle between messages.
type VehiclePositionEntity struct {
	ID       string
	Position *pb.VehiclePosition
}

func NewSource(feed string) (*Source, error) {
	f := Source{
		feed:         feed,
		entityByTrip: map[string]*pb.TripUpdate{},
	}
	return &f, nil
}

func (f *Source) GetTimestamp() uint64 {
	return f.msg.GetHeader().GetTimestamp()
}

func (f *Source) GetTrip(tid string) (*pb.TripUpdate, bool) {
	a, ok := f.entityByTrip[tid]
	if ok {
		return a, true
	}
	return nil, false
}

func (f *Source) GetVehiclePositions() []VehiclePositionEntity {
	return f.vehiclePositions
}

func (f *Source) processMessage(ctx context.Context, rtmsg *pb.FeedMessage) error {
	f.msg = rtmsg
	// The header timestamp is itself optional; a zero is no better a time than
	// no time at all, and would be served as 1970-01-01.
	defaultTimestamp := rtmsg.GetHeader().GetTimestamp()
	hasDefaultTimestamp := defaultTimestamp > 0
	a := map[string]*pb.TripUpdate{}
	var alerts []*pb.Alert
	vehiclePositions := make([]VehiclePositionEntity, 0, len(rtmsg.Entity))
	for _, ent := range rtmsg.Entity {
		if v := ent.TripUpdate; v != nil {
			if v.Timestamp == nil && hasDefaultTimestamp {
				v.Timestamp = &defaultTimestamp
			}
			tid := v.GetTrip().GetTripId()
			// TODO: GTFS-RT TripModifications — when tid is empty, fall back to
			// v.GetTrip().GetModifiedTrip().GetAffectedTripId() so modified trips
			// are indexed for downstream lookups.
			a[tid] = v
		}
		if v := ent.Alert; v != nil {
			alerts = append(alerts, v)
		}
		if v := ent.Vehicle; v != nil {
			// Not defaulted from the header, unlike a trip update: the header
			// is newer than every reading in it, so a vehicle reporting no time
			// would outrank every vehicle that reported a real one.
			vehiclePositions = append(vehiclePositions, VehiclePositionEntity{ID: ent.GetId(), Position: v})
		}
	}
	log.For(ctx).Trace().Str("feed_id", f.feed).Int("trip_updates", len(a)).Int("alerts", len(alerts)).Int("vehicle_positions", len(vehiclePositions)).Msg("rtsource: processed data")
	f.entityByTrip = a
	f.alerts = alerts
	f.vehiclePositions = vehiclePositions
	return nil
}

func (f *Source) process(ctx context.Context, rtdata []byte) error {
	if len(rtdata) == 0 {
		log.For(ctx).Trace().Str("feed_id", f.feed).Msg("rtsource: no data to process")
		return nil
	}
	rtmsg := pb.FeedMessage{}
	if err := proto.Unmarshal(rtdata, &rtmsg); err != nil {
		return err
	}
	return f.processMessage(ctx, &rtmsg)
}
