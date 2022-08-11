package tl

import (
	"strconv"
	"time"
)

// EntityError is an interface for GTFS Errors
type EntityError interface {
	Error() string
}

// Entity provides an interface for GTFS entities.
type Entity interface {
	EntityID() string
	Filename() string
	Errors() []error
	Warnings() []error
	AddError(error)
	AddWarning(error)
	SetExtra(string, string)
	Extra() map[string]string
	UpdateKeys(*EntityMap) error
}

/////////

// MinEntity provides default methods.
type MinEntity struct {
	extra        []string
	loadErrors   []error
	loadWarnings []error
}

// Filename returns the filename for this entity.
func (ent MinEntity) Filename() string { return "" }

// EntityID returns the entity ID.
func (ent MinEntity) EntityID() string { return "" }

/////////

type ExtraEntity struct {
	extra []string
}

// SetExtra adds a string key, value pair to the entity's extra fields.
func (ent *ExtraEntity) SetExtra(key string, value string) {
	ent.extra = append(ent.extra, key, value)
}

func (ent *ExtraEntity) GetExtra(key string) (string, bool) {
	for i := 0; i < len(ent.extra); i += 2 {
		if ent.extra[i] == key {
			return ent.extra[i+1], true
		}
	}
	return "", false
}

func (ent *ExtraEntity) ExtraKeys() []string {
	var ret []string
	for i := 0; i < len(ent.extra); i += 2 {
		ret = append(ret, ent.extra[i])
	}
	return ret
}

/////////

type ReferenceEntity struct {
}

// UpdateKeys updates entity referencespdates foreign keys based on an EntityMap.
func (ent *MinEntity) UpdateKeys(emap *EntityMap) error { return nil }

/////////
type ErrorEntity struct {
	loadErrors   []error
	loadWarnings []error
}

// AddError adds a loading error to the entity, e.g. from a CSV parse failure
func (ent *MinEntity) AddError(err error) {
	ent.loadErrors = append(ent.loadErrors, err)
}

// AddWarning .
func (ent *MinEntity) AddWarning(err error) {
	ent.loadWarnings = append(ent.loadWarnings, err)
}

// Errors returns validation errors.
func (ent *MinEntity) Errors() []error { return ent.loadErrors }

// Errors returns validation errors.
func (ent *MinEntity) Warnings() []error { return ent.loadWarnings }

// Filename returns the filename for this entity.
func (ent *MinEntity) Filename() string { return "" }

// EntityID returns the entity ID.
func (ent *MinEntity) EntityID() string { return "" }

// UpdateKeys updates entity referencespdates foreign keys based on an EntityMap.
func (ent *MinEntity) UpdateKeys(emap *EntityMap) error { return nil }

/////////////

type FeedVersionEntity struct {
	FeedVersionID int `csv:"-"`
}

// SetFeedVersionID sets the Entity's FeedVersionID.
func (ent *FeedVersionEntity) SetFeedVersionID(fvid int) {
	ent.FeedVersionID = fvid
}

/////////////

type DatabaseEntity struct {
	ID int `csv:"-"`
}

// SetID sets the integer ID.
func (ent *DatabaseEntity) SetID(id int) {
	ent.ID = id
}

// GetID returns the integer ID.
func (ent *DatabaseEntity) GetID() int {
	return ent.ID
}

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}

/////////////

// Timestamps .
type Timestamps struct {
	CreatedAt time.Time `csv:"-"`
	UpdatedAt time.Time `csv:"-"`
}

// UpdateTimestamps initializes or updates CreatedAt / UpdatedAt
func (ent *Timestamps) UpdateTimestamps() {
	t := time.Now().UTC()
	if ent.CreatedAt.IsZero() {
		ent.CreatedAt = t
	}
	ent.UpdatedAt = t
}

/////////////

type BaseEntity struct {
	MinEntity
	DatabaseEntity
	FeedVersionEntity
	Timestamps
}
