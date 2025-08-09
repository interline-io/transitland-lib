package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/model"
)

type validationReportResolver struct{ *Resolver }

func (r *validationReportResolver) Errors(ctx context.Context, obj *model.ValidationReport, limit *int) ([]*model.ValidationReportErrorGroup, error) {
	if len(obj.Errors) > 0 {
		return obj.Errors, nil
	}
	return LoaderFor(ctx).ValidationReportErrorGroupsByValidationReportIDs.Load(ctx, validationReportErrorGroupLoaderParam{ValidationReportID: obj.ID, Limit: limit})()
}

func (r *validationReportResolver) Warnings(ctx context.Context, obj *model.ValidationReport, limit *int) ([]*model.ValidationReportErrorGroup, error) {
	if len(obj.Warnings) > 0 {
		return obj.Warnings, nil
	}
	return nil, nil
}

func (r *validationReportResolver) Details(ctx context.Context, obj *model.ValidationReport) (*model.ValidationReportDetails, error) {
	return obj.Details, nil
}

type validationReportErrorGroupResolver struct{ *Resolver }

func (r *validationReportErrorGroupResolver) Errors(ctx context.Context, obj *model.ValidationReportErrorGroup, limit *int) ([]*model.ValidationReportError, error) {
	if len(obj.Errors) > 0 {
		return obj.Errors, nil
	}
	ret, err := LoaderFor(ctx).ValidationReportErrorExemplarsByValidationReportErrorGroupIDs.Load(ctx, validationReportErrorExemplarLoaderParam{ValidationReportGroupID: obj.ID, Limit: limit})()
	if err != nil {
		return nil, err
	}
	for _, r := range ret {
		r.GroupKey = obj.GroupKey
		r.ErrorCode = obj.ErrorCode
		r.ErrorType = obj.ErrorType
		r.Field = obj.Field
		r.Filename = obj.Filename
	}
	return ret, nil
}
