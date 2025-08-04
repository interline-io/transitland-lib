package dbfinder

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/model"
)

func licenseFilter(license *model.LicenseFilter, q sq.SelectBuilder) sq.SelectBuilder {
	return licenseFilterTable(license, q)
}

func licenseCheck(col string, v *model.LicenseValue, q sq.SelectBuilder) sq.SelectBuilder {
	if v == nil {
	} else if *v == model.LicenseValueYes {
		q = q.Where(sq.Expr("jsonb_extract_path_text(license,?) = ?", col, "yes"))
	} else if *v == model.LicenseValueUnknown {
		q = q.Where(sq.Expr("jsonb_extract_path_text(license,?) = ?", col, "unknown"))
	} else if *v == model.LicenseValueNo {
		q = q.Where(sq.Expr("jsonb_extract_path_text(license,?) = ?", col, "no"))
	} else if *v == model.LicenseValueExcludeNo {
		q = q.Where(sq.Expr("(jsonb_extract_path_text(license,?) IN (?,?) or jsonb_extract_path_text(license,?) is null)", col, "yes", "unknown", col))
	}
	return q
}

func licenseFilterTable(license *model.LicenseFilter, q sq.SelectBuilder) sq.SelectBuilder {
	if license == nil {
		return q
	}
	q = licenseCheck("commercial_use_allowed", license.CommercialUseAllowed, q)
	q = licenseCheck("share_alike_optional", license.ShareAlikeOptional, q)
	q = licenseCheck("create_derived_product", license.CreateDerivedProduct, q)
	q = licenseCheck("redistribution_allowed", license.RedistributionAllowed, q)
	q = licenseCheck("use_without_attribution", license.UseWithoutAttribution, q)
	return q
}
