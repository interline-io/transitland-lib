package rt

import (
	"time"

	"github.com/interline-io/transitland-lib/ext/sched"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/twpayne/go-geom"
)

type tripInfo struct {
	DirectionID int
	ShapeID     string
	RouteID     string
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
	tripInfo  map[string]tripInfo
	routeInfo map[string]routeInfo
	stopInfo  map[string]stopInfo
	geomCache *xy.GeomCache // shared with copier
	sched     *sched.ScheduleChecker
}

// NewValidator returns an initialized validator.
func NewValidator() *Validator {
	return &Validator{
		tripInfo:  map[string]tripInfo{},
		routeInfo: map[string]routeInfo{},
		stopInfo:  map[string]stopInfo{},
		sched:     sched.NewScheduleChecker(),
		geomCache: xy.NewGeomCache(),
	}
}

// SetGeomCache sets a shared geometry cache.
func (fi *Validator) SetGeomCache(g *xy.GeomCache) {
	fi.geomCache = g
}

// Validate gets a stream of entities from Copier to build up the cache.
func (fi *Validator) Validate(ent tl.Entity) []error {
	switch v := ent.(type) {
	case *tl.Stop:
		fi.stopInfo[v.StopID] = stopInfo{LocationType: v.LocationType}
	case *tl.Route:
		fi.routeInfo[v.RouteID] = routeInfo{
			RouteType: v.RouteType,
			AgencyID:  v.AgencyID,
		}
	case *tl.Trip:
		ti := tripInfo{
			DirectionID: v.DirectionID,
			ShapeID:     v.ShapeID.String(),
			RouteID:     v.RouteID,
		}
		fi.tripInfo[v.TripID] = ti
	case *tl.Frequency:
		a := fi.tripInfo[v.TripID]
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
		if current.GetHeader().GetTimestamp() < previous.GetHeader().GetTimestamp() {
			errs = append(errs, withField(E018, "header.timestamp"))
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
	if v := header.GetGtfsRealtimeVersion(); v == "3.0" || v == "2.0" {
		// TODO: additional version specific checks
	} else if v == "1.0" {
		//ok
	} else {
		errs = append(errs, withField(E038, ""))
	}
	//
	if v := header.GetTimestamp(); header.Timestamp == nil || v == 0 {
		errs = append(errs, withField(E048, "header.timestamp"))
	} else if !checkTimestamp(v) {
		errs = append(errs, withField(E001, "header.timestamp"))

	} else if !checkFuture(v) {
		errs = append(errs, withField(E050, "header.timestamp"))

	}
	//
	if header.Incrementality == nil {
		errs = append(errs, withField(E049, "header.incrementality"))

	} else if header.GetIncrementality() == pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, newError("FeedHeader DIFFERENTIAL incrementality is not supported", "header.incrementality"))
	}
	return errs
}

// // ValidateFeedEntity .
func (fi *Validator) ValidateFeedEntity(ent *pb.FeedEntity, current *pb.FeedMessage) (errs []error) {
	incr := current.GetHeader().GetIncrementality()
	if ent.Id == nil || ent.GetId() == "" {
		errs = append(errs, newError("FeedEntity id is required", "entity.id"))
	}
	if ent.IsDeleted != nil && incr != pb.FeedHeader_DIFFERENTIAL {
		errs = append(errs, withField(E039, "entity.is_deleted"))

	}
	if ent.TripUpdate == nil && ent.Vehicle == nil && ent.Alert == nil {
		errs = append(errs, newError("FeedEntity must provide one of TripUpdate, VehiclePosition, or Alert", "entity"))
	}
	if ent.TripUpdate != nil {
		errs = append(errs, fi.ValidateTripUpdate(ent.GetTripUpdate(), current)...)
	}
	if ent.Vehicle != nil {
		errs = append(errs, fi.ValidateVehiclePosition(ent.GetVehicle())...)
	}
	if ent.Alert != nil {
		// TODO: ValidateAlert
		// TODO: Check that route_id is not set in a TripDescriptor
	}
	return errs
}

// ValidateTripUpdate .
func (fi *Validator) ValidateTripUpdate(trip *pb.TripUpdate, current *pb.FeedMessage) (errs []error) {
	// Validate TripDescriptor
	if trip.Trip == nil {
		errs = append(errs, newError("TripDescriptor is required", "trip_update.trip"))
	} else {
		errs = append(errs, fi.validateTripDescriptor(trip.Trip)...)
	}
	if trip.Delay != nil {
		// experimental field
	}
	if trip.Timestamp != nil && !checkTimestamp(uint64(trip.GetTimestamp())) {
		errs = append(errs, withField(E001, "trip_update.timestamp"))

	}
	// Validate StopTimeUpdates
	srel := trip.GetTrip().GetScheduleRelationship()
	sts := trip.GetStopTimeUpdate()
	if len(sts) == 0 && srel != pb.TripDescriptor_CANCELED {
		errs = append(errs, withField(E041, "trip_update.trip.schedule_relationship"))
	}
	seq := uint32(0)
	visitedseq := map[uint32]int{}
	visitedstop := map[string]int{}
	prevstop := ""
	prevtime := int64(0)
	for _, st := range sts {
		if st == nil {
			continue
		}
		ss := st.StopSequence
		stopid := st.StopId
		if stopid != nil {
			s2 := *stopid
			visitedstop[s2]++
			if ss == nil && visitedstop[s2] > 1 {
				errs = append(errs, withField(E009, "trip_update.stop_time_update.stop_sequence"))
			}
			if s2 == prevstop {
				errs = append(errs, withField(E037, "trip_update.stop_time_update"))
			}
			prevstop = s2
		}
		if ss != nil {
			s2 := *ss
			visitedseq[s2]++
			if visitedseq[s2] > 1 {
				errs = append(errs, withField(E036, "trip_update.stop_time_update"))

			}
			if s2 < seq {
				errs = append(errs, withField(E002, "trip_update.stop_time_update"))

			}
			seq = s2
		}
		if st.Arrival != nil && st.Arrival.Time != nil && !checkTimestamp(uint64(st.GetArrival().GetTime())) {
			errs = append(errs, withField(E001, "trip_update.stop_time_update.arrival.time"))

		}
		if st.Departure != nil && st.Departure.Time != nil && !checkTimestamp(uint64(st.GetDeparture().GetTime())) {
			errs = append(errs, withField(E001, "trip_update.stop_time_update.departure.time"))
		}
		// if st.GetArrival().Time != nil {
		if st.Arrival != nil && st.Arrival.Time != nil {
			a := *st.Arrival.Time
			if a < prevtime {
				errs = append(errs, withField(E022, "trip_update.stop_time_update"))
			}
			prevtime = a
		}
		if st.Departure != nil && st.Departure.Time != nil {
			a := *st.Departure.Time
			if a < prevtime {
				errs = append(errs, withField(E022, "trip_update.stop_time_update"))
			}
			prevtime = a
		}
		// Check individual values
		errs = append(errs, fi.ValidateStopTimeUpdate(st, current)...)
	}
	return errs
}

// ValidateStopTimeUpdate .
func (fi *Validator) ValidateStopTimeUpdate(st *pb.TripUpdate_StopTimeUpdate, current *pb.FeedMessage) (errs []error) {
	if st.StopId == nil && st.StopSequence == nil {
		errs = append(errs, withField(E040, "trip_update.stop_time_update"))
	}
	if st.StopId != nil {
		v, ok := fi.stopInfo[*st.StopId]
		if !ok {
			errs = append(errs, withField(E011, "trip_update.stop_time_update.stop_id"))
		}
		if v.LocationType != 0 {
			errs = append(errs, withField(E015, "trip_update.stop_time_update.stop_id"))
		}
	}
	// Arrival, Departure
	switch st.GetScheduleRelationship() {
	case pb.TripUpdate_StopTimeUpdate_SCHEDULED:
		if st.Arrival == nil && st.Departure == nil {
			errs = append(errs, withField(E043, "trip_update.schedule_relationship"))
		}
		if a := st.Arrival; a != nil && (a.Time == nil && a.Delay == nil) {
			errs = append(errs, withField(E044, "trip_update.schedule_relationship"))
		}
		if a := st.Departure; a != nil && (a.Time == nil && a.Delay == nil) {
			errs = append(errs, withField(E044, "trip_update.schedule_relationship"))
		}
	case pb.TripUpdate_StopTimeUpdate_NO_DATA:
		if st.Arrival != nil || st.Departure != nil {
			errs = append(errs, withField(E042, "trip_update.schedule_relationship"))
		}
	case pb.TripUpdate_StopTimeUpdate_SKIPPED:
		// ok
	}
	if st.GetArrival().GetTime() > 0 && st.GetDeparture().GetTime() > 0 && st.GetArrival().GetTime() > st.GetDeparture().GetTime() {
		errs = append(errs, withFieldAndJson(E025, "trip_update.stop_time_update.arrival.time", st))
	}
	// ValidateStopTimeEvent .
	// TODO
	return errs
}

func (fi *Validator) validateTripDescriptor(td *pb.TripDescriptor) (errs []error) {
	if td.TripId != nil {
		tripid := *td.TripId
		v, ok := fi.tripInfo[tripid]
		if !ok {
			errs = append(errs, withField(E003, "trip_update.trip.trip_id"))
		}
		if td.DirectionId != nil && td.GetDirectionId() != uint32(v.DirectionID) {
			errs = append(errs, withField(E024, "trip_update.trip.trip_id"))
		}
		freq := false
		if freq {
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
	if td.RouteId != nil {
		if _, ok := fi.routeInfo[*td.RouteId]; !ok {
			errs = append(errs, withField(E004, "trip_update.trip.route_id"))
		}
	}
	if td.StartTime != nil {
		if st, err := tt.NewWideTime(*td.StartTime); err != nil {
			errs = append(errs, withField(E020, "trip_update.trip.start_time"))
		} else if st.Seconds > (7 * 24 * 60 * 60) {
			errs = append(errs, withField(E020, "trip_update.trip.start_time"))
		}
	}
	if td.StartDate != nil {
		if _, err := time.Parse("20060102", *td.StartDate); err != nil {
			errs = append(errs, withField(E021, "trip_update.trip.start_date"))
		}
	}
	return errs
}

func (fi *Validator) ValidateVehiclePosition(ent *pb.VehiclePosition) (errs []error) {
	// Validate stop
	if ent.StopId != nil {
		_, ok := fi.stopInfo[*ent.StopId]
		if !ok {
			errs = append(errs, withField(E011, "vehicle_position.stop_id"))
		}
	}

	// Validate position
	pos := ent.GetPosition()
	posValid := fi.validatePosition(ent.Position)
	errs = append(errs, posValid...)
	if len(posValid) == 0 {
		// Check distance from shape
		posPt := xy.Point{Lon: float64(pos.GetLongitude()), Lat: float64(pos.GetLatitude())}
		if td := ent.Trip; td != nil && td.TripId != nil {
			trip, tripOk := fi.tripInfo[td.GetTripId()]
			shp := fi.geomCache.GetShape(trip.ShapeID)
			if !tripOk {
				errs = append(errs, withField(E003, "vehicle_position.trip.trip_id"))
			} else if len(shp) == 0 {
				errs = append(errs, newError("Invalid shape_id", "trip_descriptor"))
			} else {
				nearestPoint, _ := xy.LineClosestPoint(shp, posPt)
				nearestPointDist := xy.DistanceHaversine(nearestPoint.Lon, nearestPoint.Lat, posPt.Lon, posPt.Lat)
				if nearestPointDist > 100.0 {
					shpErr := withFieldAndJson(E029, "vehicle_position.position", ent)
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

					// fmt.Printf("GEOMS: %#v\n", shpErr.geoms)
					errs = append(errs, shpErr)
				}
			}
		}
	}
	return errs
}

func (fi *Validator) validatePosition(pos *pb.Position) (errs []error) {
	if pos == nil {
		errs = append(errs, newError("Position required", "vehicle_position.position"))
		return errs
	}
	if lon := pos.GetLongitude(); pos.Longitude == nil {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.longitude", pos))
	} else if lon < -180 || lon > 180 {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.longitude", pos))
	} else if lon == 0 {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.longitude", pos))
	}
	if lat := pos.GetLatitude(); pos.Latitude == nil {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.latitude", pos))
	} else if lat < -90 || lat > 90 {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.latitude", pos))
	} else if lat == 0 {
		errs = append(errs, withFieldAndJson(E026, "vehicle_position.position.latitude", pos))
	}
	return errs
}

type VehiclePositionStats struct {
	RouteID            string   `json:"route_id"`
	AgencyID           string   `json:"agency_id"`
	TripScheduledIDs   []string `json:"trip_scheduled_ids"`
	TripScheduledCount int      `json:"trip_scheduled_count"`
	TripMatchCount     int      `json:"trip_match_count"`
}

func (fi *Validator) VehiclePositionStats(now time.Time, msg *pb.FeedMessage) ([]VehiclePositionStats, error) {
	tripHasPosition := map[string]bool{}
	for _, ent := range msg.Entity {
		vp := ent.Vehicle
		if vp == nil {
			continue
		}
		pos := vp.GetPosition()
		if td := vp.Trip; td != nil && pos != nil {
			tripId := td.GetTripId()
			tripHasPosition[tripId] = true
		}
	}
	// Return early if no VehiclePositions
	if len(tripHasPosition) == 0 {
		return nil, nil
	}
	type statAggKey struct {
		RouteID  string
		AgencyID string
	}
	statAgg := map[statAggKey]VehiclePositionStats{}
	for _, tripId := range fi.sched.ActiveTrips(now) {
		trip := fi.tripInfo[tripId]
		k := statAggKey{
			RouteID:  trip.RouteID,
			AgencyID: fi.routeInfo[trip.RouteID].AgencyID,
		}
		stat := statAgg[k]
		stat.AgencyID = k.AgencyID
		stat.RouteID = k.RouteID
		stat.TripScheduledIDs = append(stat.TripScheduledIDs, tripId)
		stat.TripScheduledCount += 1
		if tripHasPosition[tripId] {
			stat.TripMatchCount += 1
		}
		statAgg[k] = stat
	}
	var ret []VehiclePositionStats
	for _, v := range statAgg {
		ret = append(ret, v)
	}
	return ret, nil

}

type TripUpdateStats struct {
	AgencyID           string   `json:"agency_id"`
	RouteID            string   `json:"route_id"`
	TripScheduledIDs   []string `json:"trip_scheduled_ids"`
	TripScheduledCount int      `json:"trip_scheduled_count"`
	TripMatchCount     int      `json:"trip_match_count"`
}

func (fi *Validator) TripUpdateStats(now time.Time, msg *pb.FeedMessage) ([]TripUpdateStats, error) {
	tripHasUpdate := map[string]bool{}
	for _, ent := range msg.Entity {
		tu := ent.TripUpdate
		if tu == nil {
			continue
		}
		tripHasUpdate[tu.GetTrip().GetTripId()] = true
	}
	// Return early if no TripUpdates
	if len(tripHasUpdate) == 0 {
		return nil, nil
	}
	type statAggKey struct {
		AgencyID string
		RouteID  string
	}
	statAgg := map[statAggKey]TripUpdateStats{}
	for _, tripId := range fi.sched.ActiveTrips(now) {
		trip := fi.tripInfo[tripId]
		k := statAggKey{
			RouteID:  trip.RouteID,
			AgencyID: fi.routeInfo[trip.RouteID].AgencyID,
		}
		stat := statAgg[k]
		stat.AgencyID = k.AgencyID
		stat.RouteID = k.RouteID
		stat.TripScheduledIDs = append(stat.TripScheduledIDs, tripId)
		stat.TripScheduledCount += 1
		if tripHasUpdate[tripId] {
			stat.TripMatchCount += 1
		}
		statAgg[k] = stat
	}
	var ret []TripUpdateStats
	for _, v := range statAgg {
		ret = append(ret, v)
	}
	return ret, nil
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
