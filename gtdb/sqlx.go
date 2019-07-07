package gtdb

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/gotransit"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func Insert(db *sqlx.DB, table string, ent interface{}) (int, error) {
	cols, vals := getInsert(ent)
	q := sq.
		Insert(table).
		Columns(cols...).
		Values(vals...).
		Suffix("RETURNING \"id\"").
		RunWith(db).
		PlaceholderFormat(sq.Dollar)
	if sql, _, err := q.ToSql(); err == nil {
		fmt.Println(sql)
	} else {
		return 0, err
	}
	eid := 0
	err := q.QueryRow().Scan(&eid)
	fmt.Println("eid:", eid, "err:", err)
	return eid, err
}

// getInsert returns column names and a slice of placeholders or squirrel expressions.
func getInsert(ent interface{}) ([]string, []interface{}) {
	cols := make([]string, 0)
	vals := make([]interface{}, 0)
	m := reflectx.NewMapperFunc("db", toSnakeCase)
	val := reflect.ValueOf(ent).Elem()
	fm := m.FieldMap(val)
	typ := reflect.TypeOf(ent)
	fields := m.TypeMap(typ)
	for k, v := range fm {
		v2 := v.Interface()
		fi := fields.GetByPath(k)
		if _, pk := fi.Options["pk"]; pk || fi.Name == "id" || fi.Name == "ID" || fi.Field.Tag == "" {
			continue
		}
		if wrap, ok := fi.Options["insert"]; ok {
			vals = append(vals, sq.Expr(wrap, v2))
		} else {
			vals = append(vals, v2)
		}
		// fmt.Println(fi.Name)
		// fmt.Printf("%#v\n\n", fi)
		cols = append(cols, fi.Name)
	}
	return cols, vals
}

// // getSelectStar returns column names for a SELECT * query.
// func getSelectStar(ent interface{}) []string {
// 	fmap := tags.GetStructTagMap(ent)
// 	cols := make([]string, len(fmap))
// 	for _, v := range fmap {
// 		a := v.DB
// 		if v.SelectWrap != "" {
// 			a = v.SelectWrap
// 		}
// 		if a != "-" {
// 			cols = append(cols, a)
// 		}
// 	}
// 	return cols
// }

type hasTableName interface {
	TableName() string
}

func getTableName(ent interface{}) string {
	if v, ok := ent.(hasTableName); ok {
		return v.TableName()
	}
	return ""
}

// SQLXAdapter .
type SQLXAdapter struct {
	DBURL string
	db    *sqlx.DB
}

func (adapter *SQLXAdapter) Insert(ent gotransit.Entity) error {
	table := getTableName(ent)
	if table == "" {
		return errors.New("no tablename")
	}
	eid, err := Insert(adapter.db, table, ent)
	ent.SetID(eid)
	return err
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
		_, err := Insert(adapter.db, "gtfs_stop_times", &d)
		if err != nil {
			// tx.Rollback()
			return err
		}
	}
	// tx.Commit()
	return nil
}
