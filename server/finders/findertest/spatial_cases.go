package findertest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Cases duplicated from server/gql that need the Spatial capability.

func feedResolverSpatialTestcases() []testcase {
	testcases := []testcase{
		// spatial
		{
			name:         "radius",
			query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "radius 2",
			query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:  "within",
			query: `query($within:Polygon) {feeds(where: {within:$within}) {onestop_id}}`,
			vars: hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.39803791046143, 37.794626736533836},
				{-122.40106344223022, 37.792303711508595},
				{-122.3965573310852, 37.789641468930114},
				{-122.3938751220703, 37.792354581451946},
				{-122.39803791046143, 37.794626736533836},
			}}}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "bbox 1",
			query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "bbox 2",
			query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:        "bbox too large",
			query:       `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
		},
	}
	return testcases
}

func feedVersionResolverSpatialTestcases() []testcase {
	within := hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
		{-122.39803791046143, 37.794626736533836},
		{-122.40106344223022, 37.792303711508595},
		{-122.3965573310852, 37.789641468930114},
		{-122.3938751220703, 37.792354581451946},
		{-122.39803791046143, 37.794626736533836},
	}}}}
	testcases := []testcase{
		{
			name: "multiple where values",
			query: `query($within:Polygon) {
				feeds(where:{onestop_id:"BA"}) {
					all:    feed_versions {sha1} 
					all1:   feed_versions(limit:1) {sha1} 
					all2:   feed_versions(limit:2) {sha1} 
					ok:     feed_versions(where:{import_status:SUCCESS}) {sha1} 
					fail:   feed_versions(where:{import_status:ERROR}) {sha1} 	
					ip:     feed_versions(where:{import_status:IN_PROGRESS}) {sha1} 	
					within: feed_versions(where:{within:$within}) {sha1}
					covers: feed_versions(where:{covers:{start_date:"2016-12-31"}}) {sha1}
				}
			}`,
			vars: within,
			// selector:     "ok.#.sha1",
			// selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
			f: func(t *testing.T, jj string) {
				exp := []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"}
				var empty []string
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.all.#.sha1").Array()))
				assert.Equal(t, exp[0:1], astr(gjson.Get(jj, "feeds.0.all1.#.sha1").Array()))
				assert.Equal(t, exp[0:2], astr(gjson.Get(jj, "feeds.0.all2.#.sha1").Array()))
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.ok.#.sha1").Array()))
				assert.Equal(t, empty, astr(gjson.Get(jj, "feeds.0.fail.#.sha1").Array()))
				assert.Equal(t, empty, astr(gjson.Get(jj, "feeds.0.ip.#.sha1").Array()))
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.within.#.sha1").Array()))
				assert.Equal(t, []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"}, astr(gjson.Get(jj, "feeds.0.covers.#.sha1").Array()))
			},
		},
		// spatial
		{
			name:         "radius",
			query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "radius 2",
			query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"c969427f56d3a645195dd8365cde6d7feae7e99b"},
		},
		{
			name:         "within",
			query:        `query($within:Polygon) {feed_versions(where: {within:$within}) {sha1}}`,
			vars:         within,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "bbox 1",
			query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "bbox 2",
			query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		{
			name:        "bbox too large",
			query:       `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
		},
	}
	return testcases
}

func operatorResolverSpatialTestcases() []testcase {
	testcases := []testcase{
		// spatial
		{
			name:         "near 100m",
			query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "near 10000m",
			query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "within polygon",
			query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "within polygon big",
			query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id onestop_id}}`,
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			name:         "where bbox 1",
			query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "operators.#.onestop_id",
			selectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			name:         "where bbox 2",
			query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "operators.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:        "where bbox too large",
			query:       `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
		},
	}
	return testcases
}
