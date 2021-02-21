package rt

import (
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl"
)

func ne(msg string, code int) *RealtimeError {
	return &RealtimeError{
		Code: code,
		msg:  msg,
	}
}

func checkTimestamp(ts uint64) bool {
	// 1/1/1990 -> year 2038
	if ts < 631152000 || ts > (1<<31-1) {
		return false
	}
	return true
}

func checkFuture(ts uint64) bool {
	// Is timestamp more than 1 minute in the future
	if ts > uint64(time.Now().Unix()+60) {
		return false
	}
	return true
}

// ValidateFeedMessage .
func ValidateFeedMessage(fi *FeedInfo, current *pb.FeedMessage, previous *pb.FeedMessage) (errs []error) {
	if current.Header == nil {
		errs = append(errs, ne("FeedMessage Header is required", 0))
	} else {
		// Check previous Header timestamp
		if current.GetHeader().GetTimestamp() < previous.GetHeader().GetTimestamp() {
			errs = append(errs, ne("FeedMessage Header timestamp is earlier than previous update", 18))
		}
		errs = append(errs, ValidateHeader(fi, current.Header, current)...)
	}
	// TODO: Validate TripDescriptors are unique
	for _, ent := range current.GetEntity() {
		errs = append(errs, ValidateFeedEntity(fi, ent, current)...)
	}
	return errs
}

// ValidateHeader .
func ValidateHeader(fi *FeedInfo, header *pb.FeedHeader, current *pb.FeedMessage) (errs []error) {
	if v := header.GetGtfsRealtimeVersion(); v == "3.0" || v == "2.0" {
		// TODO: additional version specific checks
	} else if v == "1.0" {
		//ok
	} else {
		errs = append(errs, E038)
	}
	//
	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
		errs = append(errs, ne("FeedHeader timestamp is required", 48))
	} else if !checkTimestamp(v) {
		errs = append(errs, ne("FeedHeader timestamp is out of bounds", 1))
	} else if !checkFuture(v) {
		errs = append(errs, ne("FeedHeader timestamp is in the future", 50))
	}
	//
	if header.Incrementality == nil {
		errs = append(errs, ne("FeedHeader incrementality is required", 49))
	} else if header.GetIncrementality() == pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, ne("FeedHeader DIFFERENTIAL incrementality is not supported", 0))
	}
	return errs
}

// ValidateFeedEntity .
func ValidateFeedEntity(fi *FeedInfo, ent *pb.FeedEntity, current *pb.FeedMessage) (errs []error) {
	incr := current.GetHeader().GetIncrementality()
	if ent.Id == nil || ent.GetId() == "" {
		errs = append(errs, ne("FeedEntity id is required", 0))
	}
	if ent.IsDeleted != nil && incr != pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, ne("FeedEntity IsDeleted should only be provided for DIFFERENTIAL incrementality", 39))
	}
	if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil {
		errs = append(errs, ne("FeedEntity must provide one of TripUpdate, VehiclePosition, or Alert", 0))
	}
	if ent.TripUpdate != nil {
		errs = append(errs, ValidateTripUpdate(fi, ent.GetTripUpdate(), current)...)
	}
	if ent.Vehicle != nil {
		// TODO: ValidateVehiclePosition
	}
	if ent.Alert != nil {
		// TODO: ValidateAlert
		// TODO: Check that route_id is not set in a TripDescriptor
	}
	return errs
}

// ValidateTripUpdate .
func ValidateTripUpdate(fi *FeedInfo, trip *pb.TripUpdate, current *pb.FeedMessage) (errs []error) {
	// Validate TripDescriptor
	if trip.Trip == nil {
		errs = append(errs, ne("TripDescriptor is required", 0))
	} else {
		errs = append(errs, ValidateTripDescriptor(fi, trip.Trip, current)...)
	}
	if trip.Delay != nil {
		// experimental field
	}
	// Validate StopTimeUpdates
	srel := trip.GetTrip().GetScheduleRelationship()
	sts := trip.GetStopTimeUpdate()
	if len(sts) == 0 && srel != pb.TripDescriptor_CANCELED {
		errs = append(errs, ne("StopTimeUpdates are required unless the trip is canceled", 41))
	}
	seq := uint32(0)
	visitedseq := map[uint32]int{}
	visitedstop := map[string]int{}
	prevstop := ""
	prevtime := int64(0)
	for _, st := range sts {
		ss := st.StopSequence
		stopid := st.StopId
		if stopid != nil {
			s2 := *stopid
			visitedstop[s2]++
			if ss == nil && visitedstop[s2] > 1 {
				errs = append(errs, ne("StopTimeUpdate must specify stop_sequence when a stop is visited more than once", 9))
			}
			if s2 == prevstop {
				errs = append(errs, ne("StopTimeUpdates visits the same stop twice in a row", 37))
			}
			prevstop = s2
		}
		if ss != nil {
			s2 := *ss
			visitedseq[s2]++
			if visitedseq[s2] > 1 {
				errs = append(errs, ne("StopTimeUpdates repeats the same stop_sequence", 36))
			}
			if s2 < seq {
				errs = append(errs, ne("StopTimeUpdates not sorted by stop_sequence", 2))
			}
			seq = s2
		}
		if st.GetArrival().Time != nil {
			a := st.GetArrival().GetTime()
			if a < prevtime {
				errs = append(errs, ne("StopTimeUpdates are not increasing in time", 22))
			}
			prevtime = a
		}
		if st.GetDeparture().Time != nil {
			a := st.GetDeparture().GetTime()
			if a < prevtime {
				errs = append(errs, ne("StopTimeUpdates are not increasing in time", 22))
			}
			prevtime = a
		}
		// Check individual values
		errs = append(errs, ValidateStopTimeUpdate(fi, st, current)...)
	}
	return errs
}

// ValidateStopTimeUpdate .
func ValidateStopTimeUpdate(fi *FeedInfo, st *pb.TripUpdate_StopTimeUpdate, current *pb.FeedMessage) (errs []error) {
	if st.StopId == nil && st.StopSequence == nil {
		errs = append(errs, ne("StopTimeUpdate must specify stop_sequence or stop_id", 40))
	}
	if st.StopId != nil {
		v, ok := fi.stopLocationTypes[*st.StopId]
		if !ok {
			errs = append(errs, ne("StopTimeUpdate references unknown stop_id", 11))
		}
		if v != 0 {
			errs = append(errs, ne("StopTimeUpdate cannot reference stop where location_type is not 0", 15))
		}
	}
	// Arrival, Departure
	switch st.GetScheduleRelationship() {
	case pb.TripUpdate_StopTimeUpdate_SCHEDULED:
		if st.Arrival == nil && st.Departure == nil {
			errs = append(errs, ne("StopTimeUpdate must specify either arrival or departure when schedule_relationship is scheduled", 43))
		}
	case pb.TripUpdate_StopTimeUpdate_NO_DATA:
		if st.Arrival != nil || st.Departure != nil {
			errs = append(errs, ne("StopTimeUpdate cannot specify arrival or departure when schedule_relationship is NO_DATA", 42))
		}
	case pb.TripUpdate_StopTimeUpdate_SKIPPED:
		// ok
	}
	if st.GetArrival().GetTime() > st.GetDeparture().GetTime() {
		errs = append(errs, ne("StopTimeUpdate arrival time is later than departure time", 25))
	}
	// ValidateStopTimeEvent .
	// TODO
	return errs
}

// ValidateTripDescriptor .
func ValidateTripDescriptor(fi *FeedInfo, td *pb.TripDescriptor, current *pb.FeedMessage) (errs []error) {
	if td.TripId != nil {
		tripid := *td.TripId
		v, ok := fi.tripDirections[tripid]
		if !ok {
			errs = append(errs, ne("TripDescriptor references unknown trip_id", 3))
		}
		if td.DirectionId != nil && td.GetDirectionId() != uint32(v) {
			errs = append(errs, ne("TripDescriptor trip does not match GTFS direction", 24))
		}
		freq := false
		if freq {
			if td.StartTime == nil || td.StartDate == nil {
				errs = append(errs, ne("TripDescriptor must provide start_date and start_time for frequency based trips", 0))
			}
			// TODO: Additional frequency based trip checks
		}
	} else {
		if td.RouteId == nil || td.DirectionId == nil || td.StartDate == nil || td.StartTime == nil {
			errs = append(errs, ne("TripDescriptor must provided a trip_id or all of route_id, direction_id, start_date, and start_time", 0))
		}
		if td.GetScheduleRelationship() != pb.TripDescriptor_SCHEDULED {
			errs = append(errs, ne("TripDescriptor must be SCHEDULED if no trip_id is provided", 0))
		}
	}
	if td.RouteId != nil {
		if !fi.Contains("routes.txt", *td.RouteId) {
			errs = append(errs, ne("TripDescriptor references unknown route_id", 4))
		}
	}
	if td.StartTime != nil {
		if _, err := tl.NewWideTime(*td.StartTime); err != nil {
			errs = append(errs, ne("TripDescriptor could not parse StartTime", 20))
		}
	}
	if td.StartDate != nil {
		if _, err := time.Parse("20060102", *td.StartDate); err != nil {
			errs = append(errs, ne("TripDescriptor could not parse StartDate", 21))
		}
	}
	return errs
}
