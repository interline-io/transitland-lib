package rtfinder

import (
	"context"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/proto"
)

type Source struct {
	feed         string
	msg          *pb.FeedMessage
	entityByTrip map[string]*pb.TripUpdate
	alerts       []*pb.Alert
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

func (f *Source) processMessage(ctx context.Context, rtmsg *pb.FeedMessage) error {
	f.msg = rtmsg
	defaultTimestamp := rtmsg.GetHeader().GetTimestamp()
	a := map[string]*pb.TripUpdate{}
	var alerts []*pb.Alert
	for _, ent := range rtmsg.Entity {
		if v := ent.TripUpdate; v != nil {
			// Set default timestamp
			if v.Timestamp == nil {
				v.Timestamp = &defaultTimestamp
			}
			tid := v.GetTrip().GetTripId()
			a[tid] = v
		}
		if v := ent.Alert; v != nil {
			alerts = append(alerts, v)
		}
		// todo: vehicle positions...
	}
	log.For(ctx).Trace().Str("feed_id", f.feed).Int("trip_updates", len(a)).Int("alerts", len(alerts)).Msg("rtsource: processed data")
	f.entityByTrip = a
	f.alerts = alerts
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
