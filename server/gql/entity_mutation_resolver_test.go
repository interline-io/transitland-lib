package gql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
