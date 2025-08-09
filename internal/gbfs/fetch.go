package gbfs

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/tldb"
)

type Options struct {
	Language string
	fetch.Options
}

type Result struct {
	fetch.Result
}

func Fetch(ctx context.Context, atx tldb.Adapter, opts Options) ([]GbfsFeed, Result, error) {
	result := Result{}
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

	// Fetch system file
	systemFile := SystemFile{}
	fr, err := fetchUnmarshal(opts.FeedURL, &systemFile, reqOpts...)
	result.ResponseCode = fr.ResponseCode
	result.ResponseSHA1 = fr.ResponseSHA1
	result.ResponseSize = fr.ResponseSize
	if err != nil {
		return nil, result, err
	}

	// Fetch additional data
	var feeds []GbfsFeed
	for _, sflang := range systemFile.Data {
		if sflang == nil {
			continue
		}
		if feed, err := fetchAll(ctx, *sflang); err == nil {
			feeds = append(feeds, feed)
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

	return feeds, result, nil
}

func fetchAll(ctx context.Context, sf SystemFeeds, reqOpts ...request.RequestOption) (GbfsFeed, error) {
	ret := GbfsFeed{}
	var err error
	for _, v := range sf.Feeds {
		switch v.Name.Val {
		case "system_information":
			e := SystemInformationFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.SystemInformation = e.Data
		case "station_information":
			e := StationInformationFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.StationInformation = e.Data.Stations
		case "station_status":
			e := StationStatusFile{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			ret.StationStatus = e.Data.Stations
		case "free_bike_status":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Bikes = e.Data.Bikes
			}
		case "system_hours":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.RentalHours = e.Data.RentalHours
			}
		case "system_calendar":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Calendars = e.Data.Calendars
			}
		case "system_regions":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Regions = e.Data.Regions
			}
		case "system_alerts":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Alerts = e.Data.Alerts
			}
		case "vehicle_types":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.VehicleTypes = e.Data.VehicleTypes
			}
		case "system_pricing_plans":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.Plans = e.Data.Plans
			}
		case "geofencing_zones":
			e := GbfsFeedData{}
			_, err = fetchUnmarshal(v.URL.Val, &e, reqOpts...)
			if e.Data != nil {
				ret.GeofencingZones = e.Data.GeofencingZones
			}
		case "gbfs_versions":
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
	return ret, err
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
