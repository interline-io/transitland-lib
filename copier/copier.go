package copier

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	geomxy "github.com/twpayne/go-geom/xy"
)

// Prepare is called before general copying begins.
type Prepare interface {
	Prepare(tl.Reader, *tl.EntityMap) error
}

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
	AfterValidator(tl.Entity, *tl.EntityMap) error
}

// AfterWrite is called for after writing each entity.
type AfterWrite interface {
	AfterWrite(string, tl.Entity, *tl.EntityMap) error
}

// Extension is run after normal copying has completed.
type Extension interface {
	Copy(*Copier) error
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
	// Simplify shapes
	SimplifyShapes float64
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
	// Default error handler
	ErrorHandler ErrorHandler
	// Journey Pattern Key Function
	JourneyPatternKey func(*tl.Trip) string
	// Named extensions
	Extensions []string
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
	extensions        []Extension
	filters           []Filter
	errorValidators   []Validator
	warningValidators []Validator
	afterValidators   []AfterValidator
	afterWriters      []AfterWrite
	// book keeping
	geomCache *xy.GeomCache
	result    *Result
	*tl.EntityMap
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader tl.Reader, writer tl.Writer, opts Options) (*Copier, error) {
	copier := &Copier{}
	copier.Options = opts
	copier.Reader = reader
	copier.Writer = writer
	// Result
	result := NewResult()
	copier.result = result
	copier.geomCache = xy.NewGeomCache()
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
	// Set the default Journey Pattern function
	if copier.JourneyPatternKey == nil {
		copier.JourneyPatternKey = journeyPatternKey
	}

	// Default set of validators
	copier.AddValidator(&rules.EntityDuplicateCheck{}, 0)
	copier.AddValidator(&rules.ValidFarezoneCheck{}, 0)
	copier.AddValidator(&rules.AgencyIDConditionallyRequiredCheck{}, 0)
	copier.AddValidator(&rules.StopTimeSequenceCheck{}, 0)
	copier.AddValidator(&rules.InconsistentTimezoneCheck{}, 0)
	copier.AddValidator(&rules.ParentStationLocationTypeCheck{}, 0)

	// Default extensions
	if copier.UseBasicRouteTypes {
		copier.AddExtension(&BasicRouteTypeFilter{})
	}

	// Add extensions
	for _, extName := range opts.Extensions {
		e, err := ext.GetExtension(extName)
		if err != nil || e == nil {
			return nil, fmt.Errorf("No registered extension for '%s'", extName)
		}
		if err := copier.AddExtension(e); err != nil {
			return nil, fmt.Errorf("Failed to add extension '%s': %s", extName, err.Error())
		}
	}
	return copier, nil
}

// AddValidator adds an additional entity validator.
func (copier *Copier) AddValidator(ext Validator, level int) error {
	if v, ok := ext.(canShareGeomCache); ok {
		v.SetGeomCache(copier.geomCache)
	}
	if v, ok := ext.(Prepare); ok {
		v.Prepare(copier.Reader, copier.EntityMap)
	}
	if level == 0 {
		copier.errorValidators = append(copier.errorValidators, ext)
	} else if level == 1 {
		copier.warningValidators = append(copier.warningValidators, ext)
	} else {
		return errors.New("unknown validation level")
	}
	return nil
}

// AddExtension adds an Extension to the copy process.
func (copier *Copier) AddExtension(ext interface{}) error {
	added := false
	if v, ok := ext.(canShareGeomCache); ok {
		v.SetGeomCache(copier.geomCache)
	}
	if v, ok := ext.(Prepare); ok {
		v.Prepare(copier.Reader, copier.EntityMap)
	}
	if v, ok := ext.(Filter); ok {
		copier.filters = append(copier.filters, v)
		added = true
	}
	if v, ok := ext.(Validator); ok {
		copier.AddValidator(v, 0)
		added = true
	}
	if v, ok := ext.(AfterValidator); ok {
		copier.afterValidators = append(copier.afterValidators, v)
		added = true
	}
	if v, ok := ext.(Extension); ok {
		copier.extensions = append(copier.extensions, v)
		added = true
	}
	if v, ok := ext.(AfterWrite); ok {
		copier.afterWriters = append(copier.afterWriters, v)
		added = true
	}
	if !added {
		return errors.New("extension does not satisfy any extension interfaces")
	}
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
	if err := copier.checkEntity(ent); err != nil {
		return "", err, nil
	}
	eid, err := copier.addEntity(ent)
	return eid, nil, err
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
		log.Error("Critical error: failed to write %d entities for %s: %s", len(ents), efn, err.Error())
		return err
	}
	for i, eid := range eids {
		sid := sids[i]
		// log.Debug("%s '%s': saved -> %s", efn, sid, eid)
		copier.EntityMap.Set(efn, sid, eid)
	}
	copier.result.EntityCount[efn] += len(ents)
	// AfterWriters
	for i, eid := range eids {
		for _, v := range copier.afterWriters {
			if err := v.AfterWrite(eid, ents[i], copier.EntityMap); err != nil {
				return err
			}
		}
	}
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
	for _, ef := range copier.filters {
		if err := ef.Filter(ent, copier.EntityMap); err != nil {
			log.Debug("%s '%s' skipped by filter: %s", efn, sid, err)
			copier.result.SkipEntityFilterCount[efn]++
			return errors.New("skipped by filter")
		}
	}
	// Run Entity Validators
	for _, v := range copier.errorValidators {
		for _, err := range v.Validate(ent) {
			ent.AddError(err)
		}
	}
	for _, v := range copier.warningValidators {
		for _, err := range v.Validate(ent) {
			ent.AddWarning(err)
		}
	}
	// Perform entity level validation; includes any previous errors
	errs := ent.Errors()
	warns := ent.Warnings()
	copier.ErrorHandler.HandleEntityErrors(ent, errs, warns)
	if len(errs) > 0 && !copier.AllowEntityErrors {
		copier.result.SkipEntityErrorCount[efn]++
		return errs[0]
	}

	// UpdateKeys is handled separately from other validators.
	// It is more like a filter than an error, since it mutates entities.
	referr := ent.UpdateKeys(copier.EntityMap)
	if referr != nil {
		ent.AddError(referr)
	}
	if referr != nil && !copier.AllowReferenceErrors {
		copier.result.SkipEntityReferenceCount[efn]++
		return referr
	}

	// Handle after validators
	for _, v := range copier.afterValidators {
		if err := v.AfterValidator(ent, copier.EntityMap); err != nil {
			return err
		}
	}
	return nil
}

func (copier *Copier) addEntity(ent tl.Entity) (string, error) {
	// OK, Save
	efn := ent.Filename()
	sid := ent.EntityID()
	eid, err := copier.Writer.AddEntity(ent)
	if err != nil {
		log.Error("Critical error: failed to write %s '%s': %s -- entity dump: %#v", efn, sid, err.Error(), ent)
		return "", err
	}
	copier.EntityMap.Set(efn, sid, eid)
	copier.result.EntityCount[efn]++
	// AfterWriters
	for _, v := range copier.afterWriters {
		if err := v.AfterWrite(eid, ent, copier.EntityMap); err != nil {
			return "", err
		}
	}
	return eid, nil
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS entities from the Reader to the Writer, returning the summary as a Result.
func (copier *Copier) Copy() *Result {
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
	for e := range copier.Reader.Levels() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	return nil
}

func (copier *Copier) copyStops() error {
	// First pass for stations
	for ent := range copier.Reader.Stops() {
		if ent.LocationType == 1 {
			copier.geomCache.AddStop(ent.EntityID(), ent)
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	// Second pass for platforms, exits, and generic nodes
	for ent := range copier.Reader.Stops() {
		if ent.LocationType == 0 || ent.LocationType == 2 || ent.LocationType == 3 {
			copier.geomCache.AddStop(ent.EntityID(), ent)
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	// Third pass for boarding areas
	for ent := range copier.Reader.Stops() {
		if ent.LocationType == 4 {
			copier.geomCache.AddStop(ent.EntityID(), ent)
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	copier.logCount(&tl.Stop{})
	return nil
}

func (copier *Copier) copyFares() error {
	// FareAttributes
	for e := range copier.Reader.FareAttributes() {
		var err error
		if _, _, err = copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.FareAttribute{})

	// FareRules
	for e := range copier.Reader.FareRules() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.FareRule{})
	return nil
}

func (copier *Copier) copyPathways() error {
	// Pathways
	for e := range copier.Reader.Pathways() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Pathway{})
	return nil
}

// copyRoutes writes routes
func (copier *Copier) copyRoutes() error {
	for e := range copier.Reader.Routes() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Route{})
	return nil
}

// copyFeedInfos writes FeedInfos
func (copier *Copier) copyFeedInfos() error {
	for e := range copier.Reader.FeedInfos() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.FeedInfo{})
	return nil
}

// copyTransfers writes Transfers
func (copier *Copier) copyTransfers() error {
	for e := range copier.Reader.Transfers() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Transfer{})
	return nil
}

// copyShapes writes Shapes
func (copier *Copier) copyShapes() error {
	// Not safe for batch copy (currently)
	for ent := range copier.Reader.Shapes() {
		sid := ent.EntityID()
		if copier.SimplifyShapes > 0 {
			simplifyValue := copier.SimplifyShapes / 1e6
			pnts := ent.Geometry.FlatCoords()
			// before := len(pnts)
			stride := ent.Geometry.Stride()
			ii := geomxy.SimplifyFlatCoords(pnts, simplifyValue, stride)
			for i, j := range ii {
				if i == j*stride {
					continue
				}
				pnts[i*stride], pnts[i*stride+1] = pnts[j*stride], pnts[j*stride+1]
			}
			pnts = pnts[:len(ii)*stride]
			ent.Geometry = tl.NewLineStringFromFlatCoords(pnts)
		}
		if _, ok, err := copier.CopyEntity(&ent); err != nil {
			return err
		} else if ok == nil {
			copier.geomCache.AddShape(sid, ent)
		}
	}
	copier.logCount(&tl.Shape{})
	return nil
}

// copyFrequencies writes Frequencies
func (copier *Copier) copyFrequencies() error {
	for e := range copier.Reader.Frequencies() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Frequency{})
	return nil
}

// copyCalendars
func (copier *Copier) copyCalendars() error {
	// Get Calendars as Services
	duplicateServices := []*tl.Calendar{}
	svcs := map[string]*tl.Service{}
	for ent := range copier.Reader.Calendars() {
		if !copier.isMarked(&tl.Calendar{}) {
			continue
		}
		_, ok := svcs[ent.EntityID()]
		if ok {
			duplicateServices = append(duplicateServices, &ent)
			continue
		}
		svcs[ent.EntityID()] = tl.NewService(ent)
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
			svc.AddError(causes.NewDuplicateServiceExceptionError(ent.ServiceID, ent.Date))
		} else {
			svc.AddCalendarDate(ent)
		}
	}

	// Simplify and and adjust StartDate and EndDate
	for _, svc := range svcs {
		// Simplify generated and non-generated calendars
		if copier.SimplifyCalendars {
			if s, err := svc.Simplify(); err == nil {
				svc = s
				svcs[svc.EntityID()] = svc
			}
		}
		// Generated calendars may need their service period set...
		if svc.Generated && (svc.StartDate.IsZero() || svc.EndDate.IsZero()) {
			svc.StartDate, svc.EndDate = svc.ServicePeriod()
		}
	}

	// Write Calendars
	bt := []tl.Entity{}
	var btErr error
	for _, svc := range svcs {
		// Need to get calendar dates before ID is updated
		cds := svc.CalendarDates()
		// Skip main Calendar entity if generated and not normalizing/simplifying service IDs.
		if svc.Generated && !copier.NormalizeServiceIDs && !copier.SimplifyCalendars {
			copier.SetEntity(&svc.Calendar, svc.EntityID(), svc.ServiceID)
		} else {
			if _, _, err := copier.CopyEntity(svc); err != nil {
				return err
			}
		}
		// Copy dependent entities
		for i := range cds {
			if bt, btErr = copier.checkBatch(bt, &cds[i]); btErr != nil {
				return btErr
			}
		}
		if svc.Generated {
			copier.result.GeneratedCount["calendar.txt"]++
		}
	}
	if btErr = copier.writeBatch(bt); btErr != nil {
		return btErr
	}
	// Attempt to copy duplicate services
	for _, ent := range duplicateServices {
		if _, _, err := copier.CopyEntity(ent); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Calendar{})
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
	trips := map[string]tl.Trip{}
	duplicateTrips := []tl.Trip{}
	allTripIds := map[string]int{}
	for trip := range copier.Reader.Trips() {
		eid := trip.EntityID()
		allTripIds[eid]++
		// Skip unmarked trips to save work
		if !copier.isMarked(&trip) {
			copier.result.SkipEntityMarkedCount["trips.txt"]++
			continue
		}
		// Handle duplicate trips later
		if _, ok := trips[eid]; ok {
			trip := trip
			duplicateTrips = append(duplicateTrips, trip)
			continue
		}
		trips[eid] = trip
	}

	// Process each set of Trip/StopTimes
	stopPatterns := map[string]int{}
	stopPatternShapeIDs := map[int]string{}
	journeyPatterns := map[string]patInfo{}
	tripOffsets := map[string]int{} // used for deduplicating StopTimes
	var stbt []tl.Entity
	for sts := range copier.Reader.StopTimesByTripID() {
		if len(sts) == 0 {
			continue
		}

		// Does this trip exist?
		tripid := sts[0].TripID
		if _, ok := allTripIds[tripid]; !ok {
			// Trip doesn't exist, try to copy stop times anyway
			for i := range sts {
				copier.CopyEntity(&sts[i])
			}
			continue
		}

		// Is this trip marked?
		trip, ok := trips[tripid]
		if !ok {
			// Trip exists but is not marked
			copier.result.SkipEntityMarkedCount["stop_times.txt"] += len(sts)
			continue
		}

		// Mark trip as associated with at least 1 stop_time
		// Remaining trips will be processed later
		delete(trips, tripid)

		// Set stop times
		trip.StopTimes = sts

		// Set StopPattern
		patkey := stopPatternKey(trip.StopTimes)
		if pat, ok := stopPatterns[patkey]; ok {
			trip.StopPatternID = pat
		} else {
			trip.StopPatternID = len(stopPatterns)
			stopPatterns[patkey] = trip.StopPatternID
		}

		// Create missing shape if necessary
		if !trip.ShapeID.Valid && copier.CreateMissingShapes {
			// Note: if the trip has errors, may result in unused shapes!
			if shapeid, ok := stopPatternShapeIDs[trip.StopPatternID]; ok {
				trip.ShapeID = tl.NewOKey(shapeid)
			} else {
				if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID, time.Now().Unix()), trip.StopTimes); err != nil {
					log.Error("Error: failed to create shape for trip '%s': %s", trip.EntityID(), err)
					trip.AddWarning(err)
				} else {
					// Set ShapeID
					stopPatternShapeIDs[trip.StopPatternID] = shapeid
					trip.ShapeID = tl.NewOKey(shapeid)
				}
			}
		}

		// Interpolate stop times
		if copier.InterpolateStopTimes {
			if stoptimes2, err := copier.geomCache.InterpolateStopTimes(trip); err != nil {
				trip.AddWarning(err)
			} else {
				trip.StopTimes = stoptimes2
			}
		}

		// Set JourneyPattern
		jkey := copier.JourneyPatternKey(&trip)
		if jpat, ok := journeyPatterns[jkey]; ok {
			trip.JourneyPatternID = jpat.key
			trip.JourneyPatternOffset = trip.StopTimes[0].ArrivalTime.Seconds - jpat.firstArrival
			tripOffsets[trip.TripID] = trip.JourneyPatternOffset // do not write stop times for this trip
		} else {
			trip.JourneyPatternID = trip.TripID
			trip.JourneyPatternOffset = 0
			journeyPatterns[jkey] = patInfo{firstArrival: trip.StopTimes[0].ArrivalTime.Seconds, key: trip.JourneyPatternID}
		}

		// Validate trip entity
		if _, _, err := copier.CopyEntity(&trip); err == nil {
			if _, dedupOk := tripOffsets[trip.TripID]; dedupOk && copier.DeduplicateJourneyPatterns {
				// fmt.Println("deduplicating:", trip.TripID)
				// skip
			} else {
				for i := range trip.StopTimes {
					var err error
					stbt, err = copier.checkBatch(stbt, &trip.StopTimes[i])
					if err != nil {
						return err
					}
				}
			}
		} else {
			return err
		}
	}
	if err := copier.writeBatch(stbt); err != nil {
		return err
	}

	// Add any Trips that were not visited/did not have StopTimes
	for _, trip := range trips {
		if _, _, err := copier.CopyEntity(&trip); err == nil {
			return err
		}
	}

	// Add any duplicate trips
	for _, trip := range duplicateTrips {
		if _, _, err := copier.CopyEntity(&trip); err == nil {
			return err
		}
	}

	copier.logCount(&tl.Trip{})
	copier.logCount(&tl.StopTime{})
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
