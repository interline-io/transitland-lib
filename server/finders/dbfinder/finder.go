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

// Maximum query result limit
var MAXLIMIT = 100_000

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

func (f *Finder) DBX() tldb.Ext {
	return f.db
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

func checkLimit(limit *int) uint64 {
	return checkRange(limit, 0, MAXLIMIT)
}

func checkRange(limit *int, min, max int) uint64 {
	if limit == nil {
		return uint64(max)
	} else if *limit >= max {
		return uint64(max)
	} else if *limit < min {
		return uint64(min)
	}
	return uint64(*limit)
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

func pfJoinCheck(q sq.SelectBuilder, permFilter *model.PermFilter) sq.SelectBuilder {
	q = q.Join("feed_states fsp on fsp.feed_id = current_feeds.id").
		Where(sq.Eq{"current_feeds.deleted_at": nil})
	sqOr := sq.Or{}
	sqOr = append(sqOr, sq.Expr("fsp.public = true"))
	sqOr = append(sqOr, In("fsp.feed_id", permFilter.GetAllowedFeeds()))
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
		Limit(checkLimit(limit))
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
