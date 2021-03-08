package rules

import (
	"fmt"
)

// TODO: Unused entity checks

// UnusedEntityError reports when an entity is present but not referenced.
type UnusedEntityError struct {
	bc
}

func (e *UnusedEntityError) Error() string {
	return fmt.Sprintf("entity '%s' exists but is not referenced", e.EntityID)
}
