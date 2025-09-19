package model

import (
	"github.com/interline-io/transitland-lib/internal/gbfs"
)

type GbfsAlertTime struct {
	*gbfs.AlertTime
}

type GbfsBrandAsset struct {
	*gbfs.BrandAsset
}

type GbfsFeed struct {
	*gbfs.GbfsFeed
}

func (g *GbfsFeed) SystemInformation() *GbfsSystemInformation {
	if g.GbfsFeed == nil || g.GbfsFeed.SystemInformation == nil {
		return nil
	}
	return &GbfsSystemInformation{
		Feed:              g,
		SystemInformation: g.GbfsFeed.SystemInformation,
	}
}

func (g *GbfsFeed) StationInformation() []*GbfsStationInformation {
	if g.GbfsFeed == nil {
		return nil
	}
	var ret []*GbfsStationInformation
	for _, s := range g.GbfsFeed.StationInformation {
		if s == nil {
			continue
		}
		ret = append(ret, &GbfsStationInformation{
			Feed:               g,
			StationInformation: s,
		})
	}
	return ret
}

func (g *GbfsFeed) RentalHours() []*GbfsSystemHour {
	if g.GbfsFeed == nil {
		return nil
	}
	var ret []*GbfsSystemHour
	for _, s := range g.GbfsFeed.RentalHours {
		ret = append(ret, &GbfsSystemHour{SystemHour: s})
	}
	return ret
}

func (g *GbfsFeed) Calendars() []*GbfsSystemCalendar {
	if g.GbfsFeed == nil {
		return nil
	}
	var ret []*GbfsSystemCalendar
	for _, s := range g.GbfsFeed.Calendars {
		ret = append(ret, &GbfsSystemCalendar{SystemCalendar: s})
	}
	return ret
}

func (g *GbfsFeed) Alerts() []*GbfsSystemAlert {
	if g.GbfsFeed == nil {
		return nil
	}
	var ret []*GbfsSystemAlert
	for _, s := range g.GbfsFeed.Alerts {
		ret = append(ret, &GbfsSystemAlert{SystemAlert: s})
	}
	return ret
}

type GbfsFreeBikeStatus struct {
	Feed *GbfsFeed
	*gbfs.FreeBikeStatus
}

func (g *GbfsFreeBikeStatus) Station() *GbfsStationInformation {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.StationInformation() {
		if s == nil {
			continue
		}
		if s.StationID.Val == g.StationID.Val {
			return s
		}
	}
	return nil
}

func (g *GbfsFreeBikeStatus) HomeStation() *GbfsStationInformation {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.StationInformation() {
		if s == nil {
			continue
		}
		if s.StationID.Val == g.HomeStationID.Val {
			return s
		}
	}
	return nil
}

func (g *GbfsFreeBikeStatus) PricingPlan() *GbfsSystemPricingPlan {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.Plans {
		if s == nil {
			continue
		}
		if s.PlanID.Val == g.PricingPlanID.Val {
			return &GbfsSystemPricingPlan{SystemPricingPlan: s}
		}
	}
	return nil
}

func (g *GbfsFreeBikeStatus) VehicleType() *GbfsVehicleType {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.VehicleTypes {
		if s == nil {
			continue
		}
		if s.VehicleTypeID.Val == g.VehicleTypeID.Val {
			return &GbfsVehicleType{VehicleType: s, Feed: g.Feed}
		}
	}
	return nil
}

func (g *GbfsFreeBikeStatus) RentalUris() *GbfsRentalUris {
	if g.FreeBikeStatus == nil || g.RentalURIs == nil {
		return nil
	}
	return &GbfsRentalUris{RentalURIs: g.RentalURIs}
}

type GbfsGeofenceFeature struct {
	*gbfs.GeofenceFeature
}

type GbfsVehicleAssets struct {
	*gbfs.VehicleAssets
}

func (g *GbfsGeofenceFeature) Properties() []*GbfsGeofenceProperty {
	return nil
}

type GbfsGeofenceProperty struct {
	*gbfs.GeofenceProperty
}

func (g *GbfsGeofenceProperty) Rules() []*GbfsGeofenceRule {
	return nil
}

type GbfsGeofenceRule struct {
	*gbfs.GeofenceRule
}

func (g *GbfsGeofenceRule) VehicleType() *GbfsVehicleType {
	return nil
}

type GbfsGeofenceZone struct {
	*gbfs.GeofenceZone
}

func (g *GbfsGeofenceZone) Features() []*GbfsGeofenceFeature {
	return nil
}

type GbfsPlanPrice struct {
	*gbfs.PlanPrice
}

type GbfsRentalApps struct {
	*gbfs.RentalApps
}

func (g *GbfsRentalApps) Android() *GbfsRentalApp {
	if g.RentalApps == nil || g.RentalApps.Android == nil {
		return nil
	}
	return &GbfsRentalApp{RentalApp: g.RentalApps.Android}
}

func (g *GbfsRentalApps) Ios() *GbfsRentalApp {
	if g.RentalApps == nil || g.RentalApps.IOS == nil {
		return nil
	}
	return &GbfsRentalApp{RentalApp: g.RentalApps.IOS}
}

type GbfsRentalApp struct {
	*gbfs.RentalApp
}

type GbfsStationInformation struct {
	Feed *GbfsFeed
	*gbfs.StationInformation
}

func (g *GbfsStationInformation) Region() *GbfsSystemRegion {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.Regions {
		if s == nil {
			continue
		}
		if s.RegionID.Val == g.RegionID.Val {
			return &GbfsSystemRegion{SystemRegion: s}
		}
	}
	return nil
}

func (g *GbfsStationInformation) Status() *GbfsStationStatus {
	if g.Feed == nil {
		return nil
	}
	for _, s := range g.Feed.StationStatus {
		if s == nil {
			continue
		}
		if s.StationID.Val == g.StationID.Val {
			return &GbfsStationStatus{StationStatus: s}
		}
	}
	return nil
}

type GbfsStationStatus struct {
	Feed *GbfsFeed
	*gbfs.StationStatus
}

func (g *GbfsStationStatus) VehicleTypesAvailable() []*GbfsVehicleTypeAvailable {
	if g.StationStatus == nil {
		return nil
	}
	var ret []*GbfsVehicleTypeAvailable
	for _, s := range g.StationStatus.VehicleTypesAvailable {
		if s == nil {
			continue
		}
		ret = append(ret, &GbfsVehicleTypeAvailable{VehicleTypeAvailable: s, Feed: g.Feed})
	}
	return ret
}

func (g *GbfsStationStatus) VehicleDocksAvailable() []*GbfsVehicleDockAvailable {
	if g.StationStatus == nil {
		return nil
	}
	var ret []*GbfsVehicleDockAvailable
	for _, s := range g.StationStatus.VehicleDocksAvailable {
		if s == nil {
			continue
		}
		ret = append(ret, &GbfsVehicleDockAvailable{VehicleDockAvailable: s, Feed: g.Feed})
	}
	return ret
}

type GbfsSystemAlert struct {
	*gbfs.SystemAlert
}

func (g *GbfsSystemAlert) Times() []*GbfsAlertTime {
	if g.SystemAlert == nil {
		return nil
	}
	var ret []*GbfsAlertTime
	for _, s := range g.SystemAlert.Times {
		if s == nil {
			continue
		}
		ret = append(ret, &GbfsAlertTime{AlertTime: s})
	}
	return ret
}

type GbfsSystemCalendar struct {
	*gbfs.SystemCalendar
}

type GbfsSystemHour struct {
	*gbfs.SystemHour
}

type GbfsSystemInformation struct {
	Feed *GbfsFeed
	*gbfs.SystemInformation
}

func (g *GbfsSystemInformation) BrandAssets() *GbfsBrandAsset {
	if g.SystemInformation == nil || g.SystemInformation.BrandAssets == nil {
		return nil
	}
	return &GbfsBrandAsset{BrandAsset: g.SystemInformation.BrandAssets}
}

func (g *GbfsSystemInformation) RentalApps() *GbfsRentalApps {
	if g.SystemInformation == nil || g.SystemInformation.RentalApps == nil {
		return nil
	}
	return &GbfsRentalApps{RentalApps: g.SystemInformation.RentalApps}
}

type GbfsSystemPricingPlan struct {
	*gbfs.SystemPricingPlan
}

func (g *GbfsSystemPricingPlan) PerKmPricing() []*GbfsPlanPrice {
	if g.SystemPricingPlan == nil {
		return nil
	}
	var ret []*GbfsPlanPrice
	for _, s := range g.SystemPricingPlan.PerKmPricing {
		ret = append(ret, &GbfsPlanPrice{PlanPrice: s})
	}
	return ret
}

func (g *GbfsSystemPricingPlan) PerMinPricing() []*GbfsPlanPrice {
	if g.SystemPricingPlan == nil {
		return nil
	}
	var ret []*GbfsPlanPrice
	for _, s := range g.SystemPricingPlan.PerMinPricing {
		ret = append(ret, &GbfsPlanPrice{PlanPrice: s})
	}
	return ret
}

type GbfsSystemRegion struct {
	*gbfs.SystemRegion
}

type GbfsSystemVersion struct {
	*gbfs.SystemVersion
}

type GbfsVehicleDockAvailable struct {
	Feed *GbfsFeed
	*gbfs.VehicleDockAvailable
}

func (g *GbfsVehicleDockAvailable) VehicleTypes() []*GbfsVehicleType {
	if g.VehicleDockAvailable == nil {
		return nil
	}
	var ret []*GbfsVehicleType
	for _, s := range g.VehicleDockAvailable.VehicleTypeIDs.Val {
		for _, t := range g.Feed.VehicleTypes {
			if s == t.VehicleTypeID.Val {
				ret = append(ret, &GbfsVehicleType{VehicleType: t, Feed: g.Feed})
			}
		}
	}
	return ret
}

type GbfsVehicleType struct {
	Feed *GbfsFeed
	*gbfs.VehicleType
}

func (g *GbfsVehicleType) DefaultPricingPlan() *GbfsSystemPricingPlan {
	if g.Feed == nil || g.VehicleType == nil {
		return nil
	}
	for _, s := range g.Feed.Plans {
		if s.PlanID.Val == g.DefaultPricingPlanID.Val {
			return &GbfsSystemPricingPlan{SystemPricingPlan: s}
		}
	}
	return nil
}

func (g *GbfsVehicleType) PricingPlans() []*GbfsSystemPricingPlan {
	if g.VehicleType == nil {
		return nil
	}
	var ret []*GbfsSystemPricingPlan
	for _, t := range g.PricingPlanIDs.Val {
		for _, s := range g.Feed.Plans {
			if t == s.PlanID.Val {
				ret = append(ret, &GbfsSystemPricingPlan{SystemPricingPlan: s})
			}
		}
	}
	return ret
}

func (g *GbfsVehicleType) RentalUris() *GbfsRentalUris {
	if g.RentalURIs == nil {
		return nil
	}
	return &GbfsRentalUris{RentalURIs: g.RentalURIs}
}

func (g *GbfsVehicleType) VehicleAssets() *GbfsVehicleAssets {
	if g.VehicleType == nil {
		return nil
	}
	return &GbfsVehicleAssets{VehicleAssets: g.VehicleType.VehicleAssets}
}

type GbfsVehicleTypeAvailable struct {
	Feed *GbfsFeed
	*gbfs.VehicleTypeAvailable
}

func (g *GbfsVehicleTypeAvailable) VehicleType() *GbfsVehicleType {
	if g.Feed == nil || g.VehicleTypeAvailable == nil {
		return nil
	}
	for _, s := range g.Feed.VehicleTypes {
		if s.VehicleTypeID.Val == g.VehicleTypeID.Val {
			return &GbfsVehicleType{VehicleType: s, Feed: g.Feed}
		}

	}
	return nil
}

type GbfsRentalUris struct {
	*gbfs.RentalURIs
}
