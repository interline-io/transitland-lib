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

// SpatiaLiteAdapter provides support for SpatiaLite.
type SpatiaLiteAdapter struct {
	DBURL  string
	db     sqlx.Ext
	mapper *reflectx.Mapper
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
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	adapter.db = db
	adapter.mapper = db.Mapper
	return nil
}

// Close implements Adapter Close.
func (adapter *SpatiaLiteAdapter) Close() error {
	if a, ok := adapter.db.(canClose); ok {
		return a.Close()
	}
	return nil
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

// DBX returns the underlying Sqlx DB or Tx.
func (adapter *SpatiaLiteAdapter) DBX() sqlx.Ext {
	return adapter.db
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *SpatiaLiteAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db)
}

// Tx runs a callback inside a transaction.
func (adapter *SpatiaLiteAdapter) Tx(cb func(Adapter) error) error {
	sqlxdb, ok := adapter.db.(*sqlx.DB)
	if !ok {
		return errors.New("adapter is not *sqlx.DB")
	}
	tx, err := sqlxdb.Beginx()
	if err != nil {
		if errTx := tx.Rollback(); errTx != nil {
			return errTx
		}
		return err
	}
	adapter2 := &SpatiaLiteAdapter{DBURL: adapter.DBURL, db: tx, mapper: adapter.mapper}
	if errTx := cb(adapter2); errTx != nil {
		if err3 := tx.Rollback(); err3 != nil {
			return err3
		}
		return errTx
	}
	return tx.Commit()
}

// Find finds a single entity based on the EntityID()
func (adapter *SpatiaLiteAdapter) Find(dest interface{}, args ...interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	if err != nil {
		return err
	}
	return sqlx.Get(adapter.db, dest, qstr, args...)
}

// Get wraps sqlx.Get
func (adapter *SpatiaLiteAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, qstr, args...)
}

// Select wraps sqlx.Select
func (adapter *SpatiaLiteAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, qstr, args...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *SpatiaLiteAdapter) Insert(ent interface{}) (int, error) {
	// Keep the mapper to use cache.
	table := getTableName(ent)
	cols, vals, err := getInsert(adapter.mapper, ent)
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

// Update a single record.
func (adapter *SpatiaLiteAdapter) Update(ent interface{}, columns ...string) error {
	table := getTableName(ent)
	cols, vals, err := getInsert(adapter.mapper, ent)
	if err != nil {
		return err
	}
	colmap := make(map[string]interface{})
	for i, col := range cols {
		if len(columns) > 0 && !contains(col, columns) {
			continue
		}
		colmap[col] = vals[i]
	}
	q := sq.Update(table).SetMap(colmap).RunWith(adapter.db)
	_, err = q.Exec()
	return err
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
	cols, vals, err := getInsert(adapter.mapper, sts[0])
	if err != nil {
		return err
	}
	mapper := adapter.mapper
	return adapter.Tx(func(adapter Adapter) error {
		q, _, err := sq.Insert(table).Columns(cols...).Values(vals...).ToSql()
		if err != nil {
			return err
		}
		for _, d := range sts {
			_, vals, err := getInsert(mapper, d)
			if err != nil {
				return err
			}
			if _, err := adapter.DBX().Exec(q, vals...); err != nil {
				return err
			}
		}
		return nil
	})
}
