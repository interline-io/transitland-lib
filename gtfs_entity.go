package tl

import (
	"strconv"
	"time"
)

// EntityError is an interface for GTFS Errors
type EntityError interface {
	Error() string
}

// EntityFilter provides an interface for modifying an entity, e.g. before writing
type EntityFilter interface {
	Filter(Entity, *EntityMap) error
}

// Entity provides an interface for GTFS entities.
type Entity interface {
	EntityID() string
	Filename() string
	Errors() []error
	Warnings() []error
	AddError(error)
	AddWarning(error)
	SetID(int)
	SetExtra(string, string)
	Extra() map[string]string
	UpdateKeys(*EntityMap) error
}

// BaseEntity provides default methods.
type BaseEntity struct {
	Timestamps
	ID            int
	FeedVersionID int
	extra         []string
	loadErrors    []error
	loadWarnings  []error
	// DeletedAt     OptionalTime
}

// SetID sets the integer ID.
func (ent *BaseEntity) SetID(id int) {
	ent.ID = id
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
	if ent.extra == nil {
		ent.extra = []string{}
	}
	ent.extra = append(ent.extra, key, value)
}

// AddError adds a loading error to the entity, e.g. from a CSV parse failure
func (ent *BaseEntity) AddError(err error) {
	if ent.loadErrors == nil {
		ent.loadErrors = []error{}
	}
	ent.loadErrors = append(ent.loadErrors, err)
}

// AddWarning .
func (ent *BaseEntity) AddWarning(err error) {
	if ent.loadWarnings == nil {
		ent.loadWarnings = []error{}
	}
	ent.loadWarnings = append(ent.loadErrors, err)
}

// Warnings returns validation warnings.
func (ent *BaseEntity) Warnings() []error { return ent.loadWarnings }

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
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UpdateTimestamps initializes or updates CreatedAt / UpdatedAt
func (ent *Timestamps) UpdateTimestamps() {
	t := time.Now()
	if ent.CreatedAt.IsZero() {
		ent.CreatedAt = t
	}
	ent.UpdatedAt = t
}
