// Package copier provides tools and utilities for copying and modifying GTFS feeds.
package copier

import (
	"errors"
	"fmt"
	"io"
	"os"
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
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
	// Default error handler
	ErrorHandler ErrorHandler
	// Journey Pattern Key Function
	JourneyPatternKey func(*gtfs.Trip) string
	// Named extensions
	Extensions []string
	// Initialized extensions
	extensions []Extension
	// Error limit
	ErrorLimit int

	// Sub-logger
	Quiet     bool
	sublogger zerolog.Logger
}

func (opts *Options) AddExtension(ext Extension) {
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
		copier.AddValidator(&rules.EntityDuplicateCheck{}, 0)
		copier.AddValidator(&rules.ValidFarezoneCheck{}, 0)
		copier.AddValidator(&rules.AgencyIDConditionallyRequiredCheck{}, 0)
		copier.AddValidator(&rules.StopTimeSequenceCheck{}, 0)
		copier.AddValidator(&rules.InconsistentTimezoneCheck{}, 0)
		copier.AddValidator(&rules.ParentStationLocationTypeCheck{}, 0)
		copier.AddValidator(&rules.CalendarDuplicateDates{}, 0)
		copier.AddValidator(&rules.DuplicateFareLegRuleCheck{}, 0)
		copier.AddValidator(&rules.DuplicateFareTransferRuleCheck{}, 0)
		copier.AddValidator(&rules.DuplicateFareProductCheck{}, 0)
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
		return errors.New("extension does not satisfy any extension interfaces")
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
	return copier.CopyEntities([]tt.Entity{ent})

}

// CopyEntities validates a slice of entities and writes those that pass validation.
func (copier *Copier) CopyEntities(ents []tt.Entity) error {
	okEnts := make([]tt.Entity, 0, len(ents))
	for _, ent := range ents {
		expanded := false
		for _, f := range copier.expandFilters {
			if exp, ok, err := f.Expand(ent, copier.EntityMap); err != nil {
				// skip
			} else if ok {
				expanded = true
				if err := copier.checkEntity(ent); err == nil {
					okEnts = append(okEnts, exp...)
				}
			}
		}
		if !expanded {
			if err := copier.checkEntity(ent); err == nil {
				okEnts = append(okEnts, ent)
			}
		}
	}
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
	fns := []func() error{
		func() error { return copyEntityTypeBatch(copier, 1, copier.Reader.Agencies()) },
		func() error {
			return copyEntityTypeBatch(copier, copier.BatchSize, shapesToShapeLines(copier.Reader.ShapesByShapeID()))
		},
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Routes()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Levels()) },
		func() error {
			return copyEntityTypeBatch(copier, copier.BatchSize,
				chanFilter(copier.Reader.Stops(), func(ent gtfs.Stop) bool {
					return ent.ParentStation.Val == ""
				}),
			)
		},
		func() error {
			return copyEntityTypeBatch(copier, copier.BatchSize,
				chanFilter(copier.Reader.Stops(), func(ent gtfs.Stop) bool {
					return ent.ParentStation.Val != "" && ent.LocationType.Val != 4
				}),
			)
		},
		func() error {
			return copyEntityTypeBatch(copier, copier.BatchSize,
				chanFilter(copier.Reader.Stops(), func(ent gtfs.Stop) bool {
					return ent.ParentStation.Val != "" && ent.LocationType.Val == 4
				}),
			)
		},
		copier.copyCalendars,
		copier.copyTripsAndStopTimes,
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Pathways()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareAttributes()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareRules()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Frequencies()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Transfers()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FeedInfos()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Translations()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Attributions()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.Areas()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.StopAreas()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.RiderCategories()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareMedia()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareProducts()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareLegRules()) },
		func() error { return copyEntityTypeBatch(copier, copier.BatchSize, copier.Reader.FareTransferRules()) },
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
	duplicateServices := []*gtfs.Calendar{}
	cals := map[string]*gtfs.Calendar{}
	for ent := range copier.Reader.Calendars() {
		if !copier.isMarked(&gtfs.Calendar{}) {
			continue
		}
		if _, ok := cals[ent.EntityID()]; ok {
			// Save duplicates for later
			duplicateServices = append(duplicateServices, &ent)
			continue
		}
		cals[ent.EntityID()] = &ent
	}

	// Add the CalendarDates to Services
	for ent := range copier.Reader.CalendarDates() {
		// Check if marked
		cal := gtfs.Calendar{ServiceID: tt.NewString(ent.ServiceID.Val)}
		if !copier.isMarked(&cal) {
			continue
		}
		// Do we create a generated calendar?
		svc, ok := cals[ent.ServiceID.Val]
		if !ok {
			svc = &gtfs.Calendar{}
			svc.ServiceID.Set(ent.ServiceID.Val)
			svc.Generated.Set(true)
			svc.Monday.OrSet(0)
			svc.Tuesday.OrSet(0)
			svc.Wednesday.OrSet(0)
			svc.Thursday.OrSet(0)
			svc.Friday.OrSet(0)
			svc.Saturday.OrSet(0)
			svc.Sunday.OrSet(0)
			cals[ent.ServiceID.Val] = svc
		}
		svc.CalendarDates = append(svc.CalendarDates, ent)
	}

	// Simplify and and adjust StartDate and EndDate
	for _, cal := range cals {
		svc := service.NewService(*cal, cal.CalendarDates...)
		// Simplify generated and non-generated calendars
		if copier.SimplifyCalendars {
			if s, err := svc.Simplify(); err == nil {
				cal = &s.Calendar
				cals[svc.EntityID()] = cal
			}
		}
		// Generated calendars may need their service period set...
		if cal.Generated.Val && (cal.StartDate.IsZero() || cal.EndDate.IsZero()) {
			a, b := svc.ServicePeriod()
			cal.StartDate.Set(a)
			cal.EndDate.Set(b)
		}
	}

	// Helper function for batch copy calendars
	batchCopyCalendars := func(batchEnts []*gtfs.Calendar, ent *gtfs.Calendar, flush bool) ([]*gtfs.Calendar, error) {
		batchEnts, batchEntsAppend, batchFlushed, batchErr := checkBatch(copier, copier.BatchSize, batchEnts, ent, flush)
		if batchErr != nil {
			return nil, batchErr
		}
		if batchFlushed {
			var batchCalendarDates []*gtfs.CalendarDate
			for _, cal := range batchEntsAppend {
				// eid := cal.EntityID()
				for _, cd := range cal.CalendarDates {
					cd := cd
					cd.ServiceID.Set(cal.ServiceID.Val)
					batchCalendarDates = append(batchCalendarDates, &cd)
				}
			}
			if _, _, _, err := checkBatch(copier, copier.BatchSize, batchCalendarDates, nil, true); err != nil {
				return nil, err
			}
		}
		return batchEnts, nil
	}

	// Process calendars
	var batchCalendars []*gtfs.Calendar
	for _, cal := range cals {
		// Write un-normalized calendar dates
		if cal.Generated.Val && !copier.NormalizeServiceIDs && !copier.SimplifyCalendars {
			copier.EntityMap.SetEntity(cal, cal.EntityID(), cal.ServiceID.Val)
			var b []*gtfs.CalendarDate
			for _, cd := range cal.CalendarDates {
				cd := cd
				cd.ServiceID.Set(cal.ServiceID.Val)
				b = append(b, &cd)
			}
			if _, _, _, err := checkBatch(copier, copier.BatchSize, b, nil, true); err != nil {
				return err
			}
			continue
		}
		// Write calendar
		var batchErr error
		batchCalendars, batchErr = batchCopyCalendars(batchCalendars, cal, false)
		if batchErr != nil {
			return batchErr
		}
	}
	if _, batchErr := batchCopyCalendars(batchCalendars, nil, true); batchErr != nil {
		return batchErr
	}

	// Attempt to copy duplicate services
	if _, err := batchCopyCalendars(duplicateServices, nil, true); err != nil {
		return err
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

	// Helper function for batch copy trips
	batchCopyTrips := func(batchEnts []*gtfs.Trip, ent *gtfs.Trip, flush bool) ([]*gtfs.Trip, error) {
		batchEntsNew, batchEntsAppend, batchFlushed, batchErr := checkBatch(copier, copier.BatchSize, batchEnts, ent, flush)
		if batchErr != nil {
			return nil, batchErr
		}
		if batchFlushed {
			var batchStopTimes []tt.Entity
			for _, ent := range batchEntsAppend {
				if _, dedupOk := tripOffsets[ent.TripID.Val]; dedupOk && copier.DeduplicateJourneyPatterns {
					log.Trace().Msgf("deduplicating: %s", ent.TripID)
					continue
				}
				for _, st := range ent.StopTimes {
					batchStopTimes = append(batchStopTimes, &st)
				}
			}
			if err := copier.CopyEntities(batchStopTimes); err != nil {
				return nil, err
			}
		}
		return batchEntsNew, nil
	}

	// Process trips and stop times
	var batchTrips []*gtfs.Trip
	for sts := range copier.Reader.StopTimesByTripID() {
		if len(sts) == 0 {
			continue
		}

		// Does this trip exist?
		tripid := sts[0].TripID.Val
		if _, ok := allTripIds[tripid]; !ok {
			// Trip doesn't exist, try to copy stop times anyway
			var writeSts []*gtfs.StopTime
			for _, st := range sts {
				writeSts = append(writeSts, &st)
			}
			if _, _, _, err := checkBatch(copier, copier.BatchSize, writeSts, nil, true); err != nil {
				return err
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

		var batchErr error
		batchTrips, batchErr = batchCopyTrips(batchTrips, trip, false)
		if batchErr != nil {
			return batchErr
		}
	}
	if _, batchErr := batchCopyTrips(batchTrips, nil, true); batchErr != nil {
		return batchErr
	}

	// Add any Trips that were not visited/did not have StopTimes
	var extraTrips []*gtfs.Trip
	for _, trip := range trips {
		extraTrips = append(extraTrips, trip)
	}
	if _, err := batchCopyTrips(extraTrips, nil, true); err != nil {
		return err
	}

	// Add any duplicate trips
	if _, err := batchCopyTrips(duplicateTrips, nil, true); err != nil {
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

// Copy helpers
func copyEntityTypeBatch[
	T any,
	PT interface {
		tt.Entity
		*T
	}](copier *Copier, batchSize int, it chan T) error {
	var batchEnts []PT
	for ent := range it {
		if batchEntsNew, _, _, batchErr := checkBatch(copier, batchSize, batchEnts, &ent, false); batchErr != nil {
			return batchErr
		} else {
			batchEnts = batchEntsNew
		}
	}
	if _, _, _, batchErr := checkBatch(copier, batchSize, batchEnts, nil, true); batchErr != nil {
		return batchErr
	}
	var entType PT
	copier.logCount(entType)
	return nil
}

func checkBatch[
	T any,
	PT interface {
		tt.Entity
		*T
	}](
	copier *Copier,
	batchSize int,
	ents []PT,
	ent PT,
	flush bool,
) ([]PT, []PT, bool, error) {
	if ent != nil {
		ents = append(ents, ent)
	}
	if len(ents) >= batchSize {
		flush = true
	}
	entsNew := ents
	entsAppend := ents
	if flush {
		entsNew = nil
		writeEnts := make([]tt.Entity, 0, len(ents))
		for _, ent := range ents {
			ent := ent
			writeEnts = append(writeEnts, ent)
		}
		if err := copier.CopyEntities(writeEnts); err != nil {
			return nil, nil, false, err
		}
		ents = nil
	}
	return entsNew, entsAppend, flush, nil
}

func chanFilter[
	T any,
](it chan T, filt func(T) bool) chan T {
	out := make(chan T, 1_000)
	go func() {
		for ent := range it {
			if filt(ent) {
				// fmt.Println("ok:", ent)
				out <- ent
			} else {
				// fmt.Println("skipping:", ent)
			}
		}
		close(out)
	}()
	return out
}

func shapesToShapeLines(it chan []gtfs.Shape) chan service.ShapeLine {
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
