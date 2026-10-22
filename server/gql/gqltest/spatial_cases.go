package gqltest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Cases that need the Spatial capability.

func feedResolverSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		// spatial
		{
			Name:         "radius",
			Query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			Vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "radius 2",
			Query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			Vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA"},
		},
		{
			Name:  "within",
			Query: `query($within:Polygon) {feeds(where: {within:$within}) {onestop_id}}`,
			Vars: hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.39803791046143, 37.794626736533836},
				{-122.40106344223022, 37.792303711508595},
				{-122.3965573310852, 37.789641468930114},
				{-122.3938751220703, 37.792354581451946},
				{-122.39803791046143, 37.794626736533836},
			}}}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "bbox 1",
			Query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "bbox 2",
			Query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:        "bbox too large",
			Query:       `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
	}
	return testcases
}

func feedVersionResolverSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	within := hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
		{-122.39803791046143, 37.794626736533836},
		{-122.40106344223022, 37.792303711508595},
		{-122.3965573310852, 37.789641468930114},
		{-122.3938751220703, 37.792354581451946},
		{-122.39803791046143, 37.794626736533836},
	}}}}
	testcases := []testcase{
		{
			Name: "multiple where values",
			Query: `query($within:Polygon) {
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
			Vars: within,
			// selector:     "ok.#.sha1",
			// selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
			F: func(t *testing.T, jj string) {
				exp := []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"}
				var empty []string
				assert.Equal(t, exp, gqlcase.Strings(gjson.Get(jj, "feeds.0.all.#.sha1").Array()))
				assert.Equal(t, exp[0:1], gqlcase.Strings(gjson.Get(jj, "feeds.0.all1.#.sha1").Array()))
				assert.Equal(t, exp[0:2], gqlcase.Strings(gjson.Get(jj, "feeds.0.all2.#.sha1").Array()))
				assert.Equal(t, exp, gqlcase.Strings(gjson.Get(jj, "feeds.0.ok.#.sha1").Array()))
				assert.Equal(t, empty, gqlcase.Strings(gjson.Get(jj, "feeds.0.fail.#.sha1").Array()))
				assert.Equal(t, empty, gqlcase.Strings(gjson.Get(jj, "feeds.0.ip.#.sha1").Array()))
				assert.Equal(t, exp, gqlcase.Strings(gjson.Get(jj, "feeds.0.within.#.sha1").Array()))
				assert.Equal(t, []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"}, gqlcase.Strings(gjson.Get(jj, "feeds.0.covers.#.sha1").Array()))
			},
		},
		// spatial
		{
			Name:         "radius",
			Query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			Vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "radius 2",
			Query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			Vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"c969427f56d3a645195dd8365cde6d7feae7e99b"},
		},
		{
			Name:         "within",
			Query:        `query($within:Polygon) {feed_versions(where: {within:$within}) {sha1}}`,
			Vars:         within,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "bbox 1",
			Query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "bbox 2",
			Query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:        "bbox too large",
			Query:       `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
	}
	return testcases
}

func operatorResolverSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		// spatial
		{
			Name:         "near 100m",
			Query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "near 10000m",
			Query:        `query {operators(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "within polygon",
			Query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "within polygon big",
			Query:        `query{operators(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id onestop_id}}`,
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "where bbox 1",
			Query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{"o-9q9-bayarearapidtransit"},
		},
		{
			Name:         "where bbox 2",
			Query:        `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "operators.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:        "where bbox too large",
			Query:       `query($bbox:BoundingBox) {operators(where:{bbox:$bbox}) {onestop_id}}`,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
	}
	return testcases
}

func agencyResolverLocationSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	testFeed, testAgencyCode := "BA", "BART"
	testAgencies, err := cfg.Finder.FindAgencies(model.WithConfig(context.Background(), cfg), nil, nil, nil, &model.AgencyFilter{FeedOnestopID: &testFeed, AgencyID: &testAgencyCode})
	if err != nil {
		t.Fatal(err)
	}
	if len(testAgencies) != 1 {
		t.Fatalf("found %d active agencies with agency_id %s in %s, expected 1", len(testAgencies), testAgencyCode, testFeed)
	}
	testcases := []testcase{
		{
			Name:         "near 100m",
			Query:        `query {agencies(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {agency_id}}`,
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"BART"},
		},
		{
			Name:         "near 10000m",
			Query:        `query {agencies(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {agency_id}}`,
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us", "BART"},
		},
		{
			Name:         "within polygon",
			Query:        `query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39803791046143,37.794626736533836],[-122.40106344223022,37.792303711508595],[-122.3965573310852,37.789641468930114],[-122.3938751220703,37.792354581451946],[-122.39803791046143,37.794626736533836]]]}}){agency_id}}`,
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"BART"},
		},
		{
			Name:         "within polygon big",
			Query:        `query{agencies(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id agency_id}}`,
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us", "BART"},
		},
		{
			Name:         "where bbox 1",
			Query:        `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"BART"},
		},
		{
			Name:         "where bbox 2",
			Query:        `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{},
		},
		{
			Name:        "where bbox too large",
			Query:       `query($bbox:BoundingBox) {agencies(where:{bbox:$bbox}) {agency_id}}`,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
	}
	return testcases
}

// feedResolverGeohashFilterSpatialTestcases runs the feed spatial filter cases
// with the geohash secondary filter on. They expect the polygon-only results,
// so the filter must keep every legitimate match.
func feedResolverGeohashFilterSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			Name:         "bbox SF",
			Query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "bbox empty area",
			Query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:  "within SF polygon",
			Query: `query($within:Polygon) {feeds(where: {within:$within}) {onestop_id}}`,
			Vars: hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.39803791046143, 37.794626736533836},
				{-122.40106344223022, 37.792303711508595},
				{-122.3965573310852, 37.789641468930114},
				{-122.3938751220703, 37.792354581451946},
				{-122.39803791046143, 37.794626736533836},
			}}}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "near SF point",
			Query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			Vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "near Tampa point",
			Query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			Vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA"},
		},
	}
	return testcases
}

// feedVersionResolverGeohashFilterSpatialTestcases runs the feed version
// spatial filter cases with the geohash secondary filter on. They expect the
// polygon-only results, so the filter must keep every legitimate match.
func feedVersionResolverGeohashFilterSpatialTestcases(t testing.TB, cfg model.Config) []testcase {
	within := hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
		{-122.39803791046143, 37.794626736533836},
		{-122.40106344223022, 37.792303711508595},
		{-122.3965573310852, 37.789641468930114},
		{-122.3938751220703, 37.792354581451946},
		{-122.39803791046143, 37.794626736533836},
	}}}}
	testcases := []testcase{
		{
			Name:         "bbox SF",
			Query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "bbox empty area",
			Query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:         "within SF polygon",
			Query:        `query($within:Polygon) {feed_versions(where: {within:$within}) {sha1}}`,
			Vars:         within,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "near SF point",
			Query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			Vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "near Tampa point",
			Query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			Vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"c969427f56d3a645195dd8365cde6d7feae7e99b"},
		},
	}
	return testcases
}
