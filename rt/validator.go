package rt

import (
	"time"

	"github.com/interline-io/transitland-lib/ext/sched"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/geomcache"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
)

type tripInfo struct {
	DirectionID   int
	UsesFrequency bool
	ShapeID       string
	RouteID       string
}

type stopInfo struct {
	LocationType int
}

type routeInfo struct {
	AgencyID  string
	RouteType int
}

type rtTripKey struct {
	AgencyID string
	RouteID  string
	TripID   string
	Found    bool
	Added    bool
}

// Validator validates RT messages based on data from a static feed.
// It can be initialized through NewValidatorFromReader or through the Copier Validator interface.
type Validator struct {
	Timezone            string
	MaxDistanceFromTrip float64
	tripInfo            map[string]tripInfo
	routeInfo           map[string]routeInfo
	stopInfo            map[string]stopInfo
	staticShapeIds      map[string]bool
	// currentInlineStops holds stop_ids declared via FeedEntity.stop in the
	// FeedMessage currently being validated. GTFS-RT TripModifications
	// (experimental) lets producers publish inline Stop entities that are
	// referenced by replacement_stops and modified-trip TripUpdates without
	// appearing in the static GTFS. Rebuilt at the start of each
	// ValidateFeedMessage call.
	currentInlineStops map[string]bool
	// currentPlainTripIds holds trip_ids of every plain (non-modified)
	// TripUpdate in the FeedMessage currently being validated. Used to
	// detect when a modified-trip TripUpdate is missing the parallel plain
	// fallback the spec recommends (W103). Rebuilt at the start of each
	// ValidateFeedMessage call.
	currentPlainTripIds map[string]bool
	geomCache          tlxy.GeomCache // shared with copier
	sched              *sched.ScheduleChecker
}

// NewValidator returns an initialized validator.
func NewValidator() *Validator {
	return &Validator{
		MaxDistanceFromTrip: 100.0,
		tripInfo:            map[string]tripInfo{},
		routeInfo:           map[string]routeInfo{},
		stopInfo:            map[string]stopInfo{},
		staticShapeIds:      map[string]bool{},
		sched:               sched.NewScheduleChecker(),
		geomCache:           geomcache.NewGeomCache(),
	}
}

// SetGeomCache sets a shared geometry cache.
func (fi *Validator) SetGeomCache(g tlxy.GeomCache) {
	fi.geomCache = g
}

// Validate gets a stream of entities from Copier to build up the cache.
func (fi *Validator) Validate(ent tt.Entity) []error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		fi.Timezone = v.AgencyTimezone.Val
	case *gtfs.Stop:
		fi.stopInfo[v.StopID.Val] = stopInfo{LocationType: v.LocationType.Int()}
	case *gtfs.Route:
		fi.routeInfo[v.RouteID.Val] = routeInfo{
			RouteType: v.RouteType.Int(),
			AgencyID:  v.AgencyID.Val,
		}
	case *gtfs.Trip:
		fi.tripInfo[v.TripID.Val] = tripInfo{
			DirectionID: v.DirectionID.Int(),
			ShapeID:     v.ShapeID.String(),
			RouteID:     v.RouteID.Val,
		}
	case *gtfs.Frequency:
		a := fi.tripInfo[v.TripID.Val]
		a.UsesFrequency = true
		fi.tripInfo[v.TripID.Val] = a
	case *gtfs.Shape:
		if v.ShapeID.Val != "" {
			fi.staticShapeIds[v.ShapeID.Val] = true
		}
	case *service.ShapeLine:
		if v.ShapeID.Val != "" {
			fi.staticShapeIds[v.ShapeID.Val] = true
		}
	}

	// Validate with schedule checker
	if err := fi.sched.Validate(ent); err != nil {
		return err
	}
	return nil
}

// ValidateFeedMessage .
func (fi *Validator) ValidateFeedMessage(current *pb.FeedMessage, previous *pb.FeedMessage) (errs []error) {
	if current.Header == nil {
		errs = append(errs, newError("FeedMessage Header is required", "header"))
	} else {
		// Check previous Header timestamp
		if currentTimestamp, previousTimestamp := current.GetHeader().GetTimestamp(), previous.GetHeader().GetTimestamp(); currentTimestamp < previousTimestamp {
			errs = append(errs, withFieldAndJson(
				E018,
				"header.timestamp",
				"",
				currentTimestamp,
				current.Header,
				"Header timestamp %d (local: %s) is before previous header timestamp %d (local: %s)",
				currentTimestamp,
				toLocalTime(int64(currentTimestamp), fi.Timezone),
				previousTimestamp,
				toLocalTime(int64(previousTimestamp), fi.Timezone),
			))
		}
		errs = append(errs, fi.ValidateHeader(current.Header, current)...)
	}
	// Pre-pass: collect inline Stop entities so stop_id references in
	// TripUpdates / VehiclePositions can resolve against them as well as
	// static stops, and plain (non-modified) TripUpdate trip_ids so W103
	// can detect missing fallbacks. See currentInlineStops and
	// currentPlainTripIds on Validator for details.
	fi.currentInlineStops = map[string]bool{}
	fi.currentPlainTripIds = map[string]bool{}
	for _, ent := range current.GetEntity() {
		if s := ent.GetStop(); s != nil {
			if sid := s.GetStopId(); sid != "" {
				fi.currentInlineStops[sid] = true
			}
		}
		if tu := ent.GetTripUpdate(); tu != nil {
			td := tu.GetTrip()
			if td != nil && td.GetModifiedTrip() == nil {
				if tid := td.GetTripId(); tid != "" {
					fi.currentPlainTripIds[tid] = true
				}
			}
		}
	}
	// TODO: Validate TripDescriptors are unique
	for _, ent := range current.GetEntity() {
		errs = append(errs, fi.ValidateFeedEntity(ent, current)...)
	}
	return errs
}

// ValidateHeader .
func (fi *Validator) ValidateHeader(header *pb.FeedHeader, current *pb.FeedMessage) (errs []error) {
	if gtfsRealtimeVersion := header.GetGtfsRealtimeVersion(); gtfsRealtimeVersion == "3.0" || gtfsRealtimeVersion == "2.0" {
		// TODO: additional version specific checks
	} else if gtfsRealtimeVersion == "1.0" {
		//ok
	} else {
		errs = append(errs, withFieldAndJson(
			E038,
			"header.gtfs_realtime_version",
			"",
			gtfsRealtimeVersion,
			header,
			"Invalid realtime version: %s",
			gtfsRealtimeVersion,
		))
	}
	//
	if headerTimestamp := int64(header.GetTimestamp()); header.Timestamp == nil || headerTimestamp == 0 {
		errs = append(errs, withFieldAndJson(
			E048,
			"header.timestamp",
			"",
			headerTimestamp,
			header,
			"",
		))
	} else if !checkTimestamp(headerTimestamp) {
		errs = append(errs, withFieldAndJson(
			E001,
			"header.timestamp",
			"",
			headerTimestamp,
			header,
			"Not in POSIX time: %d",
			headerTimestamp,
		))
	} else if !checkFuture(headerTimestamp) {
		errs = append(errs, withFieldAndJson(
			E050,
			"header.timestamp",
			"",
			headerTimestamp,
			header,
			"Timestamp is in the future: %d (local: %s)",
			headerTimestamp,
			toLocalTime(headerTimestamp, fi.Timezone),
		))
	}
	//
	if headerIncrementality := header.GetIncrementality(); header.Incrementality == nil {
		errs = append(errs, withFieldAndJson(
			E049,
			"header.incrementality",
			"",
			headerIncrementality,
			header,
			"",
		))
	} else if headerIncrementality == pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, newError("FeedHeader DIFFERENTIAL incrementality is not supported", "header.incrementality"))
	}
	return errs
}

// // ValidateFeedEntity .
func (fi *Validator) ValidateFeedEntity(ent *pb.FeedEntity, current *pb.FeedMessage) (errs []error) {
	headerIncrementality := current.GetHeader().GetIncrementality()
	if ent.Id == nil || ent.GetId() == "" {
		errs = append(errs, newError("FeedEntity id is required", "entity.id"))
	}
	if ent.IsDeleted != nil && headerIncrementality != pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, withFieldAndJson(
			E039,
			"entity.is_deleted",
			"",
			ent.IsDeleted,
			ent,
			"",
		))
	}
	if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil &&
		ent.Shape == nil && ent.Stop == nil && ent.TripModifications == nil {
		errs = append(errs, newError("FeedEntity must provide one of TripUpdate, VehiclePosition, Alert, Shape, Stop, or TripModifications", "entity"))
	}
	if tripUpdate := ent.GetTripUpdate(); tripUpdate != nil {
		errs = append(errs, fi.ValidateTripUpdate(tripUpdate, current)...)
	}
	if vehicle := ent.GetVehicle(); vehicle != nil {
		errs = append(errs, fi.ValidateVehiclePosition(vehicle)...)
	}
	if alert := ent.GetAlert(); alert != nil {
		// TODO: ValidateAlert
		// TODO: Check that route_id is not set in a TripDescriptor
	}
	if tm := ent.GetTripModifications(); tm != nil {
		errs = append(errs, fi.ValidateTripModifications(tm, ent)...)
	}
	if s := ent.GetStop(); s != nil {
		errs = append(errs, fi.ValidateStop(s, ent)...)
	}
	if sh := ent.GetShape(); sh != nil {
		errs = append(errs, fi.ValidateShape(sh, ent)...)
	}
	return errs
}

// ValidateStop validates a GTFS-RT Stop FeedEntity body (experimental
// TripModifications extension). Per spec, stop_id / stop_name / stop_lat /
// stop_lon are required, and stop_id MUST differ from any static stop_id.
func (fi *Validator) ValidateStop(s *pb.Stop, ent *pb.FeedEntity) (errs []error) {
	if s.GetStopId() == "" {
		errs = append(errs, withFieldAndJson(E102, "stop.stop_id", "", "", ent, "inline Stop entity is missing required field stop_id"))
	} else if _, ok := fi.stopInfo[s.GetStopId()]; ok {
		errs = append(errs, withFieldAndJsonWarning(
			W105,
			"stop.stop_id",
			"",
			s.GetStopId(),
			ent,
			"inline Stop.stop_id '%s' collides with a stop_id defined in static GTFS; per spec it MUST be different",
			s.GetStopId(),
		))
	}
	if s.GetStopName().GetTranslation() == nil && s.StopName == nil {
		errs = append(errs, withFieldAndJson(E102, "stop.stop_name", "", "", ent, "inline Stop entity is missing required field stop_name"))
	}
	if s.StopLat == nil {
		errs = append(errs, withFieldAndJson(E102, "stop.stop_lat", "", "", ent, "inline Stop entity is missing required field stop_lat"))
	}
	if s.StopLon == nil {
		errs = append(errs, withFieldAndJson(E102, "stop.stop_lon", "", "", ent, "inline Stop entity is missing required field stop_lon"))
	}
	return errs
}

// ValidateShape validates a GTFS-RT Shape FeedEntity body (experimental
// TripModifications extension). Per spec, shape_id and encoded_polyline are
// required, and shape_id MUST differ from any static shape_id.
func (fi *Validator) ValidateShape(s *pb.Shape, ent *pb.FeedEntity) (errs []error) {
	if s.GetShapeId() == "" {
		errs = append(errs, withFieldAndJson(E103, "shape.shape_id", "", "", ent, "inline Shape entity is missing required field shape_id"))
	} else if fi.staticShapeIds[s.GetShapeId()] {
		errs = append(errs, withFieldAndJsonWarning(
			W106,
			"shape.shape_id",
			"",
			s.GetShapeId(),
			ent,
			"inline Shape.shape_id '%s' collides with a shape_id defined in static GTFS; per spec it MUST be different",
			s.GetShapeId(),
		))
	}
	if s.GetEncodedPolyline() == "" {
		errs = append(errs, withFieldAndJson(E103, "shape.encoded_polyline", "", "", ent, "inline Shape entity is missing required field encoded_polyline"))
	}
	return errs
}

// ValidateTripModifications validates a TripModifications FeedEntity body
// (GTFS-RT experimental extension). Checks that selected_trips reference real
// static trips and that replacement_stops reference stops defined in either
// static GTFS or an inline FeedEntity.stop in the same FeedMessage.
func (fi *Validator) ValidateTripModifications(tm *pb.TripModifications, ent *pb.FeedEntity) (errs []error) {
	for _, sel := range tm.GetSelectedTrips() {
		for _, tid := range sel.GetTripIds() {
			if tid == "" {
				continue
			}
			if _, ok := fi.tripInfo[tid]; !ok {
				errs = append(errs, withFieldAndJsonWarning(
					W102,
					"trip_modifications.selected_trips.trip_ids",
					"",
					tid,
					ent,
					"TripModifications selected_trips references trip_id '%s' that does not exist in static GTFS data",
					tid,
				))
			}
		}
	}
	for _, mod := range tm.GetModifications() {
		for _, rs := range mod.GetReplacementStops() {
			sid := rs.GetStopId()
			if sid == "" {
				continue
			}
			if v, ok := fi.stopInfo[sid]; ok {
				if v.LocationType != 0 {
					errs = append(errs, withFieldAndJsonWarning(
						W104,
						"trip_modifications.modifications.replacement_stops.stop_id",
						"",
						sid,
						ent,
						"TripModifications replacement_stops references stop_id '%s' with location_type=%d; per spec replacement stops MUST have location_type=0 (routable stops)",
						sid,
						v.LocationType,
					))
				}
				continue
			}
			if fi.currentInlineStops[sid] {
				continue
			}
			errs = append(errs, withFieldAndJsonWarning(
				W101,
				"trip_modifications.modifications.replacement_stops.stop_id",
				"",
				sid,
				ent,
				"TripModifications replacement_stops references stop_id '%s' that is not defined in static GTFS or as an inline FeedEntity.stop",
				sid,
			))
		}
	}
	return errs
}

// ValidateTripUpdate .
func (fi *Validator) ValidateTripUpdate(tripUpdate *pb.TripUpdate, current *pb.FeedMessage) (errs []error) {
	tripDescriptor := tripUpdate.GetTrip()
	rtKey := fi.getRtTripKey(tripDescriptor)
	agencyId := rtKey.AgencyID

	// Validate TripDescriptor
	if tripDescriptor == nil {
		errs = append(errs, newError("TripDescriptor is required", "trip_update.trip"))
	} else {
		errs = append(errs, fi.validateTripDescriptor(tripDescriptor, tripUpdate)...)
	}
	// experimental field
	// 	if tripUpdate.Delay != nil {
	// }

	if tripUpdateTimestamp := int64(tripUpdate.GetTimestamp()); tripUpdate.Timestamp != nil && !checkTimestamp(tripUpdateTimestamp) {
		errs = append(errs, withFieldAndJson(
			E001,
			"trip_update.timestamp",
			agencyId,
			tripUpdateTimestamp,
			tripUpdate,
			"TripUpdate timestamp %d is missing or not in POSIX time",
			tripUpdateTimestamp,
		))
	}

	// Validate StopTimeUpdates
	scheduleRelationship := tripDescriptor.GetScheduleRelationship()
	stopTimeUpdates := tripUpdate.GetStopTimeUpdate()
	if len(stopTimeUpdates) == 0 && scheduleRelationship != pb.TripDescriptor_CANCELED {
		errs = append(errs, withFieldAndJson(
			E041,
			"trip_update.trip.schedule_relationship",
			agencyId,
			scheduleRelationship,
			tripUpdate,
			"",
		))
	}

	// Validate sequence
	seqVisited := map[uint32]int{}
	stopVisited := map[string]int{}
	prevStopSequence := uint32(0)
	prevStopId := ""
	prevTime := int64(0)
	for _, stopTimeUpdate := range stopTimeUpdates {
		if stopTimeUpdate == nil {
			continue
		}

		// Check if this stop has been visited more than once
		if stopId := stopTimeUpdate.GetStopId(); stopId != "" {
			stopVisited[stopId]++
			if stopTimeUpdate.StopSequence == nil && stopVisited[stopId] > 1 {
				errs = append(errs, withFieldAndJson(
					E009,
					"trip_update.stop_time_update.stop_sequence",
					agencyId,
					"",
					tripUpdate,
					"",
				))
			}
			if stopId == prevStopId {
				errs = append(errs, withFieldAndJson(
					E037,
					"trip_update.stop_time_update.stop_sequence",
					agencyId,
					"",
					tripUpdate,
					"",
				))
			}
			prevStopId = stopId
		}

		// Check if this stop sequence has been visited more than once
		if stopSequence := stopTimeUpdate.GetStopSequence(); stopTimeUpdate.StopSequence != nil {
			seqVisited[stopSequence]++
			if seqVisited[stopSequence] > 1 {
				errs = append(errs, withFieldAndJson(
					E036,
					"trip_update.stop_time_update",
					agencyId,
					stopSequence,
					tripUpdate,
					"TripUpdate contains a StopTimeUpdate with a stop sequence value of %d that is the same as a previous stop sequence",
					stopSequence,
				))

			}
			if stopSequence < prevStopSequence {
				errs = append(errs, withFieldAndJson(
					E002,
					"trip_update.stop_time_update",
					agencyId,
					stopSequence,
					tripUpdate,
					"TripUpdate contains a StopTimeUpdate with a stop sequence value of %d that is less than previous stop sequence %d",
					stopSequence,
					prevStopSequence,
				))

			}
			prevStopSequence = stopSequence
		}

		// Check Arrival Time
		if arrivalTime := stopTimeUpdate.GetArrival().GetTime(); stopTimeUpdate.Arrival != nil && stopTimeUpdate.Arrival.Time != nil && !checkTimestamp(arrivalTime) {
			errs = append(errs, withFieldAndJson(
				E001,
				"trip_update.stop_time_update.arrival.time",
				agencyId,
				arrivalTime,
				tripUpdate,
				"Not in POSIX time: %d",
				arrivalTime,
			))
		}

		// Check Departure Time
		if departureTime := stopTimeUpdate.GetDeparture().GetTime(); stopTimeUpdate.Departure != nil && stopTimeUpdate.Departure.Time != nil && !checkTimestamp(departureTime) {
			errs = append(errs, withFieldAndJson(
				E001,
				"trip_update.stop_time_update.departure.time",
				agencyId,
				departureTime,
				tripUpdate,
				"Not in POSIX time: %d",
				departureTime,
			))
		}

		// Check vs. previous time
		if arrivalTime := stopTimeUpdate.GetArrival().GetTime(); stopTimeUpdate.Arrival != nil && stopTimeUpdate.Arrival.Time != nil {
			if arrivalTime < prevTime {
				errs = append(errs, withFieldAndJson(
					E022,
					"trip_update.stop_time_update",
					agencyId,
					arrivalTime,
					tripUpdate,
					"TripUpdate contains a StopTimeUpdate where arrival time %d (local: %s) was before previous time %d (local: %s)",
					arrivalTime,
					toLocalTime(arrivalTime, fi.Timezone),
					prevTime,
					toLocalTime(prevTime, fi.Timezone),
				))
			}
			prevTime = arrivalTime
		}

		// Check vs. previous time
		if departureTime := stopTimeUpdate.GetDeparture().GetTime(); stopTimeUpdate.Departure != nil && stopTimeUpdate.Departure.Time != nil {
			if departureTime < prevTime {
				errs = append(errs, withFieldAndJson(
					E022,
					"trip_update.stop_time_update",
					agencyId,
					departureTime,
					tripUpdate,
					"TripUpdate contains a StopTimeUpdate where departure time %d (local: %s) was before previous time %d (local: %s)",
					departureTime,
					toLocalTime(departureTime, fi.Timezone),
					prevTime,
					toLocalTime(prevTime, fi.Timezone),
				))
			}
			prevTime = departureTime
		}

		// Check individual values
		errs = append(errs, fi.ValidateStopTimeUpdate(stopTimeUpdate, tripUpdate, current)...)
	}
	return errs
}

// ValidateStopTimeUpdate .
func (fi *Validator) ValidateStopTimeUpdate(st *pb.TripUpdate_StopTimeUpdate, tripUpdate *pb.TripUpdate, current *pb.FeedMessage) (errs []error) {
	tripDescriptor := tripUpdate.GetTrip()
	rtKey := fi.getRtTripKey(tripDescriptor)
	agencyId := rtKey.AgencyID

	if st.StopId == nil && st.StopSequence == nil {
		errs = append(errs, withFieldAndJson(
			E040,
			"trip_update.stop_time_update",
			agencyId,
			"",
			tripUpdate,
			"",
		))
	}
	if stopId := st.GetStopId(); stopId != "" {
		v, ok := fi.stopInfo[stopId]
		if !ok && !fi.currentInlineStops[stopId] {
			errs = append(errs, withFieldAndJson(
				E011,
				"trip_update.stop_time_update.stop_id",
				agencyId,
				stopId,
				tripUpdate,
				"TripUpdate has a StopTimeUpdate that references stop '%s' that does not exist in static GTFS data",
				st.GetStopId(),
			))
		}
		if v.LocationType != 0 {
			errs = append(errs, withFieldAndJson(
				E015,
				"trip_update.stop_time_update.stop_id",
				agencyId,
				stopId,
				tripUpdate,
				"TripUpdate has a StopTimeUpdate that references stop '%s' which has location_type '%d' but must be 0",
				stopId,
				v.LocationType,
			))
		}
	}

	// Arrival, Departure
	switch st.GetScheduleRelationship() {
	case pb.TripUpdate_StopTimeUpdate_SCHEDULED:
		if st.Arrival == nil && st.Departure == nil {
			errs = append(errs, withFieldAndJson(
				E043,
				"trip_update.schedule_relationship",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
		if arrival := st.Arrival; arrival != nil && (arrival.Time == nil && arrival.Delay == nil) {
			errs = append(errs, withFieldAndJson(
				E044,
				"trip_update.schedule_relationship",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
		if departure := st.Departure; departure != nil && (departure.Time == nil && departure.Delay == nil) {
			errs = append(errs, withFieldAndJson(
				E044,
				"trip_update.schedule_relationship",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
	case pb.TripUpdate_StopTimeUpdate_NO_DATA:
		if st.Arrival != nil || st.Departure != nil {
			errs = append(errs, withFieldAndJson(
				E042,
				"trip_update.schedule_relationship",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
	case pb.TripUpdate_StopTimeUpdate_SKIPPED:
		// ok
	}

	if arrivalTime, departureTime := st.GetArrival().GetTime(), st.GetDeparture().GetTime(); arrivalTime > 0 && departureTime > 0 && arrivalTime > departureTime {
		errs = append(errs, withFieldAndJson(
			E025,
			"trip_update.stop_time_update.arrival.time",
			agencyId,
			arrivalTime,
			tripUpdate,
			"TripUpdate contains a StopTimeUpdate with arrival time %d (local: %s) after departure time %d (local: %s)",
			arrivalTime,
			toLocalTime(arrivalTime, fi.Timezone),
			departureTime,
			toLocalTime(departureTime, fi.Timezone),
		))
	}

	// ValidateStopTimeEvent .
	// TODO
	return errs
}

func (fi *Validator) validateTripDescriptor(td *pb.TripDescriptor, tripUpdate *pb.TripUpdate) (errs []error) {
	rtKey := fi.getRtTripKey(td)
	agencyId := rtKey.AgencyID

	// GTFS-RT TripModifications: when modified_trip is present, validate its selector
	// and warn on legacy-field co-occurrence regardless of which identifier branch
	// below runs. Per spec the legacy fields (trip_id, route_id, direction_id,
	// start_time, start_date) MUST be empty when modified_trip is set; W100 is a
	// warning rather than an error to remain lenient with producers in the wild.
	if mt := td.GetModifiedTrip(); mt != nil {
		if mt.GetModificationsId() == "" {
			errs = append(errs, withFieldAndJson(
				E101,
				"trip_update.trip.modified_trip.modifications_id",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
		affectedTripId := mt.GetAffectedTripId()
		affectedExists := false
		if affectedTripId == "" {
			errs = append(errs, withFieldAndJson(
				E100,
				"trip_update.trip.modified_trip.affected_trip_id",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		} else if _, ok := fi.tripInfo[affectedTripId]; !ok {
			errs = append(errs, withFieldAndJson(
				E003,
				"trip_update.trip.modified_trip.affected_trip_id",
				agencyId,
				affectedTripId,
				tripUpdate,
				"TripUpdate modified_trip references affected_trip_id '%s' that does not exist in static GTFS data",
				affectedTripId,
			))
		} else {
			affectedExists = true
		}
		if td.GetTripId() != "" || td.RouteId != nil || td.DirectionId != nil || td.StartDate != nil || td.StartTime != nil {
			errs = append(errs, withFieldAndJsonWarning(
				W100,
				"trip_update.trip.modified_trip",
				agencyId,
				"",
				tripUpdate,
				"TripDescriptor sets modified_trip alongside legacy identifier fields; per spec these MUST be empty when modified_trip is set",
			))
		}
		// W103: per spec SHOULD, a modified-trip TripUpdate should be
		// accompanied by a parallel plain TripUpdate (same trip_id, no
		// modified_trip) for consumers that don't support TripModifications.
		// Only fires when the affected trip actually exists in static — no
		// point recommending a fallback for an invalid trip.
		if affectedExists && !fi.currentPlainTripIds[affectedTripId] {
			errs = append(errs, withFieldAndJsonWarning(
				W103,
				"trip_update.trip.modified_trip",
				agencyId,
				affectedTripId,
				tripUpdate,
				"Modified-trip TripUpdate for affected_trip_id '%s' has no parallel plain TripUpdate in the same FeedMessage; per spec SHOULD, providers should also publish an unmodified TripUpdate for legacy consumers",
				affectedTripId,
			))
		}
	}

	if tripId := td.GetTripId(); tripId != "" {
		tripInfo, ok := fi.tripInfo[tripId]
		// Check trip exists
		if !ok && td.GetScheduleRelationship() == pb.TripDescriptor_ADDED {
			// ADDED trip - allowed
		} else if !ok {
			errs = append(errs, withFieldAndJson(
				E003,
				"trip_update.trip.trip_id",
				agencyId,
				tripId,
				tripUpdate,
				"TripUpdate TripDescriptor references trip '%s' that does not exist in static GTFS data",
				tripId,
			))
		}
		// Check direction
		if directionId := td.GetDirectionId(); td.DirectionId != nil && int(directionId) != tripInfo.DirectionID {
			errs = append(errs, withFieldAndJson(
				E024,
				"trip_update.trip.trip_id",
				agencyId,
				tripId,
				tripUpdate,
				"",
			))
		}
		if tripInfo.UsesFrequency {
			if td.StartTime == nil || td.StartDate == nil {
				errs = append(errs, newError("TripDescriptor must provide start_date and start_time for frequency based trips", "trip_update.trip.start_time"))
			}
			// TODO: Additional frequency based trip checks
		}
	} else if td.GetModifiedTrip() == nil {
		// Neither trip_id nor modified_trip is set; require the legacy tuple.
		if td.RouteId == nil || td.DirectionId == nil || td.StartDate == nil || td.StartTime == nil {
			errs = append(errs, newError("TripDescriptor must provide a trip_id or all of route_id, direction_id, start_date, and start_time", "trip_update.trip.trip_id"))
		}
		if td.GetScheduleRelationship() != pb.TripDescriptor_SCHEDULED {
			errs = append(errs, newError("TripDescriptor must be SCHEDULED if no trip_id is provided", "trip_update.trip.trip_id"))
		}
	}
	if routeId := td.GetRouteId(); routeId != "" {
		if _, ok := fi.routeInfo[routeId]; !ok {
			errs = append(errs, withFieldAndJson(
				E004,
				"trip_update.trip.route_id",
				agencyId,
				routeId,
				tripUpdate,
				"TripUpdate TripDescriptor references route '%s' that does not exist in static GTFS data",
				routeId,
			))
		}
	}
	if startTime := td.GetStartTime(); startTime != "" {
		if wt, err := tt.NewSecondsFromString(startTime); err != nil {
			errs = append(errs, withFieldAndJson(
				E020,
				"trip_update.trip.start_time",
				agencyId,
				startTime,
				tripUpdate,
				"",
			))
		} else if wt.Int() > (7 * 24 * 60 * 60) {
			errs = append(errs, withFieldAndJson(
				E020,
				"trip_update.trip.start_time",
				agencyId,
				startTime,
				tripUpdate,
				"",
			))
		}
	}
	if startDate := td.GetStartDate(); startDate != "" {
		if _, err := time.Parse("20060102", startDate); err != nil {
			errs = append(errs, withFieldAndJson(
				E021,
				"trip_update.trip.start_date",
				agencyId,
				"",
				tripUpdate,
				"",
			))
		}
	}
	return errs
}

func (fi *Validator) ValidateVehiclePosition(ent *pb.VehiclePosition) (errs []error) {
	tripDescriptor := ent.GetTrip()
	rtKey := fi.getRtTripKey(tripDescriptor)
	agencyId := rtKey.AgencyID

	// Validate stop
	if stopId := ent.GetStopId(); stopId != "" {
		_, ok := fi.stopInfo[stopId]
		if !ok && !fi.currentInlineStops[stopId] {
			errs = append(errs, withFieldAndJson(
				E011,
				"vehicle_position.stop_id",
				agencyId,
				stopId,
				ent,
				"VehiclePosition references stop '%s' that does not exist in static GTFS data",
				stopId,
			))
		}
	}

	// Validate position
	pos := ent.GetPosition()
	posValid := fi.validatePosition(ent.Position, ent)
	errs = append(errs, posValid...)
	if len(posValid) == 0 {
		// Check distance from shape
		posPt := tlxy.Point{Lon: float64(pos.GetLongitude()), Lat: float64(pos.GetLatitude())}
		if td := ent.Trip; td != nil && td.TripId != nil {
			tripId := td.GetTripId()
			trip, tripOk := fi.tripInfo[tripId]
			shp := fi.geomCache.GetShape(trip.ShapeID)
			if !tripOk {
				errs = append(errs, withFieldAndJson(
					E003,
					"vehicle_position.trip.trip_id",
					agencyId,
					tripId,
					ent,
					"VehiclePosition TripDescriptor references trip '%s' that does not exist in static GTFS data",
					tripId,
				))
			} else if len(shp) == 0 {
				errs = append(errs, newError("Invalid shape_id", "trip_descriptor"))
			} else {
				nearestPoint, _, _ := tlxy.LineClosestPoint(shp, posPt)
				nearestPointDist := tlxy.DistanceHaversine(nearestPoint, posPt)
				if nearestPointDist > fi.MaxDistanceFromTrip {
					shpErr := withFieldAndJson(
						E029,
						"vehicle_position.position",
						agencyId,
						"",
						ent,
						"Vehicle position (%f,%f) is %0.2f meters from trip '%s' with shape_id '%s'",
						posPt.Lon,
						posPt.Lat,
						nearestPointDist,
						td.GetTripId(),
						trip.ShapeID,
					)
					var coords []float64
					for _, p := range shp {
						coords = append(coords, p.Lon, p.Lat)
					}
					// Create geometry manually because we want XY not XYM
					shpLineGeom := geom.NewLineStringFlat(geom.XY, coords)
					shpLineGeom.SetSRID(4326)
					shpPointGeom := geom.NewPointFlat(geom.XY, []float64{posPt.Lon, posPt.Lat})
					shpPointGeom.SetSRID(4326)

					// Create geom collection
					shpGeomCollection := geom.NewGeometryCollection()
					shpGeomCollection.Push(shpLineGeom)
					shpGeomCollection.Push(shpPointGeom)
					shpErr.geom = tt.NewGeometry(shpGeomCollection)
					errs = append(errs, shpErr)
				}
			}
		}
	}
	return errs
}

func (fi *Validator) validatePosition(pos *pb.Position, vehiclePosition *pb.VehiclePosition) (errs []error) {
	tripDescriptor := vehiclePosition.GetTrip()
	rtKey := fi.getRtTripKey(tripDescriptor)
	agencyId := rtKey.AgencyID

	if pos == nil {
		errs = append(errs, newError("Position required", "vehicle_position.position"))
		return errs
	}
	if longitude := pos.GetLongitude(); pos.Longitude == nil {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.longitude",
			agencyId,
			longitude,
			vehiclePosition,
			"Invalid longitude: null",
		))
	} else if longitude < -180 || longitude > 180 {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.longitude",
			agencyId,
			longitude,
			vehiclePosition,
			"Invalid longitude: %f",
			longitude,
		))
	} else if longitude == 0 {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.longitude",
			agencyId,
			longitude,
			vehiclePosition,
			"Invalid longitude: %f",
			longitude,
		))
	}
	if latitude := pos.GetLatitude(); pos.Latitude == nil {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.latitude",
			agencyId,
			latitude,
			vehiclePosition,
			"Invalid latitude: null",
		))
	} else if latitude < -90 || latitude > 90 {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.latitude",
			agencyId,
			latitude,
			vehiclePosition,
			"Invalid latitude: %f",
			latitude,
		))
	} else if latitude == 0 {
		errs = append(errs, withFieldAndJson(
			E026,
			"vehicle_position.position.latitude",
			agencyId,
			latitude,
			vehiclePosition,
			"Invalid latitude: %f",
			latitude,
		))
	}
	return errs
}

func (fi *Validator) getRtTripKey(trip *pb.TripDescriptor) rtTripKey {
	tripId := trip.GetTripId()
	// GTFS-RT TripModifications: when the TripDescriptor uses modified_trip
	// (with no trip_id), the affected_trip_id identifies the underlying static trip.
	if tripId == "" {
		tripId = trip.GetModifiedTrip().GetAffectedTripId()
	}
	ret := rtTripKey{
		TripID: tripId,
	}
	if trip.GetScheduleRelationship() == pb.TripDescriptor_ADDED {
		ret.Added = true
	}
	if a, ok := fi.tripInfo[tripId]; ok {
		ret.RouteID = a.RouteID
		ret.Found = true
	} else if b := trip.GetRouteId(); b != "" {
		ret.RouteID = b
	}
	if a, ok := fi.routeInfo[ret.RouteID]; ok {
		ret.AgencyID = a.AgencyID
	}
	return ret
}

type EntityCounts struct {
	Alert      int
	TripUpdate int
	Vehicle    int
}

func (fi *Validator) EntityCounts(msg *pb.FeedMessage) EntityCounts {
	ret := EntityCounts{}
	for _, ent := range msg.Entity {
		if ent.Vehicle != nil {
			ret.Vehicle += 1
		}
		if ent.TripUpdate != nil {
			ret.TripUpdate += 1
		}
		if ent.Alert != nil {
			ret.Alert += 1
		}
	}
	return ret
}

func checkTimestamp(ts int64) bool {
	// 1/1/1990 -> year 2038
	if ts < 631152000 || ts > (1<<31-1) {
		return false
	}
	return true
}

func checkFuture(ts int64) bool {
	// Is timestamp more than 1 minute in the future
	return ts <= int64(time.Now().Unix()+60)
}

func toLocalTime(v int64, tzName string) string {
	utcTime := time.Unix(int64(v), 0)
	tz, err := time.LoadLocation(tzName)
	if err != nil {
		return ""
	}
	localTime := utcTime.In(tz)
	return localTime.Format("15:04:05")
}
