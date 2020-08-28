// +build cgo

package gtdb

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jmoiron/sqlx"

	// sqlite3
	"github.com/mattn/go-sqlite3"
)

// Register.
func init() {
	// Register test adapter
	adapters["sqlite3"] = func(dburl string) Adapter { return &SQLiteAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (gotransit.Reader, error) { return NewReader(url) }
	gotransit.RegisterReader("sqlite3", r)
	w := func(url string) (gotransit.Writer, error) { return NewWriter(url) }
	gotransit.RegisterWriter("sqlite3", w)
	// Handle SQL function after_feed_version_import.  -- TODO: this is temporary.
	dummy := func(fvid int) int {
		return 0
	}
	sql.Register("sqlite3_w_funcs",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("after_feed_version_import", dummy, true)
			},
		})

}

// SQLiteAdapter provides support for SQLite.
type SQLiteAdapter struct {
	DBURL string
	db    sqlx.Ext
}

// Open implements Adapter Open.
func (adapter *SQLiteAdapter) Open() error {
	dbname := strings.Split(adapter.DBURL, "://")
	if len(dbname) != 2 {
		return causes.NewSourceUnreadableError("no database filename provided", nil)
	}
	db, err := sqlx.Open("sqlite3_w_funcs", dbname[1])
	if err != nil {
		return causes.NewSourceUnreadableError("could not open database", err)
	}
	db.Mapper = mapper
	adapter.db = &queryLogger{db.Unsafe()}
	return nil
}

// Close implements Adapter Close.
func (adapter *SQLiteAdapter) Close() error {
	if a, ok := adapter.db.(canClose); ok {
		return a.Close()
	}
	return nil
}

// Create implements Adapter Create.
func (adapter *SQLiteAdapter) Create() error {
	// Dont log, used often in tests
	adb := adapter.db
	if a, ok := adapter.db.(*queryLogger); ok {
		adb = a.ext
	}
	if _, err := adb.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	schema, err := getSchema("/sqlite.sql")
	if err != nil {
		return err
	}
	_, err = adb.Exec(schema)
	return err
}

// DBX returns the underlying Sqlx DB or Tx.
func (adapter *SQLiteAdapter) DBX() sqlx.Ext {
	return adapter.db
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *SQLiteAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db)
}

// Tx runs a callback inside a transaction.
func (adapter *SQLiteAdapter) Tx(cb func(Adapter) error) error {
	var err error
	var tx *sqlx.Tx
	if a, ok := adapter.db.(canBeginx); ok {
		tx, err = a.Beginx()
	}
	if err != nil {
		return err
	}
	adapter2 := &SQLiteAdapter{DBURL: adapter.DBURL, db: &queryLogger{tx}}
	if errTx := cb(adapter2); errTx != nil {
		if err3 := tx.Rollback(); err3 != nil {
			return err3
		}
		return errTx
	}
	return tx.Commit()
}

// Find finds a single entity based on the EntityID()
func (adapter *SQLiteAdapter) Find(dest interface{}, args ...interface{}) error {
	return find(adapter, dest, args...)
}

// Get wraps sqlx.Get
func (adapter *SQLiteAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, qstr, args...)
}

// Select wraps sqlx.Select
func (adapter *SQLiteAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, qstr, args...)
}

// Update a single record.
func (adapter *SQLiteAdapter) Update(ent interface{}, columns ...string) error {
	return update(adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *SQLiteAdapter) Insert(ent interface{}) (int, error) {
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
		return 0, err
	}
	eid, err := result.LastInsertId()
	if v, ok := ent.(canSetID); ok {
		v.SetID(int(eid))
	}
	return int(eid), nil
}

// CopyInsert is an alias to MultiInsert.
func (adapter *SQLiteAdapter) CopyInsert(ents []interface{}) error {
	return adapter.MultiInsert(ents)
}

// MultiInsert provides a fast path for creating StopTimes.
func (adapter *SQLiteAdapter) MultiInsert(ents []interface{}) error {
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
