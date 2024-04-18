package rt

import (
	"fmt"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/interline-io/transitland-lib/rt/pb"
)

type RTTripStat struct {
	AgencyID                string
	RouteID                 string
	TripScheduledIDs        []string
	TripRtIDs               []string
	TripScheduledCount      int
	TripScheduledMatched    int
	TripScheduledNotMatched int
	TripRtCount             int
	TripRtMatched           int
	TripRtNotMatched        int
	// Not found / added
	TripRtNotFoundIDs   []string
	TripRtAddedIDs      []string
	TripRtNotFoundCount int
	TripRtAddedCount    int
}

type statAggKey struct {
	AgencyID string
	RouteID  string
}

func (fi *Validator) VehiclePositionStats(now time.Time, msg *pb.FeedMessage) ([]RTTripStat, error) {
	scheduledTrips := fi.sched.ActiveTrips(now)
	var rtTrips []rtTripKey
	for _, ent := range msg.Entity {
		rtEnt := ent.Vehicle
		if rtEnt == nil {
			continue
		}
		rtTrips = append(rtTrips, fi.getRtTripKey(rtEnt.GetTrip()))
	}
	if len(rtTrips) == 0 {
		return nil, nil
	}
	stats, err := fi.compareTripSets(scheduledTrips, rtTrips)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (fi *Validator) TripUpdateStats(now time.Time, msg *pb.FeedMessage) ([]RTTripStat, error) {
	scheduledTrips := fi.sched.ActiveTrips(now)
	var rtTrips []rtTripKey
	for _, ent := range msg.Entity {
		rtEnt := ent.TripUpdate
		if rtEnt == nil {
			continue
		}
		rtTrips = append(rtTrips, fi.getRtTripKey(rtEnt.GetTrip()))
	}
	if len(rtTrips) == 0 {
		return nil, nil
	}
	stats, err := fi.compareTripSets(scheduledTrips, rtTrips)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (fi *Validator) compareTripSets(scheduledTrips []string, rtTrips []rtTripKey) ([]RTTripStat, error) {
	statAgg := map[statAggKey]RTTripStat{}
	statAgg[statAggKey{}] = RTTripStat{}

	// Prepopulate with all known routes
	for routeId, v := range fi.routeInfo {
		k := statAggKey{
			RouteID:  routeId,
			AgencyID: v.AgencyID,
		}
		statAgg[k] = RTTripStat{
			RouteID:  k.RouteID,
			AgencyID: k.AgencyID,
		}
	}

	// Process scheduled trips
	for _, tripId := range scheduledTrips {
		k := statAggKey{}
		trip, ok := fi.tripInfo[tripId]
		if ok {
			k.RouteID = trip.RouteID
			k.AgencyID = fi.routeInfo[trip.RouteID].AgencyID
		} else {
			continue
		}
		stat := statAgg[k]
		stat.AgencyID = k.AgencyID
		stat.RouteID = k.RouteID
		stat.TripScheduledIDs = append(stat.TripScheduledIDs, tripId)
		statAgg[k] = stat
	}

	// Process RT entities
	for _, rtKey := range rtTrips {
		k := statAggKey{
			RouteID:  rtKey.RouteID,
			AgencyID: rtKey.AgencyID,
		}
		stat := statAgg[k]
		stat.AgencyID = k.AgencyID
		stat.RouteID = k.RouteID
		if rtKey.Found {
			stat.TripRtIDs = append(stat.TripRtIDs, rtKey.TripID)
		} else if rtKey.Added {
			stat.TripRtAddedIDs = append(stat.TripRtAddedIDs, rtKey.TripID)
		} else {
			stat.TripRtNotFoundIDs = append(stat.TripRtNotFoundIDs, rtKey.TripID)
		}
		statAgg[k] = stat
	}

	var statAggSortedKeys []statAggKey
	for k := range statAgg {
		statAggSortedKeys = append(statAggSortedKeys, k)
	}
	sort.Slice(statAggSortedKeys, func(i, j int) bool {
		a, b := statAggSortedKeys[i], statAggSortedKeys[j]
		return fmt.Sprintf("%s:%s", a.AgencyID, a.RouteID) < fmt.Sprintf("%s:%s", b.AgencyID, b.RouteID)
	})
	var ret []RTTripStat
	for _, k := range statAggSortedKeys {
		v := statAgg[k]
		scheduledSet := mapset.NewSet[string](v.TripScheduledIDs...)
		updateSet := mapset.NewSet[string](v.TripRtIDs...)
		updateNotFoundSet := mapset.NewSet[string](v.TripRtNotFoundIDs...)
		updateAddedSet := mapset.NewSet[string](v.TripRtAddedIDs...)
		tripScheduledMatched := scheduledSet.Intersect(updateSet)
		tripScheduledNotMatched := scheduledSet.Difference(updateSet)
		tripRtMatched := updateSet.Intersect(scheduledSet)
		tripRtNotMatched := updateSet.Difference(scheduledSet)
		v.TripScheduledIDs = scheduledSet.ToSlice()
		v.TripScheduledCount = scheduledSet.Cardinality()
		v.TripScheduledMatched = tripScheduledMatched.Cardinality()
		v.TripScheduledNotMatched = tripScheduledNotMatched.Cardinality()
		v.TripRtIDs = updateSet.ToSlice()
		v.TripRtCount = updateSet.Cardinality()
		v.TripRtMatched = tripRtMatched.Cardinality()
		v.TripRtNotMatched = tripRtNotMatched.Cardinality()
		v.TripRtNotFoundIDs = updateNotFoundSet.ToSlice()
		v.TripRtNotFoundCount = updateNotFoundSet.Cardinality()
		v.TripRtAddedIDs = updateAddedSet.ToSlice()
		v.TripRtAddedCount = updateAddedSet.Cardinality()
		statAgg[k] = v
		// fmt.Printf("\tagency '%s' route '%s'\n", k.AgencyID, k.RouteID)
		// fmt.Printf("\t\tsched %d %v\n", len(v.TripScheduledIDs), v.TripScheduledIDs)
		// fmt.Printf("\t\t\tsched matched: %d %v\n", tripScheduledMatched.Cardinality(), tripScheduledMatched.ToSlice())
		// fmt.Printf("\t\t\tsched not matched: %d %v\n", tripScheduledNotMatched.Cardinality(), tripScheduledNotMatched.ToSlice())
		// fmt.Printf("\t\tt %d %v\n", len(v.TripRtIDs), v.TripRtIDs)
		// fmt.Printf("\t\t\trt matched: %d %v\n", tripRtMatched.Cardinality(), tripRtMatched.ToSlice())
		// fmt.Printf("\t\t\trt not matched: %d %v\n", tripRtNotMatched.Cardinality(), tripRtNotMatched.ToSlice())
		// fmt.Printf("\tout: %#v\n", v)
		ret = append(ret, v)
	}
	return ret, nil
}
