package find

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/jmoiron/sqlx"
)

// MAXLIMIT .
const MAXLIMIT = 1000

// MustSelect runs a query or panics.
func MustSelect(db sqlx.Ext, q sq.SelectBuilder, dest interface{}) {
	qstr, qargs := q.MustSql()
	if err := sqlx.Select(db, dest, db.Rebind(qstr), qargs...); err != nil {
		panic(err)
	}
}

func checkLimit(limit *int) uint64 {
	if limit == nil {
		return MAXLIMIT
	} else if *limit >= MAXLIMIT {
		return MAXLIMIT
	}
	return uint64(*limit)
}

func checkAfter(after *int) int {
	if after == nil {
		return 0
	}
	return *after
}

func checkBool(v *bool) bool {
	if v == nil || *v == false {
		return false
	}
	return true
}

func checkFloat(v *float64, min float64, max float64) float64 {
	if v == nil || *v < min {
		return min
	} else if *v > max {
		return max
	}
	return *v
}

func atoi(v string) int {
	a, _ := strconv.Atoi(v)
	return a
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

func escapeWordsWithSuffix(v string, sfx string) []string {
	f := strings.Fields(v)
	for i, s := range f {
		f[i] = alphanumeric(s) + sfx
	}
	return f
}

func tsQuery(s string) (rank sq.Sqlizer, wc sq.Sqlizer) {
	s = strings.TrimSpace(s)
	words := []string{}
	for _, v := range escapeWordsWithSuffix(s, ":*") {
		// Minimum length 2 characters
		if len(v) > 1 {
			words = append(words, v)
		}
	}
	wordstsq := strings.Join(words, " & ")
	rank = sq.Expr("ts_rank_cd(textsearch,to_tsquery('tl',?)) as search_rank", wordstsq)
	wc = sq.Expr("t.textsearch @@ to_tsquery('tl',?)", wordstsq)
	return rank, wc
}

func lateralWrap(q sq.SelectBuilder, parent string, pkey string, ckey string, pids []int) sq.SelectBuilder {
	q = q.Where(fmt.Sprintf("%s = parent.%s", ckey, pkey))
	q2 := sq.StatementBuilder.
		Select("t.*").
		From(parent + " parent").
		JoinClause(q.Prefix("JOIN LATERAL (").Suffix(") t on true")).
		Where(sq.Eq{"parent." + pkey: pids})
	return q2
}

func quickSelect(table string, limit *int, after *int, ids []int) sq.SelectBuilder {
	return quickSelectOrder(table, limit, after, ids, "id")
}

func quickSelectOrder(table string, limit *int, after *int, ids []int, order string) sq.SelectBuilder {
	q := sq.StatementBuilder.
		Select("t.*").
		From(table + " t").
		Limit(checkLimit(limit))
	if order != "" {
		q = q.OrderBy("t." + order)
	}
	if len(ids) > 0 {
		q = q.Where(sq.Eq{"t.id": ids})
	}
	if after != nil {
		q = q.Where(sq.Gt{"t.id": *after})
	}
	return q
}

func FindFeedVersions(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.FeedVersionFilter) (ents []*model.FeedVersion, err error) {
	MustSelect(model.DB, FeedVersionSelect(limit, after, ids, where), &ents)
	return ents, nil
}

func FindFeeds(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.FeedFilter) (ents []*model.Feed, err error) {
	MustSelect(model.DB, FeedSelect(limit, after, ids, where), &ents)
	return ents, nil
}

func FindAgencies(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.AgencyFilter) (ents []*model.Agency, err error) {
	q := AgencySelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func FindRoutes(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.RouteFilter) (ents []*model.Route, err error) {
	q := RouteSelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func FindTrips(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.TripFilter) (ents []*model.Trip, err error) {
	q := TripSelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func FindStops(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.StopFilter) (ents []*model.Stop, err error) {
	q := StopSelect(limit, after, ids, where)
	if len(ids) == 0 && (where == nil || where.FeedVersionSha1 == nil) {
		q = q.Where(sq.NotEq{"active": nil})
	}
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func FindOperators(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.OperatorFilter) (ents []*model.Operator, err error) {
	q := OperatorSelect(limit, after, ids, where)
	MustSelect(model.DB, q, &ents)
	return ents, nil
}
