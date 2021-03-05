package tl

// Writer writes a GTFS feed.
type Writer interface {
	Open() error
	Close() error
	Create() error
	NewReader() (Reader, error)
	AddEntity(Entity) (string, error)
	AddEntities([]Entity) ([]string, error)
}
