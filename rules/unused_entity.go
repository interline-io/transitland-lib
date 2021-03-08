package rules

import "fmt"

// UnusedEntityError reports when an entity is present but not referenced.
type UnusedEntityError struct {
	bc
}

// NewUnusedEntityError returns a new UnusedEntityError
func NewUnusedEntityError(eid string) *UnusedEntityError {
	return &UnusedEntityError{bc: bc{EntityID: eid}}
}

func (e *UnusedEntityError) Error() string {
	return fmt.Sprintf("entity '%s' exists but is not referenced", e.EntityID)
}
