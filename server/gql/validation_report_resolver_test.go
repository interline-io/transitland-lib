package gql

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestValidationReportResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	fvsha1 := "96b67c0934b689d9085c52967365d8c233ea321d"
	q := `query($feed_version_sha1: String!, $where: ValidationReportFilter) {  feed_versions(where:{sha1:$feed_version_sha1}) {validation_reports(where:$where) {id success failure_reason includes_static includes_rt validator validator_version errors { filename error_type error_code field count errors { filename error_type error_code entity_id field line value message geometry entity_json }} }} }`
	var reportId int
	if err := cfg.Finder.DBX().QueryRowx("select rp.id from tl_validation_reports rp join feed_versions fv on fv.id = rp.feed_version_id where fv.sha1 = $1 limit 1", fvsha1).Scan(&reportId); err != nil {
		panic(err)
	}
	testcases := []testcase{
		// Saved validation reports
		{
			name:  "validation reports",
			query: q,
			vars:  hw{"feed_version_sha1": fvsha1},
			f: func(t *testing.T, jj string) {
				reports := gjson.Get(jj, "feed_versions.0.validation_reports")
				assert.Equal(t, 1, len(reports.Array()))
				report := reports.Get("0")
				assert.Equal(t, []string{"stops.txt", "stops.txt"}, astr(report.Get("errors.#.filename").Array()))
				assert.Equal(t, []string{"InvalidFieldError", "InvalidFieldError"}, astr(report.Get("errors.#.error_type").Array()))
				assert.Equal(t, []string{"1", "1"}, astr(report.Get("errors.#.count").Array()))
				var messages []string
				for _, a := range report.Get("errors").Array() {
					messages = append(messages, astr(a.Get("errors.#.message").Array())...)
				}
				expMessages := []string{
					"invalid value for field stop_lat '-200': out of bounds, less than -90.000000",
					"invalid value for field stop_lon '-200': out of bounds, less than -180.000000",
				}
				assert.ElementsMatch(t, expMessages, messages)
			},
		},
		{
			name:         "success=true",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"success": true}},
			selector:     "feed_versions.0.validation_reports.#.success",
			selectExpect: []string{"true"},
		},
		{
			name:         "success=false",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"success": false}},
			selector:     "feed_versions.0.validation_reports.#.success",
			selectExpect: []string{},
		},
		{
			name:         "includes_static=true",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_static": true}},
			selector:     "feed_versions.0.validation_reports.#.includes_static",
			selectExpect: []string{"true"},
		},
		{
			name:         "includes_static=false",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_static": false}},
			selector:     "feed_versions.0.validation_reports.#.includes_static",
			selectExpect: []string{},
		},

		{
			name:         "includes_rt=true",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_rt": true}},
			selector:     "feed_versions.0.validation_reports.#.includes_rt",
			selectExpect: []string{},
		},
		{
			name:         "includes_rt=false",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"includes_rt": false}},
			selector:     "feed_versions.0.validation_reports.#.includes_rt",
			selectExpect: []string{"false"},
		},
		{
			name:         "report_ids",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"report_ids": []int{reportId}}},
			selector:     "feed_versions.0.validation_reports.#.id",
			selectExpect: []string{strconv.Itoa(reportId)},
		},
		{
			name:         "report_ids",
			query:        q,
			vars:         hw{"feed_version_sha1": fvsha1, "where": hw{"report_ids": []int{100000}}},
			selector:     "feed_versions.0.validation_reports.#.id",
			selectExpect: []string{},
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
	queryTestcases(t, c, testcases)
}
