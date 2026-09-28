package gbfs

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// GBFS file names, as a discovery file lists them.
const (
	fileSystemInformation  = "system_information"
	fileStationInformation = "station_information"
	fileStationStatus      = "station_status"
	fileFreeBikeStatus     = "free_bike_status"
	fileVehicleStatus      = "vehicle_status"
	fileSystemHours        = "system_hours"
	fileSystemCalendar     = "system_calendar"
	fileSystemRegions      = "system_regions"
	fileSystemAlerts       = "system_alerts"
	fileVehicleTypes       = "vehicle_types"
	fileSystemPricingPlans = "system_pricing_plans"
	fileGeofencingZones    = "geofencing_zones"
	fileGbfsVersions       = "gbfs_versions"
)

type Options struct {
	Language string
	fetch.Options
}

type Result struct {
	fetch.Result
}

// Fetch fetches one system from a GBFS discovery file, upconverted to the 3.x
// shape. It returns nil if no system_information fetched.
//
// A 1.x/2.x discovery file lists the system's files once per language. The
// first language in sorted order supplies the system, and each later one only
// its translations of the text.
func Fetch(ctx context.Context, atx tldb.Adapter, opts Options) (*GbfsFeed, Result, error) {
	result := Result{}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	var reqOpts []request.RequestOption
	if opts.AllowFTPFetch {
		reqOpts = append(reqOpts, request.WithAllowFTP)
	}
	if opts.AllowLocalFetch {
		reqOpts = append(reqOpts, request.WithAllowLocal)
	}
	if opts.AllowS3Fetch {
		reqOpts = append(reqOpts, request.WithAllowS3)
	}
	if opts.AllowHTTPFetchUnfiltered {
		reqOpts = append(reqOpts, request.WithAllowHTTPUnfiltered)
	}

	// Fetch system file
	systemFile := SystemFile{}
	fr, err := fetchUnmarshal(opts.FeedURL, &systemFile, reqOpts...)
	result.ResponseCode = fr.ResponseCode
	result.ResponseSHA1 = fr.ResponseSHA1
	result.ResponseSize = fr.ResponseSize
	if err != nil {
		return nil, result, err
	}

	// Fetch additional data. A 3.x system is listed under the one key "".
	var feed *GbfsFeed
	var languages []string
	for _, lang := range slices.Sorted(maps.Keys(systemFile.Data)) {
		sf := systemFile.Data[lang]
		if sf == nil {
			continue
		}
		if feed != nil {
			sf = textFiles(sf)
		}
		f := fetchAll(ctx, *sf, reqOpts...)
		if f.SystemInformation == nil {
			continue
		}
		if lang != "" {
			// 1.x/2.x text is untagged, and a system_information may name
			// another language than the one it is listed under.
			setLanguage(&f, lang)
			languages = append(languages, lang)
		}
		if feed == nil {
			feed = &f
		} else {
			addTranslations(feed, &f)
		}
	}
	if feed != nil {
		si := feed.SystemInformation
		if len(languages) > 0 {
			si.Languages = tt.NewStrings(languages)
		}
		if len(si.Languages.Val) > 0 {
			si.Language = tt.NewString(si.Languages.Val[0])
		}
	}

	if atx != nil {
		// Prepare and save feed fetch record
		tlfetch := dmfr.FeedFetch{}
		tlfetch.FeedID = opts.FeedID
		tlfetch.URLType = opts.URLType
		tlfetch.FetchedAt.Set(opts.FetchedAt)
		if !opts.HideURL {
			tlfetch.URL = opts.FeedURL
		}
		if result.ResponseCode > 0 {
			tlfetch.ResponseCode.SetInt(result.ResponseCode)
			tlfetch.ResponseSize.SetInt(result.ResponseSize)
			tlfetch.ResponseSHA1.Set(result.ResponseSHA1)
		}
		if result.FetchError == nil {
			tlfetch.Success = true
		} else {
			tlfetch.Success = false
			tlfetch.FetchError.Set(result.FetchError.Error())
		}
		if _, err := atx.Insert(context.TODO(), &tlfetch); err != nil {
			return nil, result, err
		}
	}

	return feed, result, nil
}

// textFileNames are the files that hold translatable text, the only ones
// fetched for a system's later languages.
var textFileNames = map[string]bool{
	fileSystemInformation:  true,
	fileStationInformation: true,
	fileVehicleTypes:       true,
	fileSystemRegions:      true,
	fileSystemPricingPlans: true,
	fileSystemAlerts:       true,
}

// textFiles returns the files in sf that hold translatable text.
func textFiles(sf *SystemFeeds) *SystemFeeds {
	ret := SystemFeeds{}
	for _, v := range sf.Feeds {
		if textFileNames[v.Name.Val] {
			ret.Feeds = append(ret.Feeds, v)
		}
	}
	return &ret
}

// fetchAll fetches and decodes the files it recognizes in sf. A file that
// fails is logged and left out.
func fetchAll(ctx context.Context, sf SystemFeeds, reqOpts ...request.RequestOption) GbfsFeed {
	ret := GbfsFeed{}
	for _, v := range sf.Feeds {
		var err error
		switch v.Name.Val {
		case fileSystemInformation:
			e := SystemInformationFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.SystemInformation = e.Data
		case fileStationInformation:
			e := StationInformationFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.StationInformation = e.Data.Stations
		case fileStationStatus:
			e := StationStatusFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.StationStatus = e.Data.Stations
		case fileFreeBikeStatus:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Bikes = e.Data.Bikes
			}
		case fileVehicleStatus:
			e := VehicleStatusFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.Bikes = e.Data.Vehicles
		case fileSystemHours:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.RentalHours = e.Data.RentalHours
			}
		case fileSystemCalendar:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Calendars = e.Data.Calendars
			}
		case fileSystemRegions:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Regions = e.Data.Regions
			}
		case fileSystemAlerts:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Alerts = e.Data.Alerts
			}
		case fileVehicleTypes:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.VehicleTypes = e.Data.VehicleTypes
			}
		case fileSystemPricingPlans:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Plans = e.Data.Plans
			}
		case fileGeofencingZones:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.GeofencingZones = e.Data.GeofencingZones
			}
		case fileGbfsVersions:
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Versions = e.Data.Versions
			}
		}
		if err != nil {
			log.For(ctx).Info().Err(err).Str("url", v.URL.Val).Msgf("failed to parse %s", v.Name.Val)
		}
	}
	return ret
}

func fetchUnmarshal(url string, ent any, reqOpts ...request.RequestOption) (request.FetchResponse, error) {
	ctx := context.TODO()
	var out bytes.Buffer
	fr, err := request.AuthenticatedRequest(ctx, &out, url, reqOpts...)
	if err != nil {
		return fr, err
	}
	if err := json.Unmarshal(out.Bytes(), ent); err != nil {
		return fr, err
	}
	return fr, nil
}
