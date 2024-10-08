package stats

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

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
			log.Error().Msgf("could not set feed version id for type %T", input[i])
		}
	}
	return input
}

func CreateFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	stats, err := NewFeedStatsFromReader(reader)
	if err != nil {
		return err
	}

	// Delete any existing records
	tables := []string{
		"feed_version_file_infos",
		"feed_version_service_levels",
		"feed_version_service_windows",
		"feed_version_agency_onestop_ids",
		"feed_version_route_onestop_ids",
		"feed_version_stop_onestop_ids",
	}
	for _, table := range tables {
		q, args, err := atx.Sqrl().Delete(table).Where(sq.Eq{"feed_version_id": fvid}).ToSql()
		if err != nil {
			return err
		}
		if _, err := atx.DBX().Exec(q, args...); err != nil {
			return err
		}
	}

	// Insert FVSW
	fvsw := stats.ServiceWindow
	fvsw.FeedVersionID = fvid
	if _, err := atx.Insert(&fvsw); err != nil {
		return err
	}

	// Batch insert OSIDs
	if _, err := atx.MultiInsert(setFvid(convertToAny(stats.AgencyOnestopIDs), fvid)); err != nil {
		return err
	}
	if _, err := atx.MultiInsert(setFvid(convertToAny(stats.RouteOnestopIDs), fvid)); err != nil {
		return err
	}
	if _, err := atx.MultiInsert(setFvid(convertToAny(stats.StopOnestopIDs), fvid)); err != nil {
		return err
	}

	// Insert FVFIs
	if _, err := atx.MultiInsert(setFvid(convertToAny(stats.FileInfos), fvid)); err != nil {
		return err
	}

	// Batch insert FVSLs
	if _, err := atx.MultiInsert(setFvid(convertToAny(stats.ServiceLevels), fvid)); err != nil {
		return err
	}
	return nil
}
