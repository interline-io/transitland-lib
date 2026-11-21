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
			StopID:       toPtr(fmt.Sprintf("station-%d", time.Now().UnixNano())),
			StopName:     toOpt("station"),
			LocationType: toPtr(1),
			Geometry:     toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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
		assert.Equal(t, stopInput.StopID, &checkEnt.StopID.Val)
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
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toOpt("hello"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		}
		eid, err := finder.StopCreate(ctx, stopInput)
		if err != nil {
			t.Fatal(err)
		}
		stopUpdate := model.StopSetInput{
			ID:       toPtr(eid),
			StopID:   toPtr(fmt.Sprintf("update-%d", time.Now().UnixNano())),
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
		assert.Equal(t, stopUpdate.StopID, &checkEnt.StopID.Val)
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
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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

// In an update, null or "" clears a stop's text and null clears a number, an
// omitted field keeps its value, and the stop keeps its text search.
func TestStopUpdate_GraphQLNullClears(t *testing.T) {
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
		eid, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion:        &model.FeedVersionInput{ID: toPtr(fvs[0].ID)},
			StopID:             toPtr(fmt.Sprintf("s-%d", time.Now().UnixNano())),
			StopName:           toOpt("Platform 1"),
			StopDesc:           toOpt("By the stairs"),
			StopCode:           toOpt("A1"),
			ZoneID:             toOpt("Z1"),
			WheelchairBoarding: toOpt(1),
			Geometry:           toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		stored := func() gtfs.Stop {
			t.Helper()
			ent := gtfs.Stop{}
			ent.ID = eid
			if err := atx.Find(ctx, &ent); err != nil {
				t.Fatal(err)
			}
			return ent
		}
		before := stored()
		assert.Equal(t, "By the stairs", before.StopDesc.Val)
		assert.Equal(t, "A1", before.StopCode.Val)
		assert.Equal(t, 1, before.WheelchairBoarding.Int())

		srv := NewDefaultHandler()
		srv = model.AddConfigAndPerms(cfg, srv)
		srv = usercheck.AdminDefaultMiddleware("test")(srv)
		c := client.New(srv)
		resp := make(map[string]any)
		if err := c.Post(
			`mutation($set: StopSetInput!) { stop_update(set: $set) { id } }`,
			&resp,
			client.Var("set", map[string]any{
				"id":                  eid,
				"stop_desc":           nil,
				"stop_code":           "",
				"wheelchair_boarding": nil,
			}),
		); err != nil {
			t.Fatal(err)
		}
		after := stored()
		assert.False(t, after.StopDesc.Valid, "null should clear stop_desc")
		assert.False(t, after.StopCode.Valid, `"" should clear stop_code`)
		assert.False(t, after.WheelchairBoarding.Valid, "null should clear wheelchair_boarding")
		assert.Equal(t, "Z1", after.ZoneID.Val, "an omitted zone_id should keep its value")
		var searchable bool
		if err := sqlx.GetContext(ctx, cfg.Adapter.DBX(), &searchable, `select textsearch is not null from gtfs_stops where id = $1`, eid); err != nil {
			t.Fatal(err)
		}
		assert.True(t, searchable, "clearing text should keep the stop's textsearch vector")
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
			StopID:      toPtr(fmt.Sprintf("from-%d", time.Now().UnixNano())),
			StopName:    toOpt("from"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		toStopID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("to-%d", time.Now().UnixNano())),
			StopName:    toOpt("to"),
			Geometry:    toPtr(tt.NewPoint(-122.271, 37.803)),
		})
		if err != nil {
			t.Fatal(err)
		}
		eid, err := finder.PathwayCreate(ctx, model.PathwaySetInput{
			FeedVersion:     &fv,
			PathwayID:       toPtr(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toPtr(1),
			IsBidirectional: toPtr(0),
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
			PathwayID: toPtr(fmt.Sprintf("pw-update-%d", time.Now().UnixNano())),
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
				StopID:      toPtr(fmt.Sprintf("%s-%d", name, time.Now().UnixNano())),
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
			PathwayID:       toPtr(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toPtr(1),
			IsBidirectional: toPtr(0),
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

// In an update, null clears a pathway's optional field and leaves a required
// one, and an omitted field keeps its value.
func TestPathwayUpdate_GraphQLNullClears(t *testing.T) {
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
				StopID:      toPtr(fmt.Sprintf("%s-%d", name, time.Now().UnixNano())),
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
			PathwayID:       toPtr(fmt.Sprintf("pw-%d", time.Now().UnixNano())),
			PathwayMode:     toPtr(2),
			IsBidirectional: toPtr(1),
			TraversalTime:   toOpt(30),
			StairCount:      toOpt(24),
			SignpostedAs:    toOpt("Trains"),
			FromStop:        &model.StopSetInput{ID: &fromStopID},
			ToStop:          &model.StopSetInput{ID: &toStopID},
		})
		if err != nil {
			t.Fatal(err)
		}
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Adapter.DBX())
		stored := func() gtfs.Pathway {
			t.Helper()
			ent := gtfs.Pathway{}
			ent.ID = eid
			if err := atx.Find(ctx, &ent); err != nil {
				t.Fatal(err)
			}
			return ent
		}
		before := stored()
		assert.Equal(t, 30, before.TraversalTime.Int())
		assert.Equal(t, "Trains", before.SignpostedAs.Val)

		srv := NewDefaultHandler()
		srv = model.AddConfigAndPerms(cfg, srv)
		srv = usercheck.AdminDefaultMiddleware("test")(srv)
		c := client.New(srv)
		resp := make(map[string]any)
		if err := c.Post(
			`mutation($set: PathwaySetInput!) { pathway_update(set: $set) { id } }`,
			&resp,
			client.Var("set", map[string]any{
				"id":             eid,
				"traversal_time": nil,
				"signposted_as":  nil,
				"pathway_mode":   nil,
			}),
		); err != nil {
			t.Fatal(err)
		}
		after := stored()
		assert.False(t, after.TraversalTime.Valid, "null should clear traversal_time")
		assert.False(t, after.SignpostedAs.Valid, "null should clear signposted_as")
		assert.Equal(t, 24, after.StairCount.Int(), "an omitted stair_count should keep its value")
		assert.Equal(t, 2, after.PathwayMode.Int(), "null should leave the required pathway_mode")
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
			LevelID:     toPtr(fmt.Sprintf("L-%d", time.Now().UnixNano())),
			LevelIndex:  toPtr(0.0),
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
			StopID:       toPtr(fmt.Sprintf("station-%d", time.Now().UnixNano())),
			StopName:     toOpt("station"),
			LocationType: toPtr(1),
			Geometry:     toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		eid, err := finder.LevelCreate(ctx, model.LevelSetInput{
			FeedVersion: &fv,
			LevelID:     toPtr(fmt.Sprintf("L-%d", time.Now().UnixNano())),
			LevelIndex:  toPtr(0.0),
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
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
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

// toOpt returns an Omittable input field set to v.
func toOpt[T any](v T) graphql.Omittable[*T] {
	return graphql.OmittableOf(&v)
}
