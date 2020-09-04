package copier

import tl "github.com/interline-io/transitland-lib"

// yesMarker will always return that an entity is visited and marked.
type yesMarker struct {
}

// newYesMarker returns a new yesMarker.
func newYesMarker() *yesMarker {
	return &yesMarker{}
}

// IsMarked returns if an Entity is marked.
func (marker *yesMarker) IsMarked(filename, eid string) bool {
	return true
}

// IsVisited returns if an Entity was visited.
func (marker *yesMarker) IsVisited(filename, eid string) bool {
	return true
}

// VisitAndMark traverses the feed and marks entities.
func (marker *yesMarker) VisitAndMark(reader tl.Reader) error {
	return nil
}
