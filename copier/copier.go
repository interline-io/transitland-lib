// Package copier provides tools and utilities for copying and modifying GTFS feeds.
package copier

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/geomcache"
	"github.com/interline-io/transitland-lib/rules"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/rs/zerolog"
)

// Prepare is called before general copying begins.
type Prepare interface {
	Prepare(adapters.Reader, *tt.EntityMap) error
}

// Filter is called before validation.
type Filter interface {
	Filter(tt.Entity, *tt.EntityMap) error
}

type ExpandFilter interface {
	Expand(tt.Entity, *tt.EntityMap) ([]tt.Entity, bool, error)
}

// Validator is called for each entity.
type Validator interface {
	Validate(tt.Entity) []error
}

// AfterValidator is called for each fully validated entity before writing.
type AfterValidator interface {
	AfterValidator(tt.Entity, *tt.EntityMap) error
}

// AfterWrite is called for after writing each entity.
type AfterWrite interface {
	AfterWrite(string, tt.Entity, *tt.EntityMap) error
}

// Extension is run after normal copying has completed.
type Extension interface {
	Copy(*Copier) error
}

// ErrorHandler is called on each source file and entity; errors can be nil
type ErrorHandler interface {
	HandleEntityErrors(tt.Entity, []error, []error)
	HandleSourceErrors(string, []error, []error)
}

type errorWithContext interface {
	Context() *causes.Context
}

type canShareGeomCache interface {
	SetGeomCache(tlxy.GeomCache)
}

type hasLine interface {
	Line() int
}

////////////////////////////
////////// Copier //////////
////////////////////////////

// Options defines the settable options for a Copier.
type Options struct {
	// Batch size
	BatchSize int
	// Skip most validation filters
	NoValidators bool
	// Skip shape cache
	NoShapeCache bool
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
	// Copy extra files (requires CSV input)
	CopyExtraFiles bool
	// Simplify shapes
	SimplifyShapes float64
	// Convert route network_id to networks.txt/route_networks.txt
	NormalizeNetworks bool
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
	// Default error handler
	ErrorHandler ErrorHandler
	// Journey Pattern Key Function
	JourneyPatternKey func(*gtfs.Trip) string
	// Named extensions
	Extensions []string
	// Initialized extensions
	extensions []any
	// Error limit
	ErrorLimit int

	// Sub-logger
	Quiet     bool
	sublogger zerolog.Logger
}

func (opts *Options) AddExtension(ext any) {
	opts.extensions = append(opts.extensions, ext)
}

// Copier copies from Reader to Writer
type Copier struct {
	// Default options
	Options
	// Reader and writer
	Reader adapters.Reader
	Writer adapters.Writer
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
	expandFilters     []ExpandFilter
	// book keeping
	geomCache *geomCacheFilter
	result    *Result
	EntityMap *tt.EntityMap
}

// Quiet copy
func QuietCopy(reader adapters.Reader, writer adapters.Writer, optfns ...func(*Options)) error {
	opts := Options{
		ErrorLimit: -1,
		Quiet:      true,
	}
	for _, f := range optfns {
		f(&opts)
	}
	cp, err := NewCopier(reader, &empty.Writer{}, opts)
	if err != nil {
		return nil
	}
	if cpResult := cp.Copy(); cpResult.WriteError != nil {
		return err
	}
	return nil

}

// Copy with options builder
func Copy(reader adapters.Reader, writer adapters.Writer, optfns ...func(*Options)) error {
	opts := Options{}
	for _, f := range optfns {
		f(&opts)
	}
	cp, err := NewCopier(reader, &empty.Writer{}, opts)
	if err != nil {
		return nil
	}
	if cpResult := cp.Copy(); cpResult.WriteError != nil {
		return err
	}
	return nil
}

// NewCopier creates and initializes a new Copier.
func NewCopier(reader adapters.Reader, writer adapters.Writer, opts Options) (*Copier, error) {
	copier := &Copier{}
	copier.Options = opts
	copier.Reader = reader
	copier.Writer = writer

	// Logging
	if opts.Quiet {
		copier.Options.sublogger = log.Logger.Level(zerolog.ErrorLevel).With().Str("reader", reader.String()).Str("writer", writer.String()).Logger()
	} else {
		copier.Options.sublogger = log.Logger.With().Str("reader", reader.String()).Str("writer", writer.String()).Logger()
	}

	// Result
	result := NewResult(opts.ErrorLimit)
	copier.result = result
	copier.ErrorHandler = opts.ErrorHandler
	if copier.ErrorHandler == nil {
		copier.ErrorHandler = result
	}

	// Default Markers
	copier.Marker = newYesMarker()

	// Default EntityMap
	copier.EntityMap = tt.NewEntityMap()

	// Set the default BatchSize
	if copier.BatchSize == 0 {
		copier.BatchSize = 1_000
	}

	// Set the default Journey Pattern function
	if copier.JourneyPatternKey == nil {
		copier.JourneyPatternKey = journeyPatternKey
	}

	// Geometry cache
	copier.geomCache = &geomCacheFilter{
		NoShapeCache: opts.NoShapeCache,
		GeomCache:    geomcache.NewGeomCache(),
	}
	copier.AddExtension(copier.geomCache)

	// Default set of validators
	if !opts.NoValidators {
		copier.AddValidator(&rules.EntityDuplicateIDCheck{}, 0)
		copier.AddValidator(&rules.EntityDuplicateKeyCheck{}, 0)
		copier.AddValidator(&rules.ValidFarezoneCheck{}, 0)
		copier.AddValidator(&rules.AgencyIDConditionallyRequiredCheck{}, 0)
		copier.AddValidator(&rules.StopTimeSequenceCheck{}, 0)
		copier.AddValidator(&rules.InconsistentTimezoneCheck{}, 0)
		copier.AddValidator(&rules.ParentStationLocationTypeCheck{}, 0)
		copier.AddValidator(&rules.CalendarDuplicateDates{}, 0)
	}

	// Default extensions
	if copier.UseBasicRouteTypes {
		// Convert extended route types to basic route types
		copier.AddExtension(&filters.BasicRouteTypeFilter{})
	}
	if copier.NormalizeTimezones {
		// Normalize timezones and apply agency/stop timezones where empty
		copier.AddExtension(&filters.NormalizeTimezoneFilter{})
		copier.AddExtension(&filters.ApplyParentTimezoneFilter{})
	}
	if copier.SimplifyShapes > 0 {
		// Simplify shapes.txt
		copier.AddExtension(&filters.SimplifyShapeFilter{SimplifyValue: copier.SimplifyShapes})
	}
	if copier.NormalizeNetworks {
		// Convert routes.txt network_id to networks.txt/route_networks.txt
		copier.AddExtension(&filters.RouteNetworkIDFilter{})
	} else {
		copier.AddExtension(&filters.RouteNetworkIDCompatFilter{})
	}
	if copier.SimplifyCalendars && copier.NormalizeServiceIDs {
		// Simplify calendar and calendar dates
		copier.AddExtension(&filters.SimplifyCalendarFilter{})
	}

	// Add extensions
	for _, ext := range opts.extensions {
		if err := copier.AddExtension(ext); err != nil {
			return nil, fmt.Errorf("failed to add extension: %s", err.Error())
		}
	}
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
	copier.sublogger = g
}

// AddValidator adds an additional entity validator.
func (copier *Copier) AddValidator(ext Validator, level int) error {
	if level == 0 {
		return copier.addExtension(ext, false)
	} else if level == 1 {
		return copier.addExtension(ext, true)
	}
	return errors.New("unknown validation level")
}

// AddExtension adds an Extension to the copy process.
func (copier *Copier) AddExtension(ext interface{}) error {
	return copier.addExtension(ext, false)
}

func (copier *Copier) addExtension(ext interface{}, warning bool) error {
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
		if warning {
			copier.warningValidators = append(copier.warningValidators, v)
		} else {
			copier.errorValidators = append(copier.errorValidators, v)
		}
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
	if v, ok := ext.(ExpandFilter); ok {
		copier.expandFilters = append(copier.expandFilters, v)
		added = true
	}
	if !added {
		err := errors.New("extension does not satisfy any extension interfaces")
		log.Error().Err(err).Msg(err.Error())
		return err
	}
	return nil
}

////////////////////////////////////
////////// Helper Methods //////////
////////////////////////////////////

// Check if the entity is marked for copying.
func (copier *Copier) isMarked(ent tt.Entity) bool {
	return copier.Marker.IsMarked(ent.Filename(), ent.EntityID())
}

// CopyEntity performs validation and saves errors and warnings.
func (copier *Copier) CopyEntity(ent tt.Entity) error {
	_, err := copyEntities(copier, []tt.Entity{ent})
	return err
}

func (copier *Copier) CopyEntities(ents []tt.Entity) error {
	_, err := copyEntities(copier, ents)
	return err
}

// checkEntity is the main filter and validation check.
func (copier *Copier) checkEntity(ent tt.Entity) error {
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

	// UpdateKeys is handled separately from other validators.
	var refErrs []error
	if extEnt, ok := ent.(tt.EntityWithReferences); ok {
		if refErr := extEnt.UpdateKeys(copier.EntityMap); refErr != nil {
			refErrs = append(refErrs, refErr)
		}
	} else {
		refErrs = append(refErrs, tt.ReflectUpdateKeys(copier.EntityMap, ent)...)
	}

	// Run filter/validator/extension validators
	var extErrors []error
	var extWarnings []error
	for _, v := range copier.errorValidators {
		extErrors = append(extErrors, v.Validate(ent)...)
	}
	for _, v := range copier.warningValidators {
		extWarnings = append(extWarnings, v.Validate(ent)...)
	}

	// Associate errors with entity if it supports AddError / AddWarning
	var errs []error
	var warns []error
	if len(extErrors) > 0 || len(extWarnings) > 0 || len(refErrs) > 0 {
		if extEnt, ok := ent.(tt.EntityWithLoadErrors); ok {
			for _, err := range refErrs {
				extEnt.AddError(err)
			}
			for _, err := range extErrors {
				extEnt.AddError(err)
			}
			for _, err := range extWarnings {
				extEnt.AddWarning(err)
			}
			errs = nil
			warns = nil
		} else {
			// Otherwise just carry errors over directly
			errs = extErrors
			warns = extWarnings
			errs = append(errs, refErrs...)
		}
	}

	// Get all errors and warnings, including those added above or by data loader
	errs = append(errs, tt.CheckErrors(ent)...)
	warns = append(warns, tt.CheckWarnings(ent)...)

	// Log and set line context
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
	if len(refErrs) > 0 && !copier.AllowReferenceErrors {
		copier.result.SkipEntityReferenceCount[efn]++
		return refErrs[0]
	}

	// Handle after validators
	for _, v := range copier.afterValidators {
		if err := v.AfterValidator(ent, copier.EntityMap); err != nil {
			return err
		}
	}
	return nil
}

func (copier *Copier) writerAddEntities(okEnts []tt.Entity) error {
	if len(okEnts) == 0 {
		return nil
	}
	efn := okEnts[0].Filename()
	sids := make([]string, len(okEnts))
	for i, ent := range okEnts {
		sids[i] = ent.EntityID()
	}
	eids, err := copier.Writer.AddEntities(okEnts)
	if err != nil {
		copier.sublogger.Error().Err(err).Str("filename", efn).Msgf("critical error: failed to write %d entities", len(okEnts))
		return err
	}
	if len(eids) != len(okEnts) {
		return fmt.Errorf("expected to write %d entities, got %d", len(okEnts), len(eids))
	}
	for i, ent := range okEnts {
		sid := sids[i]
		eid := eids[i]
		copier.EntityMap.Set(efn, sid, eid)
		if entExt, ok := ent.(tt.EntityWithGroupKey); ok {
			if groupKey, groupId := entExt.GroupKey(); groupId != "" {
				copier.EntityMap.Set(fmt.Sprintf("%s:%s", efn, groupKey), groupId, groupId)
			}
		}
	}
	copier.result.EntityCount[efn] += len(okEnts)

	// AfterWriters
	for i, eid := range eids {
		for _, v := range copier.afterWriters {
			if err := v.AfterWrite(eid, okEnts[i], copier.EntityMap); err != nil {
				return err
			}
		}
	}
	return nil
}

//////////////////////////////////
////////// Copy Methods //////////
//////////////////////////////////

// Copy copies Base GTFS entities from the Reader to the Writer, returning the summary as a Result.
func (copier *Copier) Copy() *Result {
	// Handle source errors and warnings
	sourceErrors := map[string][]error{}

	copier.sublogger.Trace().Msg("Validating structure")
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
	copier.sublogger.Trace().Msg("Begin processing feed")
	r := copier.Reader
	bs := copier.BatchSize
	fns := []func() error{
		func() error { return batchCopy(copier, batchChan(r.Agencies(), 1, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Routes(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Levels(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(shapeLines(r.ShapesByShapeID()), bs, nil)) },
		func() error {
			return batchCopy(copier,
				batchChan(r.Stops(), bs, func(ent gtfs.Stop) bool {
					return ent.LocationType.Val == 1
				}),
			)
		},
		func() error {
			return batchCopy(copier,
				batchChan(r.Stops(), bs, func(ent gtfs.Stop) bool {
					lt := ent.LocationType.Val
					return lt == 0 || lt == 2 || lt == 3
				}),
			)
		},
		func() error {
			return batchCopy(copier,
				batchChan(r.Stops(), bs, func(ent gtfs.Stop) bool {
					return ent.LocationType.Val == 4
				}),
			)
		},
		copier.copyCalendars,
		copier.copyTripsAndStopTimes,
		func() error { return batchCopy(copier, batchChan(r.Pathways(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareAttributes(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareRules(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Frequencies(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Transfers(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FeedInfos(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Translations(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Attributions(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Timeframes(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Networks(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.RouteNetworks(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.Areas(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.StopAreas(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.RiderCategories(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareMedia(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareProducts(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareLegRules(), bs, nil)) },
		func() error { return batchCopy(copier, batchChan(r.FareTransferRules(), bs, nil)) },
	}
	for i := range fns {
		if err := fns[i](); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}

	for _, e := range copier.extensions {
		copier.sublogger.Trace().Msgf("Running extension Copy(): %T", e)
		if err := e.Copy(copier); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}

	if copier.CopyExtraFiles {
		copier.sublogger.Trace().Msg("Copying extra files")
		if err := copier.copyExtraFiles(); err != nil {
			copier.result.WriteError = err
			return copier.result
		}
	}

	copier.sublogger.Trace().Msg("Done")
	return copier.result
}

/////////////////////////////////////////
////////// Entity Copy Methods //////////
/////////////////////////////////////////

func (copier *Copier) copyExtraFiles() error {
	// TODO: Make more general than CSV-to-CSV
	// And clean up...
	type canFileInfos interface {
		tlcsv.Adapter
		FileInfos() ([]os.FileInfo, error)
	}
	type canAddFile interface {
		FileInfos() ([]os.FileInfo, error)
		AddFile(string, io.Reader) error
	}
	//
	csvReader, ok := copier.Reader.(*tlcsv.Reader)
	if !ok {
		return errors.New("reader does not support copying extra files")
	}
	readerAdapter, ok := csvReader.Adapter.(canFileInfos)
	if !ok {
		return errors.New("reader does not support copying extra files")
	}
	csvWriter, ok := copier.Writer.(*tlcsv.Writer)
	if !ok {
		return errors.New("writer does not support copying extra files")
	}
	writerAdapter, ok := csvWriter.WriterAdapter.(canAddFile)
	if !ok {
		return errors.New("writer does not support copying extra files")
	}
	//
	readerFiles, _ := readerAdapter.FileInfos()
	writerFiles, _ := writerAdapter.FileInfos()
	for _, rf := range readerFiles {
		found := false
		for _, wf := range writerFiles {
			if rf.Name() == wf.Name() {
				found = true
			}
		}
		if found {
			continue
		}
		copier.sublogger.Info().Str("filename", rf.Name()).Msgf("copying extra file")
		var err1 error
		var err2 error
		err1 = readerAdapter.OpenFile(rf.Name(), func(rio io.Reader) {
			err2 = writerAdapter.AddFile(rf.Name(), rio)
		})
		if err1 != nil {
			return err1
		}
		if err2 != nil {
			return err2
		}
	}
	return nil
}

// copyCalendars
func (copier *Copier) copyCalendars() error {
	// Get Calendars as grouped calendars/calendar_dates
	calDates := map[string][]gtfs.CalendarDate{}
	for ent := range copier.Reader.CalendarDates() {
		calDates[ent.ServiceID.Val] = append(calDates[ent.ServiceID.Val], ent)
	}

	// Simplify and and adjust StartDate and EndDate
	for cals := range batchChan(copier.Reader.Calendars(), copier.BatchSize, nil) {
		batchCals := make([]*gtfs.Calendar, 0, len(cals))
		cdCount := 0
		for _, cal := range cals {
			// Add CalendarDates
			cal.CalendarDates = calDates[cal.EntityID()]
			// Remove from CalendarDates, process only once
			// Left-overs will be handled as Generated Calendars below
			delete(calDates, cal.EntityID())
			batchCals = append(batchCals, &cal)
			cdCount += len(cal.CalendarDates)
		}
		// Write Calendars
		okCals, err := copyEntities(copier, batchCals)
		if err != nil {
			return err
		}
		// Write CalendarDates
		batchCalDates := make([]*gtfs.CalendarDate, 0, cdCount)
		for _, ent := range okCals {
			if cal, ok := ent.(*gtfs.Calendar); ok {
				for _, cd := range cal.CalendarDates {
					batchCalDates = append(batchCalDates, &cd)
				}
			}
		}
		if _, err := copyEntities(copier, batchCalDates); err != nil {
			return err
		}
	}

	// Process generated Calendars
	{
		batchCals := make([]*gtfs.Calendar, 0, len(calDates))
		cdCount := 0
		for serviceId, cds := range calDates {
			cal := gtfs.Calendar{}
			cal.ServiceID.Set(serviceId)
			// Set generated
			cal.Generated.Set(true)
			// Set days of week as 0
			cal.Monday.Set(0)
			cal.Tuesday.Set(0)
			cal.Wednesday.Set(0)
			cal.Thursday.Set(0)
			cal.Friday.Set(0)
			cal.Saturday.Set(0)
			cal.Sunday.Set(0)
			cal.CalendarDates = cds
			// Set StartDate, EndDate
			svc := service.NewService(cal, cal.CalendarDates...)
			a, b := svc.ServicePeriod()
			cal.StartDate.Set(a)
			cal.EndDate.Set(b)
			batchCals = append(batchCals, &cal)
			cdCount += len(cal.CalendarDates)
		}
		// Write Calendars
		var okCals []tt.Entity
		if copier.NormalizeServiceIDs {
			var err error
			okCals, err = copyEntities(copier, batchCals)
			if err != nil {
				return err
			}
		} else {
			okCals = make([]tt.Entity, 0, len(batchCals))
			for _, cal := range batchCals {
				copier.EntityMap.Set("calendar.txt", cal.ServiceID.Val, cal.ServiceID.Val)
				okCals = append(okCals, cal)
			}
		}
		// Write CalendarDates
		batchCalDates := make([]*gtfs.CalendarDate, 0, cdCount)
		for _, ent := range okCals {
			if cal, ok := ent.(*gtfs.Calendar); ok {
				for _, cd := range cal.CalendarDates {
					batchCalDates = append(batchCalDates, &cd)
				}
			}
		}
		if _, err := copyEntities(copier, batchCalDates); err != nil {
			return err
		}
	}
	copier.logCount(&gtfs.Calendar{})
	copier.logCount(&gtfs.CalendarDate{})
	return nil
}

type patInfo struct {
	key          string
	firstArrival int
}

// copyTripsAndStopTimes writes Trips and StopTimes
func (copier *Copier) copyTripsAndStopTimes() error {
	// Cache all trips in memory
	trips := map[string]*gtfs.Trip{}
	duplicateTrips := []*gtfs.Trip{}
	allTripIds := map[string]struct{}{}
	for trip := range copier.Reader.Trips() {
		eid := trip.EntityID()
		allTripIds[eid] = struct{}{}
		// Skip unmarked trips to save work
		if !copier.isMarked(&trip) {
			copier.result.SkipEntityMarkedCount["trips.txt"]++
			continue
		}
		// Handle duplicate trips later
		tripCopy := trip
		if _, ok := trips[eid]; ok {
			duplicateTrips = append(duplicateTrips, &tripCopy)
			continue
		}
		trips[eid] = &tripCopy
	}
	log.Trace().Msgf("Loaded %d trips", len(allTripIds))

	// Process each set of Trip/StopTimes
	stopPatterns := map[string]int{}
	stopPatternShapeIDs := map[int]string{}
	journeyPatterns := map[string]patInfo{}
	tripOffsets := map[string]int{} // used for deduplicating StopTimes

	// Process trips and stop times
	for stsGroup := range batchChan(copier.Reader.StopTimesByTripID(), copier.BatchSize, nil) {
		count := 0
		for _, sts := range stsGroup {
			count += len(sts)
		}
		batchTrips := make([]*gtfs.Trip, 0, len(stsGroup))
		batchStopTimes := make([]*gtfs.StopTime, 0, count)
		for _, sts := range stsGroup {
			if len(sts) == 0 {
				continue
			}

			// Does this trip exist?
			tripid := sts[0].TripID.Val
			if _, ok := allTripIds[tripid]; !ok {
				// Trip doesn't exist, try to copy stop times anyway
				for _, st := range sts {
					batchStopTimes = append(batchStopTimes, &st)
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
				trip.StopPatternID.SetInt(pat)
			} else {
				trip.StopPatternID.SetInt(len(stopPatterns))
				stopPatterns[patkey] = trip.StopPatternID.Int()
			}

			// Create missing shape if necessary
			if !trip.ShapeID.Valid && copier.CreateMissingShapes {
				// Note: if the trip has errors, may result in unused shapes!
				if shapeid, ok := stopPatternShapeIDs[trip.StopPatternID.Int()]; ok {
					trip.ShapeID.Set(shapeid)
				} else {
					if shapeid, err := copier.createMissingShape(fmt.Sprintf("generated-%d-%d", trip.StopPatternID.Val, time.Now().Unix()), trip.StopTimes); err != nil {
						copier.sublogger.Error().Err(err).Str("filename", "trips.txt").Str("source_id", trip.EntityID()).Msg("failed to create shape")
						trip.AddWarning(err)
					} else {
						// Set ShapeID
						stopPatternShapeIDs[trip.StopPatternID.Int()] = shapeid
						trip.ShapeID.Set(shapeid)
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
			jkey := copier.JourneyPatternKey(trip)
			if jpat, ok := journeyPatterns[jkey]; ok {
				trip.JourneyPatternID.Set(jpat.key)
				trip.JourneyPatternOffset.SetInt(trip.StopTimes[0].ArrivalTime.Int() - jpat.firstArrival)
				tripOffsets[trip.TripID.Val] = trip.JourneyPatternOffset.Int() // do not write stop times for this trip
			} else {
				trip.JourneyPatternID.Set(trip.TripID.Val)
				trip.JourneyPatternOffset.Set(0)
				journeyPatterns[jkey] = patInfo{firstArrival: trip.StopTimes[0].ArrivalTime.Int(), key: trip.JourneyPatternID.Val}
			}

			// Add to group
			batchTrips = append(batchTrips, trip)
		}

		// Write trips
		okTrips, err := copyEntities(copier, batchTrips)
		if err != nil {
			return err
		}

		// Process regular stop times
		for _, ent := range okTrips {
			if v, ok := ent.(*gtfs.Trip); ok {
				if _, dedupOk := tripOffsets[v.TripID.Val]; dedupOk && copier.DeduplicateJourneyPatterns {
					log.Trace().Msgf("deduplicating: %s", v.TripID)
					continue
				}
				for _, st := range v.StopTimes {
					batchStopTimes = append(batchStopTimes, &st)
				}
			}
		}

		// Write stop times
		if _, err := copyEntities(copier, batchStopTimes); err != nil {
			return err
		}
	}

	// Add any Trips that were not visited/did not have StopTimes
	if _, err := copyEntities(copier, slices.Collect(maps.Values(trips))); err != nil {
		return err
	}

	// Add any duplicate trips
	if _, err := copyEntities(copier, duplicateTrips); err != nil {
		return err
	}

	copier.logCount(&gtfs.Trip{})
	copier.logCount(&gtfs.StopTime{})
	return nil
}

////////////////////////////////////////////
////////// Entity Support Methods //////////
////////////////////////////////////////////

func (copier *Copier) logCount(ent tt.Entity) {
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

func (copier *Copier) createMissingShape(shapeID string, stoptimes []gtfs.StopTime) (string, error) {
	stopids := []string{}
	for _, st := range stoptimes {
		stopids = append(stopids, st.StopID.Val)
	}
	line, dists, err := copier.geomCache.MakeShape(stopids...)
	if err != nil {
		return "", err
	}
	var flatCoords []float64
	for i := 0; i < len(line); i++ {
		flatCoords = append(flatCoords, line[i].Lon, line[i].Lat, dists[i])
	}
	shape := service.ShapeLine{}
	shape.Generated = true
	shape.ShapeID.Set(shapeID)
	shape.Geometry = tt.NewLineStringFromFlatCoords(flatCoords)
	if writeErr := copier.CopyEntity(&shape); writeErr != nil {
		return "", writeErr
	} else {
		copier.result.GeneratedCount["shapes.txt"]++
	}
	return shape.ShapeID.Val, nil
}

func copyEntities[T tt.Entity](copier *Copier, ents []T) ([]tt.Entity, error) {
	if len(ents) == 0 {
		return nil, nil
	}
	expandedEnts := make([]tt.Entity, 0, len(ents))
	for _, ent := range ents {
		ent := ent
		expanded := false
		for _, f := range copier.expandFilters {
			if a, ok, err := f.Expand(ent, copier.EntityMap); err != nil {
				log.Error().Err(err).Msg("failed to expand")
			} else if ok {
				expanded = true
				expandedEnts = append(expandedEnts, a...)
			}
		}
		if !expanded {
			expandedEnts = append(expandedEnts, ent)
		}
	}
	// Group by filename, retaining input order
	batchedEnts := batchEntFilenames(expandedEnts)
	if len(batchedEnts) == 0 {
		batchedEnts = append(batchedEnts, expandedEnts)
	}
	// Write in filename batches
	okEnts := make([]tt.Entity, 0, len(expandedEnts))
	for _, batch := range batchedEnts {
		checkedEnts := make([]tt.Entity, 0, len(batch))
		for _, ent := range batch {
			if err := copier.checkEntity(ent); err == nil {
				checkedEnts = append(checkedEnts, ent)
			}
		}
		if err := copier.writerAddEntities(checkedEnts); err != nil {
			return nil, err
		}
		okEnts = append(okEnts, checkedEnts...)
	}
	return okEnts, nil
}

// Copy helpers
func batchCopy[
	T any,
	PT interface {
		tt.Entity
		*T
	}](
	copier *Copier,
	itBatch iter.Seq[[]T],
) error {
	for entBatch := range itBatch {
		writeEnts := make([]tt.Entity, len(entBatch))
		for i, ent := range entBatch {
			var x PT = &ent
			writeEnts[i] = x
		}
		if err := copier.CopyEntities(writeEnts); err != nil {
			return err
		}
	}
	var entType PT
	copier.logCount(entType)
	return nil
}

func batchChan[T any](it chan T, batchSize int, filt func(T) bool) iter.Seq[[]T] {
	return func(yield func([]T) bool) {
		var ents []T
		for ent := range it {
			if filt != nil && !filt(ent) {
				continue
			}
			ents = append(ents, ent)
			if len(ents) < batchSize {
				continue
			}
			if !yield(ents) {
				return
			}
			ents = nil
		}
		if len(ents) > 0 {
			yield(ents)
		}
	}
}

func batchEntFilenames(ents []tt.Entity) [][]tt.Entity {
	mixedFns := false
	lastFn := ents[0].Filename()
	for _, ent := range ents {
		fn := ent.Filename()
		if fn != lastFn {
			mixedFns = true
			break
		}
	}
	if !mixedFns {
		return nil
	}
	var batches [][]tt.Entity
	var batch []tt.Entity
	lastFn = ents[0].Filename()
	for _, ent := range ents {
		if fn := ent.Filename(); fn == lastFn {
			batch = append(batch, ent)
		} else {
			lastFn = fn
			batches = append(batches, batch)
			batch = nil
			batch = append(batch, ent)
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

func shapeLines(it chan []gtfs.Shape) chan service.ShapeLine {
	out := make(chan service.ShapeLine)
	go func() {
		for shapeEnts := range it {
			ent := service.NewShapeLineFromShapes(shapeEnts)
			out <- ent
		}
		close(out)
	}()
	return out
}

// geomCacheFilter

type geomCacheFilter struct {
	NoShapeCache bool
	*geomcache.GeomCache
}

func (e *geomCacheFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		e.GeomCache.AddStopGeom(v.EntityID(), v.ToPoint())
	case *service.ShapeLine:
		if !e.NoShapeCache {
			lm := v.Geometry.ToLineM()
			e.GeomCache.AddShapeGeom(v.EntityID(), lm.Coords, lm.Data)
		}
	}
	return nil
}
