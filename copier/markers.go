package copier

import "github.com/interline-io/gotransit"

// Marker visits and marks entities.
type Marker interface {
	VisitAndMark(gotransit.Reader) error
	IsMarked(string, string) bool
	IsVisited(string, string) bool
}
