package rt

import (
	"github.com/interline-io/transitland-lib/rt/pb"
)

// Rules defined by func(*Validator, *pb.FeedMessage, *pb.FeedMessage)

// -------- WIP -----------
// This set of rules, and the interface, is in flux.

type RuleMsg struct {
	Current  *pb.FeedMessage
	Previous *pb.FeedMessage
	Entity   interface{}
	fi       *Validator
}

func HeaderRequired(rm *RuleMsg) []error {
	var errs []error
	if rm.Current.Header == nil {
		errs = append(errs, ne("FeedMessage Header is required", 0))
	}
	return errs
}

func HeaderTimestampRequired(rm *RuleMsg) []error {
	// Check the Header timestamp is present
	var errs []error
	header := rm.Current.GetHeader()
	if header == nil {
		return nil
	}
	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
		errs = append(errs, ne("FeedHeader timestamp is required", 48))
	}
	return errs
}

func HeaderTimestampIncreasing(rm *RuleMsg) []error {
	// Check previous Header timestamp
	var errs []error
	if rm.Current.GetHeader().GetTimestamp() < rm.Previous.GetHeader().GetTimestamp() {
		errs = append(errs, ne("FeedMessage Header timestamp is earlier than previous update", 18))
	}
	return errs
}

func HeaderInvalidRealtimeVersion(rm *RuleMsg) []error {
	var errs []error
	header := rm.Current.GetHeader()
	if header == nil {
		return nil
	}
	if v := header.GetGtfsRealtimeVersion(); v == "3.0" || v == "2.0" {
		// TODO: additional version specific checks
	} else if v == "1.0" {
		//ok
	} else {
		errs = append(errs, E038)
	}
	return errs
}

func HeaderTimestampBoundsCheck(rm *RuleMsg) []error {
	var errs []error
	header := rm.Current.GetHeader()
	if header == nil {
		return nil
	}
	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
		errs = append(errs, ne("FeedHeader timestamp is required", 48))
	} else if !checkTimestamp(v) {
		errs = append(errs, ne("FeedHeader timestamp is out of bounds", 1))
	} else if !checkFuture(v) {
		errs = append(errs, ne("FeedHeader timestamp is in the future", 50))
	}
	return errs
}

func HeaderTimestampIncrementalityCheck(rm *RuleMsg) []error {
	var errs []error
	header := rm.Current.GetHeader()
	if header == nil {
		return nil
	}
	if header.Incrementality == nil {
		errs = append(errs, ne("FeedHeader incrementality is required", 49))
	} else if header.GetIncrementality() == pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, ne("FeedHeader DIFFERENTIAL incrementality is not supported", 0))
	}
	return errs
}

func FeedEntityIDCheck(rm *RuleMsg) []error {
	var errs []error
	for _, ent := range rm.Current.GetEntity() {
		if ent.Id == nil || ent.GetId() == "" {
			errs = append(errs, ne("FeedEntity id is required", 0))
		}
	}
	return errs
}

func FeedEntityIsDeletedCheck(rm *RuleMsg) []error {
	var errs []error
	incr := rm.Current.Header.GetIncrementality()
	for _, ent := range rm.Current.GetEntity() {
		if ent.IsDeleted != nil && incr != pb.FeedHeader_DIFFERENTIAL {
			errs = append(errs, ne("FeedEntity IsDeleted should only be provided for DIFFERENTIAL incrementality", 39))
		}
	}
	return errs
}

func FeedEntityCompletenessCheck(rm *RuleMsg) []error {
	var errs []error
	for _, ent := range rm.Current.GetEntity() {
		if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil {
			errs = append(errs, ne("FeedEntity must provide one of TripUpdate, VehiclePosition, or Alert", 0))
		}
	}
	return errs
}

func TripUpdateTripDescriptorRequiredCheck(rm *RuleMsg) []error {
	var errs []error
	for _, ent := range rm.Current.GetEntity() {
		trip := ent.TripUpdate
		if trip == nil {
			continue
		}
		if trip.Trip == nil {
			errs = append(errs, ne("TripDescriptor is required", 0))
		}
	}
	return errs
}

func TripDescriptorTripIDCheck(rm *RuleMsg) []error {
	var errs []error
	c := func(trip *pb.TripDescriptor) error {
		if _, ok := rm.fi.tripInfo[*trip.TripId]; !ok {
			errs = append(errs, ne("TripDescriptor references unknown trip_id", 3))
		}
		return nil
	}
	_ = c
	return errs
}
