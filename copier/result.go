package copier

import (
	"fmt"
	"sort"
	"strings"

	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

type ctx = causes.Context

type updateContext interface {
	Update(*causes.Context)
}

type hasContext interface {
	Context() *causes.Context
}

func getErrorType(err error) string {
	errtype := strings.Replace(fmt.Sprintf("%T", err), "*", "", 1)
	if len(strings.Split(errtype, ".")) > 1 {
		errtype = strings.Split(errtype, ".")[1]
	}
	return errtype
}

func getErrorFilename(err error) string {
	if v, ok := err.(hasContext); ok {
		return v.Context().Filename
	}
	return ""
}

func getErrorKey(err error) string {
	return getErrorFilename(err) + ":" + getErrorType(err)
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

// ErrorGroup helps group errors together with a maximum limit on the number stored.
type ErrorGroup struct {
	Filename  string
	ErrorType string
	Count     int
	Limit     int
	Errors    []error
}

// NewErrorGroup returns a new ErrorGroup.
func NewErrorGroup(filename string, etype string, limit int) *ErrorGroup {
	return &ErrorGroup{
		Filename:  filename,
		ErrorType: etype,
		Limit:     limit,
	}
}

// Add an error to the error group.
func (e *ErrorGroup) Add(err error) {
	if e.Count < e.Limit || e.Limit == 0 {
		e.Errors = append(e.Errors, err)
	}
	e.Count++
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
	Errors                    map[string]*ErrorGroup
	Warnings                  map[string]*ErrorGroup
	ErrorLimit                int
}

// NewResult returns a new Result.
func NewResult() *Result {
	return &Result{
		EntityCount:              map[string]int{},
		GeneratedCount:           map[string]int{},
		SkipEntityErrorCount:     map[string]int{},
		SkipEntityReferenceCount: map[string]int{},
		SkipEntityFilterCount:    map[string]int{},
		SkipEntityMarkedCount:    map[string]int{},
		Errors:                   map[string]*ErrorGroup{},
		Warnings:                 map[string]*ErrorGroup{},
		ErrorLimit:               1000,
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
			v = NewErrorGroup(getErrorFilename(err), getErrorType(err), cr.ErrorLimit)
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
			v = NewErrorGroup(getErrorFilename(err), getErrorType(err), cr.ErrorLimit)
			cr.Warnings[key] = v
		}
		v.Add(err)
	}
}

// HandleError .
func (cr *Result) HandleError(fn string, errs []error) {
	for _, err := range errs {
		key := fn + ":" + getErrorType(err)
		v, ok := cr.Errors[key]
		if !ok {
			v = NewErrorGroup(fn, getErrorType(err), cr.ErrorLimit)
			cr.Errors[key] = v
		}
		v.Add(err)
	}
}

// HandleEntityErrors .
func (cr *Result) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
	efn := ent.Filename()
	eid := ent.EntityID()
	for _, err := range errs {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid})
		}
		key := getErrorKey(err)
		v, ok := cr.Errors[key]
		if !ok {
			v = NewErrorGroup(getErrorFilename(err), getErrorType(err), cr.ErrorLimit)
			cr.Errors[key] = v
		}
		v.Add(err)
		log.Debug("error %s '%s': %s", efn, eid, err.Error())
	}
	for _, err := range warns {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid})
		}
		key := getErrorKey(err)
		v, ok := cr.Warnings[key]
		if !ok {
			v = NewErrorGroup(getErrorFilename(err), getErrorType(err), cr.ErrorLimit)
			cr.Warnings[key] = v
		}
		v.Add(err)
		log.Debug("warning %s '%s': %s", efn, eid, err.Error())
	}
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

// DisplayErrors shows individual errors in log.Info
func (cr *Result) DisplayErrors() {
	if len(cr.Errors) == 0 {
		log.Info("No errors")
		return
	}
	log.Info("Errors:")
	for _, v := range cr.Errors {
		log.Info("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
		for _, err := range v.Errors {
			log.Info("\t\t%s", errfmt(err))
		}
		remain := v.Count - len(v.Errors)
		if remain > 0 {
			log.Info("\t\t... and %d more", remain)
		}
	}
}

// DisplayWarnings shows individual warnings in log.Info
func (cr *Result) DisplayWarnings() {
	if len(cr.Warnings) == 0 {
		log.Info("No warnings")
		return
	}
	log.Info("Warnings:")
	for _, v := range cr.Warnings {
		log.Info("\tFilename: %s Type: %s Count: %d", v.Filename, v.ErrorType, v.Count)
		for _, err := range v.Errors {
			log.Info("\t\t%s", errfmt(err))
		}
		remain := v.Count - len(v.Errors)
		if remain > 0 {
			log.Info("\t\t... and %d more", remain)
		}
	}
}

// DisplaySummary shows entity and error counts in log.Info
func (cr *Result) DisplaySummary() {
	log.Info("Copied count:")
	for _, k := range sortedKeys(cr.EntityCount) {
		log.Info("\t%s: %d", k, cr.EntityCount[k])
	}
	if msiSum(cr.GeneratedCount) > 0 {
		log.Info("Generated count:")
		for _, k := range sortedKeys(cr.GeneratedCount) {
			log.Info("\t%s: %d", k, cr.GeneratedCount[k])
		}
	}
	if cr.InterpolatedStopTimeCount > 0 {
		log.Info("Interpolated stop_time count: %d", cr.InterpolatedStopTimeCount)
	}
	if msiSum(cr.SkipEntityErrorCount) > 0 {
		log.Info("Skipped with errors:")
		for _, k := range sortedKeys(cr.SkipEntityErrorCount) {
			log.Info("\t%s: %d", k, cr.SkipEntityErrorCount[k])
		}
	}
	if msiSum(cr.SkipEntityReferenceCount) > 0 {
		log.Info("Skipped with reference errors:")
		for _, k := range sortedKeys(cr.SkipEntityReferenceCount) {
			log.Info("\t%s: %d", k, cr.SkipEntityReferenceCount[k])
		}
	}
	if msiSum(cr.SkipEntityFilterCount) > 0 {
		log.Info("Skipped by filter:")
		for _, k := range sortedKeys(cr.SkipEntityFilterCount) {
			log.Info("\t%s: %d", k, cr.SkipEntityFilterCount[k])
		}
	}
	if msiSum(cr.SkipEntityMarkedCount) > 0 {
		log.Info("Skipped by marker:")
		for _, k := range sortedKeys(cr.SkipEntityMarkedCount) {
			log.Info("\t%s: %d", k, cr.SkipEntityMarkedCount[k])
		}
	}
}
