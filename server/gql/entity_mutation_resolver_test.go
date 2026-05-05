package gql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Entity mutation tests

func TestStopCreate(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toPtr("hello"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		}
		eid, err := finder.StopCreate(ctx, stopInput)
		if err != nil {
			t.Fatal(err)
		}
		checkEnt := gtfs.Stop{}
		checkEnt.ID = eid
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX())
		if err := atx.Find(ctx, &checkEnt); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stopInput.StopID, &checkEnt.StopID.Val)
		assert.Equal(t, stopInput.StopName, &checkEnt.StopName.Val)
		assert.Equal(t, stopInput.Geometry.FlatCoords(), checkEnt.Geometry.FlatCoords())
	})
}

func TestStopUpdate(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toPtr("hello"),
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
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX())
		if err := atx.Find(ctx, &checkEnt); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, stopUpdate.StopID, &checkEnt.StopID.Val)
		assert.Equal(t, stopUpdate.Geometry.FlatCoords(), checkEnt.Geometry.FlatCoords())
	})
}

func TestStopUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		stopInput := model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
			StopName:    toPtr("hello"),
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
			StopName: toPtr("hello again"),
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
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		srv, _ := NewServer()
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
		entID := int(gjson.Get(toJson(createResp), "stop_create.id").Int())
		createdUpdatedAt := gjson.Get(toJson(createResp), "stop_create.updated_at").String()
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
		updatedUpdatedAt := gjson.Get(toJson(updateResp), "stop_update.updated_at").String()
		assert.NotEmpty(t, updatedUpdatedAt, "updated_at should be returned on update")
		assert.NotEqual(t, createdUpdatedAt, updatedUpdatedAt,
			"updated_at returned via GraphQL should advance after stop_update (was %s, now %s)",
			createdUpdatedAt, updatedUpdatedAt)
	})
}

func TestPathwayUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		// Create two stops to use as endpoints
		fromStopID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("from-%d", time.Now().UnixNano())),
			StopName:    toPtr("from"),
			Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
		})
		if err != nil {
			t.Fatal(err)
		}
		toStopID, err := finder.StopCreate(ctx, model.StopSetInput{
			FeedVersion: &fv,
			StopID:      toPtr(fmt.Sprintf("to-%d", time.Now().UnixNano())),
			StopName:    toPtr("to"),
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

func TestLevelUpdate_BumpsUpdatedAt(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		atx := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX())
		fv := model.FeedVersionInput{ID: toPtr(1)}
		eid, err := finder.LevelCreate(ctx, model.LevelSetInput{
			FeedVersion: &fv,
			LevelID:     toPtr(fmt.Sprintf("L-%d", time.Now().UnixNano())),
			LevelIndex:  toPtr(0.0),
			LevelName:   toPtr("ground"),
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
			LevelName: toPtr("ground floor"),
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

func TestStopReference(t *testing.T) {
	testStr := func(t *testing.T) string { return fmt.Sprintf("%s-%d", t.Name(), time.Now().UnixNano()) }
	type stopExternalReference struct {
		ID                  int       `db:"id"`
		StopID              tt.Key    `db:"stop_id"`
		TargetFeedOnestopID tt.String `db:"target_feed_onestop_id"`
		TargetStopID        tt.String `db:"target_stop_id"`
	}
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		finder := cfg.Finder
		ctx := model.WithConfig(context.Background(), cfg)
		fv := model.FeedVersionInput{ID: toPtr(1)}
		t.Run("create with stop", func(t *testing.T) {
			valueA := testStr(t)
			valueB := testStr(t)
			stopInput := model.StopSetInput{
				FeedVersion: &fv,
				StopID:      toPtr(fmt.Sprintf("%d", time.Now().UnixNano())),
				StopName:    toPtr("hello"),
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
			if err := sqlx.GetContext(ctx, cfg.Finder.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
				StopName:    toPtr("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			var checkIds []int
			if err := sqlx.SelectContext(ctx, cfg.Finder.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
			if err := sqlx.SelectContext(ctx, cfg.Finder.DBX(), &checkIds2, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 1, len(checkIds2), "expected to have created external reference")
			ret := stopExternalReference{}
			if err := sqlx.GetContext(ctx, cfg.Finder.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
				StopName:    toPtr("hello"),
				Geometry:    toPtr(tt.NewPoint(-122.271604, 37.803664)),
			}
			eid, err := finder.StopCreate(ctx, stopInput)
			if err != nil {
				t.Fatal(err)
			}
			var checkIds []int
			if err := sqlx.SelectContext(ctx, cfg.Finder.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
			if err := sqlx.GetContext(ctx, cfg.Finder.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
				StopName:    toPtr("hello"),
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
			if err := sqlx.GetContext(ctx, cfg.Finder.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
			if err := sqlx.SelectContext(ctx, cfg.Finder.DBX(), &checkIds, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
				StopName:    toPtr("hello"),
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
			if err := sqlx.GetContext(ctx, cfg.Finder.DBX(), &ret, `select * from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
			if err := sqlx.SelectContext(ctx, cfg.Finder.DBX(), &checkIds2, `select id from tl_stop_external_references where stop_id = $1`, eid); err != nil {
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
