package copier

import (
	"sort"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// CopyResult stores Copier results and statistics.
type CopyResult struct {
	WriteError                error
	Errors                    []error
	Warnings                  []error
	InterpolatedStopTimeCount int
	EntityCount               map[string]int
	GeneratedCount            map[string]int
	SkipEntityErrorCount      map[string]int
	SkipEntityReferenceCount  map[string]int
	SkipEntityFilterCount     map[string]int
	SkipEntityMarkedCount     map[string]int
}

// NewCopyResult returns a new CopyResult.
func NewCopyResult() *CopyResult {
	return &CopyResult{
		Errors:                   []error{},
		Warnings:                 []error{},
		EntityCount:              map[string]int{},
		GeneratedCount:           map[string]int{},
		SkipEntityErrorCount:     map[string]int{},
		SkipEntityReferenceCount: map[string]int{},
		SkipEntityFilterCount:    map[string]int{},
		SkipEntityMarkedCount:    map[string]int{},
	}
}

type ctx = causes.Context

type updateContext interface {
	Update(*causes.Context)
}

// HandleSourceErrors .
func (cr *CopyResult) HandleSourceErrors(fn string, errs []error, warns []error) {
	for _, err := range errs {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: fn})
		}
		cr.Errors = append(cr.Errors, err)
	}
	for _, err := range warns {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: fn})
		}
		cr.Warnings = append(cr.Warnings, err)
	}
}

// HandleEntityErrors .
func (cr *CopyResult) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
	efn := ent.Filename()
	eid := ent.EntityID()
	for _, err := range errs {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid})
		}
		cr.Errors = append(cr.Errors, err)
	}
	for _, err := range warns {
		if v, ok := err.(updateContext); ok {
			v.Update(&ctx{Filename: efn, EntityID: eid})
		}
		cr.Warnings = append(cr.Warnings, err)
	}
}

// DisplayErrors shows individual errors in log.Info
func (cr *CopyResult) DisplayErrors() {
	keys := map[string][]error{}
	for _, err := range cr.Errors {
		efn := ""
		if v, ok := err.(errorWithContext); ok {
			ctx := v.Context()
			efn = ctx.Filename
		}
		keys[efn] = append(keys[efn], err)
	}
	log.Info("Logged errors:")
	for fn, v := range keys {
		group := map[string][]error{}
		for _, err := range v {
			eid := ""
			if v, ok := err.(errorWithContext); ok {
				ctx := v.Context()
				eid = ctx.EntityID
			}
			group[eid] = append(group[eid], err)
		}
		for k, v := range group {
			for _, err := range v {
				log.Info("\t%s '%s': %s", fn, k, err)
			}
		}
	}
}

// DisplaySummary shows entity and error counts in log.Info
func (cr *CopyResult) DisplaySummary() {
	log.Info("Copied count:")
	for _, k := range sortedKeys(cr.EntityCount) {
		log.Info("\t%s: %d", k, cr.EntityCount[k])
	}
	log.Info("Generated count:")
	for _, k := range sortedKeys(cr.GeneratedCount) {
		log.Info("\t%s: %d", k, cr.GeneratedCount[k])
	}
	log.Info("Interpolated stop_time count: %d", cr.InterpolatedStopTimeCount)
	log.Info("Skipped with errors:")
	for _, k := range sortedKeys(cr.SkipEntityErrorCount) {
		log.Info("\t%s: %d", k, cr.SkipEntityErrorCount[k])
	}
	log.Info("Skipped with reference errors:")
	for _, k := range sortedKeys(cr.SkipEntityReferenceCount) {
		log.Info("\t%s: %d", k, cr.SkipEntityReferenceCount[k])
	}
	log.Info("Skipped by filter:")
	for _, k := range sortedKeys(cr.SkipEntityFilterCount) {
		log.Info("\t%s: %d", k, cr.SkipEntityFilterCount[k])
	}
	log.Info("Skipped by marker:")
	for _, k := range sortedKeys(cr.SkipEntityMarkedCount) {
		log.Info("\t%s: %d", k, cr.SkipEntityMarkedCount[k])
	}
}

func sortedKeys(m map[string]int) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
