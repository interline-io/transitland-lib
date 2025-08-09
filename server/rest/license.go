package rest

import "strings"

type LicenseFilter struct {
	LicenseCommercialUseAllowed  string `json:"license_commercial_use_allowed"`
	LicenseShareAlikeOptional    string `json:"license_share_alike_optional"`
	LicenseCreateDerivedProduct  string `json:"license_create_derived_product"`
	LicenseRedistributionAllowed string `json:"license_redistribution_allowed"`
	LicenseUseWithoutAttribution string `json:"license_use_without_attribution"`
}

func checkLicenseFilter(lic LicenseFilter) hw {
	w := hw{}
	if v := checkLicenseFilterValue(lic.LicenseCommercialUseAllowed); v != "" {
		w["commercial_use_allowed"] = v
	}
	if v := checkLicenseFilterValue(lic.LicenseShareAlikeOptional); v != "" {
		w["share_alike_optional"] = v
	}
	if v := checkLicenseFilterValue(lic.LicenseCreateDerivedProduct); v != "" {
		w["create_derived_product"] = v
	}
	if v := checkLicenseFilterValue(lic.LicenseRedistributionAllowed); v != "" {
		w["redistribution_allowed"] = v
	}
	if v := checkLicenseFilterValue(lic.LicenseUseWithoutAttribution); v != "" {
		w["use_without_attribution"] = v
	}
	if len(w) == 0 {
		return nil
	}
	return w
}

func checkLicenseFilterValue(v string) string {
	v = strings.ToLower(v)
	switch v {
	case "yes":
		return "YES"
	case "no":
		return "NO"
	case "unknown":
		return "UNKNOWN"
	case "exclude_no":
		return "EXCLUDE_NO"
	}
	return ""
}
