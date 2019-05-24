package copier

import "github.com/interline-io/gotransit"

// marker visits and marks entities.
type marker interface {
	VisitAndMark(gotransit.Reader) error
	IsMarked(string, string) bool
	IsVisited(string, string) bool
}
