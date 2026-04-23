package tldb

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var bufferSize = 1000

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("db", tags.ToSnakeCase))

type Ext interface {
	sqlx.Ext
	sqlx.QueryerContext
	sqlx.ExecerContext
	// QueryRowContext is missing from sqlx.QueryerContext, despite having QueryRowContext
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

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
	result, err := adapter.Sqrl().
		Update(table).
		Where("id = ?", entid).
		Suffix("returning id").
		SetMap(colmap).
		ExecContext(ctx)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return errors.New("failed to update record")
	}
	return nil
}

// check for error and panic
// TODO: don't do this. panic is bad.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func getFvids(dburl string) ([]int, string, error) {
	// Split on "?" rather than using url.Parse so that driver-specific forms
	// like "sqlite3://:memory:" (where ":memory:" is not a valid host:port)
	// are accepted.
	fvids := []int{}
	base, query, hasQuery := strings.Cut(dburl, "?")
	if !hasQuery {
		return fvids, dburl, nil
	}
	vars, err := url.ParseQuery(query)
	if err != nil {
		return nil, "", err
	}
	if a, ok := vars["fvid"]; ok {
		for _, v := range a {
			fvid, err := strconv.Atoi(v)
			if err != nil {
				return nil, "", errors.New("invalid feed version id")
			}
			fvids = append(fvids, fvid)
		}
	}
	delete(vars, "fvid")
	newQuery := vars.Encode()
	if newQuery == "" {
		return fvids, base, nil
	}
	return fvids, base + "?" + newQuery, nil
}
