package rt

// Rules defined by func(*Validator, *pb.FeedMessage, *pb.FeedMessage)

// -------- WIP -----------
// This set of rules, and the interface, is in flux.

// type RuleMsg struct {
// 	Current  *pb.FeedMessage
// 	Previous *pb.FeedMessage
// 	Entity   interface{}
// 	fi       *Validator
// }

// func HeaderRequired(rm *RuleMsg) []error {
// 	var errs []error
// 	if rm.Current.Header == nil {
// 		errs = append(errs, ne("FeedMessage Header is required", "header"))
// 	}
// 	return errs
// }

// func HeaderTimestampRequired(rm *RuleMsg) []error {
// 	// Check the Header timestamp is present
// 	var errs []error
// 	header := rm.Current.GetHeader()
// 	if header == nil {
// 		return nil
// 	}
// 	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
// 		errs = append(errs, ef(E048, "header.timestamp"))
// 	}
// 	return errs
// }

// func HeaderTimestampIncreasing(rm *RuleMsg) []error {
// 	// Check previous Header timestamp
// 	var errs []error
// 	if rm.Current.GetHeader().GetTimestamp() < rm.Previous.GetHeader().GetTimestamp() {
// 		errs = append(errs, ef(E018, "header.timestamp"))
// 	}
// 	return errs
// }

// func HeaderInvalidRealtimeVersion(rm *RuleMsg) []error {
// 	var errs []error
// 	header := rm.Current.GetHeader()
// 	if header == nil {
// 		return nil
// 	}
// 	if v := header.GetGtfsRealtimeVersion(); v == "3.0" || v == "2.0" {
// 		// TODO: additional version specific checks
// 	} else if v == "1.0" {
// 		//ok
// 	} else {
// 		errs = append(errs, ef(E038, ""))
// 	}
// 	return errs
// }

// func HeaderTimestampBoundsCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	header := rm.Current.GetHeader()
// 	if header == nil {
// 		return nil
// 	}
// 	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
// 		errs = append(errs, ef(E048, "header.timestamp"))
// 	} else if !checkTimestamp(v) {
// 		errs = append(errs, ef(E001, "header.timestamp"))
// 	} else if !checkFuture(v) {
// 		errs = append(errs, ef(E050, "header.timestamp"))
// 	}
// 	return errs
// }

// func HeaderTimestampIncrementalityCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	header := rm.Current.GetHeader()
// 	if header == nil {
// 		return nil
// 	}
// 	if header.Incrementality == nil {
// 		errs = append(errs, ef(E049, "header.incrementality"))
// 	} else if header.GetIncrementality() == pb.FeedHeader_DIFFERENTIAL {
// 		errs = append(errs, ne("FeedHeader DIFFERENTIAL incrementality is not supported", "header.incrementality"))
// 	}
// 	return errs
// }

// func FeedEntityIDCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	for _, ent := range rm.Current.GetEntity() {
// 		if ent.Id == nil || ent.GetId() == "" {
// 			errs = append(errs, ne("FeedEntity id is required", "entity.id"))
// 		}
// 	}
// 	return errs
// }

// func FeedEntityIsDeletedCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	incr := rm.Current.Header.GetIncrementality()
// 	for _, ent := range rm.Current.GetEntity() {
// 		if ent.IsDeleted != nil && incr != pb.FeedHeader_DIFFERENTIAL {
// 			errs = append(errs, ef(E039, "entity.is_deleted"))
// 		}
// 	}
// 	return errs
// }

// func FeedEntityCompletenessCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	for _, ent := range rm.Current.GetEntity() {
// 		if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil {
// 			errs = append(errs, ne("FeedEntity must provide one of TripUpdate, VehiclePosition, or Alert", "entity"))
// 		}
// 	}
// 	return errs
// }

// func TripUpdateTripDescriptorRequiredCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	for _, ent := range rm.Current.GetEntity() {
// 		trip := ent.TripUpdate
// 		if trip == nil {
// 			continue
// 		}
// 		if trip.Trip == nil {
// 			errs = append(errs, ne("TripDescriptor is required", "trip_update.trip"))
// 		}
// 	}
// 	return errs
// }

// func TripDescriptorTripIDCheck(rm *RuleMsg) []error {
// 	var errs []error
// 	c := func(trip *pb.TripDescriptor) error {
// 		if _, ok := rm.fi.tripInfo[*trip.TripId]; !ok {
// 			errs = append(errs, ef(E003, "trip.trip_id"))
// 		}
// 		return nil
// 	}
// 	_ = c
// 	return errs
// }
