package kvcache

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/jmoiron/sqlx"
)

// PostgresStore implements Store and the HashStore capability over two
// unlogged tables (tl_kv_cache, tl_kv_hash). Expiry is enforced on read;
// Sweep, run periodically via Start, reclaims lapsed rows.
type PostgresStore struct {
	// Timeout bounds each database operation (default 5s).
	Timeout time.Duration
	db      sqlx.ExtContext

	tickerLock sync.Mutex
	tickerStop chan struct{}
	tickerWg   sync.WaitGroup
}

var (
	_ Store     = (*PostgresStore)(nil)
	_ HashStore = (*PostgresStore)(nil)
)

// Statements are static literals with bound parameters; the store never
// assembles SQL from variables.
const (
	pgKVGet = `SELECT value FROM tl_kv_cache WHERE key = ? AND (expires_at IS NULL OR expires_at > now())`

	pgKVGetMulti = `SELECT key, value FROM tl_kv_cache WHERE key IN (?) AND (expires_at IS NULL OR expires_at > now())`

	pgKVSetTTL = `INSERT INTO tl_kv_cache (key, value, expires_at) VALUES (?, ?, now() + (? * interval '1 second'))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at`

	pgKVSet = `INSERT INTO tl_kv_cache (key, value, expires_at) VALUES (?, ?, NULL)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at`

	pgKVSweep = `DELETE FROM tl_kv_cache WHERE expires_at IS NOT NULL AND expires_at <= now()`

	pgHashSet = `INSERT INTO tl_kv_hash (hash_key, field, value) VALUES (?, ?, ?)
ON CONFLICT (hash_key, field) DO UPDATE SET value = EXCLUDED.value`

	pgHashGetAll = `SELECT field, value FROM tl_kv_hash WHERE hash_key = ?`
)

func NewPostgresStore(db sqlx.ExtContext) *PostgresStore {
	return &PostgresStore{
		Timeout: 5 * time.Second,
		db:      db,
	}
}

func (s *PostgresStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	var value []byte
	err := s.db.QueryRowxContext(rctx, s.db.Rebind(pgKVGet), key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (s *PostgresStore) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	if len(keys) == 0 {
		return ret, nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	query, args, err := sqlx.In(pgKVGetMulti, keys)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryxContext(rctx, s.db.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		ret[key] = value
	}
	return ret, rows.Err()
}

func (s *PostgresStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	// ttl <= 0 means no expiry; branch to a separate literal rather than
	// build the expires_at expression conditionally.
	if ttl > 0 {
		_, err := s.db.ExecContext(rctx, s.db.Rebind(pgKVSetTTL), key, value, ttl.Seconds())
		return err
	}
	_, err := s.db.ExecContext(rctx, s.db.Rebind(pgKVSet), key, value)
	return err
}

func (s *PostgresStore) HSet(ctx context.Context, key string, field string, value string) error {
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	_, err := s.db.ExecContext(rctx, s.db.Rebind(pgHashSet), key, field, value)
	return err
}

func (s *PostgresStore) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	ret := map[string]string{}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	rows, err := s.db.QueryxContext(rctx, s.db.Rebind(pgHashGetAll), key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var field, value string
		if err := rows.Scan(&field, &value); err != nil {
			return nil, err
		}
		ret[field] = value
	}
	return ret, rows.Err()
}

// Sweep deletes rows whose expiry has lapsed, returning the count removed.
func (s *PostgresStore) Sweep(ctx context.Context) (int64, error) {
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	res, err := s.db.ExecContext(rctx, pgKVSweep)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Start launches a background goroutine that Sweeps every interval. It is a
// no-op if already running and panics on a non-positive interval.
func (s *PostgresStore) Start(interval time.Duration) {
	if interval <= 0 {
		panic("kvcache: non-positive interval for Start")
	}
	s.tickerLock.Lock()
	defer s.tickerLock.Unlock()
	if s.tickerStop != nil {
		return
	}
	stop := make(chan struct{})
	s.tickerStop = stop
	s.tickerWg.Add(1)
	go func() {
		defer s.tickerWg.Done()
		for {
			select {
			case <-time.After(jitter(interval)):
				if _, err := s.Sweep(context.Background()); err != nil {
					log.Trace().Err(err).Msg("kvcache: postgres sweep failed")
				}
			case <-stop:
				return
			}
		}
	}()
}

// Stop halts the background sweeper and waits for an in-flight sweep.
func (s *PostgresStore) Stop() {
	s.tickerLock.Lock()
	defer s.tickerLock.Unlock()
	if s.tickerStop == nil {
		return
	}
	close(s.tickerStop)
	s.tickerStop = nil
	s.tickerWg.Wait()
}
