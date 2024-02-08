package copier

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
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
		ErrorCode: ve.ErrorCode,
		ErrorType: errtype,
		Limit:     limit,
	}
}

func (eg *ValidationErrorGroup) Key() string {
	return eg.Filename + ":" + eg.Field + ":" + eg.ErrorType
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
	WriteError                error
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

// func entityAsJson(ent tl.Entity) map[string]any {
// 	ret := map[string]any{}
// 	entBytes, err := json.Marshal(ent)
// 	if err != nil {
// 		panic(err)
// 	}
// 	if err := json.Unmarshal(entBytes, &ret); err != nil {
// 		panic(err)
// 	}
// 	return ret
// }

// HandleEntityErrors .
func (cr *Result) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
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
	if cr.WriteError == nil && len(cr.Errors) == 0 {
		log.Infof("No errors")
		return
	}
	if cr.WriteError != nil {
		log.Infof("Write error: %s", cr.WriteError.Error())
	}
	if len(cr.Errors) > 0 {
		log.Infof("Errors:")
		for _, v := range cr.Errors {
			log.Infof("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
			for _, err := range v.Errors {
				log.Infof("\t\t%s", errfmt(err))
			}
			remain := v.Count - len(v.Errors)
			if remain > 0 {
				log.Infof("\t\t... and %d more", remain)
			}
		}
	}
}

// DisplayWarnings shows individual warnings in log.Info
func (cr *Result) DisplayWarnings() {
	if len(cr.Warnings) == 0 {
		log.Infof("No warnings")
		return
	}
	log.Infof("Warnings:")
	for _, v := range cr.Warnings {
		log.Infof("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
		for _, err := range v.Errors {
			log.Infof("\t\t%s", errfmt(err))
		}
		remain := v.Count - len(v.Errors)
		if remain > 0 {
			log.Infof("\t\t... and %d more", remain)
		}
	}
}

// DisplaySummary shows entity and error counts in log.Info
func (cr *Result) DisplaySummary() {
	log.Infof("Copied count:")
	for _, k := range sortedKeys(cr.EntityCount) {
		log.Infof("\t%s: %d", k, cr.EntityCount[k])
	}
	if msiSum(cr.GeneratedCount) > 0 {
		log.Infof("Generated count:")
		for _, k := range sortedKeys(cr.GeneratedCount) {
			log.Infof("\t%s: %d", k, cr.GeneratedCount[k])
		}
	}
	if cr.InterpolatedStopTimeCount > 0 {
		log.Infof("Interpolated stop_time count: %d", cr.InterpolatedStopTimeCount)
	}
	if msiSum(cr.SkipEntityErrorCount) > 0 {
		log.Infof("Skipped with errors:")
		for _, k := range sortedKeys(cr.SkipEntityErrorCount) {
			log.Infof("\t%s: %d", k, cr.SkipEntityErrorCount[k])
		}
	}
	if msiSum(cr.SkipEntityReferenceCount) > 0 {
		log.Infof("Skipped with reference errors:")
		for _, k := range sortedKeys(cr.SkipEntityReferenceCount) {
			log.Infof("\t%s: %d", k, cr.SkipEntityReferenceCount[k])
		}
	}
	if msiSum(cr.SkipEntityFilterCount) > 0 {
		log.Infof("Skipped by filter:")
		for _, k := range sortedKeys(cr.SkipEntityFilterCount) {
			log.Infof("\t%s: %d", k, cr.SkipEntityFilterCount[k])
		}
	}
	if msiSum(cr.SkipEntityMarkedCount) > 0 {
		log.Infof("Skipped by marker:")
		for _, k := range sortedKeys(cr.SkipEntityMarkedCount) {
			log.Infof("\t%s: %d", k, cr.SkipEntityMarkedCount[k])
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
