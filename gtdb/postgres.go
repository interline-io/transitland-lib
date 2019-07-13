package gtdb

import (
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

// SQLXAdapter .
type SQLXAdapter struct {
	DBURL string
	db    *sqlx.DB
}

func (adapter *SQLXAdapter) Open() error {
	db, err := sqlx.Open("postgres", adapter.DBURL)
	if err != nil {
		return err
	}
	adapter.db = db
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	return nil
}

func (adapter *SQLXAdapter) Close() error {
	return adapter.db.Close()
}

func (adapter *SQLXAdapter) Create() error {
	return nil
}

func (adapter *SQLXAdapter) DB() *gorm.DB {
	// Temporary compat...
	gormdb, err := gorm.Open("postgres", adapter.db.DB)
	if err != nil {
		panic(err)
	}
	return gormdb
}

func (adapter *SQLXAdapter) SetDB(db *gorm.DB) {
	ddb := db.DB()
	adapter.db = sqlx.NewDb(ddb, "postgres")
}

func (adapter *SQLXAdapter) DBX() *sqlx.DB {
	return adapter.db
}

func (adapter *SQLXAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db).PlaceholderFormat(sq.Dollar)
}

func (adapter *SQLXAdapter) Find(dest interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, _ := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	return adapter.Get(dest, qstr, args...)
}

func (adapter *SQLXAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Get(dest, adapter.db.Rebind(qstr), args...)
}

func (adapter *SQLXAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return adapter.db.Select(dest, adapter.db.Rebind(qstr), args...)
}

func (adapter *SQLXAdapter) Insert(ent interface{}) (int, error) {
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

func (adapter *SQLXAdapter) BatchInsert(ents []gotransit.Entity) error {
	sts := []*gotransit.StopTime{}
	for _, ent := range ents {
		if st, ok := ent.(*gotransit.StopTime); ok {
			sts = append(sts, st)
		} else {
			fmt.Printf("st: %#v\n", ent)
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
