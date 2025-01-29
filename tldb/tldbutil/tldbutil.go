package tldbutil

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

type Adapter = tldb.Adapter

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("db", tags.ToSnakeCase))

type CanBeginx interface {
	Beginx() (*sqlx.Tx, error)
}

type CanClose interface {
	Close() error
}

type HasTableName interface {
	TableName() string
}

type CanSetID interface {
	SetID(int)
}

type CanGetID interface {
	GetID() int
}

type CanUpdateTimestamps interface {
	UpdateTimestamps()
}

type CanSetFeedVersion interface {
	SetFeedVersionID(int)
}

func GetTableName(ent interface{}) string {
	if v, ok := ent.(HasTableName); ok {
		return v.TableName()
	}
	return ""
}

func Contains(a string, b []string) bool {
	for _, v := range b {
		if a == v {
			return true
		}
	}
	return false
}

// Find a single record.
func Find(ctx context.Context, adapter Adapter, dest interface{}) error {
	entid := 0
	if v, ok := dest.(CanGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(GetTableName(dest)).Where("id = ?", entid).ToSql()
	if err != nil {
		return err
	}
	return adapter.Get(ctx, dest, qstr, args...)
}

// update a single record.
func Update(ctx context.Context, adapter Adapter, ent interface{}, columns ...string) error {
	entid := 0
	if v, ok := ent.(CanGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	table := GetTableName(ent)
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
		if len(columns) > 0 && !Contains(col, columns) {
			continue
		}
		colmap[col] = vals[i]
	}
	result, err2 := adapter.Sqrl().
		Update(table).
		Where("id = ?", entid).
		Suffix("returning id").
		SetMap(colmap).
		ExecContext(ctx)
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return errors.New("failed to update record")
	}
	return err2
}
