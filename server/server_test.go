package server

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	model.DB = model.MustOpenDB(os.Getenv("TL_DATABASE_URL"))
	os.Exit(m.Run())
}

func toJson(m map[string]interface{}) string {
	rr, err := json.Marshal(&m)
	if err != nil {
		panic(err)
	}
	return string(rr)
}

type hw = map[string]interface{}

type expectJson struct {
	query  string
	vars   hw
	expect string
}

func TestFeedResolver(t *testing.T) {
	testcases := []expectJson{
		{`query { feeds {onestop_id}}`, hw{}, `{"feeds":[{"onestop_id":"CT"},{"onestop_id":"BA"}]}`},
	}
	c := client.New(newServer())
	for _, tc := range testcases {
		var resp map[string]interface{}
		c.MustPost(tc.query, &resp)
		assert.JSONEq(t, tc.expect, toJson(resp))
	}
}

// func TestFeedResolver(t *testing.T) {
// 	c := client.New(newServer())
// 	resp := make(map[string]interface{})
// 	c.MustPost(`query($onestop_id:String!) {
// 		feeds(where:{onestop_id:$onestop_id}) {
// 		  onestop_id
// 		  spec
// 		  urls {
// 			  static_current
// 		  }
// 		  feed_versions {
// 			sha1
// 		  }
// 		}
// 	  }`, &resp, client.Var("onestop_id", "BA"))
// 	expect := `{"feeds":[{"feed_versions":[{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}],"onestop_id":"CT","spec":"gtfs","urls":{"static_current":"test/data/external/caltrain.zip"}},{"feed_versions":[{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}],"onestop_id":"BA","spec":"gtfs","urls":{"static_current":"test/data/external/bart.zip"}}]}`
// 	assert.JSONEq(t, toJson(resp), expect)
// }
