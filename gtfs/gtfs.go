package gtfs

import (
	"strconv"
)

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}

type Reader interface {
	Stops() chan Stop
	StopTimes() chan StopTime
	Agencies() chan Agency
	Calendars() chan Calendar
	CalendarDates() chan CalendarDate
	FareAttributes() chan FareAttribute
	FareRules() chan FareRule
	FeedInfos() chan FeedInfo
	Frequencies() chan Frequency
	Routes() chan Route
	Shapes() chan Shape
	Transfers() chan Transfer
	Pathways() chan Pathway
	Levels() chan Level
	Trips() chan Trip
	Translations() chan Translation
	Attributions() chan Attribution
	Areas() chan Area
	StopAreas() chan StopArea
	FareLegRules() chan FareLegRule
	FareTransferRules() chan FareTransferRule
	FareProducts() chan FareProduct
	RiderCategories() chan RiderCategory
	FareMedia() chan FareMedia
	Timeframes() chan Timeframe
	Networks() chan Network
	RouteNetworks() chan RouteNetwork
	FareLegJoinRules() chan FareLegJoinRule
}
