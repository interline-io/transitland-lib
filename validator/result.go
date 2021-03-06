package validator

import (
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
)

// Result contains a validation report result,
type Result struct {
	copier.Result                                       // add to copier result:
	Success              bool                           `json:"success"`
	FailureReason        string                         `json:"failure_reason"`
	SHA1                 string                         `json:"sha1"`
	EarliestCalendarDate time.Time                      `json:"earliest_calendar_date"`
	LatestCalendarDate   time.Time                      `json:"latest_calendar_date"`
	Agencies             []tl.Agency                    `json:"agencies"`
	Routes               []tl.Route                     `json:"routes"`
	Stops                []tl.Stop                      `json:"stops"`
	FeedInfos            []tl.FeedInfo                  `json:"feed_infos"`
	Files                []dmfr.FeedVersionFileInfo     `json:"files"`
	ServiceLevels        []dmfr.FeedVersionServiceLevel `json:"service_levels"`
}
