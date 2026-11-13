package gql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
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

// In an update, a field left out keeps its value, a null clears it, and a
// value sets it, through the GraphQL mutation as Station Editor sends it.
func TestPathwayUpdate_GraphQLNullClears(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		srv := NewDefaultHandler()
		srv = model.AddConfigAndPerms(cfg, srv)
		srv = usercheck.AdminDefaultMiddleware("test")(srv)
		c := client.New(srv)
		fv := model.FeedVersionInput{ID: toPtr(1)}
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
		const fields = `id traversal_time stair_count min_width signposted_as`
		post := func(mutation string, set map[string]interface{}) gjson.Result {
			t.Helper()
			resp := make(map[string]interface{})
			if err := c.Post(fmt.Sprintf(`mutation($set: PathwaySetInput!) { %s(set: $set) { %s } }`, mutation, fields), &resp, client.Var("set", set)); err != nil {
				t.Fatal(err)
			}
			return gjson.Get(gqlcase.ToJSON(resp), mutation)
		}
		created := post("pathway_create", map[string]interface{}{
			"feed_version":     map[string]interface{}{"id": 1},
			"pathway_id":       fmt.Sprintf("pw-%d", time.Now().UnixNano()),
			"pathway_mode":     2,
			"is_bidirectional": 1,
			"traversal_time":   30,
			"min_width":        1.5,
			"signposted_as":    "Trains",
			"from_stop":        map[string]interface{}{"id": fromStopID},
			"to_stop":          map[string]interface{}{"id": toStopID},
		})
		id := created.Get("id").Int()

		// Left out: unchanged
		updated := post("pathway_update", map[string]interface{}{"id": id, "stair_count": 24})
		assert.Equal(t, int64(24), updated.Get("stair_count").Int())
		assert.Equal(t, int64(30), updated.Get("traversal_time").Int())
		assert.Equal(t, "Trains", updated.Get("signposted_as").String())

		// Null: cleared, and the rest unchanged
		cleared := post("pathway_update", map[string]interface{}{"id": id, "traversal_time": nil, "signposted_as": nil})
		assert.Equal(t, gjson.Null, cleared.Get("traversal_time").Type, "traversal_time should be cleared")
		assert.Empty(t, cleared.Get("signposted_as").String(), "signposted_as should be cleared")
		assert.Equal(t, int64(24), cleared.Get("stair_count").Int())
		assert.Equal(t, 1.5, cleared.Get("min_width").Float())
	})
}

// A null clears a stop's field, as Station Editor sends for "no information".
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				},
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				},
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				},
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				},
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: toPtr(valueA),
					TargetStopID:        toPtr(valueB),
				},
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
				ExternalReference: &model.StopExternalReferenceSetInput{
					TargetFeedOnestopID: nil,
				},
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
