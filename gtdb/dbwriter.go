package gtdb

import (
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
	// TODO: Move to extension Create()
	db := writer.Adapter.DB()
	writer.Adapter.Create()
	db.AutoMigrate(&gotransit.FeedVersion{})
	db.AutoMigrate(&gotransit.Stop{})
	db.AutoMigrate(&gotransit.Shape{})
	db.AutoMigrate(&gotransit.FeedInfo{})
	db.AutoMigrate(&gotransit.Frequency{})
	db.AutoMigrate(&gotransit.Trip{})
	db.AutoMigrate(&gotransit.Agency{})
	db.AutoMigrate(&gotransit.Transfer{})
	db.AutoMigrate(&gotransit.Calendar{})
	db.AutoMigrate(&gotransit.CalendarDate{})
	db.AutoMigrate(&gotransit.Route{})
	db.AutoMigrate(&gotransit.StopTime{})
	db.AutoMigrate(&gotransit.FareRule{})
	db.AutoMigrate(&gotransit.FareAttribute{})
	// FeedVersion Relationships
	// x := "RESTRICT"
	// db.Model(&gotransit.FeedVersion{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Stop{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Shape{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.FeedInfo{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Frequency{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Trip{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Agency{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Transfer{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Calendar{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.CalendarDate{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.Route{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.StopTime{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.FareRule{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// db.Model(&gotransit.FareAttribute{}).AddForeignKey("feed_version_id", "feed_versions(id)", x, x)
	// Required relations
	// db.Model(&gotransit.Trip{}).AddForeignKey("route_id", "gtfs_routes(id)", x, x)
	// db.Model(&gotransit.Trip{}).AddForeignKey("service_id", "gtfs_calendars(id)", x, x)
	// db.Model(&gotransit.CalendarDate{}).AddForeignKey("service_id", "gtfs_calendars(id)", x, x)
	// db.Model(&gotransit.FareAttribute{}).AddForeignKey("agency_id", "gtfs_agencies(id)", x, x)
	// db.Model(&gotransit.FareRule{}).AddForeignKey("fare_id", "gtfs_fare_attributes(id)", x, x)
	// db.Model(&gotransit.Frequency{}).AddForeignKey("trip_id", "gtfs_trips(id)", x, x)
	// db.Model(&gotransit.Route{}).AddForeignKey("agency_id", "gtfs_agencies(id)", x, x)
	// db.Model(&gotransit.StopTime{}).AddForeignKey("trip_id", "gtfs_trips(id)", x, x)
	// db.Model(&gotransit.StopTime{}).AddForeignKey("stop_id", "gtfs_stops(id)", x, x)
	// db.Model(&gotransit.Transfer{}).AddForeignKey("from_stop_id", "gtfs_stops(id)", x, x)
	// db.Model(&gotransit.Transfer{}).AddForeignKey("to_stop_id", "gtfs_stops(id)", x, x)
	// Optional relations
	// db.Model(&gotransit.Stop{}).AddForeignKey("parent_station", "gtfs_stops(id)", x, x)
	// db.Model(&gotransit.Trip{}).AddForeignKey("shape_id", "gtfs_shapes(id)", x, x)
	// db.Model(&gotransit.FareRule{}).AddForeignKey("route_id", "gtfs_routes(id)", x, x)
	return nil
}

// Delete any entities associated with the FeedVersion.
func (writer *Writer) Delete() error {
	// TODO: Move to extension Delete() ?
	// force feed_version_id
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
	// Type specific updates
	switch et := ent.(type) {
	case *gotransit.Stop:
		et.Geometry.Encoding = writer.Adapter.GeomEncoding()
		if len(et.ParentStation) == 0 {
			et.ParentStation = "0"
		}
	case *gotransit.Trip:
		if len(et.ShapeID) == 0 {
			et.ShapeID = "0"
		}
	case *gotransit.Shape:
		et.Geometry.Encoding = writer.Adapter.GeomEncoding()
	case *gotransit.FareRule:
		et.RouteID = "0"
	}
	// Set the FeedVersionID
	if z, ok := ent.(feedVersionSetter); ok {
		z.SetFeedVersionID(writer.FeedVersionID)
	}
	// Save
	err := writer.Adapter.Insert(ent)
	return ent.EntityID(), err
}

// AddEntities provides a generic interface for adding Entities to the database.
func (writer *Writer) AddEntities(ents []gotransit.Entity) error {
	errs := []error{}
	stoptimes := []gotransit.StopTime{}
	for _, ent := range ents {
		switch v := ent.(type) {
		case *gotransit.StopTime:
			v.SetFeedVersionID(writer.FeedVersionID)
			stoptimes = append(stoptimes, *v)
		}
	}
	if len(stoptimes) == len(ents) {
		if err := writer.Adapter.BatchInsert(&stoptimes); err != nil {
			errs = append(errs, err)
		}
	} else {
		for _, ent := range ents {
			if _, err := writer.AddEntity(ent); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
