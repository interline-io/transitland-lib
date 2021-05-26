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
	AddError(error)
	SetExtra(string, string)
	Extra() map[string]string
	UpdateKeys(*EntityMap) error
}

// BaseEntity provides default methods.
type BaseEntity struct {
	Timestamps
	ID            int `csv:"-"`
	FeedVersionID int `csv:"-"`
	extra         []string
	loadErrors    []error
	loadWarnings  []error
	// DeletedAt     OTime
}

// SetID sets the integer ID.
func (ent *BaseEntity) SetID(id int) {
	ent.ID = id
}

// GetID returns the integer ID.
func (ent *BaseEntity) GetID() int {
	return ent.ID
}

// SetFeedVersionID sets the Entity's FeedVersionID.
func (ent *BaseEntity) SetFeedVersionID(fvid int) {
	ent.FeedVersionID = fvid
}

// Extra provides any additional fields that were present.
func (ent *BaseEntity) Extra() map[string]string {
	ret := map[string]string{}
	for i := 0; i < len(ent.extra); i += 2 {
		ret[ent.extra[i]] = ent.extra[i+1]
	}
	return ret
}

// SetExtra adds a string key, value pair to the entity's extra fields.
func (ent *BaseEntity) SetExtra(key string, value string) {
	ent.extra = append(ent.extra, key, value)
}

// AddError adds a loading error to the entity, e.g. from a CSV parse failure
func (ent *BaseEntity) AddError(err error) {
	ent.loadErrors = append(ent.loadErrors, err)
}

// AddWarning .
func (ent *BaseEntity) AddWarning(err error) {
	ent.loadWarnings = append(ent.loadErrors, err)
}

// Errors returns validation errors.
func (ent *BaseEntity) Errors() []error { return ent.loadErrors }

// Filename returns the filename for this entity.
func (ent *BaseEntity) Filename() string { return "" }

// EntityID returns the entity ID.
func (ent *BaseEntity) EntityID() string { return "" }

// UpdateKeys updates entity referencespdates foreign keys based on an EntityMap.
func (ent *BaseEntity) UpdateKeys(emap *EntityMap) error { return nil }

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}

// Timestamps .
type Timestamps struct {
	CreatedAt time.Time `csv:"-"`
	UpdatedAt time.Time `csv:"-"`
}

// UpdateTimestamps initializes or updates CreatedAt / UpdatedAt
func (ent *Timestamps) UpdateTimestamps() {
	t := time.Now()
	if ent.CreatedAt.IsZero() {
		ent.CreatedAt = t
	}
	ent.UpdatedAt = t
}
