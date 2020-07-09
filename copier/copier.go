package copier

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/interline-io/gotransit/causes"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/enums"
	"github.com/interline-io/gotransit/internal/log"
)

// ErrorHandler is called on each source file and entity; errors can be nil
type ErrorHandler interface {
	HandleEntityErrors(gotransit.Entity, []error, []error)
	HandleSourceErrors(string, []error, []error)
}

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
	// Error handler, called for each entity
	ErrorHandler ErrorHandler
	// book keeping
	agencyCount         int
	extensions          []copyableExtension      // interface
	filters             []gotransit.EntityFilter // interface
	geomCache           *geomCache
	stopPatterns        map[string]int
	stopPatternShapeIDs map[int]string
	result              *CopyResult
	*gotransit.EntityMap
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader gotransit.Reader, writer gotransit.Writer) Copier {
	copier := Copier{
		Reader:               reader,
		Writer:               writer,
		BatchSize:            1000000,
		AllowEntityErrors:    false,
		AllowReferenceErrors: false,
		InterpolateStopTimes: false,
		CreateMissingShapes:  false,
		NormalizeServiceIDs:  false,
	}
	// Result
	result := NewCopyResult()
	copier.result = result
	copier.ErrorHandler = result
	// Default Markers
	copier.Marker = newYesMarker()
	// Default EntityMap
	copier.EntityMap = gotransit.NewEntityMap()
	// Default filters
	copier.filters = []gotransit.EntityFilter{}
	// Geom Cache
	copier.geomCache = newGeomCache()
	copier.stopPatterns = map[string]int{}
	copier.stopPatternShapeIDs = map[int]string{}
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

// Check if the entity is marked for copying.
func (copier *Copier) isMarked(ent gotransit.Entity) bool {
	return copier.Marker.IsMarked(ent.Filename(), ent.EntityID())
}

// CopyEntity performs validation and saves errors and warnings, returns new EntityID if written, otherwise an entity error or write error.
// An entity error means the entity was not not written because it had an error or was filtered out; not fatal.
// A write error should be considered fatal and should stop any further write attempts.
// Any errors and warnings are added to the CopyResult.
func (copier *Copier) CopyEntity(ent gotransit.Entity) (string, error, error) {
	efn := ent.Filename()
	eid := ent.EntityID()
	sid := ent.EntityID()
	if !copier.isMarked(ent) {
		copier.result.SkipEntityMarkedCount[efn]++
		return "", errors.New("skipped by marker"), nil
	}
	// Check the entity against filters.
	for _, ef := range copier.filters {
		if err := ef.Filter(ent, copier.EntityMap); err != nil {
			log.Debug("%s '%s' skipped by filter: %s", efn, eid, err)
			copier.result.SkipEntityFilterCount[efn]++
			return "", errors.New("skipped by filter"), nil
		}
	}
	// Check the entity for errors.
	valid := true
	errs := ent.Errors()
	// Check the entity for reference errors.
	referr := ent.UpdateKeys(copier.EntityMap)
	if referr != nil {
		errs = append(errs, referr)
	}
	// Check for duplicate entities.
	if _, ok := copier.EntityMap.Get(efn, sid); ok && len(sid) > 0 {
		errs = append(errs, causes.NewDuplicateIDError(sid))
	}
	// Check error tolerance flags
	if len(errs) > 0 {
		if copier.AllowEntityErrors {
			log.Debug("%s '%s' has errors, allowing: %s", efn, eid, errs)
		} else {
			log.Debug("%s '%s' has errors, skipping: %s", efn, eid, errs)
			copier.result.SkipEntityErrorCount[efn]++
			valid = false
		}
	} else if referr != nil {
		if copier.AllowReferenceErrors {
			log.Debug("%s '%s' failed to update keys, allowing: %s", efn, eid, referr)
		} else {
			log.Debug("%s '%s' failed to update keys, skipping: %s", efn, eid, referr)
			copier.result.SkipEntityReferenceCount[efn]++
			valid = false
		}
	}
	// Error handler
	copier.ErrorHandler.HandleEntityErrors(ent, errs, ent.Warnings())
	// Continue?
	if !valid && len(errs) > 0 {
		return "", errs[0], nil
	} else if !valid {
		return "", errors.New("???"), nil
	}
	// OK, Save
	eid, err := copier.Writer.AddEntity(ent)
	if err != nil {
		log.Error("Critical error: failed to write %s '%s': %s entity dump: %#v", efn, eid, err, ent)
		return "", err, err
	}
	log.Debug("%s '%s': saved -> %s", efn, sid, eid)
	copier.EntityMap.SetEntity(ent, sid, eid)
	copier.result.EntityCount[efn]++
	return eid, nil, nil
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS Entities from the Reader to the Writer, returning the summary as a CopyResult.
func (copier *Copier) Copy() *CopyResult {
	// Handle source errors and warnings
	sourceErrors := map[string][]error{}
	for _, err := range copier.Reader.ValidateStructure() {
		if v, ok := err.(errorWithContext); ok {
			fn := v.Context().Filename
			sourceErrors[fn] = append(sourceErrors[fn], err)
		}
	}
	for fn, errs := range sourceErrors {
		copier.ErrorHandler.HandleSourceErrors(fn, errs, nil)
	}
	// Note that order is important!!
	fns := []func() error{
		copier.copyAgencies,
		copier.copyRoutes,
		copier.copyPathwaysStopsAndFares,
		copier.copyCalendars,
		copier.copyShapes,
		copier.copyTripsAndStopTimes,
		copier.copyFrequencies,
		copier.copyTransfers,
		copier.copyFeedInfos,
	}
	for i := range fns {
		if err := fns[i](); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}
	for _, ext := range copier.extensions {
		if err := ext.Copy(copier); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}
	return copier.result
}

/////////////////////////////////////////
////////// Entity Copy Methods //////////
/////////////////////////////////////////

// copyAgencies writes agencies
func (copier *Copier) copyAgencies() error {
	firstTimezone := ""
	for e := range copier.Reader.Agencies() {
		// Check for Timezone consistency - add to feed errors
		if len(firstTimezone) == 0 {
			firstTimezone = e.AgencyTimezone
		} else if e.AgencyTimezone != firstTimezone {
			e.AddWarning(causes.NewInconsistentTimezoneError(e.AgencyTimezone))
		}
		// Check for conditionally required AgencyID - add to feed errors
		if len(e.AgencyID) == 0 && copier.agencyCount > 1 {
			e.AddWarning(causes.NewConditionallyRequiredFieldError("agency_id"))
		}
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.Agency{})
	return nil
}

// copyStopsAndFares writes stops and their associated fare rules
func (copier *Copier) copyPathwaysStopsAndFares() error {
	// Levels
	for e := range copier.Reader.Levels() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}

	// Stop bookkeeping
	parents := map[string]int{}
	farezones := map[string]string{}
	// Copy fn
	copyStop := func(e gotransit.Stop) error {
		sid := e.EntityID()
		// FareID
		fzid := e.ZoneID
		// Add stop, update farezones and geom cache
		// Need to keep track of parent type even if filtered out or merged
		// Actual relationship errors will be caught during UpdateKeys
		parents[sid] = e.LocationType
		// Confirm the parent station location_type != 0
		if len(e.ParentStation.Key) == 0 {
			// ok
		} else if pstype, ok := parents[e.ParentStation.Key]; !ok {
			// ParentStation not found - check during UpdateKeys
		} else if e.LocationType == 4 {
			// Boarding areas may only link to type = 0
			if pstype != 0 {
				e.AddError(causes.NewInvalidParentStationError(e.ParentStation.Key))
			}
		} else if pstype != 1 {
			// ParentStation wrong type
			e.AddError(causes.NewInvalidParentStationError(e.ParentStation.Key))
		}
		// Actually copy
		if _, ok, err := copier.CopyEntity(&e); err != nil {
			return err
		} else if ok == nil {
			// Success writing entity
			farezones[fzid] = e.ZoneID // ZoneID may be modified by CopyEntity
			copier.geomCache.AddStop(sid, e)
		}
		return nil
	}
	// Stop copying
	// First pass for stations
	for e := range copier.Reader.Stops() {
		if e.LocationType == 1 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	// Second pass for platforms, exits, and generic nodes
	for e := range copier.Reader.Stops() {
		if e.LocationType == 0 || e.LocationType == 2 || e.LocationType == 3 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	// Third pass for boarding areas
	for e := range copier.Reader.Stops() {
		if e.LocationType == 4 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	copier.logCount(&gotransit.Stop{})

	// Pathways
	for e := range copier.Reader.Pathways() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.Pathway{})

	// FareAttributes
	for e := range copier.Reader.FareAttributes() {
		// Set default agency
		if len(e.AgencyID.Key) == 0 {
			e.AgencyID.Key = copier.DefaultAgencyID
			e.AgencyID.Valid = true
			if copier.agencyCount > 1 {
				e.AddError(causes.NewConditionallyRequiredFieldError("agency_id"))
			}
		}
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.FareAttribute{})

	// FareRules
	for e := range copier.Reader.FareRules() {
		// Explicitly check if the FareID is Marked
		if !copier.isMarked(&gotransit.FareAttribute{FareID: e.FareID}) {
			continue
		}
		// Add reference errors if we didn't write a stop with this zone.
		if v, ok := farezones[e.OriginID]; ok {
			e.OriginID = v
		} else if len(e.OriginID) > 0 {
			e.AddError(causes.NewInvalidFarezoneError("origin_id", e.OriginID))
		}
		if v, ok := farezones[e.DestinationID]; ok {
			e.DestinationID = v
		} else if len(e.DestinationID) > 0 {
			e.AddError(causes.NewInvalidFarezoneError("destination_id", e.DestinationID))
		}
		if v, ok := farezones[e.ContainsID]; ok {
			e.ContainsID = v
		} else if len(e.ContainsID) > 0 && !ok {
			e.AddError(causes.NewInvalidFarezoneError("contains_id", e.ContainsID))
		}
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.FareRule{})
	return nil
}

// copyRoutes writes routes
func (copier *Copier) copyRoutes() error {
	for e := range copier.Reader.Routes() {
		// Set default agencyID
		if len(e.AgencyID) == 0 {
			e.AgencyID = copier.DefaultAgencyID
			if copier.agencyCount == 1 {
				e.AddWarning(causes.NewConditionallyRequiredFieldError("agency_id"))
			} else {
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
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.Route{})
	return nil
}

// copyCalendars copies Calendars and CalendarDates
func (copier *Copier) copyCalendars() error {
	// Calendars
	for e := range copier.Reader.Calendars() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	// Create additional Calendars
	if copier.NormalizeServiceIDs {
		if err := copier.createMissingCalendars(); err != nil {
			return err
		}
	}
	// Keep track of duplicate CalendarDates
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
		// Check for duplicates (service_id,date)
		key := calkey{
			ServiceID: e.ServiceID,
			Date:      e.Date.Format("20060102"),
		}
		if _, ok := dups[key]; ok {
			e.AddError(causes.NewDuplicateIDError(e.EntityID()))
		}
		dups[key]++
		// Allow unchecked/invalid ServiceID references.
		if !copier.NormalizeServiceIDs {
			if _, ok := copier.EntityMap.Get("calendar.txt", e.ServiceID); !ok {
				copier.EntityMap.Set("calendar.txt", e.ServiceID, e.ServiceID)
			}
		}
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.Calendar{})
	copier.logCount(&gotransit.CalendarDate{})
	return nil
}

// copyFeedInfos writes FeedInfos
func (copier *Copier) copyFeedInfos() error {
	for e := range copier.Reader.FeedInfos() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.FeedInfo{})
	return nil
}

// copyTransfers writes Transfers
func (copier *Copier) copyTransfers() error {
	for e := range copier.Reader.Transfers() {
		// Check if Transfer stops are marked
		if copier.isMarked(&gotransit.Stop{StopID: e.FromStopID}) && copier.isMarked(&gotransit.Stop{StopID: e.ToStopID}) {
			if _, _, err := copier.CopyEntity(&e); err != nil {
				return err
			}
		} else {
			copier.result.SkipEntityMarkedCount["transfers.txt"]++
		}
	}
	copier.logCount(&gotransit.Transfer{})
	return nil
}

// copyShapes writes Shapes
func (copier *Copier) copyShapes() error {
	for e := range copier.Reader.Shapes() {
		sid := e.EntityID()
		if _, ok, err := copier.CopyEntity(&e); err != nil {
			return err
		} else if ok == nil {
			copier.geomCache.AddShape(sid, e)
		}
	}
	copier.logCount(&gotransit.Shape{})
	return nil
}

// copyFrequencies writes Frequencies
func (copier *Copier) copyFrequencies() error {
	for e := range copier.Reader.Frequencies() {
		// Check if Trip is marked
		if copier.isMarked(&gotransit.Trip{TripID: e.TripID}) {
			if _, _, err := copier.CopyEntity(&e); err != nil {
				return err
			}
		} else {
			copier.result.SkipEntityMarkedCount["frequencies.txt"]++
		}
	}
	copier.logCount(&gotransit.Frequency{})
	return nil
}

// copyTripsAndStopTimes writes Trips and StopTimes
func (copier *Copier) copyTripsAndStopTimes() error {
	// Cache all trips in memory
	// If this becomes an issue, we could do a pass through trips.txt for each stop_times chunk
	alltripids := map[string]int{}
	trips := map[string]gotransit.Trip{}
	for trip := range copier.Reader.Trips() {
		eid := trip.EntityID()
		alltripids[eid]++
		// Skip unmarked trips to save work
		if !copier.isMarked(&trip) {
			copier.result.SkipEntityMarkedCount["trips.txt"]++
			continue
		}
		// We need to check for duplicate ID errors here because they're put into a map
		if _, ok := trips[eid]; ok {
			copier.ErrorHandler.HandleEntityErrors(&trip, []error{causes.NewDuplicateIDError(eid)}, nil)
			continue
		}
		trips[eid] = trip
	}
	batch := []gotransit.StopTime{}
	for stoptimes := range copier.Reader.StopTimesByTripID() {
		// Error handling for trips without stop_times is after this block
		if len(stoptimes) == 0 {
			log.Debug("Warning: StopTimesByTripID produced zero StopTimes")
			continue
		}
		// Does this trip exist?
		tripid := stoptimes[0].TripID
		if _, ok := alltripids[tripid]; !ok {
			log.Debug("stop_times referred to unknown trip: %s", tripid)
			copier.ErrorHandler.HandleEntityErrors(&stoptimes[0], []error{causes.NewInvalidReferenceError("trip_id", tripid)}, nil)
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += len(stoptimes)
			continue
		}
		// Is this trip marked?
		trip, ok := trips[tripid]
		if !ok { // trip_id exists but is not marked
			copier.result.SkipEntityMarkedCount["stop_times.txt"] += len(stoptimes)
			continue
		}
		// Marks trip as associated with at least 1 stop_time
		delete(trips, tripid)
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
		if trip.ShapeID.IsZero() && copier.CreateMissingShapes {
			// Note: if the trip has errors, may result in unused shapes!
			if shapeid, ok := copier.stopPatternShapeIDs[trip.StopPatternID]; ok {
				trip.ShapeID.Key = shapeid
				trip.ShapeID.Valid = true
			} else {
				if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID, time.Now().Unix()), stoptimes); err != nil {
					log.Error("Error: failed to create shape for trip '%s': %s", trip.EntityID(), err)
					// TODO: Is this an error or just a general info for the output? Causing SNCF to fail.
					trip.AddError(err)
				} else {
					// Set ShapeID
					copier.stopPatternShapeIDs[trip.StopPatternID] = shapeid
					trip.ShapeID.Key = shapeid
					trip.ShapeID.Valid = true
				}
			}
		}
		// Check StopTime GROUP errors; log errors with trip; can block trip
		// Example errors: less than 2 stop_times, non-increasing sequences and times, etc.
		sterrs := gotransit.ValidateStopTimes(stoptimes)
		for _, err := range sterrs {
			trip.AddError(err)
		}
		// Interpolate StopTimes if necessary - only if no other errors; log errors with trip
		if len(sterrs) == 0 && copier.InterpolateStopTimes {
			if stoptimes2, err := copier.geomCache.InterpolateStopTimes(trip, stoptimes); err != nil {
				// stwarns = append(stwarns, err)
				trip.AddWarning(err)
			} else {
				stoptimes = stoptimes2
			}
		}
		// Save trip
		if _, ok, err := copier.CopyEntity(&trip); err != nil {
			// Serious failure; return
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += len(stoptimes)
			return err
		} else if ok != nil {
			// Entity error; skip
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += len(stoptimes)
			continue
		}
		// Check individual StopTime errors
		// Similar to CopyEntity except that writing will be done in batch
		// Note: StopTimes are not currently checked by EntityFilters.
		valid := true
		efn := "stop_times.txt"
		istc := 0
		for i := 0; i < len(stoptimes); i++ {
			errs := stoptimes[i].Errors()
			referr := stoptimes[i].UpdateKeys(copier.EntityMap)
			if referr != nil {
				errs = append(errs, referr)
			}
			// Check error tolerance flags
			if len(errs) > 0 {
				if copier.AllowEntityErrors {
					log.Debug("%s '%s' has errors, allowing: %s", efn, tripid, errs)
				} else {
					log.Debug("%s '%s' has errors, skipping: %s", efn, tripid, errs)
					copier.result.SkipEntityErrorCount[efn]++
					valid = false
				}
			} else if referr != nil {
				if copier.AllowReferenceErrors {
					log.Debug("%s '%s' failed to update keys, allowing: %s", efn, tripid, referr)
				} else {
					log.Debug("%s '%s' failed to update keys, skipping: %s", efn, tripid, referr)
					copier.result.SkipEntityReferenceCount[efn]++
					valid = false
				}
			}
			// Error handler
			copier.ErrorHandler.HandleEntityErrors(&stoptimes[i], errs, stoptimes[i].Warnings())
			// Count interpolated STs for debugging/reporting
			if stoptimes[i].Interpolated > 0 {
				istc++
			}
		}
		if !valid {
			continue
		}
		copier.result.InterpolatedStopTimeCount += istc
		// OK, Everything is good to go.
		batch = append(batch, stoptimes...)
		// Write in batches
		if len(batch) >= copier.BatchSize {
			bst := []gotransit.Entity{}
			// note: "range" re-uses the same pointer.
			for i := 0; i < len(batch); i++ {
				bst = append(bst, &batch[i])
			}
			if err := copier.Writer.AddEntities(bst); err != nil {
				// Serious error, fail
				return err
			}
			log.Info("Saved %d stop_times", len(batch))
			copier.result.EntityCount["stop_times.txt"] += len(batch)
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
			// Serious error, fail
			return err
		}
		log.Info("Saved %d stop_times", len(batch))
		copier.result.EntityCount["stop_times.txt"] += len(batch)
	}
	// Add any Trips that were not visited/did not have StopTimes
	for _, trip := range trips {
		trip.AddError(causes.NewEmptyTripError(0))
		if _, _, err := copier.CopyEntity(&trip); err != nil {
			return err
		}
	}
	copier.logCount(&gotransit.Trip{})
	return nil
}

////////////////////////////////////////////
////////// Entity Support Methods //////////
////////////////////////////////////////////

func (copier *Copier) logCount(ent gotransit.Entity) {
	out := []string{}
	fn := ent.Filename()
	fnr := strings.ReplaceAll(fn, ".txt", "")
	saved := copier.result.EntityCount[fn]
	out = append(out, fmt.Sprintf("Saved %d %s", saved, fnr))
	if a, ok := copier.result.GeneratedCount[fn]; ok {
		out = append(out, fmt.Sprintf("generated %d", a))
	}
	if a, ok := copier.result.SkipEntityMarkedCount[fn]; ok {
		out = append(out, fmt.Sprintf("skipped %d as unmarked", a))
	}
	if a, ok := copier.result.SkipEntityFilterCount[fn]; ok {
		out = append(out, fmt.Sprintf("skipped %d by filter", a))
	}
	if a, ok := copier.result.SkipEntityErrorCount[fn]; ok {
		out = append(out, fmt.Sprintf("skipped %d with entity errors", a))
	}
	if a, ok := copier.result.SkipEntityReferenceCount[fn]; ok {
		out = append(out, fmt.Sprintf("skipped %d with reference errors", a))
	}
	if saved == 0 && len(out) == 1 {
		return
	}
	outs := strings.Join(out, "; ")
	log.Info(outs)
}

func (copier *Copier) createMissingShape(shapeID string, stoptimes []gotransit.StopTime) (string, error) {
	stopids := []string{}
	for _, st := range stoptimes {
		stopids = append(stopids, st.StopID)
	}
	shape, err := copier.geomCache.MakeShape(stopids...)
	if err != nil {
		return "", err
	}
	shape.ShapeID = shapeID
	if _, ok, err := copier.CopyEntity(&shape); err != nil {
		return "", err
	} else if ok == nil {
		copier.result.GeneratedCount["shapes.txt"]++
	}
	return shape.ShapeID, nil
}

// createMissingCalendars to fully normalize ServiceIDs
func (copier *Copier) createMissingCalendars() error {
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
		if _, ok := copier.GetEntity(&cal); ok {
			continue
		}
		// Do we already know this ServiceID?
		if c, ok := missing[cal.ServiceID]; ok {
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
		log.Debug("Create missing cal: %#v\n", e)
		if _, ok, err := copier.CopyEntity(&e); err != nil {
			return err
		} else if ok == nil {
			copier.result.GeneratedCount["calendar.txt"]++
		}
	}
	return nil
}
