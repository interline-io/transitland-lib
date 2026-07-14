package dbfinder

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// These limits have high maximums just for query safety
// Normal request handler limits are set in the GraphQL resolver layer

// Maximum query result limitss

// Other maximum query limits
const (
	FINDER_MAXLIMIT     = 100_000
	FINDER_DEFAULTLIMIT = 100_000
)

func finderCheckLimit(limit *int) uint64 {
	return finderCheckLimitMax(limit, FINDER_MAXLIMIT)
}

func finderCheckLimitMax(limit *int, maxLimit int) uint64 {
	alim := FINDER_DEFAULTLIMIT
	if limit != nil {
		alim = *limit
	}
	if alim < 0 {
		alim = 0
	} else if alim >= maxLimit {
		alim = maxLimit
	}
	return uint64(alim)
}

type Finder struct {
	Clock      clock.Clock
	db         tldb.Ext
	adminCache *adminCache
	fvslCache  *clock.ServiceWindowCache
}

func NewFinder(db tldb.Ext) *Finder {
	finder := &Finder{
		db:        db,
		fvslCache: clock.NewServiceWindowCache(db),
	}
	return finder
}

func (f *Finder) LoadAdmins(ctx context.Context) error {
	log.For(ctx).Trace().Msg("loading admins")
	c, err := newAdminCache(context.Background(), f.db)
	if err != nil {
		return err
	}
	f.adminCache = c
	return nil
}

func (f *Finder) PermFilter(ctx context.Context) *model.PermFilter {
	return model.PermsForContext(ctx)
}

// Helpers

func logErr(ctx context.Context, err error) error {
	if ctx.Err() == context.Canceled {
		return nil
	}
	log.For(ctx).Error().Err(err).Msg("query failed")
	return errors.New("database error")
}

func logExtendErr(ctx context.Context, size int, err error) []error {
	errs := make([]error, size)
	if ctx.Err() == context.Canceled {
		return errs
	}
	log.For(ctx).Error().Err(err).Msg("query failed")
	for i := 0; i < len(errs); i++ {
		errs[i] = errors.New("database error")
	}
	return errs
}

func arrangeBy[K comparable, T any](keys []K, ents []T, cb func(T) K) []T {
	bykey := map[K]T{}
	for _, ent := range ents {
		bykey[cb(ent)] = ent
	}
	ret := make([]T, len(keys))
	for idx, key := range keys {
		ret[idx] = bykey[key]
	}
	return ret
}

func arrangeMap[K comparable, T any, O any](keys []K, ents []T, cb func(T) (K, O)) []O {
	bykey := map[K]O{}
	for _, ent := range ents {
		k, o := cb(ent)
		bykey[k] = o
	}
	ret := make([]O, len(keys))
	for idx, key := range keys {
		ret[idx] = bykey[key]
	}
	return ret
}

func arrangeGroup[K comparable, T any](keys []K, ents []T, cb func(T) K) [][]T {
	bykey := map[K][]T{}
	for _, ent := range ents {
		k := cb(ent)
		bykey[k] = append(bykey[k], ent)
	}
	ret := make([][]T, len(keys))
	for idx, key := range keys {
		ret[idx] = bykey[key]
	}
	return ret
}

func nilOr[T any, PT *T](v PT, def T) T {
	if v == nil {
		return def
	}
	return *v
}

func ptr[T any, PT *T](v T) PT {
	a := v
	return &a
}

func kebabize(a string) string {
	return strings.ReplaceAll(strings.ToLower(a), "_", "-")
}

func tzTruncate(s time.Time, loc *time.Location) *tt.Date {
	if loc == nil {
		log.Error().Msg("tzTruncate: loc is nil, set to UTC")
		loc = time.UTC
	}
	return ptr(tt.NewDate(time.Date(s.Year(), s.Month(), s.Day(), 0, 0, 0, 0, loc)))
}

func checkFloat(v *float64, min float64, max float64) float64 {
	if v == nil || *v < min {
		return min
	} else if *v > max {
		return max
	}
	return *v
}

// unicode aware remove all non-alphanumeric characters
// this is not for escaping sql; just for preparing to_tsquery
func alphanumeric(v string) string {
	ret := []rune{}
	for _, ch := range v {
		if unicode.IsSpace(ch) {
			ret = append(ret, ' ')
		} else if unicode.IsDigit(ch) || unicode.IsLetter(ch) {
			ret = append(ret, ch)
		}
	}
	return string(ret)
}

// az09 removes any character that is not a a-z or 0-9 or _ or .
func az09(v string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\.]+`)
	return reg.ReplaceAllString(v, "")
}

func escapeWordsWithSuffix(v string, sfx string) []string {
	var ret []string
	for _, s := range strings.Fields(v) {
		aa := alphanumeric(s)
		// Minimum length 2 characters
		if len(aa) > 1 {
			ret = append(ret, aa+sfx)
		}
	}
	return ret
}

// joinImported restricts a select over imported entity rows to feed versions whose import
// completed. It is applied to the entity selects: agency, place, route, stop, trip, shape and
// pathway.
//
// Excluded: an import still running (in_progress), and an import that failed and left rows
// behind (success = false). Both columns are load-bearing and neither implies the other -- a
// failed import ends with in_progress = false, and an unimport runs against a feed version
// that still has success = true.
//
// Import and unimport do not run in a transaction -- one spanning millions of entity rows
// would pin the xmin horizon and stall autovacuum database-wide -- so they commit as they go.
// This gate is the only thing keeping their partial states unreachable.
//
// Keyed off feed_versions.id rather than the entity table, so one clause covers every table:
// each of these selects already inner joins feed_versions on its feed_version_id. Applied
// unconditionally, including on the active path -- a feed version being active does not by
// itself mean its data is intact.
//
// Feed version and validation report selects deliberately do not use this: they describe the
// import itself, and must still see failed and in-progress ones.
//
// Not applied everywhere it needs to be. These read tables in
// dmfr.GetFeedVersionTables().ImportedTables() without a gate, so they can still return rows
// from a partial import or unimport:
//
//   - The FeedVersion child resolvers -- geometry, feed_infos, locations, location_groups,
//     booking_rules -- and the stop_times reachable under locations. They hang off the
//     feed version select, which is ungated by design, so nothing upstream covers them.
//   - The operator and feed spatial filters, and the census stop buffer, which reach
//     imported tables through feed_states or by raw entity id. These cannot use this helper
//     as written: it keys off feed_versions.id, which is not in scope for them.
//
// Everything else that reads those tables is keyed by ids that can only have come from one of
// the gated selects above.
func joinImported(q sq.SelectBuilder) sq.SelectBuilder {
	return q.Join("feed_version_gtfs_imports fvgi on fvgi.feed_version_id = feed_versions.id and fvgi.success and not fvgi.in_progress")
}

func pfJoinCheck(q sq.SelectBuilder, permFilter *model.PermFilter) sq.SelectBuilder {
	q = q.Join("feed_states fsp on fsp.feed_id = current_feeds.id").
		Where(sq.Eq{"current_feeds.deleted_at": nil})
	sqOr := sq.Or{}
	sqOr = append(sqOr, sq.Expr("fsp.public = true"))
	sqOr = append(sqOr, In("fsp.feed_id", permFilter.GetAllowedFeeds()))
	if permFilter.GetIsGlobalAdmin() {
		sqOr = append(sqOr, sq.Expr("1=1")) // Global admin: allow all rows
	}
	return q.Where(sqOr)
}

func pfJoinCheckFv(q sq.SelectBuilder, permFilter *model.PermFilter) sq.SelectBuilder {
	q = q.Join("feed_states fsp on fsp.feed_id = feed_versions.feed_id").
		Where(sq.Eq{"current_feeds.deleted_at": nil}).
		Where(sq.Eq{"feed_versions.deleted_at": nil})
	sqOr := sq.Or{}
	sqOr = append(sqOr, sq.Expr("fsp.public = true"))
	sqOr = append(sqOr, In("feed_versions.feed_id", permFilter.GetAllowedFeeds()))
	sqOr = append(sqOr, In("feed_versions.id", permFilter.GetAllowedFeedVersions()))
	if permFilter.GetIsGlobalAdmin() {
		sqOr = append(sqOr, sq.Expr("1=1")) // Global admin: allow all rows
	}
	return q.Where(sqOr)
}

func In[T any](col string, val []T) sq.Sqlizer {
	if len(val) == 0 {
		return sq.Eq{col: val}
	}
	return sq.Expr(
		fmt.Sprintf("%s = ANY(?)", az09(col)),
		val,
	)
}

func tsTableQuery(table string, s string) (rank sq.Sqlizer, wc sq.Sqlizer) {
	s = strings.TrimSpace(s)
	words := append([]string{}, escapeWordsWithSuffix(s, ":*")...)
	wordstsq := strings.Join(words, " & ")
	rank = sq.Expr(
		fmt.Sprintf(`ts_rank_cd("%s".textsearch,to_tsquery('tl',?)) as search_rank`, az09(table)),
		wordstsq,
	)
	wc = sq.Expr(
		fmt.Sprintf(`"%s".textsearch @@ to_tsquery('tl',?)`, az09(table)),
		wordstsq,
	)

	return rank, wc
}

func lateralWrap(q sq.SelectBuilder, outerTable string, outerKey string, innerTable string, innerKey string, outerIds []int) sq.SelectBuilder {
	outerTable = az09(outerTable)
	outerKey = az09(outerKey)
	innerTable = az09(innerTable)
	innerKey = az09(innerKey)
	qInner := q.Where(fmt.Sprintf("%s.%s = out.%s", innerTable, innerKey, outerKey))
	q2 := sq.StatementBuilder.
		Select("t.*").
		From(outerTable + " out").
		JoinClause(qInner.Prefix("JOIN LATERAL (").Suffix(") t on true")).
		Where(In("out."+outerKey, outerIds))
	return q2
}

func quickSelect(table string, limit *int, after *model.Cursor, ids []int) sq.SelectBuilder {
	return quickSelectOrder(table, limit, after, ids, "id")
}

func quickSelectOrder(table string, limit *int, after *model.Cursor, ids []int, order string) sq.SelectBuilder {
	table = az09(table)
	order = az09(order)
	q := sq.StatementBuilder.
		Select("*").
		From(table).
		Limit(finderCheckLimit(limit))
	if order != "" {
		q = q.OrderBy(order)
	}
	if len(ids) > 0 {
		q = q.Where(In("id", ids))
	}
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"id": after.ID})
	}
	return q
}

type UseActive struct {
	active       bool
	materialized bool
}

// Active returns true if u is non-nil and active is true
func (u *UseActive) Active() bool {
	return u != nil && u.active
}

// UseTable returns the materialized table name if conditions are met, otherwise returns the base table name
func (u *UseActive) UseTable(baseTable, materializedTable string) string {
	if u != nil && u.active && u.materialized {
		return materializedTable
	}
	return baseTable
}
