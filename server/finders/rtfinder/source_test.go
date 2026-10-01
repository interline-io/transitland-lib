package rtfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestSourceProcessMessage_UnroutedTripIds(t *testing.T) {
	alert := func(ie ...*pb.EntitySelector) *pb.FeedEntity {
		return &pb.FeedEntity{Id: proto.String("alert"), Alert: &pb.Alert{InformedEntity: ie}}
	}
	vehicle := func(td *pb.TripDescriptor) *pb.FeedEntity {
		return &pb.FeedEntity{Id: proto.String("vehicle"), Vehicle: &pb.VehiclePosition{Trip: td}}
	}
	msg := &pb.FeedMessage{Entity: []*pb.FeedEntity{
		alert(&pb.EntitySelector{Trip: testTrip("A1", "")}),
		alert(&pb.EntitySelector{StopId: proto.String("FTVL"), Trip: testTrip("A2", "")}, &pb.EntitySelector{Trip: testTrip("A1", "")}),
		// Each of these names its route, so needs no lookup.
		alert(&pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("R1", "")}),
		alert(&pb.EntitySelector{Trip: testTrip("R2", "05")}),
		vehicle(testTrip("R3", "05")),
		// Nothing to look up.
		alert(&pb.EntitySelector{Trip: &pb.TripDescriptor{}}, nil),
		vehicle(nil),
		vehicle(testTrip("V1", "")),
	}}
	src, err := NewSource("f-rt")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.processMessage(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	assert.ElementsMatch(t, []string{"A1", "A2", "V1"}, src.unroutedTripIds)
}
