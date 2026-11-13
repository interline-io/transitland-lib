package dbfinder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

func (f *Finder) StopCreate(ctx context.Context, input model.StopSetInput) (int, error) {
	if input.FeedVersion == nil || input.FeedVersion.ID == nil {
		return 0, errors.New("feed_version_id required")
	}
	return createUpdateStop(ctx, input)
}

func (f *Finder) StopUpdate(ctx context.Context, input model.StopSetInput) (int, error) {
	if input.ID == nil {
		return 0, errors.New("id required")
	}
	return createUpdateStop(ctx, input)
}

func (f *Finder) StopDelete(ctx context.Context, id int) error {
	dels := []deleteRef{
		{ID: id, TableName: "tl_stop_external_references", ColumnName: "stop_id"},
	}
	ent := gtfs.Stop{}
	ent.ID = id
	return deleteEnt(ctx, &ent, dels...)
}

func createUpdateStop(ctx context.Context, input model.StopSetInput) (int, error) {
	// The stop and its external reference are saved together, or not at all
	stopId := 0
	cfg := model.ForContext(ctx)
	err := cfg.Adapter.Tx(func(atx tldb.Adapter) error {
		txcfg := cfg
		txcfg.Adapter = atx
		ctx := model.WithConfig(ctx, txcfg)
		update := isUpdate(input.ID)
		var err error
		stopId, err = createUpdateEnt(
			ctx,
			input.ID,
			fvint(input.FeedVersion),
			&gtfs.Stop{},
			func(ent *gtfs.Stop) ([]string, error) {
				var cols []string
				var err error
				cols = scanReq(&ent.StopID, input.StopID, "stop_id", update, cols, &err)
				cols = scanOpt(&ent.StopCode, input.StopCode, "stop_code", cols, &err)
				cols = scanOpt(&ent.StopDesc, input.StopDesc, "stop_desc", cols, &err)
				cols = scanOpt(&ent.StopTimezone, input.StopTimezone, "stop_timezone", cols, &err)
				cols = scanOpt(&ent.StopName, input.StopName, "stop_name", cols, &err)
				cols = scanOpt(&ent.StopURL, input.StopURL, "stop_url", cols, &err)
				cols = scanReq(&ent.LocationType, input.LocationType, "location_type", update, cols, &err)
				cols = scanOpt(&ent.WheelchairBoarding, input.WheelchairBoarding, "wheelchair_boarding", cols, &err)
				cols = scanOpt(&ent.ZoneID, input.ZoneID, "zone_id", cols, &err)
				cols = scanOpt(&ent.TtsStopName, input.TtsStopName, "tts_stop_name", cols, &err)
				cols = scanOpt(&ent.PlatformCode, input.PlatformCode, "platform_code", cols, &err)
				cols = scanOpt(&ent.StopAccess, input.StopAccess, "stop_access", cols, &err)
				if input.Geometry != nil && input.Geometry.Valid {
					cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
				}
				if v := input.Parent; v != nil {
					cols = scanRef(&ent.ParentStation, v.ID, "parent_station", cols, &err)
				}
				if v := input.Level; v != nil {
					cols = scanRef(&ent.LevelID, v.ID, "level_id", cols, &err)
				}
				return cols, err
			})
		if err != nil {
			return err
		}
		if input.ExternalReference.IsSet() {
			return setStopExternalReference(ctx, stopId, input.ExternalReference.Value())
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return stopId, nil
}

// setStopExternalReference sets a stop's external reference, replacing its
// target feed and stop, or removes it when ref or its target feed is nil.
func setStopExternalReference(ctx context.Context, stopId int, ref *model.StopExternalReferenceSetInput) error {
	db := toAtx(ctx).DBX()
	var existing model.StopExternalReference
	if err := dbutil.Get(ctx, db, sq.StatementBuilder.Select("id").From("tl_stop_external_references").Where(sq.Eq{"stop_id": stopId}), &existing); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to find the stop external reference for stop %d: %w", stopId, err)
	}
	if ref == nil || ref.TargetFeedOnestopID == nil {
		if existing.ID == 0 {
			return nil
		}
		if err := deleteEnt(ctx, &existing); err != nil {
			return fmt.Errorf("failed to delete stop external reference for stop %d: %w", stopId, err)
		}
		return nil
	}
	var fvid *int
	if err := dbutil.Get(ctx, db, sq.StatementBuilder.Select("feed_version_id").From("gtfs_stops").Where(sq.Eq{"id": stopId}), &fvid); err != nil {
		return fmt.Errorf("failed to find the feed version of stop %d: %w", stopId, err)
	}
	if _, err := createUpdateEnt(
		ctx,
		&existing.ID,
		fvid,
		&model.StopExternalReference{},
		func(ent *model.StopExternalReference) ([]string, error) {
			var cols []string
			var err error
			cols = scanCol(&ent.StopID, &stopId, "stop_id", cols, &err)
			// The target is the one given, whole: a null target stop clears the old one
			cols = scanRef(&ent.TargetFeedOnestopID, ref.TargetFeedOnestopID, "target_feed_onestop_id", cols, &err)
			cols = scanRef(&ent.TargetStopID, ref.TargetStopID, "target_stop_id", cols, &err)
			return cols, err
		},
	); err != nil {
		return fmt.Errorf("failed to create or update stop external reference for stop %d: %w", stopId, err)
	}
	return nil
}

///////////

func (f *Finder) PathwayCreate(ctx context.Context, input model.PathwaySetInput) (int, error) {
	if input.FeedVersion == nil || input.FeedVersion.ID == nil {
		return 0, errors.New("feed_version_id required")
	}
	return createUpdatePathway(ctx, input)
}

func (f *Finder) PathwayUpdate(ctx context.Context, input model.PathwaySetInput) (int, error) {
	if input.ID == nil {
		return 0, errors.New("id required")
	}
	return createUpdatePathway(ctx, input)
}

func (f *Finder) PathwayDelete(ctx context.Context, id int) error {
	ent := gtfs.Pathway{}
	ent.ID = id
	return deleteEnt(ctx, &ent)
}

func createUpdatePathway(ctx context.Context, input model.PathwaySetInput) (int, error) {
	return createUpdateEnt(
		ctx,
		input.ID,
		fvint(input.FeedVersion),
		&gtfs.Pathway{},
		func(ent *gtfs.Pathway) ([]string, error) {
			var cols []string
			var err error
			cols = scanReq(&ent.PathwayID, input.PathwayID, "pathway_id", isUpdate(input.ID), cols, &err)
			cols = scanReq(&ent.PathwayMode, input.PathwayMode, "pathway_mode", isUpdate(input.ID), cols, &err)
			cols = scanReq(&ent.IsBidirectional, input.IsBidirectional, "is_bidirectional", isUpdate(input.ID), cols, &err)
			cols = scanOpt(&ent.Length, input.Length, "length", cols, &err)
			cols = scanOpt(&ent.TraversalTime, input.TraversalTime, "traversal_time", cols, &err)
			cols = scanOpt(&ent.StairCount, input.StairCount, "stair_count", cols, &err)
			cols = scanOpt(&ent.MaxSlope, input.MaxSlope, "max_slope", cols, &err)
			cols = scanOpt(&ent.MinWidth, input.MinWidth, "min_width", cols, &err)
			cols = scanOpt(&ent.SignpostedAs, input.SignpostedAs, "signposted_as", cols, &err)
			cols = scanOpt(&ent.ReverseSignpostedAs, input.ReverseSignpostedAs, "reverse_signposted_as", cols, &err)
			if v := input.FromStop; v != nil {
				cols = scanRef(&ent.FromStopID, v.ID, "from_stop_id", cols, &err)
			}
			if v := input.ToStop; v != nil {
				cols = scanRef(&ent.ToStopID, v.ID, "to_stop_id", cols, &err)
			}
			return cols, err
		})
}

///////////

func (f *Finder) LevelCreate(ctx context.Context, input model.LevelSetInput) (int, error) {
	if input.FeedVersion == nil || input.FeedVersion.ID == nil {
		return 0, errors.New("feed_version_id required")
	}
	return createUpdateLevel(ctx, input)
}

func (f *Finder) LevelUpdate(ctx context.Context, input model.LevelSetInput) (int, error) {
	if input.ID == nil {
		return 0, errors.New("id required")
	}
	return createUpdateLevel(ctx, input)
}

func (f *Finder) LevelDelete(ctx context.Context, id int) error {
	ent := gtfs.Level{}
	ent.ID = id
	return deleteEnt(ctx, &ent)
}

// levelRow is a gtfs_levels row with the Transitland geometry and parent
// station columns. model.Level adds feed fields that the table lacks.
type levelRow struct {
	Geometry      tt.MultiPolygon
	ParentStation tt.Key
	gtfs.Level
}

func createUpdateLevel(ctx context.Context, input model.LevelSetInput) (int, error) {
	return createUpdateEnt(
		ctx,
		input.ID,
		fvint(input.FeedVersion),
		&levelRow{},
		func(ent *levelRow) ([]string, error) {
			var cols []string
			var err error
			cols = scanReq(&ent.LevelID, input.LevelID, "level_id", isUpdate(input.ID), cols, &err)
			cols = scanOpt(&ent.LevelName, input.LevelName, "level_name", cols, &err)
			cols = scanReq(&ent.LevelIndex, input.LevelIndex, "level_index", isUpdate(input.ID), cols, &err)
			cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
			if v := input.Parent; v != nil {
				cols = scanRef(&ent.ParentStation, v.ID, "parent_station", cols, &err)
			}
			return cols, err
		})
}

///////////

func checkCol[T any, P *T](val P, inval P, colname string, cols []string) []string {
	if inval != nil {
		*val = *inval
		cols = append(cols, colname)
	}
	return cols
}

type canScan interface {
	Scan(any) error
}

// scanCol sets val from inval and appends colname to cols when inval is
// non-nil, and leaves both alone when it's nil. Any Scan error is reported
// through errp, which must be non-nil; once *errp is set, later calls
// short-circuit, so callers can chain several and check one error at the end.
func scanCol[T any](val canScan, inval *T, colname string, cols []string, errp *error) []string {
	if inval == nil {
		return cols
	}
	return scanOpt(val, graphql.OmittableOf(inval), colname, cols, errp)
}

// scanOpt is scanCol for an input field that tells null from left out: a
// field left out keeps the column as it was, a null clears it, and a value
// sets it.
func scanOpt[T any](val canScan, inval graphql.Omittable[*T], colname string, cols []string, errp *error) []string {
	if *errp != nil || !inval.IsSet() {
		return cols
	}
	var v any
	if p := inval.Value(); p != nil {
		v = *p
	}
	if err := val.Scan(v); err != nil {
		*errp = fmt.Errorf("scan %s: %w", colname, err)
		return cols
	}
	return append(cols, colname)
}

// scanReq is scanOpt for a field that must keep a value. In an update a null
// is an error, rather than a clear that would leave a not-null column its
// default; in a create, where there's nothing to clear, it's left out.
func scanReq[T any](val canScan, inval graphql.Omittable[*T], colname string, update bool, cols []string, errp *error) []string {
	if inval.IsSet() && inval.Value() == nil {
		if update && *errp == nil {
			*errp = fmt.Errorf("%s can't be cleared", colname)
		}
		return cols
	}
	return scanOpt(val, inval, colname, cols, errp)
}

// scanRef sets a column to id, or clears it when id is nil, and appends
// colname to cols.
func scanRef[T any](val canScan, id *T, colname string, cols []string, errp *error) []string {
	return scanOpt(val, graphql.OmittableOf(id), colname, cols, errp)
}

// isUpdate says whether an input's id names a record to update
func isUpdate(id *int) bool {
	return id != nil && *id > 0
}

type hasTableName interface {
	TableName() string
	GetFeedVersionID() int
	SetFeedVersionID(int)
	GetID() int
	SetID(int)
}

func fvint(fvi *model.FeedVersionInput) *int {
	if fvi == nil {
		return nil
	}
	return fvi.ID
}

func toAtx(ctx context.Context) tldb.Adapter {
	return model.ForContext(ctx).Adapter
}

var notNullFilter = &tldb.NotNullFilter{}
var emptyTextFilter = &tldb.EmptyTextFilter{}

// ensure we have edit rights to fvid
func createUpdateEnt[T hasTableName](
	ctx context.Context,
	entId *int,
	fvid *int,
	baseEnt T,
	updateFunc func(baseEnt T) ([]string, error),
) (int, error) {
	update := (entId != nil && *entId > 0)
	atx := toAtx(ctx)
	retId := 0

	// Update or create?
	if update {
		// Load the ent and get the feed version ID
		// Do not use the provided fvid value
		baseEnt.SetID(*entId)
		if err := atx.Find(ctx, baseEnt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("record not found (id=%d): %w", *entId, err)
			}
			return 0, err
		}
	} else if fvid != nil {
		baseEnt.SetFeedVersionID(*fvid)
	} else {
		return 0, errors.New("id or feed_version_id required")
	}

	// Check we can edit this feed version
	if err := checkFeedEdit(ctx, baseEnt.GetFeedVersionID()); err != nil {
		return 0, err
	}

	// Update columns
	cols, err := updateFunc(baseEnt)
	if err != nil {
		return 0, err
	}
	// Fill the not-null defaults the importer writes, validate, and then, as
	// the importer does after validation, write cleared text as an empty
	// string: a null stop name or description would void the stop's text
	// search vector.
	ent, isEntity := any(baseEnt).(tt.Entity)
	if isEntity {
		if err := notNullFilter.Filter(ent, nil); err != nil {
			return 0, err
		}
	}
	if errs := tt.CheckErrors(baseEnt); len(errs) > 0 {
		return 0, errs[0]
	}
	if isEntity {
		if err := emptyTextFilter.Filter(ent, nil); err != nil {
			return 0, err
		}
	}
	// Save
	err = nil
	if update {
		retId = baseEnt.GetID()
		err = atx.Update(ctx, baseEnt, cols...)
	} else {
		retId, err = atx.Insert(ctx, baseEnt)
	}
	if err != nil {
		return 0, err
	}
	return retId, nil
}

type deleteRef struct {
	ID         int
	TableName  string
	ColumnName string
}

// ensure we have edit rights to fvid
func deleteEnt(ctx context.Context, ent hasTableName, deleteRefs ...deleteRef) error {
	// Get fvid
	entId := ent.GetID()
	fvid := ent.GetFeedVersionID()
	db := toAtx(ctx).DBX()
	if err := dbutil.Get(
		ctx,
		db,
		sq.StatementBuilder.
			Select("feed_version_id").
			From(ent.TableName()).
			Where(sq.Eq{"id": entId}),
		&fvid,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("record not found (id=%d): %w", entId, err)
		}
		return err
	}

	// // Check if we can edit fv
	if err := checkFeedEdit(ctx, fvid); err != nil {
		return err
	}

	// Delete references
	for _, ref := range deleteRefs {
		if _, err := toAtx(ctx).Sqrl().Delete(ref.TableName).Where(sq.Eq{ref.ColumnName: entId}).Exec(); err != nil {
			return fmt.Errorf("failed to delete %s %d from %s: %w", ref.ColumnName, entId, ref.TableName, err)
		}
	}

	// Delete entity
	_, err := toAtx(ctx).Sqrl().Delete(ent.TableName()).Where(sq.Eq{"id": entId}).Exec()
	return err
}

func checkFeedEdit(ctx context.Context, fvid int) error {
	if fvid <= 0 {
		return errors.New("invalid feed version id")
	}
	cfg := model.ForContext(ctx)
	if cfg.Checker == nil {
		return authz.ErrUnauthorized
	}
	ok, err := cfg.Checker.Check(ctx, authz.ObjectRef{Type: authz.FeedVersionType, ID: int64(fvid)}, authz.CanEdit)
	if err != nil {
		return err
	}
	if !ok {
		return authz.ErrUnauthorized
	}
	return nil
}
