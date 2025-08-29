package dbfinder

import (
	"context"
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
			cols = scanCol(&ent.StopID, input.StopID, "stop_id", cols)
			cols = scanCol(&ent.StopCode, input.StopCode, "stop_code", cols)
			cols = scanCol(&ent.StopDesc, input.StopDesc, "stop_desc", cols)
			cols = scanCol(&ent.StopTimezone, input.StopTimezone, "stop_timezone", cols)
			cols = scanCol(&ent.StopName, input.StopName, "stop_name", cols)
			cols = scanCol(&ent.StopURL, input.StopURL, "stop_url", cols)
			cols = scanCol(&ent.LocationType, input.LocationType, "location_type", cols)
			cols = scanCol(&ent.WheelchairBoarding, input.WheelchairBoarding, "wheelchair_boarding", cols)
			cols = scanCol(&ent.ZoneID, input.ZoneID, "zone_id", cols)
			cols = scanCol(&ent.TtsStopName, input.TtsStopName, "tts_stop_name", cols)
			cols = scanCol(&ent.PlatformCode, input.PlatformCode, "platform_code", cols)
			if input.Geometry != nil && input.Geometry.Valid {
				cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
			}
			if v := input.Parent; v != nil {
				ent.ParentStation.Scan(nil)
				cols = append(cols, "parent_station")
				scanCol(&ent.ParentStation, v.ID, "parent_station", cols)
			}
			if v := input.Level; v != nil {
				ent.LevelID.Scan(nil)
				cols = append(cols, "level_id")
				scanCol(&ent.LevelID, v.ID, "level_id", cols)
			}
			// Set some defaults
			ent.LocationType.OrSet(0)
			return cols, nil
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
					ent.StopID.SetInt(stopId)
					cols = scanCol(&ent.StopID, &stopId, "stop_id", cols)
					cols = scanCol(&ent.TargetFeedOnestopID, refInput.TargetFeedOnestopID, "target_feed_onestop_id", cols)
					cols = scanCol(&ent.TargetStopID, refInput.TargetStopID, "target_stop_id", cols)
					// cols = scanCol(&ent.Inactive, refInput.Inactive, "inactive", cols)
					return cols, nil
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
			cols = scanCol(&ent.PathwayID, input.PathwayID, "pathway_id", cols)
			cols = scanCol(&ent.PathwayMode, input.PathwayMode, "pathway_mode", cols)
			cols = scanCol(&ent.IsBidirectional, input.IsBidirectional, "is_bidirectional", cols)
			cols = scanCol(&ent.Length, input.Length, "length", cols)
			cols = scanCol(&ent.TraversalTime, input.TraversalTime, "traversal_time", cols)
			cols = scanCol(&ent.StairCount, input.StairCount, "stair_count", cols)
			cols = scanCol(&ent.MaxSlope, input.MaxSlope, "max_slope", cols)
			cols = scanCol(&ent.MinWidth, input.MinWidth, "min_width", cols)
			cols = scanCol(&ent.SignpostedAs, input.SignpostedAs, "signposted_as", cols)
			cols = scanCol(&ent.ReverseSignpostedAs, input.ReverseSignpostedAs, "reverse_signposted_as", cols)
			if v := input.FromStop; v != nil {
				cols = append(cols, "from_stop_id")
				scanCol(&ent.FromStopID, v.ID, "", nil)
			}
			if v := input.ToStop; v != nil {
				cols = append(cols, "from_stop_id")
				scanCol(&ent.ToStopID, v.ID, "", nil)
			}
			return cols, nil
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
			cols = scanCol(&ent.LevelID, input.LevelID, "level_id", cols)
			cols = scanCol(&ent.LevelName, input.LevelName, "level_name", cols)
			cols = scanCol(&ent.LevelIndex, input.LevelIndex, "level_index", cols)
			cols = checkCol(&ent.Geometry, input.Geometry, "geometry", cols)
			if v := input.Parent; v != nil {
				cols = append(cols, "parent_station")
				scanCol(&ent.ParentStation, v.ID, "", nil)
			}
			return cols, nil
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

func scanCol[T any, PT *T](val canScan, inval PT, colname string, cols []string) []string {
	if inval != nil {
		if err := val.Scan(*inval); err != nil {
			panic(err)
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
	if checker := cfg.Checker; checker == nil {
		return nil
	} else if check, err := checker.FeedVersionPermissions(ctx, &authz.FeedVersionRequest{Id: int64(fvid)}); err != nil {
		return err
	} else if !check.Actions.CanEdit {
		return authz.ErrUnauthorized
	}
	return nil
}
