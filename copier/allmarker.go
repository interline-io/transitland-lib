package copier

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/log"
)

// allMarker marks all entities in the reader.
type allMarker struct {
	fileInfos map[string]fileInfo
}

// newAllMarker returns a new allMarker.
func newAllMarker() allMarker {
	return allMarker{
		fileInfos: map[string]fileInfo{},
	}
}

// IsMarked returns if an entity is marked.
func (marker *allMarker) IsMarked(filename, eid string) bool {
	if v, ok := marker.fileInfos[filename]; ok {
		return v.IsMarked(eid)
	}
	return false
}

// IsVisited returns if an entity is visited.
func (marker *allMarker) IsVisited(filename, eid string) bool {
	if v, ok := marker.fileInfos[filename]; ok {
		return v.IsVisited(eid)
	}
	return false
}

// VisitAndMark traverses the feed and marks entities.
func (marker *allMarker) VisitAndMark(reader gotransit.Reader) error {
	// markAll marks all entities as seen & visited
	fis := map[string]fileInfo{}
	egfi := func(ent gotransit.Entity) *fileInfo {
		if fi, ok := fis[ent.Filename()]; ok {
			return &fi
		}
		fi := newFileInfo()
		fis[ent.Filename()] = fi
		return &fi
	}
	agencyInfo := egfi(&gotransit.Agency{})
	stopInfo := egfi(&gotransit.Stop{})
	routeInfo := egfi(&gotransit.Route{})
	shapeInfo := egfi(&gotransit.Shape{})
	tripInfo := egfi(&gotransit.Trip{})
	stoptimeInfo := egfi(&gotransit.StopTime{})
	calendarInfo := egfi(&gotransit.Calendar{})
	caldateInfo := egfi(&gotransit.CalendarDate{})
	transferInfo := egfi(&gotransit.Transfer{})
	fareruleInfo := egfi(&gotransit.FareRule{})
	feedinfoInfo := egfi(&gotransit.FeedInfo{})
	frequencyInfo := egfi(&gotransit.Frequency{})
	fareattributeInfo := egfi(&gotransit.FareAttribute{})
	log.Info("Marking all:")
	log.Info("\tAgencies")
	for e := range reader.Agencies() {
		agencyInfo.Visit(e.AgencyID)
		agencyInfo.Mark(e.AgencyID)
	}
	log.Info("\tRoutes")
	for e := range reader.Routes() {
		routeInfo.Visit(e.RouteID)
		routeInfo.Mark(e.RouteID)
	}
	log.Info("\tStops")
	for e := range reader.Stops() {
		stopInfo.Visit(e.StopID)
		stopInfo.Mark(e.StopID)
	}
	log.Info("\tShapes")
	for e := range reader.Shapes() {
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
	log.Info("\tTrips")
	for e := range reader.Trips() {
		tripInfo.Visit(e.TripID)
		tripInfo.Mark(e.TripID)
	}
	log.Info("\tStopTimes")
	for e := range reader.StopTimes() {
		stoptimeInfo.Visit(e.TripID)
		stoptimeInfo.Mark(e.TripID)
	}
	log.Info("\tCalendars")
	for e := range reader.Calendars() {
		calendarInfo.Visit(e.ServiceID)
		calendarInfo.Mark(e.ServiceID)
	}
	log.Info("\tCalendarDates")
	for e := range reader.CalendarDates() {
		caldateInfo.Visit(e.ServiceID)
		caldateInfo.Mark(e.ServiceID)
	}
	log.Info("\tTransfers")
	for e := range reader.Transfers() {
		transferInfo.Visit(e.FromStopID)
		transferInfo.Mark(e.FromStopID)
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
	log.Info("\tFrequencies")
	for e := range reader.Frequencies() {
		frequencyInfo.Visit(e.TripID)
		frequencyInfo.Mark(e.TripID)
	}
	log.Info("\tFeedInfos")
	for e := range reader.FeedInfos() {
		feedinfoInfo.Visit(e.FeedVersion)
		feedinfoInfo.Mark(e.FeedVersion)
	}
	marker.fileInfos = fis
	return nil
}
