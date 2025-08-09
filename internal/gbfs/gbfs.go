package gbfs

import "github.com/interline-io/transitland-lib/tt"

// Loaders

type GbfsFeed struct {
	SystemInformation  *SystemInformation    `json:"system_information,omitempty"`
	StationInformation []*StationInformation `json:"station_information,omitempty"`
	StationStatus      []*StationStatus      `json:"station_status,omitempty"`
	Versions           []*SystemVersion      `json:"versions,omitempty"`
	VehicleTypes       []*VehicleType        `json:"vehicle_types,omitempty"`
	Bikes              []*FreeBikeStatus     `json:"bikes,omitempty"`
	Regions            []*SystemRegion       `json:"regions,omitempty"`
	RentalHours        []*SystemHour         `json:"rental_hours,omitempty"`
	Calendars          []*SystemCalendar     `json:"calendars,omitempty"`
	Plans              []*SystemPricingPlan  `json:"plans,omitempty"`
	Alerts             []*SystemAlert        `json:"alerts,omitempty"`
	GeofencingZones    []*GeofenceZone       // `json:"geofencing_zones,omitempty"`
}

type GbfsFeedData struct {
	Data *GbfsFeed
}

type SystemFeeds struct {
	Feeds []*SystemFeed `json:"feeds,omitempty"`
}

type SystemFeed struct {
	Name tt.String `json:"name,omitempty"`
	URL  tt.String `json:"url,omitempty"`
}

type SystemFile struct {
	Data map[string]*SystemFeeds `json:"data,omitempty"`
}

type SystemInformationFile struct {
	Data *SystemInformation `json:"data,omitempty"`
}

type StationInformationFile struct {
	Data struct {
		Stations []*StationInformation
	}
}

type StationStatusFile struct {
	Data struct {
		Stations []*StationStatus
	}
}

///////////////

// Main types

type SystemInformation struct {
	SystemID           tt.String   `json:"system_id,omitempty"`
	Language           tt.String   `json:"language,omitempty"`
	Name               tt.String   `json:"name,omitempty"`
	ShortName          tt.String   `json:"short_name,omitempty"`
	Operator           tt.String   `json:"operator,omitempty"`
	URL                tt.String   `json:"url,omitempty"`
	PurchaseURL        tt.String   `json:"purchase_url,omitempty"`
	StartDate          tt.Date     `json:"start_date,omitempty"`
	PhoneNumber        tt.String   `json:"phone_number,omitempty"`
	Email              tt.String   `json:"email,omitempty"`
	FeedContactEmail   tt.String   `json:"feed_contact_email,omitempty"`
	Timezone           tt.String   `json:"timezone,omitempty"`
	LicenseURL         tt.String   `json:"license_url,omitempty"`
	TermsURL           tt.String   `json:"terms_url,omitempty"`
	TermsLastUpdated   tt.Date     `json:"terms_last_updated,omitempty"`
	PrivacyURL         tt.String   `json:"privacy_url,omitempty"`
	PrivacyLastUpdated tt.Date     `json:"privacy_last_updated,omitempty"`
	BrandAssets        *BrandAsset `json:"brand_assets,omitempty"`
	RentalApps         *RentalApps `json:"rental_apps,omitempty"`
}

type RentalApps struct {
	Android *RentalApp `json:"android,omitempty"`
	IOS     *RentalApp `json:"ios,omitempty"`
}

type RentalApp struct {
	StoreURI     tt.String `json:"store_uri,omitempty"`
	DiscoveryURI tt.String `json:"discovery_uri,omitempty"`
}

type BrandAsset struct {
	BrandLastModified tt.Date   `json:"brand_last_modified,omitempty"`
	BrandTermsURL     tt.String `json:"brand_terms_url,omitempty"`
	BrandImageURL     tt.String `json:"brand_image_url,omitempty"`
	BrandImageURLDark tt.String `json:"brand_image_url_dark,omitempty"`
	Color             tt.String `json:"color,omitempty"`
}

///////////////

type StationInformation struct {
	StationID         tt.String         `json:"station_id,omitempty"`
	Name              tt.String         `json:"name,omitempty"`
	ShortName         tt.String         `json:"short_name,omitempty"`
	Lat               tt.Float          `json:"lat,omitempty"`
	Lon               tt.Float          `json:"lon,omitempty"`
	Address           tt.String         `json:"address,omitempty"`
	CrossStreet       tt.String         `json:"cross_street,omitempty"`
	RegionID          tt.String         `json:"region_id,omitempty"`
	PostCode          tt.String         `json:"post_code,omitempty"`
	RentalMethods     tt.Strings        `json:"rental_methods,omitempty"`
	IsVirtualStation  tt.Bool           `json:"is_virtual_station,omitempty"`
	StationArea       tt.Geometry       `json:"station_area,omitempty"`
	ParkingType       tt.String         `json:"parking_type,omitempty"`
	ParkingHoop       tt.Int            `json:"parking_hoop,omitempty"`
	ContactPhone      tt.String         `json:"contact_phone,omitempty"`
	Capacity          tt.Int            `json:"capacity,omitempty"`
	VehicleCapacity   map[string]tt.Int `json:"vehicle_capacity,omitempty"`
	IsValetStation    tt.Bool           `json:"is_valet_station,omitempty"`
	IsChargingStation tt.Bool           `json:"is_charging_station,omitempty"`
}

///////////////

type StationStatus struct {
	StationID             tt.String               `json:"station_id,omitempty"`
	NumBikesAvailable     tt.Int                  `json:"num_bikes_available,omitempty"`
	NumBikesDisabled      tt.Int                  `json:"num_bikes_disabled,omitempty"`
	NumDocksAvailable     tt.Int                  `json:"num_docks_available,omitempty"`
	NumDocksDisabled      tt.Int                  `json:"num_docks_disabled,omitempty"`
	IsReturning           tt.Bool                 `json:"is_returning,omitempty"`
	IsRenting             tt.Bool                 `json:"is_renting,omitempty"`
	IsInstalled           tt.Bool                 `json:"is_installed,omitempty"`
	LastReported          tt.Int                  `json:"last_reported,omitempty"`
	VehicleTypesAvailable []*VehicleTypeAvailable `json:"vehicle_types_available,omitempty"`
	VehicleDocksAvailable []*VehicleDockAvailable `json:"vehicle_docks_available,omitempty"`
}

type VehicleTypeAvailable struct {
	VehicleTypeID     tt.String `json:"vehicle_type_id,omitempty"`
	Count             tt.Int    `json:"count,omitempty"`
	NumBikesDisabled  tt.Int    `json:"num_bikes_disabled,omitempty"`
	NumDocksAvailable tt.Int    `json:"num_docks_available,omitempty"`
}

type VehicleDockAvailable struct {
	VehicleTypeIDs tt.Strings `json:"vehicle_type_ids,omitempty"`
	Count          tt.Int     `json:"count,omitempty"`
}

///////////////

type SystemVersion struct {
	Version tt.String `json:"version,omitempty"`
	URL     tt.String `json:"url,omitempty"`
}

type VehicleType struct {
	VehicleTypeID        tt.String      `json:"vehicle_type_id,omitempty"`
	FormFactor           tt.String      `json:"form_factor,omitempty"`
	RiderCapacity        tt.Int         `json:"rider_capacity,omitempty"`
	CargoVolumeCapacity  tt.Int         `json:"cargo_volume_capacity,omitempty"`
	CargoLoadCapacity    tt.Int         `json:"cargo_load_capacity,omitempty"`
	PropulsionType       tt.String      `json:"propulsion_type,omitempty"`
	EcoLabel             tt.String      `json:"eco_label,omitempty"`
	CountryCode          tt.String      `json:"country_code,omitempty"`
	EcoSticker           tt.String      `json:"eco_sticker,omitempty"`
	MaxRangeMeters       tt.Float       `json:"max_range_meters,omitempty"`
	Name                 tt.String      `json:"name,omitempty"`
	VehicleAccessories   tt.Strings     `json:"vehicle_accessories,omitempty"`
	GCO2Km               tt.Int         `json:"g_CO2_km,omitempty"`
	VehicleImage         tt.String      `json:"vehicle_image,omitempty"`
	Make                 tt.String      `json:"make,omitempty"`
	Model                tt.String      `json:"model,omitempty"`
	Color                tt.String      `json:"color,omitempty"`
	WheelCount           tt.Int         `json:"wheel_count,omitempty"`
	MaxPermittedSpeed    tt.Int         `json:"max_permitted_speed,omitempty"`
	RatedPower           tt.Int         `json:"rated_power,omitempty"`
	DefaultReserveTime   tt.Int         `json:"default_reserve_time,omitempty"`
	ReturnConstraint     tt.String      `json:"return_constraint,omitempty"`
	DefaultPricingPlanID tt.String      `json:"default_pricing_plan_id,omitempty"`
	PricingPlanIDs       tt.Strings     `json:"pricing_plan_ids,omitempty"`
	VehicleAssets        *VehicleAssets `json:"vehicle_assets,omitempty"`
	RentalURIs           *RentalURIs    `json:"rental_uris,omitempty"`
}

type VehicleAssets struct {
	IconURL          tt.String `json:"icon_url,omitempty"`
	IconURLDark      tt.String `json:"icon_url_dark,omitempty"`
	IconLastModified tt.Date   `json:"icon_last_modified,omitempty"`
}

type RentalURIs struct {
	Android tt.String `json:"android,omitempty"`
	IOS     tt.String `json:"ios,omitempty"`
	Web     tt.String `json:"web,omitempty"`
}

//

type FreeBikeStatus struct {
	BikeID             tt.String   `json:"bike_id,omitempty"`
	Lat                tt.Float    `json:"lat,omitempty"`
	Lon                tt.Float    `json:"lon,omitempty"`
	IsReserved         tt.Bool     `json:"is_reserved,omitempty"`
	IsDisabled         tt.Bool     `json:"is_disabled,omitempty"`
	VehicleTypeID      tt.String   `json:"vehicle_type_id,omitempty"`
	LastReported       tt.Int      `json:"last_reported,omitempty"`
	CurrentRangeMeters tt.Float    `json:"current_range_meters,omitempty"`
	CurrentFuelPercent tt.Float    `json:"current_fuel_percent,omitempty"`
	StationID          tt.String   `json:"station_id,omitempty"`
	HomeStationID      tt.String   `json:"home_station_id,omitempty"`
	PricingPlanID      tt.String   `json:"pricing_plan_id,omitempty"`
	VehicleEquipment   tt.Strings  `json:"vehicle_equipment,omitempty"`
	AvailableUntil     tt.Int      `json:"available_until,omitempty"`
	RentalURIs         *RentalURIs `json:"rental_uris,omitempty"`
}

type SystemHour struct {
	UserTypes tt.Strings `json:"user_types,omitempty"`
	Days      tt.Strings `json:"days,omitempty"`
	StartTime tt.String  `json:"start_time,omitempty"`
	EndTime   tt.String  `json:"end_time,omitempty"`
}

type SystemCalendar struct {
	StartMonth tt.Int `json:"start_month,omitempty"`
	StartDay   tt.Int `json:"start_day,omitempty"`
	StartYear  tt.Int `json:"start_year,omitempty"`
	EndMonth   tt.Int `json:"end_month,omitempty"`
	EndDay     tt.Int `json:"end_day,omitempty"`
	EndYear    tt.Int `json:"end_year,omitempty"`
}

type SystemRegion struct {
	RegionID tt.String `json:"region_id,omitempty"`
	Name     tt.String `json:"name,omitempty"`
}

type SystemPricingPlan struct {
	PlanID        tt.String    `json:"plan_id,omitempty"`
	URL           tt.String    `json:"url,omitempty"`
	Name          tt.String    `json:"name,omitempty"`
	Currency      tt.String    `json:"currency,omitempty"`
	Price         tt.Float     `json:"price,omitempty"`
	IsTaxable     tt.Bool      `json:"is_taxable,omitempty"`
	Description   tt.String    `json:"description,omitempty"`
	SurgePricing  tt.Bool      `json:"surge_pricing,omitempty"`
	PerKmPricing  []*PlanPrice `json:"per_km_pricing,omitempty"`
	PerMinPricing []*PlanPrice `json:"per_min_pricing,omitempty"`
}

type PlanPrice struct {
	Start    tt.Int   `json:"start,omitempty"`
	Rate     tt.Float `json:"rate,omitempty"`
	Interval tt.Int   `json:"interval,omitempty"`
	End      tt.Int   `json:"end,omitempty"`
}

type SystemAlert struct {
	AlertID     tt.String    `json:"alert_id,omitempty"`
	Type        tt.String    `json:"type,omitempty"`
	StationIDs  tt.Strings   `json:"station_ids,omitempty"`
	RegionIDs   tt.Strings   `json:"region_ids,omitempty"`
	URL         tt.String    `json:"url,omitempty"`
	Summary     tt.String    `json:"summary,omitempty"`
	Description tt.String    `json:"description,omitempty"`
	LastUpdated tt.Int       `json:"last_updated,omitempty"`
	Times       []*AlertTime `json:"times,omitempty"`
}

type AlertTime struct {
	Start tt.Int `json:"start,omitempty"`
	End   tt.Int `json:"end,omitempty"`
}

type GeofenceZone struct {
	Type     tt.String
	Features []*GeofenceFeature
}

type GeofenceFeature struct {
	Type       tt.String         `json:"type,omitempty"`
	Geometry   tt.Geometry       `json:"geometry,omitempty"`
	Properties *GeofenceProperty `json:"properties,omitempty"`
}

type GeofenceProperty struct {
	Name  tt.String       `json:"name,omitempty"`
	Start tt.Int          `json:"start,omitempty"`
	End   tt.Int          `json:"end,omitempty"`
	Rules []*GeofenceRule `json:"rules,omitempty"`
}

type GeofenceRule struct {
	VehicleTypeID      tt.Strings `json:"vehicle_type_id,omitempty"`
	RideAllowed        tt.Bool    `json:"ride_allowed,omitempty"`
	RideThroughAllowed tt.Bool    `json:"ride_through_allowed,omitempty"`
	MaximumSpeedKph    tt.Int     `json:"maximum_speed_kph,omitempty"`
	StationParking     tt.Bool    `json:"station_parking,omitempty"`
}
