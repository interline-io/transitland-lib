package dbfinder

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// Each keyed finder in model.Finder fails every key of a failed batch: with a
// generic database error when its query fails, or with the context's error when
// the request is canceled or times out.
func TestFinder_KeyedErrors(t *testing.T) {
	db := testutil.MustOpenTestDB(t)
	// Every query fails on a finished transaction.
	tx, err := db.Beginx()
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	timedOut, cancelTimedOut := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelTimedOut()
	tcs := []struct {
		name   string
		ctx    context.Context
		f      *Finder
		expect string
	}{
		{"query failed", context.Background(), NewFinder(tx), "database error"},
		{"request canceled", canceled, NewFinder(db), context.Canceled.Error()},
		{"request timed out", timedOut, NewFinder(db), context.DeadlineExceeded.Error()},
	}
	methods := finderMethods(isKeyedFinder)
	require.NotEmpty(t, methods, "expected keyed finders in model.Finder")
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			for _, m := range methods {
				t.Run(m.Name, func(t *testing.T) {
					errs, _ := callFinder(tc.ctx, tc.f, m)[1].Interface().([]error)
					require.Len(t, errs, finderTestKeys)
					for _, err := range errs {
						assert.EqualError(t, err, tc.expect)
					}
				})
			}
		})
	}
}

// No grouped finder in model.Finder returns the database's own error.
func TestFinder_GroupedErrors(t *testing.T) {
	ctx := context.Background()
	// Every query fails on a finished transaction.
	tx, err := testutil.MustOpenTestDB(t).Beginx()
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	f := NewFinder(tx)
	methods := finderMethods(isGroupedFinder)
	require.NotEmpty(t, methods, "expected grouped finders in model.Finder")
	for _, m := range methods {
		t.Run(m.Name, func(t *testing.T) {
			err, _ := callFinder(ctx, f, m)[1].Interface().(error)
			require.Error(t, err)
			assert.NotErrorIs(t, err, sql.ErrTxDone)
		})
	}
}

// finderTestKeys is how many keys callFinder passes.
const finderTestKeys = 3

var (
	contextType = reflect.TypeFor[context.Context]()
	errorType   = reflect.TypeFor[error]()
	errorsType  = reflect.TypeFor[[]error]()
)

// finderMethods returns the model.Finder methods whose signatures match.
func finderMethods(match func(reflect.Type) bool) []reflect.Method {
	var ret []reflect.Method
	ft := reflect.TypeFor[model.Finder]()
	for i := range ft.NumMethod() {
		if m := ft.Method(i); match(m.Type) {
			ret = append(ret, m)
		}
	}
	return ret
}

// isKeyedFinder matches func(ctx, []K) ([]T, []error).
func isKeyedFinder(mt reflect.Type) bool {
	return mt.NumIn() == 2 && mt.In(0) == contextType && mt.In(1).Kind() == reflect.Slice &&
		mt.NumOut() == 2 && mt.Out(0).Kind() == reflect.Slice && mt.Out(1) == errorsType
}

// isGroupedFinder matches func(ctx, ..., []K) ([][]T, error).
func isGroupedFinder(mt reflect.Type) bool {
	n := mt.NumIn()
	return n >= 2 && mt.In(0) == contextType && mt.In(n-1).Kind() == reflect.Slice &&
		mt.NumOut() == 2 && mt.Out(0).Kind() == reflect.Slice && mt.Out(0).Elem().Kind() == reflect.Slice && mt.Out(1) == errorType
}

// callFinder calls a finder method with finderTestKeys zero-valued keys, and zero
// values for its other arguments.
func callFinder(ctx context.Context, f model.Finder, m reflect.Method) []reflect.Value {
	// The census geography finders read the fields a GraphQL query selects.
	ctx = graphql.WithOperationContext(ctx, &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{Field: graphql.CollectedField{Field: &ast.Field{}}})
	args := make([]reflect.Value, m.Type.NumIn())
	args[0] = reflect.ValueOf(ctx)
	for i := 1; i < len(args)-1; i++ {
		args[i] = reflect.Zero(m.Type.In(i))
	}
	args[len(args)-1] = reflect.MakeSlice(m.Type.In(len(args)-1), finderTestKeys, finderTestKeys)
	return reflect.ValueOf(f).MethodByName(m.Name).Call(args)
}

func Test_queryErrs(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	timedOut, cancelTimedOut := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelTimedOut()
	tcs := []struct {
		name   string
		ctx    context.Context
		expect string
	}{
		{"query failed", context.Background(), "database error"},
		{"request canceled", canceled, context.Canceled.Error()},
		{"request timed out", timedOut, context.DeadlineExceeded.Error()},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			errs := queryErrs(tc.ctx, 3, errors.New("connection reset"))
			require.Len(t, errs, 3)
			for _, err := range errs {
				assert.EqualError(t, err, tc.expect)
			}
		})
	}
	t.Run("no error", func(t *testing.T) {
		assert.NoError(t, queryErr(context.Background(), nil))
	})
}

func TestFinder_FindFeedVersionServiceWindow(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	dbf := NewFinder(dbutil.WithQueryLogger(db, false, 0))
	testFinder := dbf

	fvm := map[string]int{}
	fvs, err := testFinder.FindFeedVersions(ctx, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fv := range fvs {
		fvm[fv.SHA1] = fv.ID
	}
	tcs := []struct {
		name  string
		fvid  int
		start string
		end   string
		best  string
	}{
		{
			"hart",
			fvm["c969427f56d3a645195dd8365cde6d7feae7e99b"],
			"2018-02-26", // calculated
			"2018-10-21",
			"2018-07-09",
		},
		{
			"bart",
			fvm["e535eb2b3b9ac3ef15d82c56575e914575e732e0"],
			"2018-05-26", // from feed info
			"2019-07-01", // from feed info
			"2018-06-04",
		},
		{
			"caltrain",
			fvm["d2813c293bcfd7a97dde599527ae6c62c98e66c6"],
			"2018-06-18", // calculated
			"2019-10-06",
			"2018-06-18",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			fvsw, err := testFinder.FindFeedVersionServiceWindow(ctx, tc.fvid)
			start, end, best := fvsw.StartDate, fvsw.EndDate, fvsw.FallbackWeek
			if err != nil {
				t.Fatal(err)
			}
			df := "2006-01-02"
			assert.Equal(t, tc.start, start.Format(df), "did not get expected window start")
			assert.Equal(t, tc.end, end.Format(df), "did not get expected window end")
			assert.Equal(t, tc.best, best.Format(df), "did not get expected best week in window")
			if end.Before(end) {
				t.Errorf("window end date %s before window start date %s", start.Format(df), end.Format(df))
			}
			if best.Before(start) {
				t.Errorf("best date %s before window start date %s", best.Format(df), start.Format(df))
			}
			if best.After(end) {
				t.Errorf("best date %s after window end date %s", best.Format(df), end.Format(df))
			}
		})
	}
}

func Test_alphanumeric(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"ascii char", "a", "a"},
		{"ascii string", "abc", "abc"},
		{"ascii alphanumeric", "abc123", "abc123"},
		{"ascii space", "a b c", "a b c"},
		{"emdash to space", "a—b", "a b"},
		{"double space ok", "a  b", "a  b"},
		{"slash to space", "a/b", "a b"},
		{"single quote to space", "a'b", "a b"},
		{"double quote to space", "\"", " "},
		{": to space", "a:b", "a b"},
		{"* to space", "a*b", "a b"},
		{"& to space", "a&b", "a b"},
		{"| to space", "a|b", "a b"},
		{"hyphen to space", "Winston-Salem", "Winston Salem"},
		{"tab to space", "\t", " "},
		{"french", "Hôtel", "Hôtel"},
		{"combining mark kept", "Montre\u0301al", "Montre\u0301al"},
		{"devanagari vowel signs kept", "दिल्ली", "दिल्ली"},
		{"chinese", "火车", "火车"},
		{"chinese with ascii", "abc 火车 123", "abc 火车 123"},
		{"japanese", "列車", "列車"},
		{"russian", "тренироваться", "тренироваться"},
		{"hebrew", "רכבת", "רכבת"},
		{"arabic", "قطار", "قطار"},
		{"arabic with ascii", "test قطار", "test قطار"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := alphanumeric(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

func Test_escapeWordsWithSuffix(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect []string
	}{
		{"words", "san fran", []string{"san:*", "fran:*"}},
		{"punctuation splits words", "Winston-Salem", []string{"Winston:*", "Salem:*"}},
		{"short pieces dropped", "coeur d'alene", []string{"coeur:*", "alene:*"}},
		{"nothing usable", "a b !!", nil},
		{"one accented letter is one character", "é", nil},
		{"two accented letters", "éé", []string{"éé:*"}},
		{"decomposed accent composed", "Montre\u0301al", []string{"Montr\u00e9al:*"}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, escapeWordsWithSuffix(tc.value, ":*"))
		})
	}
}

func Test_az09(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"plain", "hello", "hello"},
		{"underscore", "hello_world", "hello_world"},
		{"digits", "123", "123"},
		{"remove quotes", "a'b'\"c", "abc"},
		{"remove symbols", "a!b@c#d$e%f;g(h", "abcdefgh"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := az09(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

// A date's local noon exists even on a day whose midnight a DST change skips:
// America/Santiago jumped from midnight to 01:00 on 2024-09-08.
func TestTzTruncate(t *testing.T) {
	santiago, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	tcs := []struct {
		name   string
		in     time.Time
		expect string
	}{
		{"a date scalar on the day DST starts", time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), "2024-09-08"},
		{"late on the evening before", time.Date(2024, 9, 7, 23, 30, 0, 0, santiago), "2024-09-07"},
		{"early on the morning after", time.Date(2024, 9, 9, 0, 30, 0, 0, santiago), "2024-09-09"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tzTruncate(tc.in, santiago)
			assert.Equal(t, tc.expect, got.String())
			assert.Equal(t, 12, got.Val.Hour())
		})
	}
}
