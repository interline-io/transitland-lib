package copier

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type ctx = causes.Context

type hasContext interface {
	Context() *causes.Context
}

type updateContext interface {
	Update(*causes.Context)
}

type hasGeometry interface {
	Geometry() tt.Geometry
}

type hasEntityJson interface {
	EntityJson() tt.Map
}

// ValidationErrorGroup helps group errors together with a maximum limit on the number stored.
type ValidationErrorGroup struct {
	Filename  string
	Field     string
	ErrorType string
	ErrorCode string
	GroupKey  string
	Level     int
	Count     int
	Limit     int               `db:"-"`
	Errors    []ValidationError `db:"-"`
}

func NewValidationErrorGroup(err error, limit int) *ValidationErrorGroup {
	errtype := strings.Replace(fmt.Sprintf("%T", err), "*", "", 1)
	if len(strings.Split(errtype, ".")) > 1 {
		errtype = strings.Split(errtype, ".")[1]
	}
	ve := newValidationError(err)
	return &ValidationErrorGroup{
		Filename:  ve.Filename,
		Field:     ve.Field,
		GroupKey:  ve.GroupKey,
		ErrorCode: ve.ErrorCode,
		ErrorType: errtype,
		Limit:     limit,
	}
}

func (eg *ValidationErrorGroup) Key() string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", eg.Filename, eg.Field, eg.ErrorType, eg.ErrorType, eg.GroupKey)
}

// Add an error to the error group.
func (e *ValidationErrorGroup) Add(err error) {
	if e.Count < e.Limit || e.Limit == 0 {
		e.Errors = append(e.Errors, newValidationError(err))
	}
	e.Count++
}

func getErrorKey(err error) string {
	eg := NewValidationErrorGroup(err, 0)
	return eg.Key()
}

type ValidationError struct {
	Filename   string `db:"-"`
	Field      string `db:"-"`
	ErrorCode  string `db:"-"`
	Line       int
	GroupKey   string
	Message    string
	EntityID   string
	Value      string
	Geometry   tt.Geometry
	EntityJson tt.Map
}

func (e ValidationError) Error() string {
	return e.Message
}

func newValidationError(err error) ValidationError {
	ee := ValidationError{
		Message: err.Error(),
	}
	if v, ok := err.(hasContext); ok {
		vctx := v.Context()
		ee.Line = vctx.Line
		ee.Field = vctx.Field
		ee.Filename = vctx.Filename
		ee.EntityID = vctx.EntityID
		ee.Value = vctx.Value
		ee.ErrorCode = vctx.ErrorCode
		ee.EntityJson = tt.NewMap(vctx.EntityJson)
		ee.GroupKey = vctx.GroupKey
	}
	if v, ok := err.(hasGeometry); ok {
		ee.Geometry = v.Geometry()
	}
	if v, ok := err.(hasEntityJson); ok {
		ee.EntityJson = v.EntityJson()
	}
	return ee
}

// Result stores Copier results and statistics.
type Result struct {
	InterpolatedStopTimeCount int
	EntityCount               map[string]int
	GeneratedCount            map[string]int
	SkipEntityErrorCount      map[string]int
	SkipEntityReferenceCount  map[string]int
	SkipEntityFilterCount     map[string]int
	SkipEntityMarkedCount     map[string]int
	Errors                    map[string]*ValidationErrorGroup
	Warnings                  map[string]*ValidationErrorGroup
	ErrorLimit                int
}

// ErrorThresholdResult contains the result of checking error thresholds per file.
type ErrorThresholdResult struct {
	Exceeded bool
	Details  map[string]ErrorThresholdFileResult
}

// ErrorThresholdFileResult contains threshold check results for a single file.
type ErrorThresholdFileResult struct {
	TotalCount   int
	ErrorCount   int
	ErrorPercent float64
	Threshold    float64
	Exceeded     bool
}

// CheckErrorThreshold checks if any file exceeds its error threshold percentage.
// Thresholds are specified as 0-100 (e.g., 5 means 5%, 10 means 10%).
// The thresholds map uses filename as key (e.g., "stops.txt") with "*" as the default.
// The error rate is calculated as (entity errors + reference errors) / total entities * 100.
// Entities skipped by filters or markers do not count toward the error rate.
func (cr *Result) CheckErrorThreshold(thresholds map[string]float64) ErrorThresholdResult {
	result := ErrorThresholdResult{
		Exceeded: false,
		Details:  map[string]ErrorThresholdFileResult{},
	}
	if len(thresholds) == 0 {
		return result
	}

	// Get default threshold
	defaultThreshold := thresholds["*"]

	// Collect all filenames from all count maps
	filenames := map[string]bool{}
	for fn := range cr.EntityCount {
		filenames[fn] = true
	}
	for fn := range cr.SkipEntityErrorCount {
		filenames[fn] = true
	}
	for fn := range cr.SkipEntityReferenceCount {
		filenames[fn] = true
	}

	for fn := range filenames {
		// Get threshold for this file, falling back to default
		threshold, ok := thresholds[fn]
		if !ok {
			threshold = defaultThreshold
		}
		if threshold <= 0 {
			continue // Skip files with no threshold
		}

		entityCount := cr.EntityCount[fn]
		errorCount := cr.SkipEntityErrorCount[fn]
		refErrorCount := cr.SkipEntityReferenceCount[fn]
		totalErrors := errorCount + refErrorCount
		totalCount := entityCount + totalErrors

		var errorPercent float64
		if totalCount > 0 {
			errorPercent = float64(totalErrors) / float64(totalCount) * 100
		}

		exceeded := errorPercent > threshold
		result.Details[fn] = ErrorThresholdFileResult{
			TotalCount:   totalCount,
			ErrorCount:   totalErrors,
			ErrorPercent: errorPercent,
			Threshold:    threshold,
			Exceeded:     exceeded,
		}
		if exceeded {
			result.Exceeded = true
		}
	}

	return result
}

// NewResult returns a new Result.
func NewResult(errorLimit int) *Result {
	if errorLimit < 0 {
		errorLimit = math.MaxInt
	}
	return &Result{
		EntityCount:              map[string]int{},
		GeneratedCount:           map[string]int{},
		SkipEntityErrorCount:     map[string]int{},
		SkipEntityReferenceCount: map[string]int{},
		SkipEntityFilterCount:    map[string]int{},
		SkipEntityMarkedCount:    map[string]int{},
		Errors:                   map[string]*ValidationErrorGroup{},
		Warnings:                 map[string]*ValidationErrorGroup{},
		ErrorLimit:               errorLimit,
	}
}

// HandleSourceErrors .
func (cr *Result) HandleSourceErrors(fn string, errs []error, warns []error) {
	for _, err := range errs {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: fn})
		}
		key := getErrorKey(err)
		v, ok := cr.Errors[key]
		if !ok {
			v = NewValidationErrorGroup(err, cr.ErrorLimit)
			cr.Errors[key] = v
		}
		v.Add(err)
	}
	for _, err := range warns {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: fn})
		}
		key := getErrorKey(err)
		v, ok := cr.Warnings[key]
		if !ok {
			v = NewValidationErrorGroup(err, cr.ErrorLimit)
			cr.Warnings[key] = v
		}
		v.Add(err)
	}
}

// HandleError .
func (cr *Result) HandleError(fn string, errs []error) {
	for _, err := range errs {
		key := getErrorKey(err)
		v, ok := cr.Errors[key]
		if !ok {
			v = NewValidationErrorGroup(err, cr.ErrorLimit)
			cr.Errors[key] = v
		}
		v.Add(err)
	}
}

// HandleEntityErrors .
func (cr *Result) HandleEntityErrors(ent tt.Entity, errs []error, warns []error) {
	// Get entity line, if available
	efn := ent.Filename()
	eid := ent.EntityID()
	eln := 0
	if v, ok := ent.(hasLine); ok {
		eln = v.Line()
	}

	for _, err := range errs {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid, Line: eln})
		}
		// if v, ok := err.(hasSetEntityJson); ok {
		// 	v.SetEntityJson(entityAsJson(ent))
		// }
		key := getErrorKey(err)
		v, ok := cr.Errors[key]
		if !ok {
			v = NewValidationErrorGroup(err, cr.ErrorLimit)
			v.Level = 0
			cr.Errors[key] = v
		}
		v.Add(err)
	}
	for _, err := range warns {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid, Line: eln})
		}
		// if v, ok := err.(hasSetEntityJson); ok {
		// 	v.SetEntityJson(entityAsJson(ent))
		// }
		key := getErrorKey(err)
		v, ok := cr.Warnings[key]
		if !ok {
			v = NewValidationErrorGroup(err, cr.ErrorLimit)
			v.Level = 1
			cr.Warnings[key] = v
		}
		v.Add(err)
	}
}

// DisplayErrors shows individual errors in log.Info
func (cr *Result) DisplayErrors() {
	ctx := context.TODO()
	if len(cr.Errors) == 0 {
		return
	}
	log.For(ctx).Info().Msgf("Errors:")
	for _, v := range cr.Errors {
		log.For(ctx).Info().Msgf("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
		for _, err := range v.Errors {
			log.For(ctx).Info().Msgf("\t\t%s", errfmt(err))
		}
		remain := v.Count - len(v.Errors)
		if remain > 0 {
			log.For(ctx).Info().Msgf("\t\t... and %d more", remain)
		}
	}
}

// DisplayWarnings shows individual warnings in log.Info
func (cr *Result) DisplayWarnings() {
	ctx := context.TODO()
	if len(cr.Warnings) == 0 {
		return
	}
	log.For(ctx).Info().Msgf("Warnings:")
	for _, v := range cr.Warnings {
		log.For(ctx).Info().Msgf("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
		for _, err := range v.Errors {
			log.For(ctx).Info().Msgf("\t\t%s", errfmt(err))
		}
		remain := v.Count - len(v.Errors)
		if remain > 0 {
			log.For(ctx).Info().Msgf("\t\t... and %d more", remain)
		}
	}
}

// DisplaySummary shows entity and error counts in log.Info
func (cr *Result) DisplaySummary() {
	ctx := context.TODO()
	log.For(ctx).Info().Msgf("Copied count:")
	for _, k := range sortedKeys(cr.EntityCount) {
		log.For(ctx).Info().Msgf("\t%s: %d", k, cr.EntityCount[k])
	}
	if msiSum(cr.GeneratedCount) > 0 {
		log.For(ctx).Info().Msgf("Generated count:")
		for _, k := range sortedKeys(cr.GeneratedCount) {
			log.For(ctx).Info().Msgf("\t%s: %d", k, cr.GeneratedCount[k])
		}
	}
	if cr.InterpolatedStopTimeCount > 0 {
		log.For(ctx).Info().Msgf("Interpolated stop_time count: %d", cr.InterpolatedStopTimeCount)
	}
	if msiSum(cr.SkipEntityErrorCount) > 0 {
		log.For(ctx).Info().Msgf("Skipped with errors:")
		for _, k := range sortedKeys(cr.SkipEntityErrorCount) {
			log.For(ctx).Info().Msgf("\t%s: %d", k, cr.SkipEntityErrorCount[k])
		}
	}
	if msiSum(cr.SkipEntityReferenceCount) > 0 {
		log.For(ctx).Info().Msgf("Skipped with reference errors:")
		for _, k := range sortedKeys(cr.SkipEntityReferenceCount) {
			log.For(ctx).Info().Msgf("\t%s: %d", k, cr.SkipEntityReferenceCount[k])
		}
	}
	if msiSum(cr.SkipEntityFilterCount) > 0 {
		log.For(ctx).Info().Msgf("Skipped by filter:")
		for _, k := range sortedKeys(cr.SkipEntityFilterCount) {
			log.For(ctx).Info().Msgf("\t%s: %d", k, cr.SkipEntityFilterCount[k])
		}
	}
	if msiSum(cr.SkipEntityMarkedCount) > 0 {
		log.For(ctx).Info().Msgf("Skipped by marker:")
		for _, k := range sortedKeys(cr.SkipEntityMarkedCount) {
			log.For(ctx).Info().Msgf("\t%s: %d", k, cr.SkipEntityMarkedCount[k])
		}
	}
}

func msiSum(m map[string]int) int {
	ret := 0
	for _, v := range m {
		ret += v
	}
	return ret
}

func sortedKeys(m map[string]int) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func errfmt(err error) string {
	errc, ok := err.(hasContext)
	if !ok {
		return err.Error()
	}
	c := errc.Context()
	s := err.Error()
	if c.EntityID != "" {
		s = fmt.Sprintf("entity '%s': %s", c.EntityID, s)
	}
	if cc := c.Cause(); cc != nil {
		s = s + ": " + cc.Error()
	}
	return s
}
