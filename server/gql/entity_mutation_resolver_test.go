package gql

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Entity mutation tests

func TestStopCreate(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		// stop_access is only valid on a platform with a parent_station, so create a parent station first
		parentID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion:  &fv,
			StopID:       toOpt(fmt.Sprintf("station-%d", time.Now().UnixNano())),
			StopName:     toOpt("station"),
			LocationType: toOpt(1),
			Geometry:     toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toOpt("hello"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			Parent:      &model.StopSetInput{ID: toPtr(parentID)},
			StopAccess:  toOpt(1),
		}
		eid, err := finder.StopCreate(ctx, stopInput)
		if err != nil {
			t.Fatal(err)
		}
		checkEnt := gtfs.Stop{}
		checkEnt.ID = eid
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		if err := atx.Find(ctx, &checkEnt); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stopInput.StopID.Value(), &checkEnt.StopID.Val)
		assert.Equal(t, stopInput.StopName.Value(), &checkEnt.StopName.Val)
		assert.Equal(t, stopInput.Geometry.FlatCoords(), checkEnt.Geometry.FlatCoords())
		assert.EqualValues(t, 1, checkEnt.StopAccess.Val)
		assert.True(t, checkEnt.StopAccess.Valid)
	})
}

func TestStopUpdate(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toOpt("hello"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		}
		eid, err := finder.StopCreate(ctx, stopInput)
		if err != nil {
			t.Fatal(err)
		}
		stopUpdate := model.StopSetInput{
			ID:       toPtr(eid),
			StopID:   toOpt(fmt.Sprintf("update-%d", time.Now().UnixNano())),
			Geometry: toPtr(tt.NewPoint(-122.0, 38.0)),
		}
		if _, err := finder.StopUpdate(ctx, stopUpdate); err != nil {
			t.Fatal(err)
		}
		checkEnt := gtfs.Stop{}
		checkEnt.ID = eid
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		if err := atx.Find(ctx, &checkEnt); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stopUpdate.StopID.Value(), &checkEnt.StopID.Val)
		assert.Equal(t, stopUpdate.Geometry.FlatCoords(), checkEnt.Geometry.FlatCoords())
	})
}

func TestStopUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toOpt("hello"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		}
		eid, err := finder.StopCreate(ctx, stopInput)
		if err != nil {
			t.Fatal(err)
		}
		// Capture timestamps right after create
		afterCreate := gtfs.Stop{}
		afterCreate.ID = eid
		if err := atx.Find(ctx, &afterCreate); err != nil {
			t.Fatal(err)
		}
		// Sleep so the updated_at bump is observably later than created_at
		// even on fast machines. Postgres timestamps have microsecond precision.
		time.Sleep(2 * time.Millisecond)
		stopUpdate := model.StopSetInput{
			ID:       toPtr(eid),
			StopName: toOpt("hello again"),
		}
		if _, err := finder.StopUpdate(ctx, stopUpdate); err != nil {
			t.Fatal(err)
		}
		afterUpdate := gtfs.Stop{}
		afterUpdate.ID = eid
		if err := atx.Find(ctx, &afterUpdate); err != nil {
			t.Fatal(err)
		}
		assert.True(t, afterUpdate.CreatedAt.Equal(afterCreate.CreatedAt),
			"created_at should not change on update (was %s, now %s)",
			afterCreate.CreatedAt, afterUpdate.CreatedAt)
		assert.True(t, afterUpdate.UpdatedAt.After(afterCreate.UpdatedAt),
			"updated_at should advance on update (was %s, now %s)",
			afterCreate.UpdatedAt, afterUpdate.UpdatedAt)
	})
}

func TestStopUpdate_GraphQLBumpsUpdatedAt(t *testing.T) {
	// End-to-end check that the stop_update GraphQL mutation
	// (the one called by Station Editor) actually persists an
	// advancing updated_at, observable via a follow-up GraphQL query.
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		srv := NewDefaultHandler()
		srv = model.AddConfigAndPerms(cfg, srv)
		srv = usercheck.AdminDefaultMiddleware("test")(srv)
		c := client.New(srv)
		stopID := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
		createResp := make(map[string]interface{})
		err := c.Post(
			`mutation($set: StopSetInput!) { stop_create(set: $set) { id updated_at } }`,
			&createResp,
			client.Var("set", map[string]interface{}{
				"feed_version": map[string]interface{}{"id": 1},
				"stop_id":      stopID,
				"stop_name":    "hello",
				"geometry":     map[string]interface{}{"type": "Point", "coordinates": []float64{-122.27, 37.80}},
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		entID := int(gjson.Get(gqlcase.ToJSON(createResp), "stop_create.id").Int())
		createdUpdatedAt := gjson.Get(gqlcase.ToJSON(createResp), "stop_create.updated_at").String()
		assert.NotEmpty(t, createdUpdatedAt, "updated_at should be returned on create")
		// Sleep so the bump is observably later (microsecond precision in pg).
		time.Sleep(2 * time.Millisecond)
		updateResp := make(map[string]interface{})
		err = c.Post(
			`mutation($set: StopSetInput!) { stop_update(set: $set) { id updated_at } }`,
			&updateResp,
			client.Var("set", map[string]interface{}{
				"id":        entID,
				"stop_name": "hello again",
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		updatedUpdatedAt := gjson.Get(gqlcase.ToJSON(updateResp), "stop_update.updated_at").String()
		assert.NotEmpty(t, updatedUpdatedAt, "updated_at should be returned on update")
		assert.NotEqual(t, createdUpdatedAt, updatedUpdatedAt,
			"updated_at returned via GraphQL should advance after stop_update (was %s, now %s)",
			createdUpdatedAt, updatedUpdatedAt)
	})
}

func TestPathwayUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		// Create two stops to use as endpoints
		fromStopID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toOpt(fmt.Sprintf("from-%d", time.Now().UnixNano())),
			StopName:    toOpt("from"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		toStopID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toOpt(fmt.Sprintf("to-%d", time.Now().UnixNano())),
			StopName:    toOpt("to"),
			Geometry:    toPtr(tt.NewPoint(-122.271, 37.803)),
		})
		if err != nil {
			t.Fatal(err)
		}
		eid, err := finder.PathwayCreate(ctx, model.PathwaySetInput{
			FeedVersion:     &fv,
			PathwayID:       toOpt(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toOpt(1),
			IsBidirectional: toOpt(0),
			FromStop:        &model.StopSetInput{ID: &fromStopID},
			ToStop:          &model.StopSetInput{ID: &toStopID},
		})
		if err != nil {
			t.Fatal(err)
		}
		afterCreate := gtfs.Pathway{}
		afterCreate.ID = eid
		if err := atx.Find(ctx, &afterCreate); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
		if _, err := finder.PathwayUpdate(ctx, model.PathwaySetInput{
			ID:        toPtr(eid),
			PathwayID: toOpt(fmt.Sprintf("pw-update-%d", time.Now().UnixNano())),
		}); err != nil {
			t.Fatal(err)
		}
		afterUpdate := gtfs.Pathway{}
		afterUpdate.ID = eid
		if err := atx.Find(ctx, &afterUpdate); err != nil {
			t.Fatal(err)
		}
		assert.True(t, afterUpdate.CreatedAt.Equal(afterCreate.CreatedAt),
			"created_at should not change on pathway update")
		assert.True(t, afterUpdate.UpdatedAt.After(afterCreate.UpdatedAt),
			"updated_at should advance on pathway update (was %s, now %s)",
			afterCreate.UpdatedAt, afterUpdate.UpdatedAt)
	})
}

// gqlClient is a GraphQL client against the handler, as an admin.
func gqlClient(cfg model.Config) *client.Client {
	srv := NewDefaultHandler()
	srv = model.AddConfigAndPerms(cfg, srv)
	srv = usercheck.AdminDefaultMiddleware("test")(srv)
	return client.New(srv)
}

// postSet runs a create or update mutation with its input as the $set
// variable, and returns the mutation's result.
func postSet(c *client.Client, mutation string, input string, set map[string]any, fields string) (gjson.Result, error) {
	resp := make(map[string]any)
	err := c.Post(fmt.Sprintf(`mutation($set: %s!) { %s(set: $set) { %s } }`, input, mutation, fields), &resp, client.Var("set", set))
	return gjson.Get(gqlcase.ToJSON(resp), mutation), err
}

// createTestStop creates a stop in feed version 1 and returns its id.
func createTestStop(t *testing.T, ctx context.Context, finder model.Finder, name string) int {
	t.Helper()
	id, err := finder.StopCreate(ctx, model.StopSetInput{
		FeedVersion: &model.FeedVersionInput{ID: toPtr(1)},
		StopID:      toOpt(fmt.Sprintf("%s-%d", name, time.Now().UnixNano())),
		StopName:    toOpt(name),
		Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// countStopRefs counts a stop's external references.
func countStopRefs(t *testing.T, ctx context.Context, cfg model.Config, stopID int64) int {
	t.Helper()
	n := 0
	if err := dbutil.Get(ctx, cfg.Adapter.DBX(), sq.StatementBuilder.Select("count(*)").From("tl_stop_external_references").Where(sq.Eq{"stop_id": stopID}), &n); err != nil {
		t.Fatal(err)
	}
	return n
}

// In an update, a field left out keeps its value, a null clears it, and a
// value sets it, through the GraphQL mutation.
func TestPathwayUpdate_GraphQLNullClears(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		c := gqlClient(cfg)
		fromStopID, toStopID := createTestStop(t, ctx, finder, "from"), createTestStop(t, ctx, finder, "to")
		const fields = `id traversal_time stair_count min_width signposted_as`
		created, err := postSet(c, "pathway_create", "PathwaySetInput", map[string]any{
			"feed_version":     map[string]any{"id": 1},
			"pathway_id":       fmt.Sprintf("pw-%d", time.Now().UnixNano()),
			"pathway_mode":     2,
			"is_bidirectional": 1,
			"traversal_time":   30,
			"min_width":        1.5,
			"signposted_as":    "Trains",
			"from_stop":        map[string]any{"id": fromStopID},
			"to_stop":          map[string]any{"id": toStopID},
		}, fields)
		if err != nil {
			t.Fatal(err)
		}
		id := created.Get("id").Int()
		stored := func() gtfs.Pathway {
			t.Helper()
			ent := gtfs.Pathway{}
			ent.ID = int(id)
			if err := atx.Find(ctx, &ent); err != nil {
				t.Fatal(err)
			}
			return ent
		}

		// Left out: unchanged
		if _, err := postSet(c, "pathway_update", "PathwaySetInput", map[string]any{"id": id, "stair_count": 24}, fields); err != nil {
			t.Fatal(err)
		}
		ent := stored()
		assert.Equal(t, int64(24), ent.StairCount.Val)
		assert.Equal(t, int64(30), ent.TraversalTime.Val)
		assert.Equal(t, "Trains", ent.SignpostedAs.Val)

		// Null: cleared, to NULL in the database, and the rest unchanged
		if _, err := postSet(c, "pathway_update", "PathwaySetInput", map[string]any{"id": id, "traversal_time": nil, "signposted_as": nil}, fields); err != nil {
			t.Fatal(err)
		}
		var nulls struct {
			TraversalTime bool `db:"traversal_time"`
			SignpostedAs  bool `db:"signposted_as"`
		}
		if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &nulls, `select traversal_time is null as traversal_time, signposted_as is null as signposted_as from gtfs_pathways where id = $1`, id); err != nil {
			t.Fatal(err)
		}
		assert.True(t, nulls.TraversalTime, "traversal_time should be NULL")
		assert.True(t, nulls.SignpostedAs, "signposted_as should be NULL")
		ent = stored()
		assert.Equal(t, int64(24), ent.StairCount.Val)
		assert.Equal(t, 1.5, ent.MinWidth.Val)

		// A required field can't be cleared
		_, err = postSet(c, "pathway_update", "PathwaySetInput", map[string]any{"id": id, "pathway_mode": nil}, fields)
		assert.ErrorContains(t, err, "pathway_mode")
		assert.ErrorContains(t, err, "required")
	})
}

// A field written as a variable the request didn't provide is left out, as
// the GraphQL spec says, not cleared.
func TestPathwayUpdate_GraphQLMissingVariable(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		c := gqlClient(cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		eid, err := finder.PathwayCreate(ctx, model.PathwaySetInput{
			FeedVersion:     &fv,
			PathwayID:       toOpt(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toOpt(1),
			IsBidirectional: toOpt(1),
			TraversalTime:   toOpt(30),
			FromStop:        &model.StopSetInput{ID: toPtr(createTestStop(t, ctx, finder, "from"))},
			ToStop:          &model.StopSetInput{ID: toPtr(createTestStop(t, ctx, finder, "to"))},
		})
		if err != nil {
			t.Fatal(err)
		}
		resp := make(map[string]any)
		if err := c.Post(`mutation($id: Int!, $sc: Int, $tt: Int) { pathway_update(set: {id: $id, stair_count: $sc, traversal_time: $tt}) { id } }`, &resp, client.Var("id", eid), client.Var("sc", 12)); err != nil {
			t.Fatal(err)
		}
		ent := gtfs.Pathway{}
		ent.ID = eid
		if err := atx.Find(ctx, &ent); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(12), ent.StairCount.Val)
		assert.Equal(t, int64(30), ent.TraversalTime.Val, "traversal_time, its variable not provided, should be unchanged")
	})
}

// Nested too: a reference or an external reference given only as variables
// the request didn't provide is left out, not cleared.
func TestStopUpdate_GraphQLMissingNestedVariables(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		c := gqlClient(cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		parentID, err := finder.StopCreate(ctx, model.StopSetInput{FeedVersion: &fv, StopID: toOpt(fmt.Sprintf("station-%d", time.Now().UnixNano())), StopName: toOpt("station"), LocationType: toOpt(1), Geometry: toPtr(tt.NewPoint(-122.27, 37.80))})
		if err != nil {
			t.Fatal(err)
		}
		eid, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion:       &fv,
			StopID:            toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:          toOpt("platform"),
			Geometry:          toPtr(tt.NewPoint(-122.27, 37.80)),
			Parent:            &model.StopSetInput{ID: toPtr(parentID)},
			ExternalReference: toOpt(model.StopExternalReferenceSetInput{TargetFeedOnestopID: toPtr("f-test"), TargetStopID: toPtr("x")}),
		})
		if err != nil {
			t.Fatal(err)
		}
		resp := make(map[string]any)
		if err := c.Post(`mutation($id: Int!, $name: String, $p: Int, $f: String, $s: String) {
			stop_update(set: {id: $id, stop_name: $name, parent: {id: $p}, external_reference: {target_feed_onestop_id: $f, target_stop_id: $s}}) { id }
		}`, &resp, client.Var("id", eid), client.Var("name", "renamed")); err != nil {
			t.Fatal(err)
		}
		ent := gtfs.Stop{}
		ent.ID = eid
		if err := atx.Find(ctx, &ent); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "renamed", ent.StopName.Val)
		assert.Equal(t, fmt.Sprintf("%d", parentID), ent.ParentStation.Val, "parent, its variable not provided, should be unchanged")
		assert.Equal(t, 1, countStopRefs(t, ctx, cfg, int64(eid)), "the external reference, its variables not provided, should be kept")
	})
}

// In a create, where there's nothing to clear, a null for a field that must
// keep a value is left out: location_type takes its default.
func TestStopCreate_NullRequiredIsLeftOut(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		c := gqlClient(cfg)
		created, err := postSet(c, "stop_create", "StopSetInput", map[string]any{
			"feed_version":  map[string]any{"id": 1},
			"stop_id":       fmt.Sprintf("s-%d", time.Now().UnixNano()),
			"stop_name":     "Platform",
			"location_type": nil,
			"geometry":      map[string]any{"type": "Point", "coordinates": []float64{-122.27, 37.80}},
		}, `id location_type`)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(0), created.Get("location_type").Int())
	})
}

// A stop's external reference given again replaces its target whole: a null
// target stop doesn't keep the old feed's stop.
func TestStopUpdate_ExternalReferenceReplacesTarget(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		c := gqlClient(cfg)
		created, err := postSet(c, "stop_create", "StopSetInput", map[string]any{
			"feed_version":       map[string]any{"id": 1},
			"stop_id":            fmt.Sprintf("s-%d", time.Now().UnixNano()),
			"stop_name":          "Platform",
			"geometry":           map[string]any{"type": "Point", "coordinates": []float64{-122.27, 37.80}},
			"external_reference": map[string]any{"target_feed_onestop_id": "f-a", "target_stop_id": "x"},
		}, `id`)
		if err != nil {
			t.Fatal(err)
		}
		id := created.Get("id").Int()
		if _, err := postSet(c, "stop_update", "StopSetInput", map[string]any{"id": id, "external_reference": map[string]any{"target_feed_onestop_id": "f-b", "target_stop_id": nil}}, `id`); err != nil {
			t.Fatal(err)
		}
		var ref struct {
			Feed string  `db:"target_feed_onestop_id"`
			Stop *string `db:"target_stop_id"`
		}
		if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ref, `select target_feed_onestop_id, target_stop_id from tl_stop_external_references where stop_id = $1`, id); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "f-b", ref.Feed)
		assert.Nil(t, ref.Stop, "the old feed's target stop shouldn't be kept")
	})
}

// A stop's fields cleared through GraphQL: text to an empty string, which
// keeps its text search; a number to NULL; and its external reference
// removed. A required field, or a name its location type needs, can't be.
func TestStopUpdate_GraphQLNullClears(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		c := gqlClient(cfg)
		const fields = `id`
		created, err := postSet(c, "stop_create", "StopSetInput", map[string]any{
			"feed_version":        map[string]any{"id": 1},
			"stop_id":             fmt.Sprintf("s-%d", time.Now().UnixNano()),
			"stop_name":           "Platform 1",
			"stop_desc":           "By the stairs",
			"stop_code":           "A1",
			"location_type":       0,
			"wheelchair_boarding": 1,
			"geometry":            map[string]any{"type": "Point", "coordinates": []float64{-122.27, 37.80}},
			"external_reference":  map[string]any{"target_feed_onestop_id": "f-test", "target_stop_id": "x"},
		}, fields)
		if err != nil {
			t.Fatal(err)
		}
		id := created.Get("id").Int()
		assert.Equal(t, 1, countStopRefs(t, ctx, cfg, id))

		if _, err := postSet(c, "stop_update", "StopSetInput", map[string]any{"id": id, "stop_desc": nil, "wheelchair_boarding": nil, "external_reference": nil}, fields); err != nil {
			t.Fatal(err)
		}
		var row struct {
			StopDesc           *string `db:"stop_desc"`
			WheelchairBoarding *int    `db:"wheelchair_boarding"`
			Searchable         bool    `db:"searchable"`
		}
		if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &row, `select stop_desc, wheelchair_boarding, textsearch is not null as searchable from gtfs_stops where id = $1`, id); err != nil {
			t.Fatal(err)
		}
		if assert.NotNil(t, row.StopDesc) {
			assert.Equal(t, "", *row.StopDesc, "stop_desc should be an empty string, not NULL")
		}
		assert.Nil(t, row.WheelchairBoarding, "wheelchair_boarding should be NULL")
		assert.True(t, row.Searchable, "the stop's text search vector should survive")
		assert.Equal(t, 0, countStopRefs(t, ctx, cfg, id), "the external reference should be removed")
		ent := gtfs.Stop{}
		ent.ID = int(id)
		if err := atx.Find(ctx, &ent); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "A1", ent.StopCode.Val, "stop_code, left out, should be unchanged")

		// A platform needs a name; location_type can't be cleared
		_, err = postSet(c, "stop_update", "StopSetInput", map[string]any{"id": id, "stop_name": nil}, fields)
		assert.ErrorContains(t, err, "no value for required field stop_name")
		_, err = postSet(c, "stop_update", "StopSetInput", map[string]any{"id": id, "location_type": nil}, fields)
		assert.ErrorContains(t, err, "location_type can't be cleared")
	})
}

// A null clears a stop's field through the finder too.
func TestStopUpdate_NullClears(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		eid, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion:        &fv,
			StopID:             toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:           toOpt("hello"),
			StopCode:           toOpt("A1"),
			WheelchairBoarding: toOpt(1),
			Geometry:           toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := finder.StopUpdate(ctx, model.StopSetInput{ID: toPtr(eid), WheelchairBoarding: toNull[int]()}); err != nil {
			t.Fatal(err)
		}
		ent := gtfs.Stop{}
		ent.ID = eid
		if err := atx.Find(ctx, &ent); err != nil {
			t.Fatal(err)
		}
		assert.False(t, ent.WheelchairBoarding.Valid, "wheelchair_boarding should be cleared")
		assert.Equal(t, "A1", ent.StopCode.Val, "stop_code, left out, should be unchanged")
	})
}

// A level's name can be cleared; its index, required, can't.
func TestLevelUpdate_GraphQLNullClears(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		c := gqlClient(cfg)
		const fields = `id level_name level_index`
		created, err := postSet(c, "level_create", "LevelSetInput", map[string]any{
			"feed_version": map[string]any{"id": 1},
			"level_id":     fmt.Sprintf("l-%d", time.Now().UnixNano()),
			"level_name":   "Mezzanine",
			"level_index":  -1,
		}, fields)
		if err != nil {
			t.Fatal(err)
		}
		id := created.Get("id").Int()
		cleared, err := postSet(c, "level_update", "LevelSetInput", map[string]any{"id": id, "level_name": nil}, fields)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, gjson.Null, cleared.Get("level_name").Type)
		assert.Equal(t, -1.0, cleared.Get("level_index").Float())
		_, err = postSet(c, "level_update", "LevelSetInput", map[string]any{"id": id, "level_index": nil}, fields)
		assert.ErrorContains(t, err, "level_index")
		assert.ErrorContains(t, err, "required")
	})
}

// Updating one end of a pathway changes that end and leaves the other.
func TestPathwayUpdate_Stops(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		// BART, from the standard test fixtures.
		sha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
		fvs, err := finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
		if err != nil {
			t.Fatal(err)
		}
		if len(fvs) != 1 {
			t.Fatalf("expected the bart test feed version, got %d", len(fvs))
		}
		fv := model.FeedVersionInput{ID: toPtr(fvs[0].ID)}
		newStop := func(name string) int {
			t.Helper()
			id, err := finder.StopCreate(ctx, model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%s-%d", name, time.Now().UnixNano())),
				StopName:    toOpt(name),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			})
			if err != nil {
				t.Fatal(err)
			}
			return id
		}
		fromStopID, toStopID := newStop("from"), newStop("to")
		eid, err := finder.PathwayCreate(ctx, model.PathwaySetInput{
			FeedVersion:     &fv,
			PathwayID:       toOpt(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toOpt(1),
			IsBidirectional: toOpt(0),
			FromStop:        &model.StopSetInput{ID: &fromStopID},
			ToStop:          &model.StopSetInput{ID: &toStopID},
		})
		if err != nil {
			t.Fatal(err)
		}
		update := func(input model.PathwaySetInput) gtfs.Pathway {
			t.Helper()
			input.ID = toPtr(eid)
			if _, err := finder.PathwayUpdate(ctx, input); err != nil {
				t.Fatal(err)
			}
			checkEnt := gtfs.Pathway{}
			checkEnt.ID = eid
			atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
			if err := atx.Find(ctx, &checkEnt); err != nil {
				t.Fatal(err)
			}
			return checkEnt
		}

		toStopID = newStop("new-to")
		checkEnt := update(model.PathwaySetInput{ToStop: &model.StopSetInput{ID: &toStopID}})
		assert.Equal(t, toStopID, checkEnt.ToStopID.Int(), "to_stop should change")
		assert.Equal(t, fromStopID, checkEnt.FromStopID.Int(), "from_stop should not change")

		fromStopID = newStop("new-from")
		checkEnt = update(model.PathwaySetInput{FromStop: &model.StopSetInput{ID: &fromStopID}})
		assert.Equal(t, fromStopID, checkEnt.FromStopID.Int(), "from_stop should change")
		assert.Equal(t, toStopID, checkEnt.ToStopID.Int(), "to_stop should not change")
	})
}

func TestLevelUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		eid, err := finder.LevelCreate(ctx, model.LevelSetInput{
			FeedVersion: &fv,
			LevelID:     toOpt(fmt.Sprintf("L-%d", time.Now().UnixNano())),
			LevelIndex:  toOpt(0.0),
			LevelName:   toOpt("ground"),
		})
		if err != nil {
			t.Fatal(err)
		}
		afterCreate := gtfs.Level{}
		afterCreate.ID = eid
		if err := atx.Find(ctx, &afterCreate); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
		if _, err := finder.LevelUpdate(ctx, model.LevelSetInput{
			ID:        toPtr(eid),
			LevelName: toOpt("ground floor"),
		}); err != nil {
			t.Fatal(err)
		}
		afterUpdate := gtfs.Level{}
		afterUpdate.ID = eid
		if err := atx.Find(ctx, &afterUpdate); err != nil {
			t.Fatal(err)
		}
		assert.True(t, afterUpdate.CreatedAt.Equal(afterCreate.CreatedAt),
			"created_at should not change on level update")
		assert.True(t, afterUpdate.UpdatedAt.After(afterCreate.UpdatedAt),
			"updated_at should advance on level update (was %s, now %s)",
			afterCreate.UpdatedAt, afterUpdate.UpdatedAt)
	})
}

// A level's parent station is set by parent {id}, and cleared by parent {}, as a stop's is.
func TestLevelUpdate_Parent(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		// BART, from the standard test fixtures.
		sha1 := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
		fvs, err := finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
		if err != nil {
			t.Fatal(err)
		}
		if len(fvs) != 1 {
			t.Fatalf("expected the bart test feed version, got %d", len(fvs))
		}
		fv := model.FeedVersionInput{ID: toPtr(fvs[0].ID)}
		stationID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion:  &fv,
			StopID:       toOpt(fmt.Sprintf("station-%d", time.Now().UnixNano())),
			StopName:     toOpt("station"),
			LocationType: toOpt(1),
			Geometry:     toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		eid, err := finder.LevelCreate(ctx, model.LevelSetInput{
			FeedVersion: &fv,
			LevelID:     toOpt(fmt.Sprintf("L-%d", time.Now().UnixNano())),
			LevelIndex:  toOpt(0.0),
			Parent:      &model.StopSetInput{ID: &stationID},
		})
		if err != nil {
			t.Fatal(err)
		}
		parent := func() tt.Key {
			t.Helper()
			levels, errs := finder.LevelsByIDs(ctx, []int{eid})
			if len(errs) > 0 && errs[0] != nil {
				t.Fatal(errs[0])
			}
			if len(levels) != 1 || levels[0] == nil {
				t.Fatalf("expected level %d", eid)
			}
			return levels[0].ParentStation
		}
		assert.Equal(t, stationID, parent().Int(), "parent {id} should set the parent station")
		if _, err := finder.LevelUpdate(ctx, model.LevelSetInput{
			ID:     toPtr(eid),
			Parent: &model.StopSetInput{},
		}); err != nil {
			t.Fatal(err)
		}
		assert.False(t, parent().Valid, "parent {} should clear the parent station")
	})
}

func TestStopReference(t *testing.T) {
	testStr := func(t *testing.T) string { return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()) }
	type stopExternalReference struct {
		ID                  int       `db:"id"`
		StopID              tt.Key    `db:"stop_id"`
		TargetFeedOnestopID tt.String `db:"target_feed_onestop_id"`
		TargetStopID        tt.String `db:"target_stop_id"`
	}
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		t.Run("create with stop", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toOpt("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				}),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, eid, ret.StopID.Int())
			assert.Equal(t, valueA, ret.TargetFeedOnestopID.Val)
			assert.Equal(t, valueB, ret.TargetStopID.Val)
		})
		t.Run("create with stop update", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toOpt("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			var checkIds []int
			if err := sqlx.SelectContext(ctx, cfg.Adapter.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, len(checkIds), "should not have created external reference")
			//////////
			if _, err := finder.StopUpdate(ctx, model.StopSetInput{
				ID: &eid,
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				}),
			}); err != nil {
				t.Fatal(err)
			}
			var checkIds2 []int
			if err := sqlx.SelectContext(ctx, cfg.Adapter.DBX(), &checkIds2, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 1, len(checkIds2), "expected to have created external reference")
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, eid, ret.StopID.Int())
			assert.Equal(t, valueA, ret.TargetFeedOnestopID.Val)
			assert.Equal(t, valueB, ret.TargetStopID.Val)
		})
		t.Run("update with stop update", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toOpt("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			var checkIds []int
			if err := sqlx.SelectContext(ctx, cfg.Adapter.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, len(checkIds), "should not have created external reference")
			//////////
			if _, err := finder.StopUpdate(ctx, model.StopSetInput{
				ID: &eid,
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				}),
			}); err != nil {
				t.Fatal(err)
			}
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, eid, ret.StopID.Int())
			assert.Equal(t, valueA, ret.TargetFeedOnestopID.Val)
			assert.Equal(t, valueB, ret.TargetStopID.Val)
		})
		t.Run("delete from stop delete", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toOpt("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				}),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, eid, ret.StopID.Int())
			assert.Equal(t, valueA, ret.TargetFeedOnestopID.Val)
			assert.Equal(t, valueB, ret.TargetStopID.Val)
			// Delete
			if err := finder.StopDelete(ctx, eid); err != nil {
				t.Fatal(err)
			}
			// Check the external reference is gone
			var checkIds []int
			if err := sqlx.SelectContext(ctx, cfg.Adapter.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, len(checkIds), "deleted stop should not have external reference")
		})
		t.Run("delete without stop delete", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toOpt(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toOpt("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				}),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, eid, ret.StopID.Int())
			//////////
			if _, err := finder.StopUpdate(ctx, model.StopSetInput{
				ID: &eid,
				ExternalReference: toOpt(model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: nil,
				}),
			}); err != nil {
				t.Fatal(err)
			}
			var checkIds2 []int
			if err := sqlx.SelectContext(ctx, cfg.Adapter.DBX(), &checkIds2, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, len(checkIds2), "expected to have deleted external reference")
		})
	})
}

func toPtr[T any, P *T](v T) P {
	vcopy := v
	return &vcopy
}

// toOpt is an input field set to v
func toOpt[T any](v T) graphql.Omittable[*T] {
	return graphql.OmittableOf(&v)
}

// toNull is an input field set to null, which clears it
func toNull[T any]() graphql.Omittable[*T] {
	return graphql.OmittableOf[*T](nil)
}

// Every field of the stop, level and pathway inputs that sets a column tells
// null from left out: a new field that isn't Omittable in gqlgen.yml would
// quietly keep its value when sent null.
func TestSetInputsAreOmittable(t *testing.T) {
	refs := map[string]bool{"ID": true, "FeedVersion": true, "Geometry": true, "Parent": true, "Level": true, "FromStop": true, "ToStop": true}
	for _, input := range []any{model.StopSetInput{}, model.LevelSetInput{}, model.PathwaySetInput{}} {
		typ := reflect.TypeOf(input)
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if refs[f.Name] {
				continue
			}
			if !strings.HasPrefix(f.Type.Name(), "Omittable[") {
				t.Errorf("%s.%s is %s: make it omittable in gqlgen.yml", typ.Name(), f.Name, f.Type)
			}
		}
	}
}
