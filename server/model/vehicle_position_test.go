package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func vpAt(secs int) *time.Time {
	v := time.Unix(int64(secs), 0).In(time.UTC)
	return &v
}

func vpKeys(ents []*VehiclePosition) []string {
	var got []string
	for _, ent := range ents {
		got = append(got, ent.RtFeedOnestopID+":"+ent.ID)
	}
	return got
}

func TestOrderVehiclePositions(t *testing.T) {
	// Results come back in feed and entity id order regardless of how fresh
	// each vehicle is, so a polling client's markers do not reshuffle.
	t.Run("returned in id order", func(t *testing.T) {
		ents := []*VehiclePosition{
			{RtFeedOnestopID: "f-a", ID: "old", Timestamp: vpAt(100)},
			{RtFeedOnestopID: "f-a", ID: "new", Timestamp: vpAt(300)},
			{RtFeedOnestopID: "f-a", ID: "mid", Timestamp: vpAt(200)},
		}
		assert.Equal(t, []string{"f-a:mid", "f-a:new", "f-a:old"}, vpKeys(OrderVehiclePositions(ents, nil)))
	})

	t.Run("a limit keeps the freshest", func(t *testing.T) {
		ents := []*VehiclePosition{
			{RtFeedOnestopID: "f-a", ID: "old", Timestamp: vpAt(100)},
			{RtFeedOnestopID: "f-a", ID: "new", Timestamp: vpAt(300)},
			{RtFeedOnestopID: "f-a", ID: "mid", Timestamp: vpAt(200)},
		}
		limit := 2
		assert.Equal(t, []string{"f-a:mid", "f-a:new"}, vpKeys(OrderVehiclePositions(ents, &limit)))
	})

	// A vehicle that reported no time of its own says nothing about its
	// freshness, which is a weaker claim than any timestamp at all.
	t.Run("a limit drops missing timestamps first", func(t *testing.T) {
		ents := []*VehiclePosition{
			{RtFeedOnestopID: "f-a", ID: "none"},
			{RtFeedOnestopID: "f-a", ID: "ancient", Timestamp: vpAt(1)},
		}
		limit := 1
		assert.Equal(t, []string{"f-a:ancient"}, vpKeys(OrderVehiclePositions(ents, &limit)))
	})

	t.Run("limit of zero returns nothing", func(t *testing.T) {
		ents := []*VehiclePosition{{RtFeedOnestopID: "f-a", ID: "1", Timestamp: vpAt(1)}}
		limit := 0
		assert.Empty(t, OrderVehiclePositions(ents, &limit))
	})
}

func TestOrderVehiclePositions_Dedupe(t *testing.T) {
	vp := func(fvid int, matched bool) *VehiclePosition {
		return &VehiclePosition{
			RtFeedOnestopID:   "f-rg~rt",
			ID:                "ent-1",
			FeedVersionID:     fvid,
			MatchedByEntityID: matched,
			Timestamp:         vpAt(100),
		}
	}

	// Two agencies whose static feeds share a realtime feed both claim the same
	// vehicle; it is one bus and belongs on the map once.
	t.Run("one vehicle claimed twice collapses", func(t *testing.T) {
		got := OrderVehiclePositions([]*VehiclePosition{vp(1, false), vp(2, false)}, nil)
		assert.Len(t, got, 1)
	})

	// The surviving copy decides which feed version trip, route and stop
	// resolve against, so the one matched against real GTFS ids has to win
	// whichever order the agencies were scanned in.
	t.Run("the copy matched by entity id wins", func(t *testing.T) {
		for _, ents := range [][]*VehiclePosition{
			{vp(1, false), vp(2, true)},
			{vp(2, true), vp(1, false)},
		} {
			got := OrderVehiclePositions(ents, nil)
			assert.Len(t, got, 1)
			assert.Equal(t, 2, got[0].FeedVersionID)
		}
	})

	// GTFS-RT only says an entity id should be unique within a message. When a
	// producer reuses one, the vehicles are still distinct and neither may be
	// silently dropped.
	t.Run("distinct vehicles sharing an entity id are kept", func(t *testing.T) {
		a, b := vp(1, true), vp(1, true)
		a.Vehicle = &RTVehicleDescriptor{ID: ptrTo("bus-1")}
		b.Vehicle = &RTVehicleDescriptor{ID: ptrTo("bus-2")}
		assert.Len(t, OrderVehiclePositions([]*VehiclePosition{a, b}, nil), 2)
	})
}

func ptrTo[T any](v T) *T {
	return &v
}
