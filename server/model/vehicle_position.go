package model

import (
	"sort"
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
	sortVehiclePositionsByFreshness(ents)
	ents = dedupeVehiclePositions(ents)
	if limit != nil && len(ents) > *limit {
		ents = ents[0:*limit]
	}
	sortVehiclePositionsByID(ents)
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
	k := vehiclePositionKey{rtFeedOnestopID: ent.RtFeedOnestopID, entityID: ent.ID}
	if vd := ent.Vehicle; vd != nil && vd.ID != nil {
		k.vehicleID = *vd.ID
	}
	if td := ent.TripDescriptor; td != nil && td.TripID != nil {
		k.tripID = *td.TripID
	}
	return k
}

func dedupeVehiclePositions(ents []*VehiclePosition) []*VehiclePosition {
	var ret []*VehiclePosition
	seen := make(map[vehiclePositionKey]bool, len(ents))
	for _, ent := range ents {
		key := vehiclePositionKeyOf(ent)
		if seen[key] {
			continue
		}
		seen[key] = true
		ret = append(ret, ent)
	}
	return ret
}

// sortVehiclePositionsByFreshness orders most recently reported first, so that
// truncating to a limit keeps the freshest vehicles. A vehicle reporting no
// timestamp sorts last: nothing is known about its freshness, which is a weaker
// claim than any timestamp at all.
//
// Duplicates of one vehicle tie on every key above MatchedByEntityID, so the
// copy matched against a feed version's own GTFS ids sorts ahead of one claimed
// only because the realtime feed belongs to a single operator. That copy is the
// one whose schedule the message's ids actually resolve against.
func sortVehiclePositionsByFreshness(ents []*VehiclePosition) {
	sort.Slice(ents, func(i, j int) bool {
		a, b := ents[i], ents[j]
		if !timesEqual(a.Timestamp, b.Timestamp) {
			return moreRecent(a.Timestamp, b.Timestamp)
		}
		ka, kb := vehiclePositionKeyOf(a), vehiclePositionKeyOf(b)
		if ka != kb {
			return lessVehiclePositionKey(ka, kb)
		}
		if a.MatchedByEntityID != b.MatchedByEntityID {
			return a.MatchedByEntityID
		}
		return a.FeedVersionID < b.FeedVersionID
	})
}

// sortVehiclePositionsByID is the order results are returned in: stable between
// polls, and total, so that an unstable sort cannot reorder equal elements.
func sortVehiclePositionsByID(ents []*VehiclePosition) {
	sort.Slice(ents, func(i, j int) bool {
		ka, kb := vehiclePositionKeyOf(ents[i]), vehiclePositionKeyOf(ents[j])
		if ka != kb {
			return lessVehiclePositionKey(ka, kb)
		}
		return ents[i].FeedVersionID < ents[j].FeedVersionID
	})
}

func lessVehiclePositionKey(a vehiclePositionKey, b vehiclePositionKey) bool {
	if a.rtFeedOnestopID != b.rtFeedOnestopID {
		return a.rtFeedOnestopID < b.rtFeedOnestopID
	}
	if a.entityID != b.entityID {
		return a.entityID < b.entityID
	}
	if a.vehicleID != b.vehicleID {
		return a.vehicleID < b.vehicleID
	}
	return a.tripID < b.tripID
}

func timesEqual(a *time.Time, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

// moreRecent reports whether a is newer than b, treating an absent timestamp as
// older than every present one.
func moreRecent(a *time.Time, b *time.Time) bool {
	if a == nil || b == nil {
		return b == nil && a != nil
	}
	return a.After(*b)
}
