package model

import (
	"log"
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/tldb"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
var mapper = reflectx.NewMapperFunc("db", toSnakeCase)

// TODO: replace with middleware or configuration
var DB canBeginx

func MustOpenDB(url string) canBeginx {
	db, err := sqlx.Open("postgres", url)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	db.Mapper = mapper
	db = db.Unsafe()
	return tldb.NewQueryLogger(db)
}

func Sqrl(db sqlx.Ext) sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db)
}

type canBeginx interface {
	sqlx.Ext
	Beginx() (*sqlx.Tx, error)
}

func Tx(cb func(sqlx.Ext) error) error {
	tx, err := DB.Beginx()
	if err != nil {
		panic(err)
	}
	if err := cb(tx); err != nil {
		if errTx := tx.Rollback(); errTx != nil {
			panic(errTx)
		}
		return err
	}
	return tx.Commit()
}

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
