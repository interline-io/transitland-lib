package gtdb

import (
	"strconv"

	"github.com/interline-io/gotransit"
	"github.com/jinzhu/gorm"
)

// Writer takes a Reader and saves it to a database.
type Writer struct {
	FeedVersionID int
	Adapter       Adapter
}

// NewWriter returns a Writer appropriate for the given connection url.
func NewWriter(dburl string) (*Writer, error) {
	return &Writer{Adapter: NewAdapter(dburl)}, nil
}

// Open the database.
func (writer *Writer) Open() error {
	return writer.Adapter.Open()
}

// Close the database.
func (writer *Writer) Close() error {
	return writer.Adapter.Close()
}

// Where returns a *gorm.DB with base clauses already set.
func (writer *Writer) Where() *gorm.DB {
	db := writer.Adapter.DB()
	if writer.FeedVersionID > 0 {
		db = db.Where("feed_version_id = ?", writer.FeedVersionID)
	}
	return db
}

// NewReader returns a new Reader with the same adapter.
func (writer *Writer) NewReader() (gotransit.Reader, error) {
	reader := Reader{
		FeedVersionID: writer.FeedVersionID,
		Adapter:       writer.Adapter,
		PageSize:      1000,
	}
	reader.Adapter.SetDB(writer.Adapter.DB())
	return &reader, nil
}

// Create the database.
func (writer *Writer) Create() error {
	// TODO: Load schema
	return nil
}

// Delete any entities associated with the FeedVersion.
func (writer *Writer) Delete() error {
	// TODO: Move to extension Delete() ?
	db := writer.Where().Where("feed_version_id = ?", writer.FeedVersionID)
	db.Delete(&gotransit.Stop{})
	db.Delete(&gotransit.Shape{})
	db.Delete(&gotransit.FeedInfo{})
	db.Delete(&gotransit.Frequency{})
	db.Delete(&gotransit.Trip{})
	db.Delete(&gotransit.Agency{})
	db.Delete(&gotransit.Transfer{})
	db.Delete(&gotransit.Calendar{})
	db.Delete(&gotransit.CalendarDate{})
	db.Delete(&gotransit.Route{})
	db.Delete(&gotransit.StopTime{})
	db.Delete(&gotransit.FareRule{})
	db.Delete(&gotransit.FareAttribute{})
	return nil
}

type feedVersionSetter interface {
	SetFeedVersionID(int)
}

// AddEntity writes an entity to the database.
func (writer *Writer) AddEntity(ent gotransit.Entity) (string, error) {
	// Set the FeedVersionID
	if z, ok := ent.(feedVersionSetter); ok {
		z.SetFeedVersionID(writer.FeedVersionID)
	}
	// Save
	eid, err := writer.Adapter.Insert("", ent)
	return strconv.Itoa(eid), err
}

// AddEntities provides a generic interface for adding Entities to the database.
func (writer *Writer) AddEntities(ents []gotransit.Entity) error {
	for _, ent := range ents {
		if z, ok := ent.(feedVersionSetter); ok {
			z.SetFeedVersionID(writer.FeedVersionID)
		}
	}
	return writer.Adapter.BatchInsert("gtfs_stop_times", ents)
}
