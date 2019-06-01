package copier

// Marker visits and marks entities.
type Marker interface {
	IsMarked(string, string) bool
	IsVisited(string, string) bool
}
