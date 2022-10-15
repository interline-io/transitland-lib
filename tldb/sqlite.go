//go:build cgo
// +build cgo

package tldb

import (
	"database/sql"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/schema"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/jmoiron/sqlx"

	// sqlite3
	"github.com/mattn/go-sqlite3"
)

// Register.
func init() {
	// Register test adapter
	adapters["sqlite3"] = func(dburl string) Adapter { return &SQLiteAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	ext.RegisterReader("sqlite3", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("sqlite3", w)
	// Dummy handlers for SQL functions.
	sql.Register("sqlite3_w_funcs",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return nil
			},
		},
	)
}

// SQLiteAdapter provides support for SQLite.
type SQLiteAdapter struct {
	DBURL string
	db    sqlx.Ext
}

// Open the database.
func (adapter *SQLiteAdapter) Open() error {
	dbname := strings.Split(adapter.DBURL, "://")
	if len(dbname) != 2 {
		return causes.NewSourceUnreadableError("no database filename provided", nil)
	}
	db, err := sqlx.Open("sqlite3_w_funcs", dbname[1])
	if err != nil {
		return causes.NewSourceUnreadableError("could not open database", err)
	}
	db.Mapper = MapperCache.Mapper
	adapter.db = &QueryLogger{Ext: db.Unsafe()}
	return nil
}

// Close the database.
func (adapter *SQLiteAdapter) Close() error {
	if a, ok := adapter.db.(canClose); ok {
		return a.Close()
	}
	return nil
}

// Create the database if necessary.
func (adapter *SQLiteAdapter) Create() error {
	// Dont log, used often in tests
	adb := adapter.db
	if a, ok := adapter.db.(*QueryLogger); ok {
		adb = a.Ext
	}
	if _, err := adb.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	_, err := adb.Exec(schema.SqliteSchema)
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
	adapter2 := &SQLiteAdapter{DBURL: adapter.DBURL, db: &QueryLogger{Ext: tx}}
	if errTx := cb(adapter2); errTx != nil {
		if err3 := tx.Rollback(); err3 != nil {
			return err3
		}
		return errTx
	}
	return tx.Commit()
}

// TableExists returns true if the requested table exists
func (adapter *SQLiteAdapter) TableExists(t string) (bool, error) {
	qstr := `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?;`
	checkName := ""
	err := sqlx.Get(adapter.db, &checkName, adapter.db.Rebind(qstr), t)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return checkName == t, err
}

// Find finds a single entity based on the EntityID()
func (adapter *SQLiteAdapter) Find(dest interface{}) error {
	return find(adapter, dest)
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
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	return update(adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *SQLiteAdapter) Insert(ent interface{}) (int, error) {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	table := getTableName(ent)
	header, err := MapperCache.GetHeader(ent)
	if err != nil {
		return 0, err
	}
	vals, err := MapperCache.GetInsert(ent, header)
	if err != nil {
		return 0, err
	}
	q := sq.
		Insert(table).
		Columns(header...).
		Values(vals...).
		RunWith(adapter.db)
	result, err := q.Exec()
	if err != nil {
		return 0, err
	}
	eid, err := result.LastInsertId()
	return int(eid), err
}

// MultiInsert inserts multiple entities.
func (adapter *SQLiteAdapter) MultiInsert(ents []interface{}) ([]int, error) {
	retids := []int{}
	if len(ents) == 0 {
		return retids, nil
	}
	table := getTableName(ents[0])
	header, err := MapperCache.GetHeader(ents[0])
	if err != nil {
		return retids, nil
	}
	vals, err := MapperCache.GetInsert(ents[0], header)
	if err != nil {
		return retids, err
	}
	q, _, err := sq.Insert(table).Columns(header...).Values(vals...).ToSql()
	if err != nil {
		return retids, err
	}
	// Does not work well in tests
	// if err := adapter.Tx(func(adapter Adapter) error {
	db := adapter.DBX()
	for _, d := range ents {
		if v, ok := d.(canUpdateTimestamps); ok {
			v.UpdateTimestamps()
		}
		vals, err := MapperCache.GetInsert(d, header)
		if err != nil {
			return retids, err
		}
		result, err := db.Exec(q, vals...)
		if err != nil {
			return retids, err
		}
		eid, err := result.LastInsertId()
		if err != nil {
			return retids, err
		}
		retids = append(retids, int(eid))
	}
	// }); err != nil {
	// 	return retids, err
	// }
	return retids, nil
}

// CopyInsert uses MultiInsert.
func (adapter *SQLiteAdapter) CopyInsert(ents []interface{}) error {
	_, err := adapter.MultiInsert(ents)
	return err
}
