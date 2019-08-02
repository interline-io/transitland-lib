package gtdb

import (
	"database/sql"
	"errors"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/mattn/go-sqlite3"
)

// Register.
func init() {
	sql.Register("spatialite", &sqlite3.SQLiteDriver{Extensions: []string{"mod_spatialite"}})
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
	if _, err := adapter.db.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	schema, err := getSchema("/spatialite.sql")
	if err != nil {
		return err
	}
	_, err = adapter.db.Exec(schema)
	return err
}

// DB returns the underlying Sql DB.
func (adapter *SpatiaLiteAdapter) DB() *sql.DB {
	return adapter.db.DB
}

// DBX returns the underlying Sqlx DB.
func (adapter *SpatiaLiteAdapter) DBX() *sqlx.DB {
	return adapter.db
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *SpatiaLiteAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db)
}

// Find finds a single entity based on the EntityID()
func (adapter *SpatiaLiteAdapter) Find(dest interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	if err != nil {
		return err
	}
	return adapter.db.Get(dest, qstr, args...)
}

// Get wraps sqlx.Get
func (adapter *SpatiaLiteAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Get(dest, qstr, args...)
}

// Select wraps sqlx.Select
func (adapter *SpatiaLiteAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Select(dest, qstr, args...)
}

// Insert builds and executes an insert statement for the given entity.
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
	sts := []*gotransit.StopTime{}
	for _, ent := range ents {
		if st, ok := ent.(*gotransit.StopTime); ok {
			sts = append(sts, st)
		}
	}
	if len(sts) == 0 {
		return errors.New("presently only StopTimes are supported")
	}
	table := getTableName(sts[0])
	cols, vals, err := getInsert(adapter.db.Mapper, sts[0])
	if err != nil {
		return err
	}
	tx, err := adapter.db.Begin()
	if err != nil {
		tx.Rollback()
		return err
	}
	q, _, err := sq.Insert(table).Columns(cols...).Values(vals...).ToSql()
	for _, d := range sts {
		_, vals, err := getInsert(adapter.db.Mapper, d)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(q, vals...); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
