package extract

// fileInfo helps manage details about a GTFS table
type fileInfo struct {
	Visited map[string]int
	Marked  map[string]int
}

func newFileInfo() fileInfo {
	return fileInfo{
		Visited: map[string]int{},
		Marked:  map[string]int{},
	}
}

func (fi *fileInfo) IsMarked(id string) bool {
	_, ok := fi.Marked[id]
	return ok
}

func (fi *fileInfo) Mark(id string) {
	fi.Marked[id]++
}

func (fi *fileInfo) Unmark(id string) {
	delete(fi.Marked, id)
}

func (fi *fileInfo) Visit(id string) {
	fi.Visited[id]++
}

func (fi *fileInfo) IsVisited(id string) bool {
	_, ok := fi.Visited[id]
	return ok
}
