package rest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/resolvers"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

const LON = 37.803613
const LAT = -122.271556

func TestMain(m *testing.M) {
	g := os.Getenv("TL_TEST_SERVER_DATABASE_URL")
	if g == "" {
		fmt.Println("TL_TEST_SERVER_DATABASE_URL not set, skipping")
		return
	}
	model.DB = model.MustOpenDB(g)
	os.Exit(m.Run())
}

// Test helpers

func testRestConfig() restConfig {
	cfg := config.Config{}
	srv, _ := resolvers.NewServer(cfg)
	return restConfig{srv: srv, Config: cfg}
}

func toJson(m map[string]interface{}) string {
	rr, _ := json.Marshal(&m)
	return string(rr)
}

type testRest struct {
	name         string
	h            apiHandler
	format       string
	selector     string
	expectSelect []string
	expectLength int
}

func testquery(t *testing.T, cfg restConfig, tc testRest) {
	data, err := makeRequest(cfg, tc.h, tc.format)
	if err != nil {
		t.Error(err)
		return
	}
	jj := string(data)
	if tc.selector != "" {
		a := []string{}
		for _, v := range gjson.Get(jj, tc.selector).Array() {
			a = append(a, v.String())
		}
		if len(tc.expectSelect) > 0 {
			if len(a) == 0 {
				t.Errorf("selector '%s' returned zero elements", tc.selector)
			} else {
				if !assert.ElementsMatch(t, a, tc.expectSelect) {
					fmt.Printf("got %#v -- expect %#v\n\n", a, tc.expectSelect)
				}
			}
		} else {
			if len(a) != tc.expectLength {
				t.Errorf("got %d elements, expected %d", len(a), tc.expectLength)
			}
		}
	}
}
