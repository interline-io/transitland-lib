package copier

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// ErrorHandler is called on each source file and entity; errors can be nil
type ErrorHandler interface {
	HandleEntityErrors(tl.Entity, []error, []error)
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

// Options defines the settable options for a Copier.
type Options struct {
	// Batch size
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
	// Simplify Calendars that use mostly CalendarDates
	SimplifyCalendars bool
	// Convert extended route types to primitives
	UseBasicRouteTypes bool
	// Default AgencyID
	DefaultAgencyID string
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
}

// Copier copies from Reader to Writer
type Copier struct {
	// Default options
	Options
	// Reader and writer
	Reader tl.Reader
	Writer tl.Writer
	// Entity selection strategy
	Marker Marker
	// Error handler, called for each entity
	ErrorHandler ErrorHandler
	// book keeping
	agencyCount  int
	extensions   []copyableExtension // interface
	filters      []tl.EntityFilter   // interface
	geomCache    *geomCache
	result       *CopyResult
	duplicateMap *tl.EntityMap
	*tl.EntityMap
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader tl.Reader, writer tl.Writer, opts Options) Copier {
	copier := Copier{}
	copier.Options = opts
	copier.Reader = reader
	copier.Writer = writer
	// Result
	result := NewCopyResult()
	copier.result = result
	copier.ErrorHandler = result
	// Default Markers
	copier.Marker = newYesMarker()
	// Default EntityMap
	copier.EntityMap = tl.NewEntityMap()
	// Check for duplicate IDs
	copier.duplicateMap = tl.NewEntityMap()
	// Default filters
	copier.filters = []tl.EntityFilter{}
	// Geom Cache
	copier.geomCache = newGeomCache()
	// Set the default BatchSize
	if copier.BatchSize == 0 {
		copier.BatchSize = 1000000 // TODO: 1_000_000 requires Go 1.13
	}
	// Set the DefaultAgencyID from the Reader
	if copier.DefaultAgencyID == "" {
		for e := range copier.Reader.Agencies() {
			copier.DefaultAgencyID = e.AgencyID
			copier.agencyCount++
		}
	}
	return copier
}

// AddExtension adds an Extension to the copy process.
func (copier *Copier) AddExtension(e ext.Extension) error {
	extc, ok := e.(copyableExtension)
	if !ok {
		return fmt.Errorf("Extension does not provide Copy method")
	}
	copier.extensions = append(copier.extensions, extc)
	return nil
}

// AddEntityFilter adds an EntityFilter to the copy process.
func (copier *Copier) AddEntityFilter(ef tl.EntityFilter) error {
	copier.filters = append(copier.filters, ef)
	return nil
}

////////////////////////////////////
////////// Helper Methods //////////
////////////////////////////////////

// Check if the entity is marked for copying.
func (copier *Copier) isMarked(ent tl.Entity) bool {
	return copier.Marker.IsMarked(ent.Filename(), ent.EntityID())
}

// CopyEntity performs validation and saves errors and warnings, returns new EntityID if written, otherwise an entity error or write error.
// An entity error means the entity was not not written because it had an error or was filtered out; not fatal.
// A write error should be considered fatal and should stop any further write attempts.
// Any errors and warnings are added to the CopyResult.
func (copier *Copier) CopyEntity(ent tl.Entity) (string, error, error) {
	efn := ent.Filename()
	sid := ent.EntityID()
	if err := copier.checkEntity(ent); err != nil {
		return "", err, nil
	}
	// OK, Save
	eid, err := copier.Writer.AddEntity(ent)
	if err != nil {
		log.Error("Critical error: failed to write %s '%s': %s entity dump: %#v", efn, sid, err, ent)
		return "", err, err
	}
	log.Debug("%s '%s': saved -> %s", efn, sid, eid)
	copier.EntityMap.Set(efn, sid, eid)
	copier.result.EntityCount[efn]++
	return eid, nil, nil
}

// writeBatch does housekeeping for writing multiple entities.
func (copier *Copier) writeBatch(ents []tl.Entity) error {
	if len(ents) == 0 {
		return nil
	}
	efn := ents[0].Filename()
	sids := []string{}
	for _, ent := range ents {
		sids = append(sids, ent.EntityID())
	}
	// OK, Save
	eids, err := copier.Writer.AddEntities(ents)
	if err != nil {
		log.Error("Critical error: failed to write %d entities for %s", len(ents), efn)
		return err
	}
	for i, eid := range eids {
		sid := sids[i]
		log.Debug("%s '%s': saved -> %s", efn, sid, eid)
		copier.EntityMap.Set(efn, sid, eid)
	}
	copier.result.EntityCount[efn] += len(ents)
	// Return an emtpy slice and no error
	return nil
}

func (copier *Copier) checkBatch(ents []tl.Entity, ent tl.Entity) ([]tl.Entity, error) {
	if err := copier.checkEntity(ent); err != nil {
		return ents, nil
	}
	ents = append(ents, ent)
	if len(ents) < copier.BatchSize {
		return ents, nil
	}
	return nil, copier.writeBatch(ents)
}

func (copier *Copier) checkEntity(ent tl.Entity) error {
	efn := ent.Filename()
	if !copier.isMarked(ent) {
		copier.result.SkipEntityMarkedCount[efn]++
		return errors.New("skipped by marker")
	}
	// Check the entity against filters.
	sid := ent.EntityID() // source ID
	for _, ef := range copier.filters {
		if err := ef.Filter(ent, copier.EntityMap); err != nil {
			log.Debug("%s '%s' skipped by filter: %s", efn, sid, err)
			copier.result.SkipEntityFilterCount[efn]++
			return errors.New("skipped by filter")
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
	eid := ent.EntityID()
	if _, ok := copier.duplicateMap.Get(efn, eid); ok && len(eid) > 0 {
		errs = append(errs, causes.NewDuplicateIDError(eid))
	} else {
		copier.duplicateMap.Set(efn, eid, eid)
	}
	// Check error tolerance flags
	if len(errs) > 0 {
		if copier.AllowEntityErrors {
			log.Debug("%s '%s' has errors, allowing: %s", efn, sid, errs)
		} else {
			log.Debug("%s '%s' has errors, skipping: %s", efn, sid, errs)
			copier.result.SkipEntityErrorCount[efn]++
			valid = false
		}
	} else if referr != nil {
		if copier.AllowReferenceErrors {
			log.Debug("%s '%s' failed to update keys, allowing: %s", efn, sid, referr)
		} else {
			log.Debug("%s '%s' failed to update keys, skipping: %s", efn, sid, referr)
			copier.result.SkipEntityReferenceCount[efn]++
			valid = false
		}
	}
	// Error handler
	copier.ErrorHandler.HandleEntityErrors(ent, errs, ent.Warnings())
	// Continue?
	if !valid && len(errs) > 0 {
		return errs[0]
	} else if !valid {
		return errors.New("???")
	}
	return nil
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS entities from the Reader to the Writer, returning the summary as a CopyResult.
func (copier *Copier) Copy() *CopyResult {
	// Handle source errors and warnings
	sourceErrors := map[string][]error{}
	for _, err := range copier.Reader.ValidateStructure() {
		fn := ""
		if v, ok := err.(errorWithContext); ok {
			fn = v.Context().Filename
		}
		sourceErrors[fn] = append(sourceErrors[fn], err)
	}
	for fn, errs := range sourceErrors {
		copier.ErrorHandler.HandleSourceErrors(fn, errs, nil)
	}
	// Note that order is important!!
	fns := []func() error{
		copier.copyAgencies,
		copier.copyRoutes,
		copier.copyLevels,
		copier.copyStops,
		copier.copyPathways,
		copier.copyFares,
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
	for _, e := range copier.extensions {
		if err := e.Copy(copier); err != nil {
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
		// Agencies are not currently safe for batch writing because agency_id is conditionally required.
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Agency{})
	return nil
}

// copyLevels writes levels.
func (copier *Copier) copyLevels() error {
	// Levels
	bt := []tl.Entity{}
	for e := range copier.Reader.Levels() {
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	return nil
}

func (copier *Copier) copyStops() error {
	// Copy fn
	bt := []tl.Entity{}
	parents := map[string]int{}
	farezones := map[string]string{}
	copyStop := func(ent tl.Stop) error {
		// Add stop, update farezones and geom cache
		// Need to keep track of parent type even if filtered out or merged
		// Actual relationship errors will be caught during UpdateKeys
		sid := ent.EntityID()
		parents[sid] = ent.LocationType
		farezones[sid] = ent.ZoneID
		copier.geomCache.AddStop(sid, ent)
		// Confirm the parent station location_type != 0
		if len(ent.ParentStation.Key) == 0 {
			// ok
		} else if pstype, ok := parents[ent.ParentStation.Key]; !ok {
			// ParentStation not found - check during UpdateKeys
		} else if ent.LocationType == 4 {
			// Boarding areas may only link to type = 0
			if pstype != 0 {
				ent.AddError(causes.NewInvalidParentStationError(ent.ParentStation.Key))
			}
		} else if pstype != 1 {
			// ParentStation wrong type
			ent.AddError(causes.NewInvalidParentStationError(ent.ParentStation.Key))
		}
		e := ent
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
		return nil
	}

	// First pass for stations
	for e := range copier.Reader.Stops() {
		if e.LocationType == 1 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}

	// Second pass for platforms, exits, and generic nodes
	bt = nil
	for e := range copier.Reader.Stops() {
		if e.LocationType == 0 || e.LocationType == 2 || e.LocationType == 3 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}

	// Third pass for boarding areas
	bt = nil
	for e := range copier.Reader.Stops() {
		if e.LocationType == 4 {
			if err := copyStop(e); err != nil {
				return err
			}
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}

	// Check farezones
	for sid, zone := range farezones {
		if _, ok := copier.EntityMap.Get("stops.txt", sid); ok {
			copier.EntityMap.Set("zone_ids", zone, zone)
		}
	}
	copier.logCount(&tl.Stop{})
	return nil
}

func (copier *Copier) copyFares() error {
	// FareAttributes
	bt := []tl.Entity{}
	for e := range copier.Reader.FareAttributes() {
		// Set default agency
		if len(e.AgencyID.Key) == 0 {
			e.AgencyID.Key = copier.DefaultAgencyID
			e.AgencyID.Valid = true
			if copier.agencyCount > 1 {
				e.AddError(causes.NewConditionallyRequiredFieldError("agency_id"))
			}
		}
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.FareAttribute{})

	// FareRules
	bt = nil
	for e := range copier.Reader.FareRules() {
		// Explicitly check if the FareID is Marked
		if !copier.isMarked(&tl.FareAttribute{FareID: e.FareID}) {
			continue
		}
		// TODO: Move this into UpdateKeys()
		// Add reference errors if we didn't write a stop with this zone.
		if v, ok := copier.EntityMap.Get("zone_ids", e.OriginID); ok {
			e.OriginID = v
		} else if len(e.OriginID) > 0 {
			e.AddError(causes.NewInvalidFarezoneError("origin_id", e.OriginID))
		}
		if v, ok := copier.EntityMap.Get("zone_ids", e.DestinationID); ok {
			e.DestinationID = v
		} else if len(e.DestinationID) > 0 {
			e.AddError(causes.NewInvalidFarezoneError("destination_id", e.DestinationID))
		}
		if v, ok := copier.EntityMap.Get("zone_ids", e.ContainsID); ok {
			e.ContainsID = v
		} else if len(e.ContainsID) > 0 && !ok {
			e.AddError(causes.NewInvalidFarezoneError("contains_id", e.ContainsID))
		}
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.FareRule{})
	return nil
}

func (copier *Copier) copyPathways() error {
	// Pathways
	bt := []tl.Entity{}
	for e := range copier.Reader.Pathways() {
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.Pathway{})
	return nil
}

// copyRoutes writes routes
func (copier *Copier) copyRoutes() error {
	bt := []tl.Entity{}
	for e := range copier.Reader.Routes() {
		var err error
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
			if rt, ok := enum.GetBasicRouteType(e.RouteType); ok {
				e.RouteType = rt.Code
			} else {
				e.AddError(causes.NewInvalidFieldError("route_type", strconv.Itoa(e.RouteType), fmt.Errorf("cannot convert route_type %d to basic route type", e.RouteType)))
			}
		}
		e := e
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.Route{})
	return nil
}

// copyCalendars
func (copier *Copier) copyCalendars() error {
	// Get the basic Calendars as Services
	// This is essentially the same as NewServicesFromReader,
	// but we need additional error checking and accounting.
	svcs := map[string]*tl.Service{}
	for ent := range copier.Reader.Calendars() {
		if !copier.isMarked(&tl.Calendar{}) {
			continue
		}
		_, ok := svcs[ent.ServiceID]
		if ok {
			copier.ErrorHandler.HandleEntityErrors(&ent, []error{causes.NewDuplicateIDError(ent.ServiceID)}, nil)
			continue
		}
		svcs[ent.ServiceID] = tl.NewService(ent)
	}

	// Add the CalendarDates to Services
	for ent := range copier.Reader.CalendarDates() {
		cal := tl.Calendar{
			ServiceID: ent.ServiceID,
			Generated: true,
		}
		if !copier.isMarked(&cal) {
			continue
		}
		svc, ok := svcs[ent.ServiceID]
		if !ok {
			svc = tl.NewService(cal)
			svcs[ent.ServiceID] = svc
		}
		if _, ok := svc.Exception(ent.Date); ok {
			copier.ErrorHandler.HandleEntityErrors(&ent, []error{causes.NewDuplicateIDError(ent.ServiceID)}, nil)
			continue
		}
		svc.AddCalendarDate(ent)
	}

	// Write Calendars
	var err error
	bt := []tl.Entity{}
	for _, svc := range svcs {
		// Simplify generated and non-generated calendars
		if copier.SimplifyCalendars {
			if s, err := svc.Simplify(); err == nil {
				svc = s
				svcs[svc.ServiceID] = svc
			}
		}
		// Generated calendars may need their service period set...
		if svc.Generated && (svc.StartDate.IsZero() || svc.EndDate.IsZero()) {
			svc.StartDate, svc.EndDate = svc.ServicePeriod()
		}
		// Write out generated calendars IF normalizing OR simplifying
		if svc.Generated {
			if copier.NormalizeServiceIDs || copier.SimplifyCalendars {
				// Add entity
			} else {
				// Don't add entity, but update emap, so dates don't error
				copier.SetEntity(&svc.Calendar, svc.ServiceID, svc.ServiceID)
				continue
			}
		}
		if bt, err = copier.checkBatch(bt, &svc.Calendar); err != nil {
			return err
		}
		if svc.Generated {
			copier.result.GeneratedCount["calendar.txt"]++
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.Calendar{})

	// Write CalendarDates
	bt = nil
	for _, svc := range svcs {
		for _, cd := range svc.CalendarDates() {
			if bt, err = copier.checkBatch(bt, &cd); err != nil {
				return err
			}
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.CalendarDate{})
	return nil
}

// copyFeedInfos writes FeedInfos
func (copier *Copier) copyFeedInfos() error {
	bt := []tl.Entity{}
	for e := range copier.Reader.FeedInfos() {
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.FeedInfo{})
	return nil
}

// copyTransfers writes Transfers
func (copier *Copier) copyTransfers() error {
	bt := []tl.Entity{}
	for e := range copier.Reader.Transfers() {
		// Check if Transfer stops are marked
		if !copier.isMarked(&tl.Stop{StopID: e.FromStopID}) && copier.isMarked(&tl.Stop{StopID: e.ToStopID}) {
			copier.result.SkipEntityMarkedCount["transfers.txt"]++
			continue
		}
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.Transfer{})
	return nil
}

// copyShapes writes Shapes
func (copier *Copier) copyShapes() error {
	// Not safe for batch copy (currently)
	for e := range copier.Reader.Shapes() {
		sid := e.EntityID()
		if _, ok, err := copier.CopyEntity(&e); err != nil {
			return err
		} else if ok == nil {
			copier.geomCache.AddShape(sid, e)
		}
	}
	copier.logCount(&tl.Shape{})
	return nil
}

// copyFrequencies writes Frequencies
func (copier *Copier) copyFrequencies() error {
	bt := []tl.Entity{}
	for e := range copier.Reader.Frequencies() {
		// Check if Trip is marked
		if !copier.isMarked(&tl.Trip{TripID: e.TripID}) {
			copier.result.SkipEntityMarkedCount["frequencies.txt"]++
			continue
		}
		e := e
		var err error
		if bt, err = copier.checkBatch(bt, &e); err != nil {
			return err
		}
	}
	if err := copier.writeBatch(bt); err != nil {
		return err
	}
	copier.logCount(&tl.Frequency{})
	return nil
}

type patInfo struct {
	key          string
	firstArrival int
}

// copyTripsAndStopTimes writes Trips and StopTimes
func (copier *Copier) copyTripsAndStopTimes() error {
	// Cache all trips in memory
	// If this becomes an issue, we could do a pass through trips.txt for each stop_times chunk
	alltripids := map[string]int{}
	trips := map[string]tl.Trip{}
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

	// Process each set of Trip/StopTimes
	stopPatterns := map[string]int{}
	stopPatternShapeIDs := map[int]string{}
	journeyPatterns := map[string]patInfo{}
	batchCount := 0
	tripbt := []tl.Entity{}
	stbt := []tl.StopTime{}
	writeBatch := func() error {
		// Write Trips
		if err := copier.writeBatch(tripbt); err != nil {
			return err
		}
		log.Info("Saved %d trips", len(tripbt))
		// Perform StopTime validation
		stbt2 := []tl.Entity{}
		for i := range stbt {
			if err := copier.checkEntity(&stbt[i]); err == nil {
				stbt2 = append(stbt2, &stbt[i])
				if stbt[i].Interpolated.Int32 > 0 {
					copier.result.InterpolatedStopTimeCount++
				}
			}
		}
		if err := copier.writeBatch(stbt2); err != nil {
			return err
		}
		log.Info("Saved %d stop_times", len(stbt2))
		tripbt = nil
		stbt = nil
		batchCount = 0
		return nil
	}

	for stoptimes := range copier.Reader.StopTimesByTripID() {
		// Write batch
		if batchCount+len(stoptimes) >= copier.BatchSize {
			if err := writeBatch(); err != nil {
				return err
			}
		}

		// Error handling for trips without stop_times is after this block
		if len(stoptimes) == 0 {
			continue
		}
		// Does this trip exist?
		tripid := stoptimes[0].TripID
		if _, ok := alltripids[tripid]; !ok {
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
		// Mark trip as associated with at least 1 stop_time
		delete(trips, tripid)

		// Set StopPattern
		patkey := stopPatternKey(stoptimes)
		if pat, ok := stopPatterns[patkey]; ok {
			trip.StopPatternID = pat
		} else {
			trip.StopPatternID = len(stopPatterns)
			stopPatterns[patkey] = trip.StopPatternID
		}

		// Do we need to create a shape for this trip
		if trip.ShapeID.IsZero() && copier.CreateMissingShapes {
			// Note: if the trip has errors, may result in unused shapes!
			if shapeid, ok := stopPatternShapeIDs[trip.StopPatternID]; ok {
				trip.ShapeID.Key = shapeid
				trip.ShapeID.Valid = true
			} else {
				if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID, time.Now().Unix()), stoptimes); err != nil {
					log.Error("Error: failed to create shape for trip '%s': %s", trip.EntityID(), err)
					trip.AddError(err)
				} else {
					// Set ShapeID
					stopPatternShapeIDs[trip.StopPatternID] = shapeid
					trip.ShapeID.Key = shapeid
					trip.ShapeID.Valid = true
				}
			}
		}

		// Check StopTime GROUP errors; log errors with trip; can block trip
		// Example errors: less than 2 stop_times, non-increasing sequences and times, etc.
		sterrs := tl.ValidateStopTimes(stoptimes)
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

		// Set JourneyPattern
		jkey := journeyPatternKey(trip, stoptimes)
		stlen := len(stoptimes)
		if jpat, ok := journeyPatterns[jkey]; ok {
			trip.JourneyPatternID = jpat.key
			trip.JourneyPatternOffset = stoptimes[0].ArrivalTime - jpat.firstArrival
			if copier.DeduplicateJourneyPatterns {
				stoptimes = nil
			}
		} else {
			trip.JourneyPatternID = trip.TripID
			trip.JourneyPatternOffset = 0
			journeyPatterns[jkey] = patInfo{firstArrival: stoptimes[0].ArrivalTime, key: trip.JourneyPatternID}
		}

		// Validate trip & add to batch
		if err := copier.checkEntity(&trip); err == nil {
			tripbt = append(tripbt, &trip)
		} else {
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += stlen
			continue
		}
		// Add StopTimes to batch -- final validation after writing trips
		for i := range stoptimes {
			stbt = append(stbt, stoptimes[i])
		}
		batchCount += stlen
	}
	// Add any Trips that were not visited/did not have StopTimes
	for _, trip := range trips {
		trip := trip
		trip.AddError(causes.NewEmptyTripError(0))
		if err := copier.checkEntity(&trip); err == nil {
			tripbt = append(tripbt, &trip)
		}
	}
	// Write last entities
	if err := writeBatch(); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////
////////// Entity Support Methods //////////
////////////////////////////////////////////

func (copier *Copier) logCount(ent tl.Entity) {
	out := []string{}
	fn := ent.Filename()
	fnr := strings.ReplaceAll(fn, ".txt", "")
	saved := copier.result.EntityCount[fn]
	out = append(out, fmt.Sprintf("Saved %d %s", saved, fnr))
	if a, ok := copier.result.GeneratedCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("generated %d", a))
	}
	if a, ok := copier.result.SkipEntityMarkedCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d as unmarked", a))
	}
	if a, ok := copier.result.SkipEntityFilterCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d by filter", a))
	}
	if a, ok := copier.result.SkipEntityErrorCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d with entity errors", a))
	}
	if a, ok := copier.result.SkipEntityReferenceCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d with reference errors", a))
	}
	if saved == 0 && len(out) == 1 {
		return
	}
	outs := strings.Join(out, "; ")
	log.Info(outs)
}

func (copier *Copier) createMissingShape(shapeID string, stoptimes []tl.StopTime) (string, error) {
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
