package gtdb

import (
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// PostgresAdapter connects to a Postgres/PostGIS database.
type PostgresAdapter struct {
	DBURL  string
	db     sqlx.Ext
	mapper *reflectx.Mapper
}

// Open the adapter.
func (adapter *PostgresAdapter) Open() error {
	if adapter.db != nil {
		return nil
	}
	db, err := sqlx.Open("postgres", adapter.DBURL)
	if err != nil {
		return err
	}
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	adapter.db = db
	adapter.mapper = db.Mapper
	return nil
}

// Close the adapter.
func (adapter *PostgresAdapter) Close() error {
	return nil
	// return adapter.db.Close()
}

// Create an initial database schema.
func (adapter *PostgresAdapter) Create() error {
	if _, err := adapter.db.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	schema, err := getSchema("/postgres.pgsql")
	if err != nil {
		return err
	}
	_, err = adapter.db.Exec(schema)
	return err

}

// DBX returns *sqlx.DB
func (adapter *PostgresAdapter) DBX() sqlx.Ext {
	return adapter.db
}

// Tx runs a callback inside a transaction.
func (adapter *PostgresAdapter) Tx(cb func(Adapter) error) error {
	sqlxdb, ok := adapter.db.(*sqlx.DB)
	if !ok {
		return errors.New("adapter is not *sqlx.DB")
	}
	tx, err := sqlxdb.Beginx()
	if err != nil {
		tx.Rollback()
		return err
	}
	adapter2 := &PostgresAdapter{DBURL: adapter.DBURL, db: tx, mapper: adapter.mapper}
	if err2 := cb(adapter2); err2 != nil {
		tx.Rollback()
		return err2
	}
	tx.Commit()
	return nil
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *PostgresAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db).PlaceholderFormat(sq.Dollar)
}

// Find finds a single entity based on the EntityID()
func (adapter *PostgresAdapter) Find(dest interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	if err != nil {
		return err
	}
	return adapter.Get(dest, qstr, args...)
}

// Get wraps sqlx.Get
func (adapter *PostgresAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Select wraps sqlx.Select
func (adapter *PostgresAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *PostgresAdapter) Insert(ent interface{}) (int, error) {
	if v, ok := ent.(*gotransit.FareAttribute); ok {
		v.Transfers = "0" // TODO: Keep?
	}
	table := getTableName(ent)
	cols, vals, err := getInsert(adapter.mapper, ent)
	if err != nil {
		return 0, err
	}
	q := sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		Suffix("RETURNING \"id\"").
		PlaceholderFormat(sq.Dollar).
		RunWith(adapter.db)
	eid := 0
	if err = q.QueryRow().Scan(&eid); err != nil {
		return 0, err
	}
	if v, ok := ent.(canSetID); ok {
		v.SetID(eid)
	}
	return eid, err
}

// Update a single record
func (adapter *PostgresAdapter) Update(ent interface{}, columns ...string) error {
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
	q := sq.
		Update(table).
		SetMap(colmap).
		PlaceholderFormat(sq.Dollar).
		RunWith(adapter.db)
	_, err = q.Exec()
	return err
}

// BatchInsert builds and executes a multi-insert statement for the given entities.
func (adapter *PostgresAdapter) BatchInsert(ents []gotransit.Entity) error {
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
	cols, _, err := getInsert(adapter.mapper, sts[0])
	table := "gtfs_stop_times"
	q := sq.Insert(table).Columns(cols...)
	for _, d := range sts {
		_, vals, _ := getInsert(adapter.mapper, d)
		q = q.Values(vals...)
	}
	_, err = q.PlaceholderFormat(sq.Dollar).RunWith(adapter.db).Exec()
	return err
}
