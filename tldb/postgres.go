package tldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func init() {
	// Register driver
	adapters["postgres"] = func(dburl string) Adapter { return &PostgresAdapter{DBURL: dburl} }
	adapters["postgresql"] = func(dburl string) Adapter { return &PostgresAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	ext.RegisterReader("postgres", r)
	ext.RegisterReader("postgresql", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("postgres", w)
	ext.RegisterWriter("postgresql", w)
}

// PostgresAdapter connects to a Postgres/PostGIS database.
type PostgresAdapter struct {
	pgxpool *pgxpool.Pool
	DBURL   string
	db      sqlx.Ext
}

func NewPostgresAdapterFromDBX(db sqlx.Ext) *PostgresAdapter {
	return &PostgresAdapter{DBURL: "", db: db}
}

// Open the adapter.
func (adapter *PostgresAdapter) Open() error {
	if adapter.db != nil {
		return nil
	}
	pool, err := pgxpool.New(context.Background(), adapter.DBURL)
	if err != nil {
		return err
	}
	db := sqlx.NewDb(stdlib.OpenDBFromPool(pool), "pgx")
	db.Mapper = MapperCache.Mapper
	adapter.db = &QueryLogger{Ext: db.Unsafe()}
	adapter.pgxpool = pool
	return nil
}

// Close the adapter.
func (adapter *PostgresAdapter) Close() error {
	return nil
}

// Create an initial database schema.
func (adapter *PostgresAdapter) Create() error {
	if _, err := adapter.db.Exec("SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	return errors.New("please run postgres migrations manually")
}

// DBX returns sqlx.Ext
func (adapter *PostgresAdapter) DBX() sqlx.Ext {
	return adapter.db
}

// Tx runs a callback inside a transaction.
func (adapter *PostgresAdapter) Tx(cb func(Adapter) error) error {
	var err error
	var tx *sqlx.Tx
	// Special check for wrapped connections
	commit := false
	switch a := adapter.db.(type) {
	case *sqlx.Tx:
		tx = a
	case *QueryLogger:
		if b, ok := a.Ext.(*sqlx.Tx); ok {
			tx = b
		}
	}
	// If we aren't already in a transaction, begin one, and commit at end
	if a, ok := adapter.db.(canBeginx); tx == nil && ok {
		tx, err = a.Beginx()
		commit = true
	}
	if err != nil {
		return err
	}
	adapter2 := &PostgresAdapter{DBURL: adapter.DBURL, db: &QueryLogger{Ext: tx}}
	if err2 := cb(adapter2); err2 != nil {
		if commit {
			if errTx := tx.Rollback(); errTx != nil {
				return errTx
			}
		}
		return err2
	}
	if commit {
		return tx.Commit()
	}
	return nil
}

// Sqrl returns a properly configured Squirrel StatementBuilder.
func (adapter *PostgresAdapter) Sqrl() sq.StatementBuilderType {
	return sq.StatementBuilder.RunWith(adapter.db).PlaceholderFormat(sq.Dollar)
}

// TableExists returns true if the requested table exists
func (adapter *PostgresAdapter) TableExists(t string) (bool, error) {
	qstr := `SELECT EXISTS ( SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = ?);`
	exists := false
	err := sqlx.Get(adapter.db, &exists, adapter.db.Rebind(qstr), t)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists, err
}

// Find finds a single entity based on the EntityID()
func (adapter *PostgresAdapter) Find(dest interface{}) error {
	return find(adapter, dest)
}

// Get wraps sqlx.Get
func (adapter *PostgresAdapter) Get(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Get(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Select wraps sqlx.Select
func (adapter *PostgresAdapter) Select(dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.Select(adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Update a single entity.
func (adapter *PostgresAdapter) Update(ent interface{}, columns ...string) error {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	return update(adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *PostgresAdapter) Insert(ent interface{}) (int, error) {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	table := getTableName(ent)
	header, err := MapperCache.GetHeader(ent)
	if err != nil {
		return 0, err
	}
	vals, err := MapperCache.GetInsert(ent, header)
	if err != nil {
		return 0, err
	}
	var eid sql.NullInt64
	q := adapter.Sqrl().
		Insert(table).
		Columns(header...).
		Values(vals...)
	if _, ok := ent.(canSetID); ok {
		err = q.Suffix(`RETURNING "id"`).QueryRow().Scan(&eid)
	} else {
		_, err = q.Exec()
	}
	if err != nil {
		return 0, err
	}
	if v, ok := ent.(canSetID); ok {
		v.SetID(int(eid.Int64))
	}
	return int(eid.Int64), err
}

// MultiInsert builds and executes a multi-insert statement for the given entities.
func (adapter *PostgresAdapter) MultiInsert(ents []interface{}) ([]int, error) {
	retids := []int{}
	if len(ents) == 0 {
		return retids, nil
	}
	for _, ent := range ents {
		if v, ok := ent.(canUpdateTimestamps); ok {
			v.UpdateTimestamps()
		}
	}
	header, err := MapperCache.GetHeader(ents[0])
	table := getTableName(ents[0])
	_, setid := ents[0].(canSetID)
	batchSize := 65536 / (len(header) + 1)
	for i := 0; i < len(ents); i += batchSize {
		batch := ents[i:min(i+batchSize, len(ents))]
		q := adapter.Sqrl().Insert(table).Columns(header...)
		for _, d := range batch {
			vals, _ := MapperCache.GetInsert(d, header)
			q = q.Values(vals...)
		}
		if setid {
			q = q.Suffix(`RETURNING "id"`)
			rows, err := q.Query()
			if err != nil {
				return retids, err
			}
			defer rows.Close()
			var eid sql.NullInt64
			for rows.Next() {
				err := rows.Scan(&eid)
				if err != nil {
					return retids, err
				}
				retids = append(retids, int(eid.Int64))
			}
		} else {
			_, err = q.Exec()
		}
	}
	return retids, err
}

// CopyInsert inserts data using COPY.
func (adapter *PostgresAdapter) CopyInsert(ents []interface{}) error {
	if len(ents) == 0 {
		return nil
	}
	for _, ent := range ents {
		if v, ok := ent.(canUpdateTimestamps); ok {
			v.UpdateTimestamps()
		}
	}

	// Create a temporary table
	table := getTableName(ents[0])
	tableTmp := fmt.Sprintf("tmp_%s_t%d_r%d", table, time.Now().Unix(), rand.Int31())
	header, err := MapperCache.GetHeader(ents[0])
	if err != nil {
		return err
	}

	// Convert to rows
	valRows := make([][]any, len(ents))
	for i, d := range ents {
		vals, err := MapperCache.GetInsert(d, header)
		if err != nil {
			return err
		}
		valRows[i] = vals
	}

	// Prepare txn
	ctx := context.Background()
	pgxtx, err := adapter.pgxpool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	// Create temp table
	if _, err := pgxtx.Exec(ctx, fmt.Sprintf("CREATE UNLOGGED TABLE %s (LIKE %s)", tableTmp, table)); err != nil {
		return err
	}

	// Copy into temp table
	rowCount, err := pgxtx.CopyFrom(ctx, pgx.Identifier{tableTmp}, header, pgx.CopyFromRows(valRows))
	if err != nil {
		return err
	}
	log.Trace().Int64("count", rowCount).Str("table", tableTmp).Msg("copied rows into temp table")
	if err := pgxtx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("ok")

	// Must run in transaction
	execErr := adapter.Tx(func(atx Adapter) error {
		// Insert temp table into main table
		_, copyErr := atx.DBX().Exec(fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", table, tableTmp))
		return copyErr
	})

	// Drop temporary table regardless of outcome
	adapter.pgxpool.Exec(ctx, fmt.Sprintf("DROP TABLE %s", tableTmp))

	// Return error from main INSERT
	return execErr
}
