package validator

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tldb"
)

type resultReport struct {
	FeedVersionID tt.Int
	StaticSHA1    tt.String
	StaticURL     tt.String
	tl.DatabaseEntity
	tl.Timestamps
}

func (e *resultReport) TableName() string {
	return "tl_validator_reports"
}

type resultErrorGroup struct {
	ReportID  int
	Filename  string
	ErrorType string
	Count     int
	tl.DatabaseEntity
}

func (e *resultErrorGroup) TableName() string {
	return "tl_validator_error_groups"
}

type resultErrorError struct {
	ErrorGroupID int
	Error        string
	tl.DatabaseEntity
}

func (e *resultErrorError) TableName() string {
	return "tl_validator_errors"
}

func WriteResult(atx tldb.Adapter, fvid int, result *Result) error {
	rr := resultReport{}
	if fvid > 0 {
		rr.FeedVersionID = tt.NewInt(fvid)
	}
	_, err := atx.Insert(&rr)
	if err != nil {
		return err
	}
	for _, errGroup := range result.Result.Errors {
		eg := resultErrorGroup{
			ReportID:  rr.ID,
			Filename:  errGroup.Filename,
			ErrorType: errGroup.ErrorType,
			Count:     errGroup.Count,
		}
		if _, err := atx.Insert(&eg); err != nil {
			return err
		}
		for _, e := range errGroup.Errors {
			if _, err := atx.Insert(&resultErrorError{
				ErrorGroupID: eg.ID,
				Error:        e.Error(),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
