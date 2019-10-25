package copier

import "github.com/interline-io/gotransit"

// CopyResult stores Copier results and statistics.
type CopyResult struct {
	Errors   []error
	Warnings []error
	Count    map[string]int
}

// NewCopyResult returns a new CopyResult.
func NewCopyResult() *CopyResult {
	return &CopyResult{
		Errors:   []error{},
		Warnings: []error{},
		Count:    map[string]int{},
	}
}

// AddError adds an error to the result.
func (cr *CopyResult) AddError(err error) {
	cr.Errors = append(cr.Errors, err)
}

// AddWarning adds a warning to the result.
func (cr *CopyResult) AddWarning(err error) {
	cr.Warnings = append(cr.Warnings, err)
}

// AddEntity updates the statistics to note an Entity was successfully copied.
func (cr *CopyResult) AddEntity(ent gotransit.Entity) {
	cr.Count[ent.Filename()]++
}

// AddCount adds to the entity counter.
func (cr *CopyResult) AddCount(filename string, count int) {
	cr.Count[filename] += count
}
