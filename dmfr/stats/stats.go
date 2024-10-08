package stats

import (
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

type FeedVersionStats struct {
	ServiceWindow    dmfr.FeedVersionServiceWindow
	ServiceLevels    []dmfr.FeedVersionServiceLevel
	AgencyOnestopIDs []dmfr.FeedVersionAgencyOnestopID
	RouteOnestopIDs  []dmfr.FeedVersionRouteOnestopID
	StopOnestopIDs   []dmfr.FeedVersionStopOnestopID
	FileInfos        []dmfr.FeedVersionFileInfo
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

///////

//////////
