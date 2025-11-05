package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

func (f *Finder) ValidationReportsByFeedVersionIDs(ctx context.Context, limit *int, where *model.ValidationReportFilter, keys []int) ([][]*model.ValidationReport, error) {
	q := sq.StatementBuilder.
		Select("*").
		From("tl_validation_reports").
		Limit(finderCheckLimit(limit)).
		OrderBy("tl_validation_reports.created_at desc, tl_validation_reports.id desc")
	if where != nil {
		if len(where.ReportIds) > 0 {
			q = q.Where(In("tl_validation_reports.id", where.ReportIds))
		}
		if where.Success != nil {
			q = q.Where(sq.Eq{"success": where.Success})
		}
		if where.Validator != nil {
			q = q.Where(sq.Eq{"validator": where.Validator})
		}
		if where.ValidatorVersion != nil {
			q = q.Where(sq.Eq{"validator_version": where.ValidatorVersion})
		}
		if where.IncludesRt != nil {
			q = q.Where(sq.Eq{"includes_rt": where.IncludesRt})
		}
		if where.IncludesStatic != nil {
			q = q.Where(sq.Eq{"includes_static": where.IncludesStatic})
		}
	}
	var ents []*model.ValidationReport
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			q,
			"feed_versions",
			"id",
			"tl_validation_reports",
			"feed_version_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.ValidationReport) int { return ent.FeedVersionID }), err
}

func (f *Finder) ValidationReportErrorGroupsByValidationReportIDs(ctx context.Context, limit *int, keys []int) ([][]*model.ValidationReportErrorGroup, error) {
	var ents []*model.ValidationReportErrorGroup
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("tl_validation_report_error_groups", limit, nil, nil),
			"tl_validation_reports",
			"id",
			"tl_validation_report_error_groups",
			"validation_report_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.ValidationReportErrorGroup) int { return ent.ValidationReportID }), err
}

func (f *Finder) ValidationReportErrorExemplarsByValidationReportErrorGroupIDs(ctx context.Context, limit *int, keys []int) ([][]*model.ValidationReportError, error) {
	var ents []*model.ValidationReportError
	err := dbutil.Select(ctx,
		f.db,
		lateralWrap(
			quickSelect("tl_validation_report_error_exemplars", limit, nil, nil),
			"tl_validation_report_error_groups",
			"id",
			"tl_validation_report_error_exemplars",
			"validation_report_error_group_id",
			keys,
		),
		&ents,
	)
	return arrangeGroup(keys, ents, func(ent *model.ValidationReportError) int { return ent.ValidationReportErrorGroupID }), err
}
