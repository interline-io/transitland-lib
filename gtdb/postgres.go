package gtdb

import (
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var mapper = reflectx.NewMapperFunc("db", toSnakeCase)

// PostgresAdapter connects to a Postgres/PostGIS database.
type PostgresAdapter struct {
	DBURL string
	db    sqlx.Ext
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
	db.Mapper = mapper
	adapter.db = &queryLogger{db.Unsafe()}
	return nil
}

// Close the adapter.
func (adapter *PostgresAdapter) Close() error {
	if a, ok := adapter.db.(canClose); ok {
		return a.Close()
	}
	return nil
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
	var err error
	var tx *sqlx.Tx
	if a, ok := adapter.db.(canBeginx); ok {
		tx, err = a.Beginx()
	}
	if err != nil {
		return err
	}
	adapter2 := &PostgresAdapter{DBURL: adapter.DBURL, db: &queryLogger{tx}}
	if err2 := cb(adapter2); err2 != nil {
		if errTx := tx.Rollback(); errTx != nil {
			return errTx
		}
		return err2
	}
	return tx.Commit()
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *PostgresAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db).PlaceholderFormat(sq.Dollar)
}

// Find finds a single entity based on the EntityID()
func (adapter *PostgresAdapter) Find(dest interface{}, args ...interface{}) error {
	return find(adapter, dest, args...)
}

// Get wraps sqlx.Get
func (adapter *PostgresAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Select wraps sqlx.Select
func (adapter *PostgresAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Update a single record
func (adapter *PostgresAdapter) Update(ent interface{}, columns ...string) error {
	return update(adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *PostgresAdapter) Insert(ent interface{}) (int, error) {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	if v, ok := ent.(*gotransit.FareAttribute); ok {
		v.Transfers = "0" // TODO: Keep?
	}
	table := getTableName(ent)
	cols, vals, err := getInsert(ent)
	if err != nil {
		return 0, err
	}
	eid := 0
	err = sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		Suffix("RETURNING \"id\"").
		PlaceholderFormat(sq.Dollar).
		RunWith(adapter.db).
		QueryRow().
		Scan(&eid)
	if err != nil {
		return 0, err
	}
	if v, ok := ent.(canSetID); ok {
		v.SetID(eid)
	}
	return eid, err
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
	cols, _, err := getInsert(sts[0])
	table := "gtfs_stop_times"
	q := sq.Insert(table).Columns(cols...)
	for _, d := range sts {
		_, vals, _ := getInsert(d)
		q = q.Values(vals...)
	}
	_, err = q.PlaceholderFormat(sq.Dollar).RunWith(adapter.db).Exec()
	return err
}
