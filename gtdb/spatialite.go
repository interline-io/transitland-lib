package gtdb

import (
	"database/sql"
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
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	return nil
}

// Close implements Adapter Close.
func (adapter *SpatiaLiteAdapter) Close() error {
	return adapter.db.Close()
}

// Create implements Adapter Create.
func (adapter *SpatiaLiteAdapter) Create() error {
	_, err := sqlx.LoadFile(adapter.DBX(), "../schema/spatialite.sql")
	return err
}

// DB provides the underlying gorm DB.
func (adapter *SpatiaLiteAdapter) DB() *gorm.DB {
	gormdb, err := gorm.Open("spatialite", adapter.db.DB)
	if err != nil {
		panic(err)
	}
	return gormdb
}

// SetDB sets the database handle.
func (adapter *SpatiaLiteAdapter) SetDB(db *gorm.DB) {
	a := db.DB()
	b := sqlx.NewDb(a, "spatialite")
	adapter.db = b
}

func (adapter *SpatiaLiteAdapter) DBX() *sqlx.DB {
	return adapter.db
}

func (adapter *SpatiaLiteAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db)
}

func (adapter *SpatiaLiteAdapter) Find(dest interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, _ := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	return adapter.db.Get(dest, qstr, args...)
}

func (adapter *SpatiaLiteAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Get(dest, qstr, args...)
}

func (adapter *SpatiaLiteAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Select(dest, qstr, args...)
}

func (adapter *SpatiaLiteAdapter) Insert(ent interface{}) (int, error) {
	// Keep the mapper to use cache.
	table := getTableName(ent)
	cols, vals, err := getInsert(adapter.db.Mapper, ent)
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
func (adapter *SpatiaLiteAdapter) BatchInsert(ents []gotransit.Entity) error {
	if len(ents) == 0 {
		return nil
	}
	cols, _, err := getInsert(adapter.db.Mapper, ents[0])
	table := "gtfs_stop_times"
	q := sq.Insert(table).Columns(cols...)
	for _, d := range ents {
		_, vals, _ := getInsert(adapter.db.Mapper, d)
		q = q.Values(vals...)
	}
	_, err = q.RunWith(adapter.db).Exec()
	return err
}
