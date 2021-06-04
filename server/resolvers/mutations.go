package resolvers

import (
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/server/auth"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/jmoiron/sqlx"
)

// Fetch adds a feed version to the database.
func Fetch(cfg config.Config, src io.Reader, feedURL *string, feed string, user *auth.User) (*model.FeedVersionFetchResult, error) {
	if user == nil {
		return nil, errors.New("no user")
	}
	opts := fetch.Options{
		FeedID:    feed,
		Directory: cfg.GtfsDir,
		S3:        cfg.GtfsS3Bucket,
		CreatedBy: tl.NewOString(user.Name),
	}
	if src != nil {
		// Prepare reader
		tmpfile, err := ioutil.TempFile("", "validator-upload")
		if err != nil {
			// This should result in a failed request
			return nil, err
		}
		io.Copy(tmpfile, src)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		opts.FeedURL = tmpfile.Name()
	} else if feedURL != nil {
		opts.FeedURL = *feedURL
	}
	// Run fetch command in txn
	adapter := tldb.NewPostgresAdapterFromDBX(model.DB)
	var fr fetch.Result
	err := adapter.Tx(func(atx tldb.Adapter) error {
		var fe error
		fr, fe = fetch.DatabaseFetch(atx, opts)
		return fe
	})
	if err != nil {
		return nil, err
	}
	mr := model.FeedVersionFetchResult{
		FoundSHA1:    fr.FoundSHA1,
		FoundDirSHA1: fr.FoundDirSHA1,
	}
	if fr.FetchError == nil {
		mr.FeedVersion = &model.FeedVersion{FeedVersion: fr.FeedVersion}
		mr.FetchError = nil
	} else {
		return nil, fr.FetchError
	}
	return &mr, nil
}

// Import loads a feed version into the database.
func Import(cfg config.Config, feedVersionSHA1 string, user *auth.User) (*model.FeedVersionImportResult, error) {
	adapter := tldb.NewPostgresAdapterFromDBX(model.DB)
	fvid := 0
	if err := adapter.Get(&fvid, "SELECT id FROM feed_versions WHERE sha1 = ?", feedVersionSHA1); err != nil {
		return nil, err
	}
	if err := checkEditableFv(model.DB, fvid); err != nil {
		return nil, err
	}
	// Run import command in txn
	opts := importer.Options{
		FeedVersionID: fvid,
		Directory:     cfg.GtfsDir,
		S3:            cfg.GtfsS3Bucket,
	}
	var fr importer.Result
	err := adapter.Tx(func(atx tldb.Adapter) error {
		var fe error
		fr, fe = importer.MainImportFeedVersion(adapter, opts)
		return fe
	})
	if err != nil {
		return nil, err
	}
	mr := model.FeedVersionImportResult{
		Success: fr.FeedVersionImport.Success,
	}
	return &mr, nil
}

func UpdateFeedVersion(id int, values model.FeedVersionSetInput, user *auth.User) (*model.FeedVersion, error) {
	// Update fv in txn
	ret := &model.FeedVersion{}
	err := model.Tx(func(db sqlx.Ext) error {
		if err := checkEditableFv(db, id); err != nil {
			return err
		}
		ents, err := find.FindFeedVersions(model.DB, nil, nil, []int{id}, nil)
		if err != nil {
			return err
		}
		fv := ents[0]
		if values.Name != nil {
			fv.Name = tl.NewOString(*values.Name)
		} else {
			fv.Name.Valid = false
		}
		if values.Description != nil {
			fv.Description = tl.NewOString(*values.Description)
		} else {
			fv.Description.Valid = false
		}
		if _, err := model.Sqrl(model.DB).Update("feed_versions").Set("name", fv.Name).Set("description", fv.Description).Where(sq.Eq{"id": id}).Exec(); err != nil {
			return err
		}
		ret = fv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// UnimportFeedVersion
func UnimportFeedVersion(id int) (*model.FeedVersionUnimportResult, error) {
	// Set of tables to delete where feed_version_id = fvid
	tables := []string{
		// derived entities
		"tl_agency_geometries",
		"tl_agency_places",
		"tl_route_geometries",
		"tl_route_stops",
		"tl_route_headways",
		"tl_feed_version_geometries",
		"tl_agency_onestop_ids",
		"tl_route_onestop_ids",
		"tl_stop_onestop_ids",
		// stop times
		"gtfs_stop_times",
		// anonymous entities
		"gtfs_transfers",
		"gtfs_calendar_dates",
		"gtfs_feed_infos",
		"gtfs_frequencies",
		"gtfs_pathways",
		// extensions
		"ext_faresv2_fare_capping",
		"ext_faresv2_fare_containers",
		"ext_faresv2_fare_leg_rules",
		"ext_faresv2_fare_products",
		"ext_faresv2_fare_timeframes",
		"ext_faresv2_fare_transfer_rules",
		"ext_faresv2_rider_categories",
		"ext_plus_calendar_attributes",
		"ext_plus_directions",
		"ext_plus_fare_rider_categories",
		"ext_plus_farezone_attributes",
		"ext_plus_realtime_routes",
		"ext_plus_realtime_stops",
		"ext_plus_realtime_trips",
		"ext_plus_rider_categories",
		"ext_plus_stop_attributes",
		"ext_plus_timepoints",
		// named entities
		"gtfs_fare_rules",
		"gtfs_fare_attributes",
		"gtfs_trips",
		"gtfs_shapes",
		"gtfs_calendars",
		"gtfs_routes",
		"gtfs_stops",
		"gtfs_agencies",
		"gtfs_levels",
		// editor records
		"tl_stop_external_references",
		"tl_ext_fare_networks",
		"tl_ext_gtfs_stops",
	}
	// Run in txn
	err := model.Tx(func(db sqlx.Ext) error {
		if err := checkEditableFv(db, id); err != nil {
			return err
		}
		where := sq.Eq{"feed_version_id": id}
		for _, table := range tables {
			_, err := model.Sqrl(db).Delete(table).Where(where).Exec()
			if err != nil {
				return err
			}
		}
		if _, err := model.Sqrl(db).Delete("feed_version_gtfs_imports").Where(where).Exec(); err != nil {
			return err
		}
		if _, err := model.Sqrl(db).Update("feed_states").Set("feed_version_id", nil).Where(where).Exec(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &model.FeedVersionUnimportResult{Success: true}, nil
}

func FeedVersionDelete(id int) (*model.FeedVersionDeleteResult, error) {
	tables := []string{
		"feed_version_file_infos",
		"feed_version_service_levels",
	}
	err := model.Tx(func(db sqlx.Ext) error {
		if err := checkEditableFv(db, id); err != nil {
			return err
		}
		where := sq.Eq{"feed_version_id": id}
		checkid := 0
		if err := model.Sqrl(db).Select("id").From("feed_version_gtfs_imports").Where(where).QueryRow().Scan(&checkid); err == nil {
			return errors.New("must unimport before deleting")
		}
		for _, table := range tables {
			if _, err := model.Sqrl(db).Delete(table).Where(where).Exec(); err != nil {
				return err
			}
		}
		if _, err := model.Sqrl(db).Delete("feed_versions").Where(sq.Eq{"id": id}).Exec(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return &model.FeedVersionDeleteResult{Success: false}, err
	}
	return &model.FeedVersionDeleteResult{Success: true}, nil
}

func checkEditableFv(db sqlx.Ext, id int) error {
	ents, err := find.FindFeedVersions(db, nil, nil, []int{id}, nil)
	if err != nil {
		return err
	} else if len(ents) != 1 {
		return errors.New("no such feed version")
	} else {
		feedId := ents[0].FeedID
		if feedEnts, err := find.FindFeeds(db, nil, nil, []int{feedId}, nil); err != nil {
			return err
		} else if len(feedEnts) != 1 {
			return errors.New("no such feed")
		}
		//  else if feedEnts[0].FeedID != "user" {
		// 	return errors.New("only user feeds may be modified")
		// }
	}
	return nil
}

type hasContext interface {
	Context() *causes.Context
}

func checkurl(address string) bool {
	if address == "" {
		return false
	}
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		return true
	}
	return false
}

// ValidateUpload takes a file Reader and produces a validation package containing errors, warnings, file infos, service levels, etc.
func ValidateUpload(cfg config.Config, src io.Reader, feedURL *string, rturls []string, user *auth.User) (*model.ValidationResult, error) {
	// Check inputs
	rturlsok := []string{}
	for _, rturl := range rturls {
		if checkurl(rturl) {
			rturlsok = append(rturlsok, rturl)
		}
	}
	rturls = rturlsok
	if feedURL == nil || !checkurl(*feedURL) {
		feedURL = nil
	}
	//////
	result := model.ValidationResult{}
	result.EarliestCalendarDate = time.Now()
	result.LatestCalendarDate = time.Now()

	var reader tl.Reader
	if src != nil {
		// Prepare reader
		var err error
		tmpfile, err := ioutil.TempFile("", "validator-upload")
		if err != nil {
			// This should result in a failed request
			return nil, err
		}
		io.Copy(tmpfile, src)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		reader, err = tlcsv.NewReader(tmpfile.Name())
		if err != nil {
			result.FailureReason = "Could not read file"
			return &result, nil
		}
	} else if feedURL != nil {
		var err error
		reader, err = tlcsv.NewReader(*feedURL)
		if err != nil {
			result.FailureReason = "Could not load URL"
			return &result, nil
		}
	} else {
		result.FailureReason = "No feed specified"
		return &result, nil
	}

	if err := reader.Open(); err != nil {
		result.FailureReason = "Could not read file"
		return &result, nil
	}

	// Perform validation
	opts := validator.Options{
		BestPractices:            true,
		CheckFileLimits:          true,
		IncludeServiceLevels:     true,
		IncludeRouteGeometries:   true,
		IncludeEntities:          true,
		IncludeEntitiesLimit:     10000,
		ValidateRealtimeMessages: rturls,
	}
	if cfg.ValidateLargeFiles {
		opts.CheckFileLimits = false
	}

	checker, err := validator.NewValidator(reader, opts)
	if err != nil {
		result.FailureReason = "Could not validate file"
		return &result, nil
	}
	r, err := checker.Validate()
	if err != nil {
		result.FailureReason = "Could not validate file"
		return &result, nil
	}

	// Some mapping is necessary because most gql models have some extra fields not in the base tl models.
	result.Success = r.Success
	result.FailureReason = r.FailureReason
	result.Sha1 = r.SHA1
	result.EarliestCalendarDate = r.EarliestCalendarDate
	result.LatestCalendarDate = r.LatestCalendarDate
	for _, eg := range r.Errors {
		if eg == nil {
			continue
		}
		eg2 := model.ValidationResultErrorGroup{
			Filename:  eg.Filename,
			ErrorType: eg.ErrorType,
			Count:     eg.Count,
			Limit:     eg.Limit,
		}
		for _, err := range eg.Errors {
			err2 := model.ValidationResultError{
				Filename: eg.Filename,
				Message:  err.Error(),
			}
			if v, ok := err.(hasContext); ok {
				c := v.Context()
				err2.EntityID = c.EntityID
				err2.Field = c.Field
			}
			eg2.Errors = append(eg2.Errors, &err2)
		}
		result.Errors = append(result.Errors, eg2)
	}
	for _, eg := range r.Warnings {
		if eg == nil {
			continue
		}
		eg2 := model.ValidationResultErrorGroup{
			Filename:  eg.Filename,
			ErrorType: eg.ErrorType,
			Count:     eg.Count,
			Limit:     eg.Limit,
		}
		for _, err := range eg.Errors {
			err2 := model.ValidationResultError{
				Filename: eg.Filename,
				Message:  err.Error(),
			}
			if v, ok := err.(hasContext); ok {
				c := v.Context()
				err2.EntityID = c.EntityID
				err2.Field = c.Field
			}
			eg2.Errors = append(eg2.Errors, &err2)
		}
		result.Warnings = append(result.Warnings, eg2)
	}
	for _, v := range r.FeedInfos {
		result.FeedInfos = append(result.FeedInfos, model.FeedInfo{FeedInfo: v})
	}
	for _, v := range r.Files {
		result.Files = append(result.Files, model.FeedVersionFileInfo{FeedVersionFileInfo: v})
	}
	for _, v := range r.ServiceLevels {
		result.ServiceLevels = append(result.ServiceLevels, model.FeedVersionServiceLevel{FeedVersionServiceLevel: v})
	}
	for _, v := range r.Agencies {
		result.Agencies = append(result.Agencies, model.Agency{Agency: v})
	}
	for _, v := range r.Routes {
		result.Routes = append(result.Routes, model.Route{Geometry: v.Geometry, Route: v})
	}
	for _, v := range r.Stops {
		result.Stops = append(result.Stops, model.Stop{Stop: v})
	}
	return &result, nil
}
