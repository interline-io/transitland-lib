package tldb

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func init() {
	// Register driver
	adapterFactories["postgres"] = func(dburl string) Adapter { return &PostgresAdapter{DBURL: dburl} }
	adapterFactories["postgresql"] = func(dburl string) Adapter { return &PostgresAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (adapters.Reader, error) { return NewReader(url) }
	ext.RegisterReader("postgres", r)
	ext.RegisterReader("postgresql", r)
	w := func(url string) (adapters.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("postgres", w)
	ext.RegisterWriter("postgresql", w)
}

// PostgresAdapter connects to a Postgres/PostGIS database.
type PostgresAdapter struct {
	DBURL string
	db    Ext
}

func NewPostgresAdapterFromDBX(db Ext) *PostgresAdapter {
	return &PostgresAdapter{DBURL: "", db: db}
}

// Open the adapter.
func (adapter *PostgresAdapter) Open() error {
	if adapter.db != nil {
		return nil
	}
	pool, err := pgxpool.New(context.TODO(), adapter.DBURL)
	if err != nil {
		return err
	}
	db := sqlx.NewDb(stdlib.OpenDBFromPool(pool), "pgx")
	db.Mapper = MapperCache.Mapper
	adapter.db = &QueryLogger{Ext: db.Unsafe()}
	return nil
}

// Close the adapter.
func (adapter *PostgresAdapter) Close() error {
	return nil
}

// Create an initial database schema.
func (adapter *PostgresAdapter) Create() error {
	ctx := context.TODO()
	if _, err := adapter.db.ExecContext(ctx, "SELECT * FROM feed_versions LIMIT 0"); err == nil {
		return nil
	}
	return errors.New("please run postgres migrations manually")
}

// DBX returns sqlx.Ext
func (adapter *PostgresAdapter) DBX() Ext {
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
	if err := cb(&PostgresAdapter{DBURL: adapter.DBURL, db: &QueryLogger{Ext: tx}}); err != nil {
		if commit {
			if errTx := tx.Rollback(); errTx != nil {
				return errTx
			}
		}
		return err
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
func (adapter *PostgresAdapter) Find(ctx context.Context, dest interface{}) error {
	return find(ctx, adapter, dest)
}

// Get wraps sqlx.Get
func (adapter *PostgresAdapter) Get(ctx context.Context, dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.GetContext(context.TODO(), adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Select wraps sqlx.Select
func (adapter *PostgresAdapter) Select(ctx context.Context, dest interface{}, qstr string, args ...interface{}) error {
	return sqlx.SelectContext(ctx, adapter.db, dest, adapter.db.Rebind(qstr), args...)
}

// Update a single entity.
func (adapter *PostgresAdapter) Update(ctx context.Context, ent interface{}, columns ...string) error {
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	return update(ctx, adapter, ent, columns...)
}

// Insert builds and executes an insert statement for the given entity.
func (adapter *PostgresAdapter) Insert(ctx context.Context, ent interface{}) (int, error) {
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
		err = q.Suffix(`RETURNING "id"`).QueryRowContext(ctx).Scan(&eid)
	} else {
		_, err = q.ExecContext(ctx)
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
func (adapter *PostgresAdapter) MultiInsert(ctx context.Context, ents []interface{}) ([]int, error) {
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
			_, err = q.ExecContext(ctx)
			for range batch {
				retids = append(retids, 0)
			}
		}
	}
	for i := 0; i < len(ents); i++ {
		ent := ents[i]
		if v, ok := ent.(canSetID); ok {
			v.SetID(retids[i])
		}
	}
	return retids, err
}
