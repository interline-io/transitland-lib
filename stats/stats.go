package stats

import (
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/adapters/tlcsv"
	"github.com/interline-io/transitland-lib/adapters/tldb"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
)

type FeedVersionStats struct {
	ServiceWindow    dmfr.FeedVersionServiceWindow
	ServiceLevels    []dmfr.FeedVersionServiceLevel
	AgencyOnestopIDs []dmfr.FeedVersionAgencyOnestopID
	RouteOnestopIDs  []dmfr.FeedVersionRouteOnestopID
	StopOnestopIDs   []dmfr.FeedVersionStopOnestopID
	FileInfos        []dmfr.FeedVersionFileInfo
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
	if err := copier.QuietCopy(reader, &empty.Writer{}, func(o *copier.Options) {
		o.Quiet = false
		o.NoShapeCache = true
		o.NoValidators = true
		o.AddExtension(fvslBuilder)
		o.AddExtension(fvswBuilder)
		o.AddExtension(osidBuilder)
	}); err != nil {
		return ret, err
	}

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

func (ext *FeedVersionOnestopIDBuilder) Copy(*copier.Copier) error {
	return nil
}

func NewFeedVersionOnestopIDBuilder() *FeedVersionOnestopIDBuilder {
	return &FeedVersionOnestopIDBuilder{
		OnestopIDBuilder: builders.NewOnestopIDBuilder(),
	}
}

//////////

func CreateFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	stats, err := NewFeedStatsFromReader(reader)
	if err != nil {
		return err
	}

	fvt := dmfr.GetFeedVersionTables()

	// Delete any existing records
	tables := fvt.FetchStatDerivedTables
	for _, table := range tables {
		if err := tldb.FeedVersionTableDelete(atx, table, fvid, false); err != nil {
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
