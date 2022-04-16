package dmfr

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func lineTrimSpaces(v string) string {
	stripSpace := []string{}
	for _, line := range strings.Split(v, "\n") {
		line = strings.ReplaceAll(line, "\": ", "\":")
		line = strings.ReplaceAll(line, "\", ", "\",")
		stripSpace = append(stripSpace, strings.TrimSpace(line))
	}
	return strings.Join(stripSpace, "")
}

func TestRawRegistry_Write(t *testing.T) {
	tcs := []struct {
		name   string
		data   string
		output string
	}{
		{
			"feed",
			`{"feeds":[{"id":"test"}]}`,
			`{"feeds":[{"id":"test"}]}`,
		},
		{
			"feed sorted",
			`{"feeds":[{"id":"z"},{"id":"a"}]}`,
			`{"feeds":[{"id":"a"},{"id":"z"}]}`,
		},
		{
			"feed id required",
			`{"feeds":[{"spec":"gtfs"}]}`,
			`{"feeds":[{"id":"","spec":"gtfs"}]}`,
		},
		{
			"empty feeds removed",
			`{"feeds":[]}`,
			`{}`,
		},
		{
			"languages sorted alpha",
			`{"feeds":[{"id":"a","languages":["z","a"]}]}`,
			`{"feeds":[{"id":"a","languages":["a","z"]}]}`,
		},
		{
			"tags sorted alpha",
			`{"feeds":[{"id":"a","tags":{"z":"z","a":"a"}}]}`,
			`{"feeds":[{"id":"a","tags":{"a":"a","z":"z"}}]}`,
		},
		{
			"empty tags removed",
			`{"feeds":[{"id":"a","tags":{}}]}`,
			`{"feeds":[{"id":"a"}]}`,
		},
		{
			"empty struct fields removed",
			`{"feeds":[{"id":"a","urls":{}}]}`,
			`{"feeds":[{"id":"a"}]}`,
		},
		{
			"empty languages removed",
			`{"feeds":[{"id":"a","languages":[]}]}`,
			`{"feeds":[{"id":"a"}]}`,
		},
		{
			"nested operators sorted",
			`{"feeds":[{"id":"a","operators":[{"onestop_id":"z"},{"onestop_id":"a"}]}]}`,
			`{"feeds":[{"id":"a","operators":[{"onestop_id":"a"},{"onestop_id":"z"}]}]}`,
		},
		{
			"nested operators associations sorted",
			`{"feeds":[{"id":"a","operators":[{"onestop_id":"a","associated_feeds":[{"feed_onestop_id":"z"},{"feed_onestop_id":"a"}]}]}]}`,
			`{"feeds":[{"id":"a","operators":[{"onestop_id":"a","associated_feeds":[{"feed_onestop_id":"a"},{"feed_onestop_id":"z"}]}]}]}`,
		},
		{
			"operators sorted",
			`{"operators":[{"onestop_id":"z"},{"onestop_id":"a"}]}`,
			`{"operators":[{"onestop_id":"a"},{"onestop_id":"z"}]}`,
		},
		{
			"operators associated feeds",
			`{"operators":[{"onestop_id":"a","associated_feeds":[{"feed_onestop_id":"z"},{"feed_onestop_id":"a"}]}]}`,
			`{"operators":[{"onestop_id":"a","associated_feeds":[{"feed_onestop_id":"a"},{"feed_onestop_id":"z"}]}]}`,
		},
		{
			"field order matches struct",
			`{"feeds":[{"urls":{"static_current":"x"},"spec":"gtfs","id":"test","name":"ok","languages":["en"]}]}`,
			`{"feeds":[{"id":"test","name":"ok","spec":"gtfs","urls":{"static_current":"x"},"languages":["en"]}]}`,
		},
		{"empty", `{}`, `{}`},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ReadRawRegistry(bytes.NewBuffer([]byte(tc.data)))
			if err != nil {
				t.Fatal(err)
			}
			outbuf := bytes.NewBuffer(nil)
			if err := r.Write(outbuf); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, lineTrimSpaces(tc.output), lineTrimSpaces(outbuf.String()))
		})
	}
}
