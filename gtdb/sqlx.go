package gtdb

import (
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
)

// SQLXAdapter .
type SQLXAdapter struct {
	DBURL string
	db    *sqlx.DB
}

func (adapter *SQLXAdapter) Insert(table string, ent interface{}) (int, error) {
	if table == "" {
		table = getTableName(ent)
	}
	if table == "" {
		return 0, errors.New("no tablename")
	}
	cols, vals := getInsert(ent)
	q := sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		Suffix("RETURNING \"id\"").
		RunWith(adapter.db).
		PlaceholderFormat(sq.Dollar)
	if sql, _, err := q.ToSql(); err == nil {
		fmt.Println(sql)
	} else {
		return 0, err
	}
	eid := 0
	err := q.QueryRow().Scan(&eid)
	fmt.Println("eid:", eid, "err:", err)
	if v, ok := ent.(canSetID); ok {
		v.SetID(eid)
	}
	return eid, err
}

func (adapter *SQLXAdapter) Find(ent gotransit.Entity) error {
	table := getTableName(ent)
	q := sq.Select("*").Where("id = ?", 18).From(table).Limit(1).PlaceholderFormat(sq.Dollar)
	sql, args, err := q.ToSql()
	if err != nil {
		return err
	}
	return adapter.db.Get(ent, sql, args...)
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

func (adapter *SQLXAdapter) BatchInsert(stoptimes *[]gotransit.StopTime) error {
	// db := adapter.db
	objArr := *stoptimes
	// tx := db.Begin()
	for _, d := range objArr {
		// err := tx.Create(&d).Error
		_, err := adapter.Insert("gtfs_stop_times", &d)
		if err != nil {
			// tx.Rollback()
			return err
		}
	}
	// tx.Commit()
	return nil
}
