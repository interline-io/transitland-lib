package resolvers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) {
	model.DB = model.MustOpenDB(os.Getenv("TL_DATABASE_URL"))
	os.Exit(m.Run())
}

// Test helpers

func toJson(m map[string]interface{}) string {
	rr, err := json.Marshal(&m)
	if err != nil {
		panic(err)
	}
	return string(rr)
}

type hw = map[string]interface{}

type testcase struct {
	name         string
	query        string
	vars         hw
	expect       string
	selector     string
	expectSelect []string
}

func testquery(t *testing.T, c *client.Client, tc testcase) {
	var resp map[string]interface{}
	opts := []client.Option{}
	for k, v := range tc.vars {
		opts = append(opts, client.Var(k, v))
	}
	c.MustPost(tc.query, &resp, opts...)
	jj := toJson(resp)
	if tc.expect != "" {
		if !assert.JSONEq(t, tc.expect, jj) {
			fmt.Printf("got %s -- expect %s\n", jj, tc.expect)
		}
	}
	if tc.selector != "" {
		a := []string{}
		for _, v := range gjson.Get(jj, tc.selector).Array() {
			a = append(a, v.String())
		}
		if len(a) == 0 && tc.expectSelect == nil {
			t.Errorf("selector '%s' returned zero elements", tc.selector)
		} else {
			if !assert.ElementsMatch(t, a, tc.expectSelect) {
				fmt.Printf("got %#v -- expect %#v\n\n", a, tc.expectSelect)
			}
		}
	}
}
