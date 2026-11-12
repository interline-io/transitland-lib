package stats

import (
	"context"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// Stat names accepted by WriteOptions.Stats. Each maps to one or more
// FetchStatDerivedTables; only the selected stats are deleted and re-inserted.
const (
	StatFileInfos      = "file_infos"
	StatServiceLevels  = "service_levels"
	StatServiceWindows = "service_windows"
	StatOnestopIDs     = "onestop_ids"
	StatGeohash        = "geohash"
	StatGeometry       = "geometry"
)

// AllStats lists every supported stat name in the order they are written.
var AllStats = []string{
	StatFileInfos,
	StatServiceLevels,
	StatServiceWindows,
	StatOnestopIDs,
	StatGeohash,
	StatGeometry,
}

// statTables maps a stat name to the FetchStatDerivedTables it writes. The
// union across all entries must equal dmfr.GetFeedVersionTables().FetchStatDerivedTables;
// TestStatRegistrationConsistency enforces this.
var statTables = map[string][]string{
	StatFileInfos:      {"feed_version_file_infos"},
	StatServiceLevels:  {"feed_version_service_levels"},
	StatServiceWindows: {"feed_version_service_windows"},
	StatOnestopIDs: {
		"feed_version_agency_onestop_ids",
		"feed_version_route_onestop_ids",
		"feed_version_stop_onestop_ids",
	},
	StatGeohash:  {"tl_feed_version_geohashes"},
	StatGeometry: {"tl_feed_version_geometries"},
}

// WriteOptions configures which stats WriteFeedVersionStats persists. Builders
// always run regardless of selection; only the database delete/insert step is
// gated, so callers can target a single stat without churning unrelated records.
type WriteOptions struct {
	// Subset of stat names to write; empty means all.
	Stats []string
}

// ValidateStatNames returns an error if any name in names is not a recognized
// stat. Empty/nil is valid and means "all stats".
func ValidateStatNames(names []string) error {
	for _, s := range names {
		if _, ok := statTables[s]; !ok {
			return fmt.Errorf("unknown stat name %q (valid: %v)", s, AllStats)
		}
	}
	return nil
}

func (o WriteOptions) resolveStats() (map[string]bool, error) {
	if err := ValidateStatNames(o.Stats); err != nil {
		return nil, err
	}
	enabled := map[string]bool{}
	src := o.Stats
	if len(src) == 0 {
		src = AllStats
	}
	for _, s := range src {
		enabled[s] = true
	}
	return enabled, nil
}

type FeedVersionStats struct {
	ServiceWindow    dmfr.FeedVersionServiceWindow
	ServiceLevels    []dmfr.FeedVersionServiceLevel
	AgencyOnestopIDs []dmfr.FeedVersionAgencyOnestopID
	RouteOnestopIDs  []dmfr.FeedVersionRouteOnestopID
	StopOnestopIDs   []dmfr.FeedVersionStopOnestopID
	FileInfos        []dmfr.FeedVersionFileInfo
	GeohashCells     map[string]int
	Geometry         tt.Polygon
}

func NewFeedStatsFromReader(reader adapters.Reader) (FeedVersionStats, error) {
	ret := FeedVersionStats{}

	// File Infos - only for CSV readers
	var err error
	if v, ok := reader.(*tlcsv.Reader); ok {
		ret.FileInfos, err = NewFeedVersionFileInfosFromReader(v)
		if err != nil {
			return ret, err
		}
	}

	// Use builders to gather other statistics
	fvslBuilder := NewFeedVersionServiceLevelBuilder()
	fvswBuilder := NewFeedVersionServiceWindowBuilder()
	osidBuilder := NewFeedVersionOnestopIDBuilder()
	geohashBuilder := builders.NewFeedVersionGeohashBuilder()
	geometryBuilder := builders.NewFeedVersionGeometryBuilder()
	if _, err := copier.QuietCopy(
		context.TODO(),
		reader, &empty.Writer{},
		func(o *copier.Options) {
			o.NoShapeCache = true
			o.NoValidators = true
			o.AddExtension(fvslBuilder)
			o.AddExtension(fvswBuilder)
			o.AddExtension(osidBuilder)
			o.AddExtension(geohashBuilder)
			o.AddExtension(geometryBuilder)
		},
	); err != nil {
		return ret, err
	}
	ret.GeohashCells = geohashBuilder.Cells()
	ret.Geometry = geometryBuilder.Geometry()

	// Service levels
	ret.ServiceLevels, err = fvslBuilder.ServiceLevels()
	if err != nil {
		return ret, err
	}

	// Service window
	ret.ServiceWindow, err = fvswBuilder.ServiceWindow()
	if err != nil {
		return ret, err
	}

	// Service window: Default week
	ret.ServiceWindow.FallbackWeek, err = ServiceLevelDefaultWeek(ret.ServiceWindow.FeedStartDate, ret.ServiceWindow.FeedStartDate, ret.ServiceLevels)
	if err != nil {
		return ret, err
	}

	// OnestopIDs
	for _, osid := range osidBuilder.AgencyOnestopIDs() {
		ret.AgencyOnestopIDs = append(ret.AgencyOnestopIDs, dmfr.FeedVersionAgencyOnestopID{
			EntityID:  osid.AgencyID,
			OnestopID: osid.OnestopID,
		})
	}
	for _, osid := range osidBuilder.RouteOnestopIDs() {
		ret.RouteOnestopIDs = append(ret.RouteOnestopIDs, dmfr.FeedVersionRouteOnestopID{
			EntityID:  osid.RouteID,
			OnestopID: osid.OnestopID,
		})
	}
	for _, osid := range osidBuilder.StopOnestopIDs() {
		ret.StopOnestopIDs = append(ret.StopOnestopIDs, dmfr.FeedVersionStopOnestopID{
			EntityID:  osid.StopID,
			OnestopID: osid.OnestopID,
		})
	}
	return ret, nil
}

///////

type FeedVersionOnestopIDBuilder struct {
	*builders.OnestopIDBuilder
}

func (ext *FeedVersionOnestopIDBuilder) Copy(adapters.EntityCopier) error {
	return nil
}

func NewFeedVersionOnestopIDBuilder() *FeedVersionOnestopIDBuilder {
	return &FeedVersionOnestopIDBuilder{
		OnestopIDBuilder: builders.NewOnestopIDBuilder(),
	}
}

//////////

func CreateFeedStats(ctx context.Context, atx tldb.Adapter, reader *tlcsv.Reader, fvid int, opts WriteOptions) error {
	stats, err := NewFeedStatsFromReader(reader)
	if err != nil {
		return err
	}
	return WriteFeedVersionStats(ctx, atx, stats, fvid, opts)
}

func WriteFeedVersionStats(ctx context.Context, atx tldb.Adapter, stats FeedVersionStats, fvid int, opts WriteOptions) error {
	enabled, err := opts.resolveStats()
	if err != nil {
		return err
	}

	for _, stat := range AllStats {
		if !enabled[stat] {
			continue
		}
		for _, table := range statTables[stat] {
			if err := FeedVersionTableDelete(ctx, atx, table, fvid, false); err != nil {
				return err
			}
		}
	}

	if enabled[StatServiceWindows] {
		fvsw := stats.ServiceWindow
		fvsw.FeedVersionID = fvid
		if _, err := atx.Insert(ctx, &fvsw); err != nil {
			return err
		}
	}

	if enabled[StatOnestopIDs] {
		if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(stats.AgencyOnestopIDs), fvid)); err != nil {
			return err
		}
		if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(stats.RouteOnestopIDs), fvid)); err != nil {
			return err
		}
		if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(stats.StopOnestopIDs), fvid)); err != nil {
			return err
		}
	}

	if enabled[StatFileInfos] {
		if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(stats.FileInfos), fvid)); err != nil {
			return err
		}
	}

	if enabled[StatServiceLevels] {
		if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(stats.ServiceLevels), fvid)); err != nil {
			return err
		}
	}

	if enabled[StatGeohash] {
		if err := writeFeedVersionGeohashInserts(ctx, atx, fvid, stats.GeohashCells); err != nil {
			return err
		}
	}

	if enabled[StatGeometry] && stats.Geometry.Valid {
		ent := builders.FeedVersionGeometry{Geometry: stats.Geometry}
		if _, err := atx.MultiInsert(ctx, setFvid([]any{&ent}, fvid)); err != nil {
			return err
		}
	}
	return nil
}

// ReadFeedVersionStats returns the stats WriteFeedVersionStats wrote for a feed
// version.
func ReadFeedVersionStats(ctx context.Context, atx tldb.Adapter, fvid int) (FeedVersionStats, error) {
	ret := FeedVersionStats{GeohashCells: map[string]int{}}
	where := sq.Eq{"feed_version_id": fvid}
	windows := []dmfr.FeedVersionServiceWindow{}
	if err := selectStats(ctx, atx, &windows, atx.Sqrl().Select("*").From("feed_version_service_windows").Where(where)); err != nil {
		return ret, err
	}
	if len(windows) > 0 {
		ret.ServiceWindow = windows[0]
	}
	// FeedVersionServiceLevel has no fields for the Postgres table's route columns.
	levels := atx.Sqrl().
		Select("id", "feed_version_id", "start_date", "end_date", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday").
		From("feed_version_service_levels").
		Where(where).
		OrderBy("id")
	if err := selectStats(ctx, atx, &ret.ServiceLevels, levels); err != nil {
		return ret, err
	}
	if err := selectStats(ctx, atx, &ret.AgencyOnestopIDs, atx.Sqrl().Select("*").From("feed_version_agency_onestop_ids").Where(where).OrderBy("entity_id")); err != nil {
		return ret, err
	}
	if err := selectStats(ctx, atx, &ret.RouteOnestopIDs, atx.Sqrl().Select("*").From("feed_version_route_onestop_ids").Where(where).OrderBy("entity_id")); err != nil {
		return ret, err
	}
	if err := selectStats(ctx, atx, &ret.StopOnestopIDs, atx.Sqrl().Select("*").From("feed_version_stop_onestop_ids").Where(where).OrderBy("entity_id")); err != nil {
		return ret, err
	}
	if err := selectStats(ctx, atx, &ret.FileInfos, atx.Sqrl().Select("*").From("feed_version_file_infos").Where(where).OrderBy("id")); err != nil {
		return ret, err
	}
	cells := []builders.FeedVersionGeohash{}
	if err := selectStats(ctx, atx, &cells, atx.Sqrl().Select("*").From("tl_feed_version_geohashes").Where(where)); err != nil {
		return ret, err
	}
	for _, cell := range cells {
		ret.GeohashCells[cell.Geohash.Val] = cell.StopCount
	}
	// FeedVersionGeometry has no field for the table's centroid column.
	geoms := []builders.FeedVersionGeometry{}
	if err := selectStats(ctx, atx, &geoms, atx.Sqrl().Select("feed_version_id", "geometry").From("tl_feed_version_geometries").Where(where)); err != nil {
		return ret, err
	}
	if len(geoms) > 0 {
		ret.Geometry = geoms[0].Geometry
	}
	return ret, nil
}

// selectStats runs q and scans its rows into dest.
func selectStats(ctx context.Context, atx tldb.Adapter, dest any, q sq.SelectBuilder) error {
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return err
	}
	return atx.Select(ctx, dest, qstr, qargs...)
}

func writeFeedVersionGeohashInserts(ctx context.Context, atx tldb.Adapter, fvid int, cells map[string]int) error {
	if len(cells) == 0 {
		return nil
	}
	var ents []builders.FeedVersionGeohash
	for cell, count := range cells {
		ents = append(ents, builders.FeedVersionGeohash{
			Geohash:   tt.NewString(cell),
			StopCount: count,
		})
	}
	if _, err := atx.MultiInsert(ctx, setFvid(convertToAny(ents), fvid)); err != nil {
		return err
	}
	return nil
}

func convertToAny[T any](input []T) []any {
	var ret []any
	for i := 0; i < len(input); i++ {
		ret = append(ret, &input[i])
	}
	return ret
}

type canSetFeedVersion interface {
	SetFeedVersionID(int)
}

func setFvid(input []any, fvid int) []any {
	for i := 0; i < len(input); i++ {
		if v, ok := input[i].(canSetFeedVersion); ok {
			v.SetFeedVersionID(fvid)
		} else {
			log.For(context.TODO()).Error().Msgf("could not set feed version id for type %T", input[i])
		}
	}
	return input
}
