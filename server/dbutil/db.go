package dbutil

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	sq "github.com/irees/squirrel"

	"github.com/interline-io/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
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

// ConfigureDB sets up common database configuration
func ConfigureDB(sqlDb *sql.DB) (*sqlx.DB, error) {
	db := sqlx.NewDb(sqlDb, "pgx")
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		log.Error().Err(err).Msgf("could not connect to database")
		return nil, err
	}
	db.Mapper = reflectx.NewMapperFunc("db", toSnakeCase)
	return db.Unsafe(), nil
}

func OpenPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Error().Err(err).Msg("could not open database pool")
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		log.Error().Err(err).Msg("could not ping database pool")
		return nil, err
	}
	return pool, nil
}

func OpenDBPool(ctx context.Context, url string) (*pgxpool.Pool, *sqlx.DB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	db, err := ConfigureDB(stdlib.OpenDBFromPool(pool))
	if err != nil {
		return nil, nil, err
	}
	return pool, db, nil
}

func OpenDB(url string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	return ConfigureDB(db.DB)
}

// Select runs a query and reads results into dest.
func Select(ctx context.Context, db sqlx.Ext, q sq.SelectBuilder, dest interface{}) error {
	q = q.PlaceholderFormat(sq.Dollar)
	qstr, qargs, err := q.ToSql()
	if err == nil {
		if a, ok := db.(sqlx.QueryerContext); ok {
			err = sqlx.SelectContext(ctx, a, dest, qstr, qargs...)
		} else {
			err = sqlx.Select(db, dest, qstr, qargs...)
		}
	}
	if ctx.Err() == context.Canceled {
		log.Trace().Err(err).Str("query", qstr).Interface("args", qargs).Msg("query canceled")
	} else if err != nil {
		log.Error().Err(err).Str("query", qstr).Interface("args", qargs).Msg("query failed")
	}
	return err
}

// Get runs a query and reads results into dest.
func Get(ctx context.Context, db sqlx.Ext, q sq.SelectBuilder, dest interface{}) error {
	q = q.PlaceholderFormat(sq.Dollar)
	qstr, qargs, err := q.ToSql()
	if err == nil {
		if a, ok := db.(sqlx.QueryerContext); ok {
			err = sqlx.GetContext(ctx, a, dest, qstr, qargs...)
		} else {
			err = sqlx.Get(db, dest, qstr, qargs...)
		}
	}
	if ctx.Err() == context.Canceled {
		log.Trace().Err(err).Str("query", qstr).Interface("args", qargs).Msg("query canceled")
	} else if err != nil {
		log.Error().Err(err).Str("query", qstr).Interface("args", qargs).Msg("query failed")
	}
	return err
}

// EscapeLike escapes SQL LIKE/ILIKE wildcard characters (%, _, and \) in a string
// and optionally adds prefix/suffix wildcards for pattern matching.
func EscapeLike(s string, prefix bool, suffix bool) string {
	// Escape backslash first, then the wildcards
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	if prefix {
		s = "%" + s
	}
	if suffix {
		s = s + "%"
	}
	return s
}
