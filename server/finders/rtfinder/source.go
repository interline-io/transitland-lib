package rtfinder

import (
	"context"
	"sync"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/set"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"google.golang.org/protobuf/proto"
)

type Source struct {
	feed string
	msg  *pb.FeedMessage
	// tripUpdates holds every trip update naming a trip_id: a feed can report
	// several runs of one trip, such as last night's late run and this morning's.
	tripUpdates      map[string][]*pb.TripUpdate
	alerts           []alertEntity
	vehiclePositions []VehiclePositionEntity
	// unroutedTripIds are the trip_ids this message names without a route_id,
	// which only the static trip can place on a route.
	unroutedTripIds []string
	// tripRoutes holds unroutedTripIds resolved to route_ids, by feed version id,
	// for as long as this message is current.
	tripRoutesOnce sync.Once
	tripRoutes     *kvcache.Cache[int, map[string]string]
}

// VehiclePositionEntity pairs a vehicle position with the id of the FeedEntity
// that carried it. Every id on the message itself is optional, so the entity id
// is the only key a consumer can rely on to follow a vehicle between messages.
type VehiclePositionEntity struct {
	ID       string
	Position *pb.VehiclePosition
}

// alertEntity pairs an alert with the id of the FeedEntity that carried it,
// which identifies the alert within its feed.
type alertEntity struct {
	ID    string
	Alert *pb.Alert
}

func NewSource(feed string) (*Source, error) {
	f := Source{
		feed:        feed,
		tripUpdates: map[string][]*pb.TripUpdate{},
	}
	return &f, nil
}

func (f *Source) GetTimestamp() uint64 {
	return f.msg.GetHeader().GetTimestamp()
}

// GetTrips returns the trip updates naming a trip_id, one for each run reported.
func (f *Source) GetTrips(tid string) []*pb.TripUpdate {
	return f.tripUpdates[tid]
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
	a := map[string][]*pb.TripUpdate{}
	var alerts []alertEntity
	vehiclePositions := make([]VehiclePositionEntity, 0, len(rtmsg.Entity))
	unrouted := set.New[string]()
	addUnrouted := func(td *pb.TripDescriptor) {
		if td.GetRouteId() == "" && td.GetTripId() != "" {
			unrouted.Add(td.GetTripId())
		}
	}
	for _, ent := range rtmsg.Entity {
		if v := ent.TripUpdate; v != nil {
			if v.Timestamp == nil && hasDefaultTimestamp {
				v.Timestamp = &defaultTimestamp
			}
			tid := v.GetTrip().GetTripId()
			a[tid] = append(a[tid], v)
		}
		if v := ent.Alert; v != nil {
			alerts = append(alerts, alertEntity{ID: ent.GetId(), Alert: v})
			for _, s := range v.GetInformedEntity() {
				if s.GetRouteId() == "" {
					addUnrouted(s.GetTrip())
				}
			}
		}
		if v := ent.Vehicle; v != nil {
			// Not defaulted from the header, unlike a trip update: the header
			// is newer than every reading in it, so a vehicle reporting no time
			// would outrank every vehicle that reported a real one.
			vehiclePositions = append(vehiclePositions, VehiclePositionEntity{ID: ent.GetId(), Position: v})
			addUnrouted(v.GetTrip())
		}
	}
	log.For(ctx).Trace().Str("feed_id", f.feed).Int("trip_updates", len(a)).Int("alerts", len(alerts)).Int("vehicle_positions", len(vehiclePositions)).Msg("rtsource: processed data")
	f.tripUpdates = a
	f.alerts = alerts
	f.vehiclePositions = vehiclePositions
	f.unroutedTripIds = unrouted.ToSlice()
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
