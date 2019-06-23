package copier

import (
	"fmt"
	"strconv"

	"github.com/interline-io/gotransit/causes"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/enums"
	"github.com/interline-io/gotransit/internal/log"
)

type copyableExtension interface {
	Copy(*Copier) error
}

type errorWithContext interface {
	Context() *causes.Context
}

// CopyError wraps an underlying GTFS Error with the filename and entity ID.
type CopyError struct {
	filename string
	entityID string
	cause    error
}

// NewCopyError returns a new CopyError error with filename and id set.
func NewCopyError(efn string, eid string, err error) *CopyError {
	return &CopyError{
		filename: efn,
		entityID: eid,
		cause:    err,
	}
}

// Error returns the error string.
func (ce *CopyError) Error() string {
	return fmt.Sprintf("%s '%s': %s", ce.filename, ce.entityID, ce.cause)
}

// Cause returns the underlying GTFS Error
func (ce *CopyError) Cause() error {
	return ce.cause
}

// Context returns the error Context
func (ce *CopyError) Context() *causes.Context {
	return &causes.Context{
		Filename: ce.filename,
		EntityID: ce.entityID,
	}
}

////////////////////////////
////////// Copier //////////
////////////////////////////

// Copier copies from Reader to Writer
type Copier struct {
	Reader    gotransit.Reader
	Writer    gotransit.Writer
	BatchSize int
	// Attempt to save an entity that returns validation errors
	AllowEntityErrors    bool
	AllowReferenceErrors bool
	// Interpolate any missing StopTime values: ArrivalTime/DepartureTime/ShapeDistTraveled
	InterpolateStopTimes bool
	// Create a stop-to-stop Shape for Trips without a ShapeID.
	CreateMissingShapes bool
	// Create missing Calendar entries
	NormalizeServiceIDs bool
	// Convert extended route types to primitives
	UseBasicRouteTypes bool
	// Default AgencyID
	DefaultAgencyID string
	// Entity selection strategy
	Marker Marker
	// book keeping
	agencyCount  int
	extensions   []copyableExtension      // interface
	filters      []gotransit.EntityFilter // interface
	geomCache    *geomCache
	stopPatterns map[string]int
	*CopyResult
	*gotransit.EntityMap
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader gotransit.Reader, writer gotransit.Writer) Copier {
	copier := Copier{
		Reader:               reader,
		Writer:               writer,
		BatchSize:            1000,
		AllowEntityErrors:    false,
		AllowReferenceErrors: false,
		InterpolateStopTimes: false,
		CreateMissingShapes:  false,
		NormalizeServiceIDs:  false,
	}
	// Result
	copier.CopyResult = NewCopyResult()
	// Default Markers
	copier.Marker = newYesMarker()
	// Default EntityMap
	copier.EntityMap = gotransit.NewEntityMap()
	// Default filters
	copier.filters = []gotransit.EntityFilter{}
	// Geom Cache
	copier.geomCache = newGeomCache()
	copier.stopPatterns = map[string]int{}
	// Set the DefaultAgencyID from the Reader
	copier.DefaultAgencyID = ""
	for e := range copier.Reader.Agencies() {
		copier.DefaultAgencyID = e.AgencyID
		copier.agencyCount++
	}
	return copier
}

// AddExtension adds an Extension to the copy process.
func (copier *Copier) AddExtension(ext gotransit.Extension) error {
	extc, ok := ext.(copyableExtension)
	if !ok {
		return fmt.Errorf("ext does not provide Copy method")
	}
	copier.extensions = append(copier.extensions, extc)
	return nil
}

// AddEntityFilter adds an EntityFilter to the copy process.
func (copier *Copier) AddEntityFilter(ef gotransit.EntityFilter) error {
	copier.filters = append(copier.filters, ef)
	return nil
}

////////////////////////////////////
////////// Helper methods //////////
////////////////////////////////////

func (copier *Copier) isMarked(ent gotransit.Entity) bool {
	// Check if the entity is marked for copying.
	return copier.Marker.IsMarked(ent.Filename(), ent.EntityID())
}

// CopyEntity performs validation and saves errors and warnings, returns new EntityID and true if written.
// Any errors and warnings are added to the CopyResult.
func (copier *Copier) CopyEntity(ent gotransit.Entity) (string, bool) {
	efn := ent.Filename()
	eid := ent.EntityID()
	sid := ent.EntityID()
	if !copier.isMarked(ent) {
		return "", false
	}
	ret := true
	// Check the entity against filters.
	for _, ef := range copier.filters {
		if err := ef.Filter(ent, copier.EntityMap); err != nil {
			log.Debug("%s '%s' skipped by filter: %s", efn, eid, err)
			ret = false
		}
	}
	// Check the entity for errors.
	if errs := ent.Errors(); len(errs) > 0 {
		for _, i := range errs {
			copier.AddError(NewCopyError(efn, eid, i))
		}
		if copier.AllowEntityErrors {
			log.Debug("%s '%s' has errors, allowing: %s", efn, eid, errs)
		} else {
			log.Debug("%s '%s' has errors, skipping: %s", efn, eid, errs)
			ret = false
		}
	}
	// Check the entity for warnings.
	if warns := ent.Warnings(); len(warns) > 0 {
		// warnings
		for _, i := range warns {
			copier.AddWarning(NewCopyError(efn, eid, i))
		}
	}
	// Check the entity for reference errors.
	if err := ent.UpdateKeys(copier.EntityMap); err != nil {
		copier.AddError(NewCopyError(efn, eid, err))
		if copier.AllowReferenceErrors {
			log.Debug("%s '%s' failed to update keys, allowing: %s", efn, eid, err)
		} else {
			log.Debug("%s '%s' failed to update keys, skipping: %s", efn, eid, err)
			ret = false
		}
	}
	// Refresh EntityID after UpdateKeys/Filters
	eid = ent.EntityID()
	// Check for duplicate entities.
	if _, ok := copier.EntityMap.Get(ent); ok && len(eid) > 0 {
		copier.CopyResult.AddError(NewCopyError(ent.Filename(), eid, causes.NewDuplicateIDError(eid)))
		ret = false
	}
	if ret == false {
		return "", false
	}
	// OK, Save
	eid, err := copier.Writer.AddEntity(ent)
	if err != nil {
		log.Info("Warning: failed to write %s '%s': %s", efn, eid, err)
		copier.AddError(NewCopyError("", efn, err))
		return "", false
	}
	copier.EntityMap.Set(ent, sid, eid)
	copier.CopyResult.AddEntity(ent)
	return eid, true
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS Entities from the Reader to the Writer, returning the summary as a CopyResult.
func (copier *Copier) Copy() *CopyResult {
	for _, err := range copier.Reader.ValidateStructure() {
		copier.AddError(err)
	}
	// Note that order is important!!
	copier.copyAgencies()
	copier.copyRoutes()
	copier.copyStopsAndFares()
	copier.copyCalendars()
	copier.copyShapes()
	copier.copyTripsAndStopTimes()
	copier.copyFrequencies()
	copier.copyTransfers()
	copier.copyFeedInfos()
	for _, ext := range copier.extensions {
		ext.Copy(copier)
	}
	return copier.CopyResult
}

/////////////////////////////////////////
////////// Entity Copy Methods //////////
/////////////////////////////////////////

// copyAgencies writes agencies
func (copier *Copier) copyAgencies() {
	firstTimezone := ""
	for e := range copier.Reader.Agencies() {
		// Check for Timezone consistency - add to feed errors
		if len(firstTimezone) == 0 {
			firstTimezone = e.AgencyTimezone
		} else if e.AgencyTimezone != firstTimezone {
			copier.AddWarning(NewCopyError("", "agency.txt", causes.NewInconsistentTimezoneError(e.AgencyTimezone)))
		}
		// Check for conditionally required AgencyID - add to feed errors
		if len(e.AgencyID) == 0 && copier.agencyCount > 1 {
			copier.AddWarning(NewCopyError("", "agency.txt", causes.NewConditionallyRequiredFieldError("agency_id")))
		}
		copier.CopyEntity(&e)
	}
}

// copyStopsAndFares writes stops and their associated fare rules
func (copier *Copier) copyStopsAndFares() {
	// Stop bookkeeping
	parents := map[string]int{}
	farezones := map[string]int{}
	// First pass for stations
	for e := range copier.Reader.Stops() {
		if e.LocationType != 1 {
			continue
		}
		// Add stop, update farezones and geom cache
		sid := e.EntityID()
		if _, ok := copier.CopyEntity(&e); ok {
			farezones[e.ZoneID]++
			copier.geomCache.AddStop(sid, e)
			parents[sid] = e.LocationType
		}
	}
	// Second pass for platforms, exits, and generic nodes
	boards := []gotransit.Stop{}
	for e := range copier.Reader.Stops() {
		if e.LocationType == 1 {
			continue
		}
		// Save boarding areas for last
		if e.LocationType == 4 {
			boards = append(boards, e)
			continue
		}
		// Confirm the parent station location_type != 0
		if len(e.ParentStation) == 0 {
			// ok
		} else if pstype, ok := parents[e.ParentStation]; ok && pstype != 1 {
			// ParentStation found, not correct LocationType
			e.AddError(causes.NewInvalidParentStationError(e.ParentStation))
		} else if !ok {
			// ParentStation not found
			e.AddError(causes.NewInvalidParentStationError(e.ParentStation))
		}
		// Add stop, update farezones and geom cache
		sid := e.EntityID()
		if _, ok := copier.CopyEntity(&e); ok {
			farezones[e.ZoneID]++
			copier.geomCache.AddStop(sid, e)
		}
	}
	// Finally, boarding areas
	for _, e := range boards {
		// Confirm the parent station location_type != 0
		if len(e.ParentStation) == 0 {
			// ok
		} else if pstype := parents[e.ParentStation]; pstype != 0 {
			e.AddError(causes.NewInvalidParentStationError(e.ParentStation))
		}
		// Add stop, update farezones and geom cache
		sid := e.EntityID()
		if _, ok := copier.CopyEntity(&e); ok {
			farezones[e.ZoneID]++
			copier.geomCache.AddStop(sid, e)
		}
	}
	// FareAttributes
	for e := range copier.Reader.FareAttributes() {
		if len(e.AgencyID) == 0 {
			e.AgencyID = copier.DefaultAgencyID // todo: as else below?
			if copier.agencyCount > 1 {
				e.AddError(causes.NewConditionallyRequiredFieldError("agency_id"))
			}
		}
		copier.CopyEntity(&e)
	}
	// FareRules
	for e := range copier.Reader.FareRules() {
		// Explicitly check if the FareID is Marked
		// FareAttributes are named entities and it's up to the Marker
		// TODO: Should I just check the EntityMap instead?
		//     Do I care if it is marked, or it if was actually written OK?
		//     Same pattern for CalendarDates?
		if !copier.isMarked(&gotransit.FareAttribute{FareID: e.FareID}) {
			continue
		}
		// Add reference errors if we didn't add this farezone to the output
		if _, ok := farezones[e.OriginID]; len(e.OriginID) > 0 && !ok {
			e.AddError(causes.NewInvalidFarezoneError("origin_id", e.OriginID))
		}
		if _, ok := farezones[e.DestinationID]; len(e.DestinationID) > 0 && !ok {
			e.AddError(causes.NewInvalidFarezoneError("destination_id", e.DestinationID))
		}
		if _, ok := farezones[e.ContainsID]; len(e.ContainsID) > 0 && !ok {
			e.AddError(causes.NewInvalidFarezoneError("contains_id", e.ContainsID))
		}
		copier.CopyEntity(&e)
	}
}

// copyRoutes writes routes
func (copier *Copier) copyRoutes() {
	for e := range copier.Reader.Routes() {
		// Set default agencyID
		if len(e.AgencyID) == 0 {
			e.AgencyID = copier.DefaultAgencyID // todo - as else below?
			if copier.agencyCount > 1 {
				e.AddError(causes.NewConditionallyRequiredFieldError("agency_id"))
			}
		}
		// Use basic route types
		if copier.UseBasicRouteTypes {
			if rt, ok := enums.GetBasicRouteType(e.RouteType); ok {
				e.RouteType = rt.Code
			} else {
				e.AddError(causes.NewInvalidFieldError("route_type", strconv.Itoa(e.RouteType), fmt.Errorf("cannot convert route_type %d to basic route type", e.RouteType)))
			}
		}
		copier.CopyEntity(&e)
	}
}

// copyCalendars copies Calendars and CalendarDates
func (copier *Copier) copyCalendars() {
	// Calendars
	for e := range copier.Reader.Calendars() {
		copier.CopyEntity(&e)
	}
	// Create additional Calendars
	if copier.NormalizeServiceIDs {
		copier.createMissingCalendars()
	}
	// TODO: Make Entity method
	type calkey struct {
		ServiceID string
		Date      string
	}
	dups := map[calkey]int{}
	// Add CalendarDates
	for e := range copier.Reader.CalendarDates() {
		if !copier.isMarked(&gotransit.Calendar{ServiceID: e.ServiceID}) {
			continue
		}
		key := calkey{
			ServiceID: e.ServiceID,
			Date:      e.Date.Format("20060102"),
		}
		if _, ok := dups[key]; ok {
			copier.AddError(NewCopyError(e.Filename(), e.EntityID(), causes.NewDuplicateIDError(e.EntityID())))
			continue
		}
		dups[key]++
		copier.CopyEntity(&e)
	}
}

// copyFeedInfos writes FeedInfos
func (copier *Copier) copyFeedInfos() {
	for e := range copier.Reader.FeedInfos() {
		copier.CopyEntity(&e)
	}
}

// copyTransfers writes Transfers
func (copier *Copier) copyTransfers() {
	for e := range copier.Reader.Transfers() {
		// Check if Transfer stops are marked
		if copier.isMarked(&gotransit.Stop{StopID: e.FromStopID}) && copier.isMarked(&gotransit.Stop{StopID: e.ToStopID}) {
			copier.CopyEntity(&e)
		}
	}
}

// copyShapes writes Shapes
func (copier *Copier) copyShapes() {
	for e := range copier.Reader.ShapeLinesByShapeID() {
		sid := e.EntityID()
		if _, ok := copier.CopyEntity(&e); ok {
			copier.geomCache.AddShape(sid, e)
		}
	}
}

// copyFrequencies writes Frequencies
func (copier *Copier) copyFrequencies() {
	for e := range copier.Reader.Frequencies() {
		// Check if Trip is marked
		if copier.isMarked(&gotransit.Trip{TripID: e.TripID}) {
			copier.CopyEntity(&e)
		}
	}
}

// copyTripsAndStopTimes writes Trips and StopTimes
func (copier *Copier) copyTripsAndStopTimes() {
	// Check trips for visited, and check for errors
	// They will be updated before write.
	alltripids := map[string]int{}
	trips := map[string]gotransit.Trip{}
	for trip := range copier.Reader.Trips() {
		eid := trip.EntityID()
		alltripids[eid]++
		if !copier.isMarked(&trip) {
			continue
		}
		if _, ok := trips[eid]; ok {
			copier.AddError(NewCopyError("trips.txt", eid, causes.NewDuplicateIDError(eid)))
			continue
		}
		trips[eid] = trip
	}
	batch := []gotransit.StopTime{}
	for stoptimes := range copier.Reader.StopTimesByTripID() {
		if len(stoptimes) == 0 {
			log.Info("Warning: StopTimesByTripID produced zero StopTimes")
			continue
		}
		// Does this trip exist?
		tripid := stoptimes[0].TripID
		if _, ok := alltripids[tripid]; !ok {
			copier.AddError(NewCopyError("stop_times.txt", tripid, causes.NewInvalidReferenceError("trip_id", tripid)))
			continue
		}
		// Is this trip marked?
		trip, ok := trips[tripid]
		if !ok {
			continue // trip_id exists but is not marked
		} else {
			delete(trips, tripid) // check trips without StopTimes later
		}
		// Check for errors
		if len(stoptimes) < 2 {
			trip.AddError(causes.NewEmptyTripError(len(stoptimes)))
		}
		// Set StopPattern
		patkey := stopPatternKey(stoptimes)
		if pat, ok := copier.stopPatterns[patkey]; ok {
			// log.Debug("trip %s stop_pattern %d", tripid, pat)
			trip.StopPatternID = pat
		} else {
			pat := len(copier.stopPatterns)
			copier.stopPatterns[patkey] = pat
			// log.Debug("trip %s stop_pattern new %d", tripid, pat)
			trip.StopPatternID = pat
		}
		// Do we need to create a shape for this trip
		if len(trip.ShapeID) == 0 && copier.CreateMissingShapes {
			// Note: if the trip has errors, may result in unused shapes!
			shapeid, err := copier.createMissingShape(stoptimes)
			if err != nil {
				copier.AddError(NewCopyError("trips.txt", tripid, err))
			} else {
				trip.ShapeID = shapeid
			}
		}
		// Save trip
		if _, ok := copier.CopyEntity(&trip); !ok {
			continue
		}
		// Validate StopTimes, as a group
		sterrs := []error{}
		stwarns := []error{}
		streferrs := []error{}
		// StopTime errors
		for _, st := range stoptimes {
			sterrs = append(sterrs, st.Errors()...)
			stwarns = append(stwarns, st.Warnings()...)
		}
		// []StopTime errors
		sterrs = append(sterrs, gotransit.ValidateStopTimes(stoptimes)...)
		// Interpolate StopTimes if necessary - only if no other errors
		if len(sterrs) == 0 && copier.InterpolateStopTimes {
			if stoptimes2, err := copier.geomCache.InterpolateStopTimes(trip, stoptimes); err != nil {
				stwarns = append(stwarns, err)
			} else {
				stoptimes = stoptimes2
			}
		}
		for i := 0; i < len(stoptimes); i++ {
			if err := stoptimes[i].UpdateKeys(copier.EntityMap); err != nil {
				streferrs = append(streferrs, err)
			}
		}
		// Add errors
		for _, err := range sterrs {
			copier.AddError(NewCopyError("stop_times.txt", tripid, err))
		}
		for _, err := range stwarns {
			copier.AddWarning(NewCopyError("stop_times.txt", tripid, err))
		}
		for _, err := range streferrs {
			copier.AddError(NewCopyError("stop_times.txt", tripid, err))
		}
		// Should we continue?
		if len(sterrs) > 0 && !copier.AllowEntityErrors {
			continue
		}
		if len(streferrs) > 0 && !copier.AllowReferenceErrors {
			continue
		}
		// After updateKeys
		// for _, st := range stoptimes {
		// 	if err := copier.filterEntity(&st); err != nil {
		// 		// log.Debug("%s '%s' skipped by filter: %s", efn, eid, err)
		// 		// return false
		// 	}
		// }
		// OK, Everything is OK to go.
		batch = append(batch, stoptimes...)
		// Write in batches
		if len(batch) >= copier.BatchSize {
			bst := []gotransit.Entity{}
			// note: "range" re-uses the same pointer.
			for i := 0; i < len(batch); i++ {
				bst = append(bst, &batch[i])
			}
			if err := copier.Writer.AddEntities(bst); err != nil {
				copier.AddError(NewCopyError("", "stop_times.txt", err))
			}
			batch = nil
		}
	}
	// Write final batch
	if len(batch) > 0 {
		bst := []gotransit.Entity{}
		for i := 0; i < len(batch); i++ {
			bst = append(bst, &batch[i])
		}
		if err := copier.Writer.AddEntities(bst); err != nil {
			copier.AddError(NewCopyError("", "stop_times.txt", err))
		}
	}
	// Add any Trips that were not visited/did not have StopTimes
	for _, trip := range trips {
		errs := gotransit.ValidateStopTimes([]gotransit.StopTime{})
		for _, err := range errs {
			copier.AddError(NewCopyError("trips.txt", trip.TripID, err))
		}
		copier.CopyEntity(&trip)
	}
}

////////////////////////////////////////////
////////// Entity Support Methods //////////
////////////////////////////////////////////

func (copier *Copier) createMissingShape(stoptimes []gotransit.StopTime) (string, error) {
	stopids := []string{}
	for _, st := range stoptimes {
		stopids = append(stopids, st.StopID)
	}
	shape, err := copier.geomCache.MakeShape(stopids...)
	if err != nil {
		return "", err
	}
	eid, err := copier.Writer.AddEntity(&shape)
	if err != nil {
		copier.AddError(NewCopyError("", shape.Filename(), err))
	}
	return eid, nil
}

// createMissingCalendars to fully normalize ServiceIDs
func (copier *Copier) createMissingCalendars() {
	// Prepare to create missing Calendars
	missing := map[string]gotransit.Calendar{}
	for e := range copier.Reader.CalendarDates() {
		cal := gotransit.Calendar{
			ServiceID: e.ServiceID,
			Generated: true,
			StartDate: e.Date,
			EndDate:   e.Date,
		}
		if !copier.isMarked(&cal) {
			continue
		}
		// Do we already have this Calendar?
		if _, ok := copier.Get(&cal); ok {
			continue
		}
		// Do we already know this ServiceID?
		if c, ok := missing[e.ServiceID]; ok {
			cal = c
		}
		// Update the date range
		if e.ExceptionType == 1 {
			if e.Date.After(cal.EndDate) {
				cal.EndDate = e.Date
			}
			if e.Date.Before(cal.StartDate) {
				cal.StartDate = e.Date
			}
		}
		missing[e.ServiceID] = cal
	}
	// Create the missing Calendars
	for _, e := range missing {
		// log.Debug("create missing cal: %#v\n", e)
		eid, err := copier.Writer.AddEntity(&e)
		if err != nil {
			copier.AddError(NewCopyError("", e.Filename(), err))
		}
		copier.Set(&e, e.EntityID(), eid)
	}
}
