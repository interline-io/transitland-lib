package gtdb

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	// Log
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
	// Drivers
)

// Register.
func init() {
	sql.Register("spatialite", &sqlite3.SQLiteDriver{Extensions: []string{"mod_spatialite"}})
	d, ok := gorm.GetDialect("sqlite3")
	if ok {
		gorm.RegisterDialect("spatialite", d)
	}
}

// SpatiaLiteAdapter provides implementation details for SpatiaLite.
type SpatiaLiteAdapter struct {
	DBURL string
	db    *sqlx.DB
}

// Open implements Adapter Open.
func (adapter *SpatiaLiteAdapter) Open() error {
	dbname := strings.Split(adapter.DBURL, "://")
	if len(dbname) != 2 {
		return causes.NewSourceUnreadableError("no database filename provided", nil)
	}
	db, err := sqlx.Open("spatialite", dbname[1])
	if err != nil {
		return causes.NewSourceUnreadableError("could not open database", err)
	}
	adapter.db = db
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
	a := db.DB()
	b := sqlx.NewDb(a, "spatialite")
	adapter.db = b
}

// GeomEncoding returns 1, the encoding internal format code for SpatiaLite blobs.
func (adapter *SpatiaLiteAdapter) GeomEncoding() int {
	return 0
}

// DB provides the underlying gorm DB.
func (adapter *SpatiaLiteAdapter) DB() *gorm.DB {
	gormdb, err := gorm.Open("spatialite", adapter.db.DB)
	if err != nil {
		panic(err)
	}
	return gormdb
}

func (adapter *SpatiaLiteAdapter) Insert(table string, ent interface{}) (int, error) {
	if table == "" {
		table = getTableName(ent)
	}
	if table == "" {
		return 0, errors.New("no tablename")
	}
	cols, vals := getInsert(ent)
	q := sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		RunWith(adapter.db)
	if sql, _, err := q.ToSql(); err == nil {
		fmt.Println(sql)
	} else {
		return 0, err
	}
	result, err := q.Exec()
	if err != nil {
		return 0, err
	}
	eid, err := result.LastInsertId()
	if v, ok := ent.(canSetID); ok {
		v.SetID(int(eid))
	}
	return int(eid), nil
}

// BatchInsert provides a fast path for creating StopTimes.
func (adapter *SpatiaLiteAdapter) BatchInsert(stoptimes *[]gotransit.StopTime) error {
	objArr := *stoptimes
	// tx := db.Begin()
	for _, d := range objArr {
		_, err := adapter.Insert("gtfs_stop_times", &d)
		if err != nil {
			// tx.Rollback()
			return err
		}
	}
	// tx.Commit()
	return nil
}
