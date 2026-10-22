// Package gqlcase defines GraphQL resolver test cases, and runs them against a
// client. See server/gql/RESOLVER_TEST_GUIDE.md for how to write a case.
package gqlcase

import (
	"encoding/json"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// TestcaseSelector checks the values a gjson path selects from a query result.
type TestcaseSelector struct {
	Selector          string
	Expect            []string
	ExpectUnique      []string
	ExpectContains    []string
	ExpectCount       int
	ExpectUniqueCount int
}

// Testcase is a GraphQL query and the checks its result must pass.
type Testcase struct {
	Name        string
	Query       string
	Vars        map[string]any
	Expect      string
	User        string
	ExpectError bool
	// ExpectErrorContains expects an error containing this text.
	ExpectErrorContains string
	// Sel checks several selectors against the same result.
	Sel []TestcaseSelector
	// F runs custom assertions on the result's JSON.
	F func(*testing.T, string)
	// The Select fields are a single selector check, the same as one entry in
	// Sel.
	Selector                string
	SelectExpect            []string
	SelectExpectUnique      []string
	SelectExpectContains    []string
	SelectExpectCount       int
	SelectExpectUniqueCount int
}

// QueryTestcases runs each case as a subtest.
func QueryTestcases(t *testing.T, c *client.Client, tcs []Testcase) {
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			QueryTestcase(t, c, tc)
		})
	}
}

// QueryTestcase runs a case's query and checks its result.
func QueryTestcase(t *testing.T, c *client.Client, tc Testcase) {
	tested := false
	var resp map[string]interface{}
	opts := []client.Option{}
	for k, v := range tc.Vars {
		opts = append(opts, client.Var(k, v))
	}
	expectError := tc.ExpectError || tc.ExpectErrorContains != ""
	if err := c.Post(tc.Query, &resp, opts...); err != nil {
		if expectError {
			tested = true // expectError counts as a test
			if tc.ExpectErrorContains != "" {
				assert.ErrorContains(t, err, tc.ExpectErrorContains)
			}
		} else {
			t.Error(err)
			return
		}
	} else if expectError {
		t.Error("expected error")
	}
	jj := ToJSON(resp)
	if tc.Expect != "" {
		tested = true
		if !assert.JSONEq(t, tc.Expect, jj) {
			t.Errorf("got %s -- expect %s\n", jj, tc.Expect)
		}
	}
	if tc.F != nil {
		tested = true
		tc.F(t, jj)
	}
	if tc.Selector != "" {
		tc.Sel = append(tc.Sel, TestcaseSelector{
			Selector:          tc.Selector,
			Expect:            tc.SelectExpect,
			ExpectCount:       tc.SelectExpectCount,
			ExpectUnique:      tc.SelectExpectUnique,
			ExpectContains:    tc.SelectExpectContains,
			ExpectUniqueCount: tc.SelectExpectUniqueCount,
		})
	}
	for _, sel := range tc.Sel {
		a := []string{}
		for _, v := range gjson.Get(jj, sel.Selector).Array() {
			a = append(a, v.String())
		}
		if sel.ExpectCount != 0 {
			tested = true
			if len(a) != sel.ExpectCount {
				t.Errorf("selector returned %d elements, expected %d", len(a), sel.ExpectCount)
			}
		}
		if sel.ExpectUniqueCount != 0 {
			tested = true
			mm := map[string]int{}
			for _, v := range a {
				mm[v] += 1
			}
			if len(mm) != sel.ExpectUniqueCount {
				t.Errorf("selector returned %d unique elements, expected %d", len(mm), sel.ExpectUniqueCount)
			}
		}
		if sel.ExpectUnique != nil {
			tested = true
			mm := map[string]int{}
			for _, v := range a {
				mm[v] += 1
			}
			var keys []string
			for k := range mm {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, sel.ExpectUnique, keys)
		}
		if sel.Expect != nil {
			tested = true
			if !assert.ElementsMatch(t, sel.Expect, a) {
				t.Errorf("got %#v -- expect %#v\n\n", a, sel.Expect)
			}
		}
		if sel.ExpectContains != nil {
			tested = true
			mm := map[string]bool{}
			for _, v := range a {
				mm[v] = true
			}
			for _, v := range sel.ExpectContains {
				if !mm[v] {
					t.Errorf("expected result to contain %q", v)
				}
			}
		}
	}
	if !tested {
		t.Errorf("no test performed, check test case")
	}
}

// BenchmarkTestcases runs each case's query as a sub-benchmark.
func BenchmarkTestcases(b *testing.B, c *client.Client, tcs []Testcase) {
	for _, tc := range tcs {
		b.Run(tc.Name, func(b *testing.B) {
			benchmarkTestcase(b, c, tc)
		})
	}
}

func benchmarkTestcase(b *testing.B, c *client.Client, tc Testcase) {
	opts := []client.Option{}
	for k, v := range tc.Vars {
		opts = append(opts, client.Var(k, v))
	}
	var resp map[string]any
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.MustPost(tc.Query, &resp, opts...)
	}
}

// ToJSON returns a query result as JSON.
func ToJSON(m map[string]interface{}) string {
	rr, _ := json.Marshal(&m)
	return string(rr)
}

// Strings returns the string values of gjson results.
func Strings(a []gjson.Result) []string {
	var ret []string
	for _, b := range a {
		ret = append(ret, b.String())
	}
	return ret
}
