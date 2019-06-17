package copier

import (
	gt "github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/log"
)

// visitedMarker is a Marker that visits all entities and marks those that are global or referenced by another Entity.
type visitedMarker struct {
	fileInfos map[string]fileInfo
}

// newVisitedMarker returns a new visitedMarker.
func newVisitedMarker() visitedMarker {
	return visitedMarker{
		fileInfos: map[string]fileInfo{},
	}
}

// IsMarked returns if an Entity is marked.
func (marker *visitedMarker) IsMarked(filename, eid string) bool {
	if v, ok := marker.fileInfos[filename]; ok {
		return v.IsMarked(eid)
	}
	return false
}

// IsVisited returns if an Entity was visited.
func (marker *visitedMarker) IsVisited(filename, eid string) bool {
	if v, ok := marker.fileInfos[filename]; ok {
		return v.IsVisited(eid)
	}
	return false
}

// VisitAndMark traverses the feed and marks entities.
func (marker *visitedMarker) VisitAndMark(reader gt.Reader) error {
	// markVisited adds all visited entities to the visited table
	// func (*Copier) markVisited(reader gt.Reader) error {
	// A bit tediuous, but...
	fis := map[string]fileInfo{}
	egfi := func(ent gt.Entity) *fileInfo {
		if fi, ok := fis[ent.Filename()]; ok {
			return &fi
		}
		fi := newFileInfo()
		fis[ent.Filename()] = fi
		return &fi
	}
	agencyInfo := egfi(&gt.Agency{})
	stopInfo := egfi(&gt.Stop{})
	routeInfo := egfi(&gt.Route{})
	shapeInfo := egfi(&gt.Shape{})
	tripInfo := egfi(&gt.Trip{})
	stoptimeInfo := egfi(&gt.StopTime{})
	calendarInfo := egfi(&gt.Calendar{})
	caldateInfo := egfi(&gt.CalendarDate{})
	transferInfo := egfi(&gt.Transfer{})
	fareruleInfo := egfi(&gt.FareRule{})
	feedinfoInfo := egfi(&gt.FeedInfo{})
	frequencyInfo := egfi(&gt.Frequency{})
	fareattributeInfo := egfi(&gt.FareAttribute{})
	serviceInfo := newFileInfo() // internal
	// Let's get started.
	log.Info("Marking visited:")
	// Be explicit about IDs.
	// Selected agencies
	log.Info("\tAgencies")
	for e := range reader.Agencies() {
		eid := e.AgencyID
		agencyInfo.Visit(eid)
		agencyInfo.Mark(eid)
		log.Trace("visited agency: '%s'", eid)
	}
	// Selected routes
	log.Info("\tRoutes")
	for e := range reader.Routes() {
		eid := e.RouteID
		routeInfo.Visit(eid)
		if len(e.AgencyID) == 0 {
			log.Debug("visited route, no agency_id: '%s'", eid)
			routeInfo.Mark(eid)
		} else if agencyInfo.IsMarked(e.AgencyID) {
			log.Trace("visited route: '%s'", eid)
			routeInfo.Mark(eid)
		} else {
			log.Debug("skipping route: '%s'", eid)
		}
	}
	// Selected trips
	log.Info("\tTrips")
	for e := range reader.Trips() {
		eid := e.TripID
		tripInfo.Visit(eid)
		if routeInfo.IsMarked(e.RouteID) {
			log.Trace("visited trip: '%s'", eid)
			tripInfo.Mark(eid)
		} else {
			log.Debug("skipping trip: '%s'", eid)
		}
	}
	// Selected stops and trip counter
	log.Info("\tStopTimes")
	stopCounter := map[string]int{}
	for e := range reader.StopTimes() {
		stoptimeInfo.Visit(e.TripID)
		if tripInfo.IsMarked(e.TripID) {
			stoptimeInfo.Mark(e.TripID)
			stopCounter[e.StopID]++
		} else {
			log.Debug("skipping stop_times for unvisited trip: '%s'", e.TripID)
		}
	}
	log.Info("\tStops")
	for e := range reader.Stops() {
		eid := e.StopID
		stopInfo.Visit(eid)
		if _, ok := stopCounter[eid]; ok {
			stopInfo.Mark(eid)
			// can be referenced multiple times..
			eps := e.ParentStation
			if len(eps) > 0 && !stopInfo.IsMarked(eps) {
				stopInfo.Mark(eps)
			}
		}
	}
	// Services, pruning trips
	shapeCounter := map[string]int{} // get totals later
	for e := range reader.Trips() {
		eid := e.TripID
		if stoptimeInfo.IsMarked(eid) {
			serviceInfo.Mark(e.ServiceID)
			shapeCounter[e.ShapeID]++
		} else {
			// TODO: Include trips without stops?
			log.Debug("pruning trip: '%s'", e.TripID)
			tripInfo.Unmark(eid)
		}
	}
	// Get more values for displaying counts.
	log.Info("\tCalendars")
	for e := range reader.Calendars() {
		eid := e.ServiceID
		calendarInfo.Visit(eid)
		if serviceInfo.IsMarked(eid) {
			log.Trace("visited calendar: '%s'", eid)
			calendarInfo.Mark(eid)
		}
	}
	log.Info("\tCalendarDates")
	for e := range reader.CalendarDates() {
		eid := e.ServiceID
		caldateInfo.Visit(eid)
		if serviceInfo.IsMarked(eid) {
			caldateInfo.Mark(eid)
		}
	}
	// Update with actual counts
	log.Info("\tShapes")
	for e := range reader.Shapes() {
		if _, ok := shapeCounter[e.ShapeID]; ok {
			log.Trace("visited shape: '%s'", e.ShapeID)
			// Measure per coordinate, if provided
			if e.Geometry != nil {
				numc := e.Geometry.NumCoords()
				shapeInfo.Visited[e.ShapeID] = numc
				shapeInfo.Marked[e.ShapeID] = numc
			} else {
				shapeInfo.Visit(e.ShapeID)
				shapeInfo.Mark(e.ShapeID)
			}
		}
	}
	// Other tables -- mark all
	log.Info("\tTransfers")
	for e := range reader.Transfers() {
		transferInfo.Visit(e.FromStopID)
		transferInfo.Mark(e.FromStopID)
	}
	log.Info("\tFrequencies")
	for e := range reader.Frequencies() {
		frequencyInfo.Visit(e.TripID)
		frequencyInfo.Mark(e.TripID)
	}
	log.Info("\tFareRules")
	for e := range reader.FareRules() {
		fareruleInfo.Visit(e.FareID)
		fareruleInfo.Mark(e.FareID)
	}
	log.Info("\tFareAttributes")
	for e := range reader.FareAttributes() {
		fareattributeInfo.Visit(e.FareID)
		fareattributeInfo.Mark(e.FareID)
	}
	log.Info("\tFeedInfos")
	for e := range reader.FeedInfos() {
		feedinfoInfo.Visit(e.FeedVersion)
		feedinfoInfo.Mark(e.FeedVersion)
	}
	marker.fileInfos = fis
	return nil
}
