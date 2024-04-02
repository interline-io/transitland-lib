package rt

import (
	"fmt"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/interline-io/transitland-lib/ext/sched"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
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

// Validator validates RT messages based on data from a static feed.
// It can be initialized through NewValidatorFromReader or through the Copier Validator interface.
type Validator struct {
	Timezone            string
	MaxDistanceFromTrip float64
	tripInfo            map[string]tripInfo
	routeInfo           map[string]routeInfo
	stopInfo            map[string]stopInfo
	geomCache           *xy.GeomCache // shared with copier
	sched               *sched.ScheduleChecker
}

// NewValidator returns an initialized validator.
func NewValidator() *Validator {
	return &Validator{
		MaxDistanceFromTrip: 100.0,
		tripInfo:            map[string]tripInfo{},
		routeInfo:           map[string]routeInfo{},
		stopInfo:            map[string]stopInfo{},
		sched:               sched.NewScheduleChecker(),
		geomCache:           xy.NewGeomCache(),
	}
}

// SetGeomCache sets a shared geometry cache.
func (fi *Validator) SetGeomCache(g *xy.GeomCache) {
	fi.geomCache = g
}

// Validate gets a stream of entities from Copier to build up the cache.
func (fi *Validator) Validate(ent tl.Entity) []error {
	switch v := ent.(type) {
	case *tl.Agency:
		fi.Timezone = v.AgencyTimezone
	case *tl.Stop:
		fi.stopInfo[v.StopID] = stopInfo{LocationType: v.LocationType}
	case *tl.Route:
		fi.routeInfo[v.RouteID] = routeInfo{
			RouteType: v.RouteType,
			AgencyID:  v.AgencyID,
		}
	case *tl.Trip:
		fi.tripInfo[v.TripID] = tripInfo{
			DirectionID: v.DirectionID,
			ShapeID:     v.ShapeID.String(),
			RouteID:     v.RouteID,
		}
	case *tl.Frequency:
		a := fi.tripInfo[v.TripID]
		a.UsesFrequency = true
		fi.tripInfo[v.TripID] = a
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
	if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil {
		errs = append(errs, newError("FeedEntity must provide one of TripUpdate, VehiclePosition, or Alert", "entity"))
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
	return errs
}

// ValidateTripUpdate .
func (fi *Validator) ValidateTripUpdate(tripUpdate *pb.TripUpdate, current *pb.FeedMessage) (errs []error) {
	tripDescriptor := tripUpdate.GetTrip()
	agencyId := fi.getAgencyId(tripDescriptor.GetTripId(), tripDescriptor.GetRouteId())

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
	agencyId := fi.getAgencyId(tripDescriptor.GetTripId(), tripDescriptor.GetRouteId())

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
		if !ok {
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
	agencyId := fi.getAgencyId(td.GetTripId(), td.GetRouteId())
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
	} else {
		if td.RouteId == nil || td.DirectionId == nil || td.StartDate == nil || td.StartTime == nil {
			errs = append(errs, newError("TripDescriptor must provided a trip_id or all of route_id, direction_id, start_date, and start_time", "trip_update.trip.trip_id"))
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
		if wt, err := tt.NewWideTime(startTime); err != nil {
			errs = append(errs, withFieldAndJson(
				E020,
				"trip_update.trip.start_time",
				agencyId,
				startTime,
				tripUpdate,
				"",
			))
		} else if wt.Seconds > (7 * 24 * 60 * 60) {
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
	agencyId := fi.getAgencyId(tripDescriptor.GetTripId(), tripDescriptor.GetRouteId())

	// Validate stop
	if stopId := ent.GetStopId(); stopId != "" {
		_, ok := fi.stopInfo[stopId]
		if !ok {
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
		posPt := xy.Point{Lon: float64(pos.GetLongitude()), Lat: float64(pos.GetLatitude())}
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
				nearestPoint, _ := xy.LineClosestPoint(shp, posPt)
				nearestPointDist := xy.DistanceHaversine(nearestPoint.Lon, nearestPoint.Lat, posPt.Lon, posPt.Lat)
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
					shpErr.geom = tt.Geometry{Geometry: shpGeomCollection, Valid: true}
					errs = append(errs, shpErr)
				}
			}
		}
	}
	return errs
}

func (fi *Validator) validatePosition(pos *pb.Position, vehiclePosition *pb.VehiclePosition) (errs []error) {
	tripDescriptor := vehiclePosition.GetTrip()
	agencyId := fi.getAgencyId(tripDescriptor.GetTripId(), tripDescriptor.GetRouteId())

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

type VehiclePositionStats struct {
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
}

func (fi *Validator) VehiclePositionStats(now time.Time, msg *pb.FeedMessage) ([]VehiclePositionStats, error) {
	scheduledTrips := fi.sched.ActiveTrips(now)
	var rtTrips []string
	for _, ent := range msg.Entity {
		tu := ent.Vehicle
		if tu == nil {
			continue
		}
		tripId := tu.GetTrip().GetTripId()
		rtTrips = append(rtTrips, tripId)
	}
	stats, err := fi.compareTripSets(scheduledTrips, rtTrips)
	if err != nil {
		return nil, err
	}
	// Use direct conversion
	var ret []VehiclePositionStats
	for _, s := range stats {
		ret = append(ret, VehiclePositionStats(s))
	}
	return ret, nil
}

// Note: db for TripScheduledMatched
type TripUpdateStats struct {
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
}

func (fi *Validator) TripUpdateStats(now time.Time, msg *pb.FeedMessage) ([]TripUpdateStats, error) {
	scheduledTrips := fi.sched.ActiveTrips(now)
	var rtTrips []string
	for _, ent := range msg.Entity {
		tu := ent.TripUpdate
		if tu == nil {
			continue
		}
		tripId := tu.GetTrip().GetTripId()
		rtTrips = append(rtTrips, tripId)
	}
	stats, err := fi.compareTripSets(scheduledTrips, rtTrips)
	if err != nil {
		return nil, err
	}
	// Use direct conversion
	var ret []TripUpdateStats
	for _, s := range stats {
		ret = append(ret, TripUpdateStats(s))
	}
	return ret, nil
}

type rtTripStat struct {
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
}

func (fi *Validator) compareTripSets(scheduledTrips []string, rtTrips []string) ([]rtTripStat, error) {
	type statAggKey struct {
		AgencyID string
		RouteID  string
	}
	statAgg := map[statAggKey]rtTripStat{}

	// Process scheduled trips
	for _, tripId := range scheduledTrips {
		trip, ok := fi.tripInfo[tripId]
		if !ok {
			continue
		}
		k := statAggKey{
			RouteID:  trip.RouteID,
			AgencyID: fi.routeInfo[trip.RouteID].AgencyID,
		}
		stat := statAgg[k]
		stat.AgencyID = k.AgencyID
		stat.RouteID = k.RouteID
		// fmt.Println("found sched:", k, tripId)
		stat.TripScheduledIDs = append(stat.TripScheduledIDs, tripId)
		statAgg[k] = stat
	}

	// Process RT entities
	for _, tripId := range rtTrips {
		trip, ok := fi.tripInfo[tripId]
		if !ok {
			continue
		}
		k := statAggKey{
			RouteID:  trip.RouteID,
			AgencyID: fi.routeInfo[trip.RouteID].AgencyID,
		}
		// fmt.Println("found rt:", k, tripId)
		stat := statAgg[k]
		stat.TripRtIDs = append(stat.TripRtIDs, tripId)
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
	var ret []rtTripStat
	for _, k := range statAggSortedKeys {
		v := statAgg[k]
		scheduledSet := mapset.NewSet[string](v.TripScheduledIDs...)
		updateSet := mapset.NewSet[string](v.TripRtIDs...)
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

func (fi *Validator) getAgencyId(tripId string, routeId string) string {
	agencyId := ""
	if trip, ok := fi.tripInfo[tripId]; ok {
		routeId = trip.RouteID
	}
	if route, ok := fi.routeInfo[routeId]; ok {
		agencyId = route.AgencyID
	}
	return agencyId
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
	if ts > int64(time.Now().Unix()+60) {
		return false
	}
	return true
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
