package dbfinder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/jmoiron/sqlx"
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
	stopId, err := createUpdateEnt(
		ctx,
		input.ID,
		fvint(input.FeedVersion),
		&gtfs.Stop{},
		func(ent *gtfs.Stop) ([]string, error) {
			var cols []string
			var err error
			cols = scanCol(&ent.StopID, input.StopID, "stop_id", cols, &err)
			cols = scanCol(&ent.StopCode, input.StopCode, "stop_code", cols, &err)
			cols = scanCol(&ent.StopDesc, input.StopDesc, "stop_desc", cols, &err)
			cols = scanCol(&ent.StopTimezone, input.StopTimezone, "stop_timezone", cols, &err)
			cols = scanCol(&ent.StopName, input.StopName, "stop_name", cols, &err)
			cols = scanCol(&ent.StopURL, input.StopURL, "stop_url", cols, &err)
			cols = scanCol(&ent.LocationType, input.LocationType, "location_type", cols, &err)
			cols = scanCol(&ent.WheelchairBoarding, input.WheelchairBoarding, "wheelchair_boarding", cols, &err)
			cols = scanCol(&ent.ZoneID, input.ZoneID, "zone_id", cols, &err)
			cols = scanCol(&ent.TtsStopName, input.TtsStopName, "tts_stop_name", cols, &err)
			cols = scanCol(&ent.PlatformCode, input.PlatformCode, "platform_code", cols, &err)
			if input.Geometry != nil && input.Geometry.Valid {
				cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
			}
			if v := input.Parent; v != nil {
				ent.ParentStation.Scan(nil)
				cols = append(cols, "parent_station")
				cols = scanCol(&ent.ParentStation, v.ID, "", cols, &err)
			}
			if v := input.Level; v != nil {
				ent.LevelID.Scan(nil)
				cols = append(cols, "level_id")
				cols = scanCol(&ent.LevelID, v.ID, "", cols, &err)
			}
			// Set some defaults
			ent.LocationType.OrSet(0)
			return cols, err
		})
	if err != nil {
		return 0, err
	}
	if refInput := input.ExternalReference; refInput != nil {
		// Check if we have an existing stop external reference for this stop
		var stopRefCheck model.StopExternalReference
		var fvidCheck *int
		sqlx.GetContext(ctx, toAtx(ctx).DBX(), &stopRefCheck, `select id from tl_stop_external_references where stop_id = $1`, stopId)
		sqlx.GetContext(ctx, toAtx(ctx).DBX(), &fvidCheck, `select feed_version_id from gtfs_stops where id = $1`, stopId)
		if refInput.TargetFeedOnestopID != nil {
			// Update using normal method
			if _, err := createUpdateEnt(
				ctx,
				&stopRefCheck.ID,
				fvidCheck,
				&model.StopExternalReference{},
				func(ent *model.StopExternalReference) ([]string, error) {
					var cols []string
					var err error
					ent.StopID.SetInt(stopId)
					cols = scanCol(&ent.StopID, &stopId, "stop_id", cols, &err)
					cols = scanCol(&ent.TargetFeedOnestopID, refInput.TargetFeedOnestopID, "target_feed_onestop_id", cols, &err)
					cols = scanCol(&ent.TargetStopID, refInput.TargetStopID, "target_stop_id", cols, &err)
					// cols = scanCol(&ent.Inactive, refInput.Inactive, "inactive", cols, &err)
					return cols, err
				},
			); err != nil {
				return 0, fmt.Errorf("failed to create or update stop external reference for stop %d: %w", stopId, err)
			}
		} else if stopRefCheck.ID > 0 {
			// Delete
			if err := deleteEnt(ctx, &stopRefCheck); err != nil {
				return 0, fmt.Errorf("failed to delete stop external reference for stop %d: %w", stopId, err)
			}
		}
	}
	return stopId, err
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
			cols = scanCol(&ent.PathwayID, input.PathwayID, "pathway_id", cols, &err)
			cols = scanCol(&ent.PathwayMode, input.PathwayMode, "pathway_mode", cols, &err)
			cols = scanCol(&ent.IsBidirectional, input.IsBidirectional, "is_bidirectional", cols, &err)
			cols = scanCol(&ent.Length, input.Length, "length", cols, &err)
			cols = scanCol(&ent.TraversalTime, input.TraversalTime, "traversal_time", cols, &err)
			cols = scanCol(&ent.StairCount, input.StairCount, "stair_count", cols, &err)
			cols = scanCol(&ent.MaxSlope, input.MaxSlope, "max_slope", cols, &err)
			cols = scanCol(&ent.MinWidth, input.MinWidth, "min_width", cols, &err)
			cols = scanCol(&ent.SignpostedAs, input.SignpostedAs, "signposted_as", cols, &err)
			cols = scanCol(&ent.ReverseSignpostedAs, input.ReverseSignpostedAs, "reverse_signposted_as", cols, &err)
			if v := input.FromStop; v != nil {
				cols = append(cols, "from_stop_id")
				cols = scanCol(&ent.FromStopID, v.ID, "", cols, &err)
			}
			if v := input.ToStop; v != nil {
				cols = append(cols, "from_stop_id")
				cols = scanCol(&ent.ToStopID, v.ID, "", cols, &err)
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

func createUpdateLevel(ctx context.Context, input model.LevelSetInput) (int, error) {
	return createUpdateEnt(
		ctx,
		input.ID,
		fvint(input.FeedVersion),
		&model.Level{},
		func(ent *model.Level) ([]string, error) {
			var cols []string
			var err error
			cols = scanCol(&ent.LevelID, input.LevelID, "level_id", cols, &err)
			cols = scanCol(&ent.LevelName, input.LevelName, "level_name", cols, &err)
			cols = scanCol(&ent.LevelIndex, input.LevelIndex, "level_index", cols, &err)
			cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
			if v := input.Parent; v != nil {
				cols = append(cols, "parent_station")
				cols = scanCol(&ent.ParentStation, v.ID, "", cols, &err)
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

// scanCol assigns inval into val (via val.Scan) and appends colname to cols
// when inval is non-nil. Any Scan error is reported through errp, which must
// be non-nil; once *errp is set, subsequent scanCol calls short-circuit so
// callers can chain several and inspect a single error at the end.
func scanCol[T any, PT *T](val canScan, inval PT, colname string, cols []string, errp *error) []string {
	if *errp != nil {
		return cols
	}
	if inval != nil {
		if err := val.Scan(*inval); err != nil {
			*errp = fmt.Errorf("scan %s: %w", colname, err)
			return cols
		}
		cols = append(cols, colname)
	}
	return cols
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
	return postgres.NewPostgresAdapterFromDBX(model.ForContext(ctx).Finder.DBX())
}

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

	// Validate
	if errs := tt.CheckErrors(baseEnt); len(errs) > 0 {
		return 0, errs[0]
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
	db := model.ForContext(ctx).Finder.DBX()
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
