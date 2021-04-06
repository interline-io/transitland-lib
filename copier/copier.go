package copier

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// Filter is called before validation.
type Filter interface {
	Filter(tl.Entity, *tl.EntityMap) error
}

// Validator is called for each entity.
type Validator interface {
	Validate(tl.Entity) []error
}

// AfterValidator is called for each fully validated entity before writing.
type AfterValidator interface {
	AfterValidator(tl.Entity) error
}

// Extension is run after normal copying has completed.
type Extension interface {
	Copy(*Copier) error
}

// AfterCopy is called after normal copying and extensions have completed.
type AfterCopy interface {
	AfterCopy(*Copier) error
}

// ErrorHandler is called on each source file and entity; errors can be nil
type ErrorHandler interface {
	HandleEntityErrors(tl.Entity, []error, []error)
	HandleSourceErrors(string, []error, []error)
}

type errorWithContext interface {
	Context() *causes.Context
}

type canShareGeomCache interface {
	SetGeomCache(*xy.GeomCache)
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
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
	// Default error handler
	ErrorHandler ErrorHandler
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
	// Exts
	Extensions        []Extension
	Filters           []Filter
	ErrorValidators   []Validator
	WarningValidators []Validator
	AfterValidators   []AfterValidator
	AfterCopiers      []AfterCopy
	// book keeping
	agencyCount int
	geomCache   *xy.GeomCache
	result      *Result
	*tl.EntityMap
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader tl.Reader, writer tl.Writer, opts Options) Copier {
	copier := Copier{}
	copier.Options = opts
	copier.Reader = reader
	copier.Writer = writer
	// Result
	result := NewResult()
	copier.result = result
	copier.ErrorHandler = opts.ErrorHandler
	if copier.ErrorHandler == nil {
		copier.ErrorHandler = result
	}
	// Default Markers
	copier.Marker = newYesMarker()
	// Default EntityMap
	copier.EntityMap = tl.NewEntityMap()
	// Set the default BatchSize
	if copier.BatchSize == 0 {
		copier.BatchSize = 1_000_000
	}
	// Default filters
	copier.Filters = []Filter{}
	if copier.UseBasicRouteTypes {
		copier.Filters = append(copier.Filters, &BasicRouteTypeFilter{})
	}
	// Default set of validators
	copier.ErrorValidators = append(copier.ErrorValidators,
		&rules.EntityErrorCheck{},
		&rules.EntityDuplicateCheck{},
		&rules.ValidFarezoneCheck{},
		&rules.AgencyIDConditionallyRequiredCheck{},
		&rules.StopTimeSequenceCheck{},
		&rules.InconsistentTimezoneCheck{},
		&rules.ParentStationLocationTypeCheck{},
	)
	return copier
}

// AddValidator adds an additional entity validator.
func (copier *Copier) AddValidator(e Validator, level int) error {
	if level == 0 {
		copier.ErrorValidators = append(copier.ErrorValidators, e)
	} else if level == 1 {
		copier.WarningValidators = append(copier.WarningValidators, e)
	} else {
		return errors.New("unknown validation level")
	}
	return nil
}

// AddExtension adds an Extension to the copy process.
func (copier *Copier) AddExtension(ext Extension) error {
	copier.Extensions = append(copier.Extensions, ext)
	return nil
}

// AddEntityFilter adds an EntityFilter to the copy process.
func (copier *Copier) AddEntityFilter(ef tl.EntityFilter) error {
	copier.Filters = append(copier.Filters, ef)
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
// Any errors and warnings are added to the Result.
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
	copier.EntityMap.Set(efn, sid, eid)
	copier.result.EntityCount[efn]++
	return eid, nil, nil
}

// writeBatch handles writing a batch of entities, all of the same kind.
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
		// log.Debug("%s '%s': saved -> %s", efn, sid, eid)
		copier.EntityMap.Set(efn, sid, eid)
	}
	copier.result.EntityCount[efn] += len(ents)
	// Return an emtpy slice and no error
	return nil
}

// checkBatch adds an entity to the current batch and calls writeBatch if above batch size.
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

// checkEntity is the main filter and validation check.
func (copier *Copier) checkEntity(ent tl.Entity) error {
	efn := ent.Filename()
	if !copier.isMarked(ent) {
		copier.result.SkipEntityMarkedCount[efn]++
		return errors.New("skipped by marker")
	}
	// Check the entity against filters.
	sid := ent.EntityID() // source ID
	for _, ef := range copier.Filters {
		if err := ef.Filter(ent, copier.EntityMap); err != nil {
			log.Debug("%s '%s' skipped by filter: %s", efn, sid, err)
			copier.result.SkipEntityFilterCount[efn]++
			return errors.New("skipped by filter")
		}
	}
	// Run Entity Validators
	var errs []error
	var warns []error
	for _, v := range copier.ErrorValidators {
		errs = append(errs, v.Validate(ent)...)
	}
	for _, v := range copier.WarningValidators {
		warns = append(warns, v.Validate(ent)...)
	}
	// UpdateKeys is handled separately from other validators.
	// It is more like a filter than an error, since it mutates entities.
	referr := ent.UpdateKeys(copier.EntityMap)
	if referr != nil {
		errs = append(errs, referr)
	}
	// Error handler
	copier.ErrorHandler.HandleEntityErrors(ent, errs, warns)
	if referr != nil && !copier.AllowReferenceErrors {
		copier.result.SkipEntityReferenceCount[efn]++
		return referr
	}
	if len(errs) > 0 && !copier.AllowEntityErrors {
		copier.result.SkipEntityErrorCount[efn]++
		return errs[0]
	}
	// Handle after validators
	for _, v := range copier.AfterValidators {
		if err := v.AfterValidator(ent); err != nil {
			return err
		}
	}
	return nil
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS entities from the Reader to the Writer, returning the summary as a Result.
func (copier *Copier) Copy() *Result {
	copier.geomCache = xy.NewGeomCache()
	for _, e := range copier.ErrorValidators {
		if v, ok := e.(canShareGeomCache); ok {
			v.SetGeomCache(copier.geomCache)
		}
	}
	for _, e := range copier.WarningValidators {
		if v, ok := e.(canShareGeomCache); ok {
			v.SetGeomCache(copier.geomCache)
		}
	}
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
	for _, e := range copier.Extensions {
		if err := e.Copy(copier); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}
	for _, e := range copier.AfterCopiers {
		if err := e.AfterCopy(copier); err != nil {
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
	for e := range copier.Reader.Agencies() {
		// agency validation depends on other agencies; don't batch write.
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
	copyStop := func(ent tl.Stop) error {
		sid := ent.EntityID()
		copier.geomCache.AddStop(sid, ent)
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
	copier.logCount(&tl.Stop{})
	return nil
}

func (copier *Copier) copyFares() error {
	// FareAttributes
	bt := []tl.Entity{}
	for e := range copier.Reader.FareAttributes() {
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
			copier.geomCache.AddSimplifiedShape(sid, e, 0.000005)
		}
	}
	copier.logCount(&tl.Shape{})
	return nil
}

// copyFrequencies writes Frequencies
func (copier *Copier) copyFrequencies() error {
	bt := []tl.Entity{}
	for e := range copier.Reader.Frequencies() {
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

// copyCalendars
func (copier *Copier) copyCalendars() error {
	// Get Calendars as Services
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
			copier.ErrorHandler.HandleEntityErrors(&ent, []error{causes.NewDuplicateServiceExceptionError(ent.ServiceID, ent.Date)}, nil)
			continue
		}
		svc.AddCalendarDate(ent)
	}

	// Simplify and and adjust StartDate and EndDate
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
	}

	// Write Calendars
	var err error
	bt := []tl.Entity{}
	for _, svc := range svcs {
		// Skip main Calendar entity if generated and not normalizing service IDs.
		if svc.Generated && !copier.NormalizeServiceIDs && !copier.SimplifyCalendars {
			copier.SetEntity(&svc.Calendar, svc.ServiceID, svc.ServiceID)
			continue
		}
		// Validate as Service, with attached exceptions, for better validation.
		if bt, err = copier.checkBatch(bt, svc); err != nil {
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
			cd := cd
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
				if stbt[i].Interpolated.Int > 0 {
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

	for sts := range copier.Reader.StopTimesByTripID() {
		// Write batch
		if batchCount+len(sts) >= copier.BatchSize {
			if err := writeBatch(); err != nil {
				return err
			}
		}

		// Error handling for trips without stop_times is after this block
		if len(sts) == 0 {
			continue
		}
		// Does this trip exist?
		tripid := sts[0].TripID
		if _, ok := alltripids[tripid]; !ok {
			copier.ErrorHandler.HandleEntityErrors(&sts[0], []error{causes.NewInvalidReferenceError("trip_id", tripid)}, nil)
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += len(sts)
			continue
		}
		// Is this trip marked?
		trip, ok := trips[tripid]
		if !ok { // trip_id exists but is not marked
			copier.result.SkipEntityMarkedCount["stop_times.txt"] += len(sts)
			continue
		}
		// Mark trip as associated with at least 1 stop_time
		// We have to process these below because they won't come up via reader.StopTimesByTripID()
		delete(trips, tripid)

		// Associate StopTimes
		trip.StopTimes = sts

		// Set StopPattern
		patkey := stopPatternKey(trip.StopTimes)
		if pat, ok := stopPatterns[patkey]; ok {
			trip.StopPatternID = pat
		} else {
			trip.StopPatternID = len(stopPatterns)
			stopPatterns[patkey] = trip.StopPatternID
		}

		// Do we need to create a shape for this trip
		if !trip.ShapeID.Valid && copier.CreateMissingShapes {
			// Note: if the trip has errors, may result in unused shapes!
			if shapeid, ok := stopPatternShapeIDs[trip.StopPatternID]; ok {
				trip.ShapeID.Key = shapeid
				trip.ShapeID.Valid = true
			} else {
				if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID, time.Now().Unix()), trip.StopTimes); err != nil {
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

		// Interpolate StopTimes
		if copier.InterpolateStopTimes {
			if stoptimes2, err := copier.geomCache.InterpolateStopTimes(trip); err != nil {
				trip.AddWarning(err)
			} else {
				trip.StopTimes = stoptimes2
			}
		}

		// Set JourneyPattern
		jkey := journeyPatternKey(trip, trip.StopTimes)
		stlen := len(trip.StopTimes)
		if jpat, ok := journeyPatterns[jkey]; ok {
			trip.JourneyPatternID = jpat.key
			trip.JourneyPatternOffset = trip.StopTimes[0].ArrivalTime - jpat.firstArrival
			if copier.DeduplicateJourneyPatterns {
				trip.StopTimes = nil
			}
		} else {
			trip.JourneyPatternID = trip.TripID
			trip.JourneyPatternOffset = 0
			journeyPatterns[jkey] = patInfo{firstArrival: trip.StopTimes[0].ArrivalTime, key: trip.JourneyPatternID}
		}

		// Validate trip & add to batch
		if err := copier.checkEntity(&trip); err == nil {
			tripbt = append(tripbt, &trip)
		} else {
			copier.result.SkipEntityReferenceCount["stop_times.txt"] += stlen
			continue
		}
		for i := range trip.StopTimes {
			stbt = append(stbt, trip.StopTimes[i])
		}
		batchCount += stlen
	}

	// Add any Trips that were not visited/did not have StopTimes
	for _, trip := range trips {
		trip := trip
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
