package gtdb

import (
	"reflect"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx/reflectx"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
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
			a := strings.Replace(wrap, "@", ",", -1)
			vals = append(vals, sq.Expr(a, v2))
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

type canSetID interface {
	SetID(int)
}
