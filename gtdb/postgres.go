package gtdb

import (
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// PostgresAdapter .
type PostgresAdapter struct {
	DBURL string
	db    *sqlx.DB
}

func (adapter *PostgresAdapter) Open() error {
	db, err := sqlx.Open("postgres", adapter.DBURL)
	if err != nil {
		return err
	}
	adapter.db = db
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	return nil
}

func (adapter *PostgresAdapter) Close() error {
	return adapter.db.Close()
}

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

func (adapter *PostgresAdapter) DB() *sqlx.DB {
	return adapter.db
}

func (adapter *PostgresAdapter) DBX() *sqlx.DB {
	return adapter.db
}

func (adapter *PostgresAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db).PlaceholderFormat(sq.Dollar)
}

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

func (adapter *PostgresAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Get(dest, adapter.db.Rebind(qstr), args...)
}

func (adapter *PostgresAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Select(dest, adapter.db.Rebind(qstr), args...)
}

func (adapter *PostgresAdapter) Insert(ent interface{}) (int, error) {
	table := getTableName(ent)
	cols, vals, err := getInsert(adapter.db.Mapper, ent)
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

func (adapter *PostgresAdapter) BatchInsert(ents []gotransit.Entity) error {
	sts := []*gotransit.StopTime{}
	for _, ent := range ents {
		if st, ok := ent.(*gotransit.StopTime); ok {
			sts = append(sts, st)
		}
	}
	if len(sts) == 0 {
		return errors.New("presently only StopTimes are supported")
	}
	cols, _, err := getInsert(adapter.db.Mapper, sts[0])
	table := "gtfs_stop_times"
	q := sq.Insert(table).Columns(cols...)
	for _, d := range sts {
		_, vals, _ := getInsert(adapter.db.Mapper, d)
		q = q.Values(vals...)
	}
	_, err = q.PlaceholderFormat(sq.Dollar).RunWith(adapter.db).Exec()
	return err
}
