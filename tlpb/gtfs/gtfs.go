package gtfs

type EnumValue int32

type WheelchairAccessEnum int32

type BikeAccessEnum int32

type BoardAccessEnum int32

type PickupAccessEnum int32

type StopLocationTypeEnum int32

type RouteTypeEnum int32

type TripDirectionEnum int32

type StopTimepointEnum int32

type CalendarExceptionTypeEnum int32

type FrequencyExactTimeEnum int32

type TransferTypeEnum int32

type PathwayDirectionalityEnum int32

type PathwayModeEnum int32

type BookingRuleTypeEnum int32

type FareMediaTypeEnum int32

type FareTransferTypeEnum int32

type DurationLimitTypeEnum int32

type FareAttributeTransferTypeEnum int32

type PaymentMethodEnum int32

type AttributionRoleEnum int32

type FeedEntity struct {
	Agency Agency
	Stop Stop
	Route Route
	Trip Trip
	StopTime StopTime
	Shape Shape
	Service Service
}

type Agency struct {
	DatabaseEntity
	AgencyID Key
	AgencyName String
	AgencyUrl Url
	AgencyTimezone Timezone
	AgencyLang Language
	AgencyPhone Phone
	AgencyFareUrl Url
	AgencyEmail Email
}

type Stop struct {
	DatabaseEntity
	StopID Key
	StopCode String
	StopName String
	TtsStopName String
	StopDesc String
	StopLat Float
	StopLon Float
	ZoneID String
	StopUrl Url
	LocationType StopLocationType
	ParentStation Reference
	StopTimezone Timezone
	WheelchairBoarding WheelchairAccess
	LevelID Reference
	PlatformCode String
}

type Route struct {
	DatabaseEntity
	RouteID Key
	AgencyID Reference
	RouteShortName String
	RouteLongName String
	RouteDesc String
	RouteType RouteType
	RouteUrl Url
	RouteColor Color
	RouteTextColor Color
	RouteSortOrder Int
	ContinuousPickup PickupAccess
	ContinuousDropOff PickupAccess
	NetworkID String
}

type Trip struct {
	DatabaseEntity
	RouteID Reference
	ServiceID Reference
	TripID Key
	TripHeadsign String
	TripShortName String
	DirectionID TripDirection
	BlockID String
	ShapeID Reference
	WheelchairAccessible WheelchairAccess
	BikesAllowed BikeAccess
}

type StopTime struct {
	DatabaseEntity
	TripID Reference
	ArrivalTime Seconds
	DepartureTime Seconds
	StopID Reference
	StopSequence Int
	StopHeadsign String
	ContinuousPickup PickupAccess
	ContinuousDropOff PickupAccess
	ShapeDistTraveled Float
	Timepoint StopTimepoint
	LocationID Reference
	LocationGroupID Reference
	StartPickupDropOffWindow Seconds
	EndPickupDropOffWindow Seconds
	PickupType PickupAccess
	DropOffType PickupAccess
	PickupBookingRuleID Reference
	DropOffBookingRuleID Reference
}

type Calendar struct {
	DatabaseEntity
	ServiceID Key
	StartDate Date
	EndDate Date
	Monday Bool
	Tuesday Bool
	Wednesday Bool
	Thursday Bool
	Friday Bool
	Saturday Bool
	Sunday Bool
}

type CalendarDate struct {
	DatabaseEntity
	ServiceID Reference
	Date Date
	ExceptionType CalendarExceptionType
}

type FareAttribute struct {
	DatabaseEntity
	FareID Key
	Price Money
	CurrencyType Currency
	PaymentMethod PaymentMethod
	Transfers FareAttributeTransferType
	AgencyID Reference
	TransferDuration Int
}

type FareRule struct {
	DatabaseEntity
	FareID Key
	RouteID Reference
	OriginID Reference
	DestinationID Reference
	ContainsID Reference
}

type Timeframe struct {
	DatabaseEntity
	TimeframeGroupID Key
	StartTime Seconds
	EndTime Seconds
	ServiceID Reference
}

type FareMedia struct {
	DatabaseEntity
	FareMediaID Key
	FareMediaName String
	FareMediaType FareMediaType
}

type FareProduct struct {
	DatabaseEntity
	FareProductID Key
	FareProductName String
	FareMediaID Reference
	Amount Money
	Currency Currency
}

type FareLegRule struct {
	DatabaseEntity
	LegGroupID Key
	NetworkID Reference
	FromAreaID Reference
	ToAreaID Reference
	FromTimeframeGroupID Reference
	ToTimeframeGroupID Reference
	FareProductID Reference
	RuleProirity Int
}

type FareTransferRule struct {
	DatabaseEntity
	FromLegGroupID Reference
	ToLegGroupID Reference
	TransferCount Int
	DurationLimitType DurationLimitType
	FareTransferType FareTransferType
	FareProductID Reference
}

type Area struct {
	DatabaseEntity
	AreaID Key
	AreaName String
}

type StopArea struct {
	DatabaseEntity
	AreaID Reference
	StopID Reference
}

type Network struct {
	DatabaseEntity
	NetworkID Key
	NetworkName String
}

type RouteNetwork struct {
	DatabaseEntity
	NetworkID Reference
	RouteID Reference
}

type Frequency struct {
	DatabaseEntity
	TripID Reference
	StartTime Seconds
	EndTime Seconds
	HeadwaySecs Int
	ExactTime FrequencyExactTime
}

type Transfer struct {
	DatabaseEntity
	FromStopID Reference
	ToStopID Reference
	FromRouteID Reference
	ToRouteID Reference
	FromTripID Reference
	ToTripID Reference
	TransferType TransferType
	MinTransferTime Int
}

type Pathway struct {
	DatabaseEntity
	PathwayID Key
	FromStopID Reference
	ToStopID Reference
	PathwayMode PathwayMode
	IsBidirectional PathwayDirectionality
	Length Float
	TraversalTime Int
	StairCount Int
	MaxSlope Float
	MinWidth Float
	SignpostedAs String
	ReverseSignpostedAs String
}

type Level struct {
	DatabaseEntity
	LevelID Key
	LevelIndex Float
	LevelName String
}

type LocationGroup struct {
	DatabaseEntity
	LocationGroupID Key
	LocationGroupName String
}

type LocationGroupStop struct {
	DatabaseEntity
	LocationGroupID Reference
	StopID Reference
}

type BookingRule struct {
	DatabaseEntity
	BookingRuleID Key
	BookingType BookingRuleType
	PriorNoticeDurationMin Int
	PriorNoticeDurationMax Int
	PriorNoticeLastDay Int
	PriorNoticeLastTime Seconds
	PriorNoticeStartDay Int
	PriorNoticeStartTime Seconds
	PriorNoticeServiceID Reference
	Message String
	PickupMessage String
	DropOffMessage String
	PhoneNumber String
	InfoUrl Url
	BookingUrl Url
}

type Translation struct {
	DatabaseEntity
	TableName String
	FieldName String
	Language Language
	Translation String
	RecordID String
	RecordSubID String
	FieldValue String
}

type FeedInfo struct {
	DatabaseEntity
	FeedPublisherName String
	FeedPublisherUrl Url
	FeedLang Language
	DefaultLang Language
	FeedStartDate Date
	FeedEndDate Date
	FeedContactEmail Email
	FeedContactUrl Url
}

type Attribution struct {
	DatabaseEntity
	AttributionID Key
	AgencyID Reference
	RouteID Reference
	TripID Reference
	OrganizationName String
	IsProducer AttributionRole
	IsOperator AttributionRole
	IsAuthority AttributionRole
	AttributionUrl Url
	AttributionEmail Email
	AttributionPhone Phone
}

type ShapePoint struct {
	ShapeID Key
	ShapePtLat float64
	ShapePtLon float64
	ShapePtSequence int32
	ShapeDistTraveled float64
}

type Service struct {
	DatabaseEntity
	ServiceID Key
	StartDate Date
	EndDate Date
	Added Date
	Removed Date
	Monday Bool
	Tuesday Bool
	Wednesday Bool
	Thursday Bool
	Friday Bool
	Saturday Bool
	Sunday Bool
}

type Shape struct {
	DatabaseEntity
	ShapeID Key
	Geometry LineString
}

type Point struct {
	Lon float64
	Lat float64
}

type LineString struct {
	Stride uint32
	Coordinates float64
}

type DatabaseEntity struct {
	ID int64
	FeedVersionID int64
}

type Date struct {
	Year int32
	Month int32
	Day int32
}

type Timestamp struct { Option[int64] }

type Seconds struct { Option[int64] }

type Key struct { Option[string] }

type Timezone struct { Option[string] }

type Reference struct { Option[string] }

type Url struct { Option[string] }

type Email struct { Option[string] }

type Color struct { Option[string] }

type Money struct {
	Units int64
	Nanos int64
}

type Currency struct { Option[string] }

type Language struct { Option[string] }

type Phone struct { Option[string] }

type Float struct { Option[float64] }

type String struct { Option[string] }

type Int struct { Option[int64] }

type Bool struct { Option[bool] }

type WheelchairAccess struct { Option[WheelchairAccessEnum] }

type BikeAccess struct { Option[BikeAccessEnum] }

type BoardAccess struct { Option[BoardAccessEnum] }

type PickupAccess struct { Option[PickupAccessEnum] }

type StopLocationType struct { Option[StopLocationTypeEnum] }

type RouteType struct { Option[RouteTypeEnum] }

type TripDirection struct { Option[TripDirectionEnum] }

type StopTimepoint struct { Option[StopTimepointEnum] }

type CalendarExceptionType struct { Option[CalendarExceptionTypeEnum] }

type FrequencyExactTime struct { Option[FrequencyExactTimeEnum] }

type TransferType struct { Option[TransferTypeEnum] }

type PathwayDirectionality struct { Option[PathwayDirectionalityEnum] }

type PathwayMode struct { Option[PathwayModeEnum] }

type BookingRuleType struct { Option[BookingRuleTypeEnum] }

type FareMediaType struct { Option[FareMediaTypeEnum] }

type FareTransferType struct { Option[FareTransferTypeEnum] }

type DurationLimitType struct { Option[DurationLimitTypeEnum] }

type FareAttributeTransferType struct { Option[FareAttributeTransferTypeEnum] }

type PaymentMethod struct { Option[PaymentMethodEnum] }

type AttributionRole struct { Option[AttributionRoleEnum] }

