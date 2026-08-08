package model

import (
	"slices"
	"strings"
	"time"
)

// OrderVehiclePositions deduplicates, truncates and orders a set of vehicle
// positions for return. A nil limit returns every vehicle.
//
// A vehicle claimed by more than one agency, which happens when their static
// feeds share a realtime feed, collapses to a single result. Freshness decides
// which copy survives and what a limit drops, but the returned order is by feed
// and entity id: a client polling for changes needs its markers to stay put
// rather than reshuffle every time a timestamp advances.
func OrderVehiclePositions(ents []*VehiclePosition, limit *int) []*VehiclePosition {
	slices.SortFunc(ents, compareVehiclePositionFreshness)
	ents = dedupeVehiclePositions(ents)
	if limit != nil && len(ents) > *limit {
		ents = ents[0:*limit]
	}
	slices.SortFunc(ents, compareVehiclePositionID)
	return ents
}

// vehiclePositionKey identifies a vehicle across polls and across the agencies
// that may each claim it. The GTFS-RT entity id is only required to be unique
// within a message, so the descriptors are part of the key too: a producer that
// reuses an entity id for two vehicles should not have one of them silently
// dropped.
type vehiclePositionKey struct {
	rtFeedOnestopID string
	entityID        string
	vehicleID       string
	tripID          string
}

func vehiclePositionKeyOf(ent *VehiclePosition) vehiclePositionKey {
	return vehiclePositionKey{
		rtFeedOnestopID: ent.RtFeedOnestopID,
		entityID:        ent.ID,
		vehicleID:       vehiclePositionVehicleID(ent),
		tripID:          vehiclePositionTripID(ent),
	}
}

func dedupeVehiclePositions(ents []*VehiclePosition) []*VehiclePosition {
	ret := make([]*VehiclePosition, 0, len(ents))
	seen := make(map[vehiclePositionKey]struct{}, len(ents))
	for _, ent := range ents {
		key := vehiclePositionKeyOf(ent)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ret = append(ret, ent)
	}
	return ret
}

// compareVehiclePositionFreshness orders most recently reported first, so that
// truncating to a limit keeps the freshest vehicles. A vehicle reporting no
// timestamp sorts last: nothing is known about its freshness, which is a weaker
// claim than any timestamp at all. Duplicates of one vehicle tie until
// MatchedByEntityID, which is what decides between them.
func compareVehiclePositionFreshness(a *VehiclePosition, b *VehiclePosition) int {
	if c := vehiclePositionTime(b).Compare(vehiclePositionTime(a)); c != 0 {
		return c
	}
	if c := compareVehiclePositionID(a, b); c != 0 {
		return c
	}
	if a.MatchedByEntityID != b.MatchedByEntityID {
		if a.MatchedByEntityID {
			return -1
		}
		return 1
	}
	return a.FeedVersionID - b.FeedVersionID
}

// compareVehiclePositionID is the order results are returned in: stable between
// polls, and total once duplicates have been collapsed.
func compareVehiclePositionID(a *VehiclePosition, b *VehiclePosition) int {
	if c := strings.Compare(a.RtFeedOnestopID, b.RtFeedOnestopID); c != 0 {
		return c
	}
	if c := strings.Compare(a.ID, b.ID); c != 0 {
		return c
	}
	if c := strings.Compare(vehiclePositionVehicleID(a), vehiclePositionVehicleID(b)); c != 0 {
		return c
	}
	return strings.Compare(vehiclePositionTripID(a), vehiclePositionTripID(b))
}

// vehiclePositionTime treats an absent timestamp as the zero time, which orders
// it after every reported one.
func vehiclePositionTime(ent *VehiclePosition) time.Time {
	if ent.Timestamp == nil {
		return time.Time{}
	}
	return *ent.Timestamp
}

func vehiclePositionVehicleID(ent *VehiclePosition) string {
	if vd := ent.Vehicle; vd != nil && vd.ID != nil {
		return *vd.ID
	}
	return ""
}

func vehiclePositionTripID(ent *VehiclePosition) string {
	if td := ent.TripDescriptor; td != nil && td.TripID != nil {
		return *td.TripID
	}
	return ""
}
