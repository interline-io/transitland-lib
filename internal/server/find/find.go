package find

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	sq "github.com/Masterminds/squirrel"
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

// az09 removes any character that is not a a-z or 0-9 or _
func az09(v string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9_]+")
	return reg.ReplaceAllString(v, "")
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
	rank = sq.Expr("ts_rank_cd(t.textsearch,to_tsquery('tl',?)) as search_rank", wordstsq)
	wc = sq.Expr("t.textsearch @@ to_tsquery('tl',?)", wordstsq)
	return rank, wc
}

func lateralWrap(q sq.SelectBuilder, parent string, pkey string, ckey string, pids []int) sq.SelectBuilder {
	parent = az09(parent)
	pkey = az09(pkey)
	ckey = az09(ckey)
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
	table = az09(table)
	order = az09(order)
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
