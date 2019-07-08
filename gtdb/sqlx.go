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
	m     *reflectx.Mapper
}

func (adapter *SQLXAdapter) Open() error {
	db, err := sqlx.Open("postgres", adapter.DBURL)
	if err != nil {
		return err
	}
	adapter.db = db
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

func (adapter *SQLXAdapter) SetDB(*gorm.DB) {
}

func (adapter *SQLXAdapter) GeomEncoding() int {
	return 0
}

func (adapter *SQLXAdapter) Insert(table string, ent interface{}) (int, error) {
	if table == "" {
		table = getTableName(ent)
	}
	if table == "" {
		return 0, errors.New("no tablename")
	}
	// Keep the mapper to use cache.
	if adapter.m == nil {
		adapter.m = reflectx.NewMapperFunc("db", toSnakeCase)
	}
	cols, vals, err := getInsert(adapter.m, ent)
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

func (adapter *SQLXAdapter) BatchInsert(table string, ents []gotransit.Entity) error {
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
	if adapter.m == nil {
		adapter.m = reflectx.NewMapperFunc("db", toSnakeCase)
	}
	cols, _, err := getInsert(adapter.m, sts[0])
	q := sq.Insert(table).Columns(cols...)
	for _, d := range sts {
		_, vals, _ := getInsert(adapter.m, d)
		q = q.Values(vals...)
	}
	_, err = q.PlaceholderFormat(sq.Dollar).RunWith(adapter.db).Exec()
	if err != nil {
		panic(err)
	}
	return err
}
