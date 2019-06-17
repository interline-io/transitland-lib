package gtdb

import (
	"database/sql"
	"strings"

	// Log
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/log"

	// GORM
	"github.com/jinzhu/gorm"

	// Postgres
	_ "github.com/jinzhu/gorm/dialects/postgres"
	// Spatialite
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	// Drivers
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// Register.
func init() {
	// Register spatialite with GORM
	sql.Register("spatialite", &sqlite3.SQLiteDriver{Extensions: []string{"mod_spatialite"}})
	d, ok := gorm.GetDialect("sqlite3")
	if ok {
		gorm.RegisterDialect("spatialite", d)
	}
}

// NewAdapter returns a Adapter for the given dburl.
func NewAdapter(dburl string) Adapter {
	if strings.HasPrefix(dburl, "postgres://") {
		return &PostGISAdapter{DBURL: dburl}
	} else if strings.HasPrefix(dburl, "sqlite3://") {
		return &SpatiaLiteAdapter{DBURL: dburl}
	}
	return nil
}

// Adapter implements details specific to each backend.
type Adapter interface {
	Open() error
	Close() error
	Create() error
	DB() *gorm.DB
	SetDB(*gorm.DB)
	GeomEncoding() int
	BatchInsert(*[]gotransit.StopTime) error
}

// PostGISAdapter implements details specific to PostGIS.
type PostGISAdapter struct {
	DBURL string
	db    *gorm.DB
}

// SetDB sets the database handle.
func (adapter *PostGISAdapter) SetDB(db *gorm.DB) {
	adapter.db = db
}

// GeomEncoding returns 0, the internal encoding format for EWKB.
func (adapter *PostGISAdapter) GeomEncoding() int {
	return 0
}

// Open implements Adapter Open
func (adapter *PostGISAdapter) Open() error {
	db, err := gorm.Open("postgres", adapter.DBURL)
	if err != nil {
		return causes.NewSourceUnreadableError("could not open database", err)
	}
	adapter.db = db
	if log.Level <= 5 {
		db.LogMode(true)
	}
	return err
}

// Close implements Adapter Close
func (adapter *PostGISAdapter) Close() error {
	return adapter.db.Close()
}

// Create implements Adapter Create
func (adapter *PostGISAdapter) Create() error {
	adapter.db.Exec("CREATE EXTENSION IF NOT EXISTS postgis")
	return nil
}

// DB returns the gorm DB.
func (adapter *PostGISAdapter) DB() *gorm.DB {
	return adapter.db
}

// BatchInsert provides a fast path for creating StopTimes.
func (adapter *PostGISAdapter) BatchInsert(stoptimes *[]gotransit.StopTime) error {
	db := adapter.db
	objArr := *stoptimes
	if len(objArr) == 0 {
		return nil
	}
	// Create the columns
	mainObj := objArr[0]
	mainScope := db.NewScope(mainObj)
	mainFields := mainScope.Fields()
	fields := make([]string, 0, len(mainFields))
	for i := range mainFields {
		// If primary key has blank value (0 for int, "" for string, nil for interface ...), skip it.
		// If field is ignore field, skip it.
		if (mainFields[i].IsPrimaryKey && mainFields[i].IsBlank) || (mainFields[i].IsIgnored) {
			continue
		}
		fields = append(fields, mainFields[i].DBName)
	}
	// Prepare the COPY statement
	pqdb := db.DB()
	txn, err := pqdb.Begin()
	if err != nil {
		return err
	}
	stmt, err := txn.Prepare(pq.CopyIn(mainObj.TableName(), fields...))
	if err != nil {
		return err
	}
	// Add records to COPY
	for _, obj := range objArr {
		scope := db.NewScope(obj)
		fields := scope.Fields()
		for i := range fields {
			if (fields[i].IsPrimaryKey && fields[i].IsBlank) || (fields[i].IsIgnored) {
				continue
			}
			scope.AddToVars(fields[i].Field.Interface())
		}
		_, err = stmt.Exec(scope.SQLVars...)
		if err != nil {
			return err
		}
	}
	// exec and commit
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	err = stmt.Close()
	if err != nil {
		return err
	}
	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}

//////////////////////

// SpatiaLiteAdapter provides implementation details for SpatiaLite.
type SpatiaLiteAdapter struct {
	DBURL string
	db    *gorm.DB
}

// Open implements Adapter Open.
func (adapter *SpatiaLiteAdapter) Open() error {
	dbname := strings.Split(adapter.DBURL, "://")
	if len(dbname) != 2 {
		return causes.NewSourceUnreadableError("no database filename provided", nil)
	}
	db, err := gorm.Open("spatialite", dbname[1])
	if err != nil {
		return causes.NewSourceUnreadableError("could not open database", err)
	}
	adapter.db = db
	if log.Level <= 5 {
		db.LogMode(true)
	}
	return nil
}

// Close implements Adapter Close.
func (adapter *SpatiaLiteAdapter) Close() error {
	return adapter.db.Close()
}

// Create implements Adapter Create.
func (adapter *SpatiaLiteAdapter) Create() error {
	db := adapter.db
	// Init SpatiaLite
	db.Exec("SELECT InitSpatialMetaData(1)")
	// Yuck :( This is the ONLY WAY to add a geometry column in SpatiaLite.
	db.Exec(`CREATE TABLE "gtfs_stops" ("id" integer primary key autoincrement,"feed_version_id" integer,"created_at" datetime,"updated_at" datetime,"stop_id" varchar(255),"stop_name" varchar(255),"stop_code" varchar(255),"stop_desc" varchar(255),"stop_lat" real,"stop_lon" real,"zone_id" varchar(255),"stop_url" varchar(255),"location_type" integer,"parent_station" varchar(255),"stop_timezone" varchar(255),"wheelchair_boarding" integer )`)
	db.Exec("SELECT AddGeometryColumn('gtfs_stops', 'geometry', 4326, 'POINT', 'XY', 0);")
	// Yuck again :(
	db.Exec(`CREATE TABLE "gtfs_shapes" ("id" integer primary key autoincrement,"feed_version_id" integer,"created_at" datetime,"updated_at" datetime,"shape_id" varchar(255),"shape_pt_lat" real,"shape_pt_lon" real,"shape_pt_sequence" integer,"shape_dist_traveled" real )`)
	db.Exec("SELECT AddGeometryColumn('gtfs_shapes', 'geometry', 4326, 'LINESTRING', 'XYM', 1);")
	return nil
}

// SetDB sets the database handle.
func (adapter *SpatiaLiteAdapter) SetDB(db *gorm.DB) {
	adapter.db = db.New()
}

// GeomEncoding returns 1, the encoding internal format code for SpatiaLite blobs.
func (adapter *SpatiaLiteAdapter) GeomEncoding() int {
	return 1
}

// DB provides the underlying gorm DB.
func (adapter *SpatiaLiteAdapter) DB() *gorm.DB {
	return adapter.db
}

// BatchInsert provides a fast path for creating StopTimes.
func (adapter *SpatiaLiteAdapter) BatchInsert(stoptimes *[]gotransit.StopTime) error {
	db := adapter.db
	objArr := *stoptimes
	tx := db.Begin()
	for _, d := range objArr {
		err := tx.Create(&d).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}
