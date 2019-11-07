package gtdb

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
)

// Register.
func init() {
	sql.Register("spatialite", &sqlite3.SQLiteDriver{Extensions: []string{"mod_spatialite"}})
}

// SpatiaLiteAdapter provides support for SpatiaLite.
type SpatiaLiteAdapter struct {
	DBURL string
	db    sqlx.Ext
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
	db.Mapper = mapper
	adapter.db = &queryLogger{db.Unsafe()}
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
	// Dont log, used often in tests
	adb := adapter.db
	if a, ok := adapter.db.(*queryLogger); ok {
		adb = a.ext
	}
	if _, err := adb.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	schema, err := getSchema("/spatialite.sql")
	if err != nil {
		return err
	}
	_, err = adb.Exec(schema)
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
	var err error
	var tx *sqlx.Tx
	if a, ok := adapter.db.(canBeginx); ok {
		tx, err = a.Beginx()
	}
	if err != nil {
		return err
	}
	adapter2 := &SpatiaLiteAdapter{DBURL: adapter.DBURL, db: &queryLogger{tx}}
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
	return find(adapter, dest, args...)
}

// Get wraps sqlx.Get
func (adapter *SpatiaLiteAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, qstr, args...)
}

// Select wraps sqlx.Select
func (adapter *SpatiaLiteAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, qstr, args...)
}

// Update a single record.
func (adapter *SpatiaLiteAdapter) Update(ent interface{}, columns ...string) error {
	return update(adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *SpatiaLiteAdapter) Insert(ent interface{}) (int, error) {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	table := getTableName(ent)
	cols, vals, err := getInsert(ent)
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
		panic(err)
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
	table := getTableName(ents[0])
	cols, vals, err := getInsert(ents[0])
	if err != nil {
		return err
	}
	q, _, err := sq.Insert(table).Columns(cols...).Values(vals...).ToSql()
	if err != nil {
		return err
	}
	// return adapter.Tx(func(adapter Adapter) error {
	db := adapter.DBX()
	for _, d := range ents {
		_, vals, err := getInsert(d)
		if err != nil {
			return err
		}
		if _, err := db.Exec(q, vals...); err != nil {
			return err
		}
	}
	return nil
	// })
}
