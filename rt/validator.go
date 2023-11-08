package rt

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type tripInfo struct {
	UsesFrequency bool
	DirectionID   int
	ServiceID     string
	ShapeID       string
	StartTime     tt.WideTime
	EndTime       tt.WideTime
}

type stopInfo struct {
	LocationType int
}

type routeInfo struct {
	RouteType int
}

// Validator validates RT messages based on data from a static feed.
// It can be initialized through NewValidatorFromReader or through the Copier Validator interface.
type Validator struct {
	tripInfo  map[string]tripInfo
	routeInfo map[string]routeInfo
	stopInfo  map[string]stopInfo
	services  map[string]*tl.Service
	geomCache *xy.GeomCache // shared with copier
}

// NewValidator returns an initialized validator.
func NewValidator() *Validator {
	return &Validator{
		tripInfo:  map[string]tripInfo{},
		routeInfo: map[string]routeInfo{},
		stopInfo:  map[string]stopInfo{},
		services:  map[string]*tl.Service{},
		geomCache: xy.NewGeomCache(),
	}
}

// NewValidatorFromReader returns a Validator with data from a Reader.
func NewValidatorFromReader(reader tl.Reader) (*Validator, error) {
	fi := NewValidator()
	for v := range reader.Stops() {
		fi.stopInfo[v.StopID] = stopInfo{LocationType: v.LocationType}
	}
	for v := range reader.Routes() {
		fi.routeInfo[v.RouteID] = routeInfo{RouteType: v.RouteType}
	}
	for v := range reader.Trips() {
		fi.tripInfo[v.TripID] = tripInfo{DirectionID: v.DirectionID}
	}
	for v := range reader.Frequencies() {
		a := fi.tripInfo[v.TripID]
		a.UsesFrequency = true
		fi.tripInfo[v.TripID] = a
	}
	return fi, nil
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
		fi.routeInfo[v.RouteID] = routeInfo{RouteType: v.RouteType}
	case *tl.Service:
		fi.services[v.ServiceID] = v
	case *tl.Trip:
		ti := tripInfo{
			DirectionID: v.DirectionID,
			ServiceID:   v.ServiceID,
			ShapeID:     v.ShapeID.String(),
		}
		if len(v.StopTimes) > 0 {
			ti.StartTime = v.StopTimes[0].DepartureTime
			ti.EndTime = v.StopTimes[len(v.StopTimes)-1].ArrivalTime
		}
		fi.tripInfo[v.TripID] = ti
	case *tl.Frequency:
		a := fi.tripInfo[v.TripID]
		a.UsesFrequency = true
		fi.tripInfo[v.TripID] = a
	}
	return nil
}

// ValidateFeedMessage .
func (fi *Validator) ValidateFeedMessage(current *pb.FeedMessage, previous *pb.FeedMessage) (errs []error) {
	if current.Header == nil {
		errs = append(errs, ne("FeedMessage Header is required", 0))
	} else {
		// Check previous Header timestamp
		if current.GetHeader().GetTimestamp() < previous.GetHeader().GetTimestamp() {
			errs = append(errs, ne("FeedMessage Header timestamp is earlier than previous update", 18))
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

// // ValidateFeedEntity .
func (fi *Validator) ValidateFeedEntity(ent *pb.FeedEntity, current *pb.FeedMessage) (errs []error) {
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
		errs = append(errs, ne("TripDescriptor is required", 0))
	} else {
		errs = append(errs, fi.ValidateTripDescriptor(trip.Trip, current)...)
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
		if st == nil {
			continue
		}
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
		// if st.GetArrival().Time != nil {
		if st.Arrival != nil && st.Arrival.Time != nil {
			a := *st.Arrival.Time
			if a < prevtime {
				errs = append(errs, ne("StopTimeUpdates are not increasing in time", 22))
			}
			prevtime = a
		}
		if st.Departure != nil && st.Departure.Time != nil {
			a := *st.Departure.Time
			if a < prevtime {
				errs = append(errs, ne("StopTimeUpdates are not increasing in time", 22))
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
		errs = append(errs, ne("StopTimeUpdate must specify stop_sequence or stop_id", 40))
	}
	if st.StopId != nil {
		v, ok := fi.stopInfo[*st.StopId]
		if !ok {
			errs = append(errs, ne("StopTimeUpdate references unknown stop_id", 11))
		}
		if v.LocationType != 0 {
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
	if st.GetArrival().GetTime() > 0 && st.GetDeparture().GetTime() > 0 && st.GetArrival().GetTime() > st.GetDeparture().GetTime() {
		errs = append(errs, ne("StopTimeUpdate arrival time is later than departure time", 25))
	}
	// ValidateStopTimeEvent .
	// TODO
	return errs
}

// ValidateTripDescriptor .
func (fi *Validator) ValidateTripDescriptor(td *pb.TripDescriptor, current *pb.FeedMessage) (errs []error) {
	if td.TripId != nil {
		tripid := *td.TripId
		v, ok := fi.tripInfo[tripid]
		if !ok {
			errs = append(errs, ne("TripDescriptor references unknown trip_id", 3))
		}
		if td.DirectionId != nil && td.GetDirectionId() != uint32(v.DirectionID) {
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
		if _, ok := fi.routeInfo[*td.RouteId]; !ok {
			errs = append(errs, ne("TripDescriptor references unknown route_id", 4))
		}
	}
	if td.StartTime != nil {
		if _, err := tt.NewWideTime(*td.StartTime); err != nil {
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

func (fi *Validator) ActiveTrips(now time.Time) []string {
	var ret []string
	nowWt := tt.NewWideTimeFromSeconds(now.Hour()*3600 + now.Minute()*60 + now.Second())
	nowSvc := map[string]bool{}
	tripHasUpdate := map[string]bool{}
	msgTripIds := map[string]bool{}
	for k, v := range fi.tripInfo {
		svc, ok := fi.services[v.ServiceID]
		if !ok {
			// log.Debug().
			// 	Str("service", v.ServiceID).
			// 	Str("trip", k).
			// 	Msg("no service, skipping")
			continue
		}
		sched, ok := nowSvc[svc.ServiceID]
		if !ok {
			sched = svc.IsActive(now)
			nowSvc[svc.ServiceID] = sched
		}
		if !sched {
			// log.Debug().
			// 	Str("date", now.Format("2006-02-03")).
			// 	Str("service", v.ServiceID).
			// 	Str("trip", k).
			// 	Msg("not scheduled, skipping")
			continue
		}
		if v.StartTime.Seconds > nowWt.Seconds || v.EndTime.Seconds < nowWt.Seconds {
			// log.Debug().
			// 	Str("date", now.Format("2006-02-03")).
			// 	Str("cur_time", nowWt.String()).
			// 	Str("trip_start", v.StartTime.String()).
			// 	Str("trip_end", v.EndTime.String()).
			// 	Str("service", v.ServiceID).
			// 	Str("trip", k).
			// 	Msg("outside time, skipping")
			continue
		}
		ret = append(ret, k)
		tripHasUpdate[k] = false
		if msgTripIds[k] {
			tripHasUpdate[k] = true
		}
	}
	return ret
}

type TripUpdateStats struct {
	TripScheduledCount   int       `json:"trip_scheduled_count,omitempty"`
	TripUpdateMatchCount int       `json:"trip_update_match_count,omitempty"`
	Date                 time.Time `json:"date,omitempty"`
}

func (fi *Validator) TripUpdateStats(now time.Time, msg *pb.FeedMessage) (TripUpdateStats, error) {
	schedTrips := fi.ActiveTrips(now)
	tripHasUpdate := map[string]bool{}
	msgTripIds := map[string]bool{}
	for _, ent := range msg.Entity {
		if tu := ent.TripUpdate; tu != nil {
			msgTripIds[tu.GetTrip().GetTripId()] = true
		}
	}
	for _, k := range schedTrips {
		tripHasUpdate[k] = false
		if msgTripIds[k] {
			tripHasUpdate[k] = true
		}
	}
	tuCount := 0
	for _, v := range tripHasUpdate {
		if v {
			tuCount += 1
		}
	}
	return TripUpdateStats{
		TripScheduledCount:   len(tripHasUpdate),
		TripUpdateMatchCount: tuCount,
		Date:                 now,
	}, nil
}

func (fi *Validator) ValidateVehiclePosition(ent *pb.VehiclePosition) (errs []error) {
	pos := ent.GetPosition()
	posPt := xy.Point{Lon: float64(pos.GetLongitude()), Lat: float64(pos.GetLatitude())}
	tripHasPosition := map[string]bool{}
	if td := ent.Trip; td != nil && pos != nil {
		tripId := td.GetTripId()
		trip, ok := fi.tripInfo[tripId]
		shp := fi.geomCache.GetShape(trip.ShapeID)
		tripHasPosition[tripId] = true
		if ok && trip.ShapeID != "" && len(shp) > 0 {
			fmt.Println("Vehicle position:", posPt)
			nearestPoint, _ := xy.LineClosestPoint(shp, posPt)
			fmt.Println("\ttrip:", tripId, "shape:", trip.ShapeID)
			fmt.Println("\tnearestPoint:", nearestPoint, "dist:", xy.DistanceHaversine(nearestPoint.Lon, nearestPoint.Lat, posPt.Lon, posPt.Lat))
		}
	}
	for _, schedTrip := range fi.ActiveTrips(time.Now()) {
		_ = schedTrip
	}
	return nil
}

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
