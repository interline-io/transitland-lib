package tldb

import (
	"errors"

	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/jmoiron/sqlx/reflectx"
)

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("db", tags.ToSnakeCase))

type hasTableName interface {
	TableName() string
}

type canSetID interface {
	SetID(int)
}

type canGetID interface {
	GetID() int
}

type canUpdateTimestamps interface {
	UpdateTimestamps()
}

type canSetFeedVersion interface {
	SetFeedVersionID(int)
}

func getTableName(ent interface{}) string {
	if v, ok := ent.(hasTableName); ok {
		return v.TableName()
	}
	return ""
}

func contains(a string, b []string) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}

// find a single record.
func find(adapter Adapter, dest interface{}) error {
	entid := 0
	if v, ok := dest.(canGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", entid).ToSql()
	if err != nil {
		return err
	}
	return adapter.Get(dest, qstr, args...)
}

// update a single record.
func update(adapter Adapter, ent interface{}, columns ...string) error {
	entid := 0
	if v, ok := ent.(canGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	table := getTableName(ent)
	header, err := MapperCache.GetHeader(ent)
	if err != nil {
		return err
	}
	vals, err := MapperCache.GetInsert(ent, header)
	if err != nil {
		return err
	}
	colmap := make(map[string]interface{})
	for i, col := range header {
		if len(columns) > 0 && !contains(col, columns) {
			continue
		}
		colmap[col] = vals[i]
	}
	_, err2 := adapter.Sqrl().
		Update(table).
		Where("id = ?", entid).
		SetMap(colmap).
		Exec()
	return err2
}
