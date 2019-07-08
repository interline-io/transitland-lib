package gtdb

import (
	"database/sql"
	"errors"
	"strings"

	// Log
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
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
	m     *reflectx.Mapper
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
	// Keep the mapper to use cache.
	if adapter.m == nil {
		adapter.m = reflectx.NewMapperFunc("db", toSnakeCase)
	}
	cols, vals, err := getInsert(adapter.m, ent)
	if err != nil {
		return 0, err
	}
	q := sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		RunWith(adapter.db)
	if sql, _, err := q.ToSql(); err == nil {
		_ = sql
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
func (adapter *SpatiaLiteAdapter) BatchInsert(table string, ents []gotransit.Entity) error {
	if len(ents) == 0 {
		return nil
	}
	if adapter.m == nil {
		adapter.m = reflectx.NewMapperFunc("db", toSnakeCase)
	}
	cols, _, err := getInsert(adapter.m, ents[0])
	q := sq.Insert(table).Columns(cols...)
	for _, d := range ents {
		_, vals, _ := getInsert(adapter.m, d)
		q = q.Values(vals...)
	}
	_, err = q.RunWith(adapter.db).Exec()
	return err
}
