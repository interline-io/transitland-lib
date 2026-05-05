package tlcsv

import (
	"cmp"
	"sort"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/internal/tags"
)

// compareCells compares two raw CSV cells under the given kind. Empty or
// unparseable values rank as the greatest value (NULLS LAST in asc,
// NULLS FIRST in desc — the SQL default).
func compareCells(a, b string, kind tags.SortKind) int {
	switch kind {
	case tags.SortKindInt:
		return compareNumeric(a, b, func(s string) (int64, error) {
			return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		})
	case tags.SortKindFloat:
		return compareNumeric(a, b, func(s string) (float64, error) {
			return strconv.ParseFloat(strings.TrimSpace(s), 64)
		})
	case tags.SortKindDate:
		// GTFS dates are YYYYMMDD; lexicographic compare is correct.
		return compareEmptyAsGreatest(a, b)
	default:
		return strings.Compare(a, b)
	}
}

func compareNumeric[T cmp.Ordered](a, b string, parse func(string) (T, error)) int {
	av, aerr := parse(a)
	bv, berr := parse(b)
	switch {
	case aerr != nil && berr != nil:
		return 0
	case aerr != nil:
		return 1
	case berr != nil:
		return -1
	}
	return cmp.Compare(av, bv)
}

// compareEmptyAsGreatest treats empty strings as +∞: last in asc, first in desc.
func compareEmptyAsGreatest(a, b string) int {
	switch {
	case a == "" && b == "":
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	}
	return strings.Compare(a, b)
}

type sortKey struct {
	idx  int
	kind tags.SortKind
}

func resolveHeaderKeys(header []string, cols []*tags.FieldInfo) []sortKey {
	var keys []sortKey
	for _, fi := range cols {
		for i, h := range header {
			if h == fi.Name {
				keys = append(keys, sortKey{idx: i, kind: fi.Kind})
				break
			}
		}
	}
	return keys
}

func sortRows(rows [][]string, keys []sortKey, descending bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		for _, k := range keys {
			a, b := "", ""
			if k.idx < len(rows[i]) {
				a = rows[i][k.idx]
			}
			if k.idx < len(rows[j]) {
				b = rows[j][k.idx]
			}
			c := compareCells(a, b, k.kind)
			if c == 0 {
				continue
			}
			if descending {
				return c > 0
			}
			return c < 0
		}
		return false
	})
}
