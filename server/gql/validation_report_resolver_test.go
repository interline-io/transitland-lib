package gql

import (
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestValidationReportResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	fvsha1 := "96b67c0934b689d9085c52967365d8c233ea321d"
	q := `query($feed_version_sha1: String!, $where: ValidationReportFilter) {  feed_versions(where:{sha1:$feed_version_sha1}) {validation_reports(where:$where) {id success failure_reason includes_static includes_rt validator validator_version errors { filename error_type error_code field count errors { filename error_type error_code entity_id field line value message geometry entity_json }} }} }`
	var reportId int
	if err := cfg.Adapter.DBX().QueryRowx("select rp.id from tl_validation_reports rp join feed_versions fv on fv.id = rp.feed_version_id where fv.sha1 = $1 limit 1", fvsha1).Scan(&reportId); err != nil {
		panic(err)
	}
	testcases := []testcase{
		// Saved validation reports
		{
			Name:  "validation reports",
			Query: q,
			Vars:  hw{"feed_version_sha1": fvsha1},
			F: func(t *testing.T, jj string) {
				reports := gjson.Get(jj, "feed_versions.0.validation_reports")
				assert.Equal(t, 1, len(reports.Array()))
				report := reports.Get("0")
				assert.Equal(t, []string{"stops.txt", "stops.txt"}, gqlcase.Strings(report.Get("errors.#.filename").Array()))
				assert.Equal(t, []string{"InvalidFieldError", "InvalidFieldError"}, gqlcase.Strings(report.Get("errors.#.error_type").Array()))
				assert.Equal(t, []string{"1", "1"}, gqlcase.Strings(report.Get("errors.#.count").Array()))
				var messages []string
				for _, a := range report.Get("errors").Array() {
					messages = append(messages, gqlcase.Strings(a.Get("errors.#.message").Array())...)
				}
				expMessages := []string{
					"invalid value for field stop_lat '-200': out of bounds, less than -90.000000",
					"invalid value for field stop_lon '-200': out of bounds, less than -180.000000",
				}
				assert.ElementsMatch(t, expMessages, messages)
			},
		},
		{
			Name:         "success=true",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"success": true}},
			Selector:     "feed_versions.0.validation_reports.#.success",
			SelectExpect: []string{"true"},
		},
		{
			Name:         "success=false",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"success": false}},
			Selector:     "feed_versions.0.validation_reports.#.success",
			SelectExpect: []string{},
		},
		{
			Name:         "includes_static=true",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_static": true}},
			Selector:     "feed_versions.0.validation_reports.#.includes_static",
			SelectExpect: []string{"true"},
		},
		{
			Name:         "includes_static=false",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_static": false}},
			Selector:     "feed_versions.0.validation_reports.#.includes_static",
			SelectExpect: []string{},
		},

		{
			Name:         "includes_rt=true",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_rt": true}},
			Selector:     "feed_versions.0.validation_reports.#.includes_rt",
			SelectExpect: []string{},
		},
		{
			Name:         "includes_rt=false",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_rt": false}},
			Selector:     "feed_versions.0.validation_reports.#.includes_rt",
			SelectExpect: []string{"false"},
		},
		{
			Name:         "report_ids",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"report_ids": []int{reportId}}},
			Selector:     "feed_versions.0.validation_reports.#.id",
			SelectExpect: []string{strconv.Itoa(reportId)},
		},
		{
			Name:         "report_ids",
			Query:        q,
			Vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"report_ids": []int{100000}}},
			Selector:     "feed_versions.0.validation_reports.#.id",
			SelectExpect: []string{},
		},
		// TODO:
		// {
		// 	name:  "entity_json",
		// 	query: q,
		// 	vars:  hw{"feed_version_sha1": fvsha1},
		// 	f: func(f *testing.T, jj string) {
		// 		fmt.Println("jj:", jj)
		// 	},
		// },
	}
	gqlcase.QueryTestcases(t, c, testcases)
}
