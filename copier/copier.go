package copier

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/rs/zerolog"
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
	// Normalize timezones, e.g. US/Pacific -> America/Los_Angeles
	NormalizeTimezones bool
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
	// Sub-logger
	sublogger zerolog.Logger
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
	// Logging
	copier.Options.sublogger = log.Logger.With().Str("reader", reader.String()).Str("writer", writer.String()).Logger()

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
	copier.AddValidator(&rules.CalendarDuplicateDates{}, 0)

	// Default extensions
	if copier.UseBasicRouteTypes {
		// Convert extended route types to basic route types
		copier.AddExtension(&BasicRouteTypeFilter{})
	}
	if copier.NormalizeTimezones {
		// Normalize timezones and apply agency/stop timezones where empty
		copier.AddExtension(&NormalizeTimezoneFilter{})
		copier.AddExtension(&ApplyParentTimezoneFilter{})
	}

	// Add extensions
	for _, extName := range opts.Extensions {
		extName, extArgs, err := ext.ParseExtensionArgs(extName)
		if err != nil {
			return nil, err
		}
		e, err := ext.GetExtension(extName, extArgs)
		if err != nil {
			return nil, fmt.Errorf("error creating extension '%s' with args '%s': %s", extName, extArgs, err.Error())
		} else if e == nil {
			return nil, fmt.Errorf("no registered extension for '%s'", extName)
		}
		if err := copier.AddExtension(e); err != nil {
			return nil, fmt.Errorf("failed to add extension '%s': %s", extName, err.Error())
		}
	}
	return copier, nil
}

func (copier *Copier) SetLogger(g zerolog.Logger) {
	copier.sublogger = log.Logger.With().Str("reader", "test").Str("writer", "test").Logger()
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

// CopyEntities validates a slice of entities and writes those that pass validation.
func (copier *Copier) CopyEntities(ents []tl.Entity) error {
	var okEnts []tl.Entity
	for _, ent := range ents {
		if err := copier.checkEntity(ent); err == nil {
			okEnts = append(okEnts, ent)
		}
	}
	return copier.writeBatch(okEnts)
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
		copier.sublogger.Error().Err(err).Str("filename", efn).Msgf("critical error: failed to write %d entities", len(ents))
		return err
	}
	for i, eid := range eids {
		sid := sids[i]
		// copier.sublogger.Trace().Str("filename", efn).Str("source_id", sid).Str("output_id", eid).Msg("saved")
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
			copier.result.SkipEntityFilterCount[efn]++
			copier.sublogger.Debug().Str("filename", efn).Str("source_id", sid).Str("cause", err.Error()).Msg("skipped by filter")
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

	// UpdateKeys is handled separately from other validators.
	referr := ent.UpdateKeys(copier.EntityMap)
	if referr != nil {
		ent.AddError(referr)
	}

	// Perform entity level validation; includes any previous errors
	errs := ent.Errors()
	warns := ent.Warnings()
	for _, err := range warns {
		copier.sublogger.Debug().Str("filename", efn).Str("source_id", sid).Str("cause", err.Error()).Msg("warning")
	}
	for _, err := range errs {
		copier.sublogger.Debug().Str("filename", efn).Str("source_id", sid).Str("cause", err.Error()).Msg("error")
	}
	copier.ErrorHandler.HandleEntityErrors(ent, errs, warns)

	// Check strictness
	if len(errs) > 0 && !copier.AllowEntityErrors {
		copier.result.SkipEntityErrorCount[efn]++
		return errs[0]
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
		copier.sublogger.Error().Err(err).Str("filename", efn).Str("source_id", sid).Msgf("critical error: failed to write -- entity dump %#v", ent)
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
		copier.copyTranslations,
		copier.copyAttributions,
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
	copier.logCount(&tl.Level{})
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
			ent.Geometry = tt.NewLineStringFromFlatCoords(pnts)
		}
		if _, entErr, writeErr := copier.CopyEntity(&ent); writeErr != nil {
			return writeErr
		} else if entErr == nil {
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

// copyAttributions writes Attributions
func (copier *Copier) copyAttributions() error {
	for e := range copier.Reader.Attributions() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Attribution{})
	return nil
}

// copyTranslations writes Translations
func (copier *Copier) copyTranslations() error {
	for e := range copier.Reader.Translations() {
		if _, _, err := copier.CopyEntity(&e); err != nil {
			return err
		}
	}
	copier.logCount(&tl.Translation{})
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
			// save duplicates for later
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
		svc.AddCalendarDate(ent)
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
	var bt []tl.Entity
	var btErr error
	for _, svc := range svcs {
		cid := svc.EntityID()
		// Skip main Calendar entity if generated and not normalizing/simplifying service IDs.
		if svc.Generated && !copier.NormalizeServiceIDs && !copier.SimplifyCalendars {
			copier.SetEntity(&svc.Calendar, svc.EntityID(), svc.ServiceID)
		} else {
			if _, entErr, writeErr := copier.CopyEntity(svc); writeErr != nil {
				return writeErr
			} else if entErr != nil {
				// do not write calendar dates if service had error
				continue
				// cds = nil
			}
		}
		// Copy dependent entities
		cds := svc.CalendarDates()
		for i := range cds {
			cds[i].ServiceID = cid
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
				if _, _, err := copier.CopyEntity(&sts[i]); err != nil {
					return err
				}
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
				trip.ShapeID = tl.NewKey(shapeid)
			} else {
				if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID, time.Now().Unix()), trip.StopTimes); err != nil {
					copier.sublogger.Error().Err(err).Str("filename", "trips.txt").Str("source_id", trip.EntityID()).Msg("failed to create shape")
					trip.AddWarning(err)
				} else {
					// Set ShapeID
					stopPatternShapeIDs[trip.StopPatternID] = shapeid
					trip.ShapeID = tl.NewKey(shapeid)
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
		if _, entErr, writeErr := copier.CopyEntity(&trip); writeErr != nil {
			return writeErr
		} else if entErr == nil {
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
	evt := copier.sublogger.Info().Str("filename", fn).Int("saved", saved)
	if a, ok := copier.result.GeneratedCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("generated %d", a))
		evt = evt.Int("generated", a)
	}
	if a, ok := copier.result.SkipEntityMarkedCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d as unmarked", a))
		evt = evt.Int("skipped_marker", a)
	}
	if a, ok := copier.result.SkipEntityFilterCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d by filter", a))
		evt = evt.Int("skipped_filter", a)
	}
	if a, ok := copier.result.SkipEntityErrorCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d with entity errors", a))
		evt = evt.Int("entity_errors", a)
	}
	if a, ok := copier.result.SkipEntityReferenceCount[fn]; ok && a > 0 {
		out = append(out, fmt.Sprintf("skipped %d with reference errors", a))
		evt = evt.Int("reference_errors", a)
	}
	if saved == 0 && len(out) == 1 {
		return
	}
	outs := strings.Join(out, "; ")
	evt.Msg(outs)
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
	if _, entErr, writeErr := copier.CopyEntity(&shape); writeErr != nil {
		return "", writeErr
	} else if entErr == nil {
		copier.result.GeneratedCount["shapes.txt"]++
	}
	return shape.ShapeID, nil
}
