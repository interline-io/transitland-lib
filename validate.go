package gotransit

import "github.com/interline-io/gotransit/internal/tags"

// ValidateTags validates an Entity using the Entity's struct tags.
func ValidateTags(ent interface{}) []error {
	return tags.ValidateTags(ent)
}
