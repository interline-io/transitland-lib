// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	_ "embed"

	"github.com/interline-io/transitland-lib/tl/gtfs"
)

type Agency = gtfs.Agency
type Area = gtfs.Area
type Attribution = gtfs.Attribution
type Calendar = gtfs.Calendar
type CalendarDate = gtfs.CalendarDate
type FareAttribute = gtfs.FareAttribute
type FareLegRule = gtfs.FareLegRule
type FareMedia = gtfs.FareMedia
type FareProduct = gtfs.FareProduct
type FareTransferRule = gtfs.FareTransferRule
type FeedInfo = gtfs.FeedInfo
type Frequency = gtfs.Frequency
type Level = gtfs.Level
type Pathway = gtfs.Pathway
type RiderCategory = gtfs.RiderCategory
type Route = gtfs.Route
type Shape = gtfs.Shape
type Stop = gtfs.Stop
type StopArea = gtfs.StopArea
type StopTime = gtfs.StopTime
type Trip = gtfs.Trip
type FareRule = gtfs.FareRule
type Translation = gtfs.Translation
type Transfer = gtfs.Transfer

//////////

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	StopTimesByTripID(...string) chan []StopTime
	String() string
	// Entities
	ReadEntities(c interface{}) error
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
}

// Writer writes a GTFS feed.
type Writer interface {
	Open() error
	Close() error
	Create() error
	Delete() error
	NewReader() (Reader, error)
	AddEntity(Entity) (string, error)
	AddEntities([]Entity) ([]string, error)
	String() string
}

type WriterWithExtraColumns interface {
	Writer
	WriteExtraColumns(bool)
}

// Entity Types

// Entity provides an interface for GTFS entities.
type Entity interface {
	EntityID() string
	Filename() string
}

type EntityWithID interface {
	GetID() int
}

type EntityWithExtra interface {
	SetExtra(string, string)
	GetExtra(string) (string, bool)
	ClearExtra()
	ExtraKeys() []string
}

type EntityWithErrors interface {
	Errors() []error
	Warnings() []error
	AddError(error)
	AddWarning(error)
}
