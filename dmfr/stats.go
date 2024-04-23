package dmfr

import (
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

type FeedVersionStats struct {
	ServiceWindow    FeedVersionServiceWindow
	ServiceLevels    []FeedVersionServiceLevel
	AgencyOnestopIDs []FeedVersionAgencyOnestopID
	RouteOnestopIDs  []FeedVersionRouteOnestopID
	StopOnestopIDs   []FeedVersionStopOnestopID
	FileInfos        []FeedVersionFileInfo
}

func NewFeedStatsFromReader(reader tl.Reader) (FeedVersionStats, error) {
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
		ret.AgencyOnestopIDs = append(ret.AgencyOnestopIDs, FeedVersionAgencyOnestopID{
			EntityID:  osid.AgencyID,
			OnestopID: osid.OnestopID,
		})
	}
	for _, osid := range osidBuilder.RouteOnestopIDs() {
		ret.RouteOnestopIDs = append(ret.RouteOnestopIDs, FeedVersionRouteOnestopID{
			EntityID:  osid.RouteID,
			OnestopID: osid.OnestopID,
		})
	}
	for _, osid := range osidBuilder.StopOnestopIDs() {
		ret.StopOnestopIDs = append(ret.StopOnestopIDs, FeedVersionStopOnestopID{
			EntityID:  osid.StopID,
			OnestopID: osid.OnestopID,
		})
	}
	return ret, nil
}
