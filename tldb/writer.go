package tldb

import (
	"context"
	"errors"
	"strconv"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

var notNullFilter = &NotNullFilter{}

// OpenWriter opens & creates a db writer
func OpenWriter(dburl string, create bool) (*Writer, error) {
	// Writer
	writer, err := NewWriter(dburl)
	if err != nil {
		return nil, err
	}
	if err := writer.Open(); err != nil {
		return nil, err
	}
	if create {
		if err := writer.Create(); err != nil {
			return nil, err
		}
	}
	return writer, nil
}

// Writer takes a Reader and saves it to a database.
type Writer struct {
	FeedVersionID   int
	Adapter         Adapter
	defaultAgencyID int // required for routes
}

// NewWriter returns a Writer appropriate for the given connection url.
func NewWriter(dburl string) (*Writer, error) {
	fvids, newurl, err := getFvids(dburl)
	if err != nil {
		return nil, err
	}
	adapter := newAdapter(newurl)
	if adapter == nil {
		return nil, errors.New("no adapter")
	}
	writer := &Writer{Adapter: adapter}
	if len(fvids) > 0 {
		writer.FeedVersionID = fvids[0]
	}
	return writer, nil
}

func (writer *Writer) String() string {
	return "db"
}

// Open the database.
func (writer *Writer) Open() error {
	return writer.Adapter.Open()
}

// Close the database.
func (writer *Writer) Close() error {
	if writer.Adapter == nil {
		return nil
	}
	return writer.Adapter.Close()
}

// NewReader returns a new Reader with the same adapter.
func (writer *Writer) NewReader() (adapters.Reader, error) {
	reader := Reader{
		FeedVersionIDs: []int{writer.FeedVersionID},
		Adapter:        writer.Adapter,
		PageSize:       1000,
	}
	return &reader, nil
}

// Create the database.
func (writer *Writer) Create() error {
	return writer.Adapter.Create()
}

// Delete any entities associated with the FeedVersion.
func (writer *Writer) Delete() error {
	// TODO: Move to extension Delete() ?
	return nil
}

// AddEntity writes an entity to the database.
func (writer *Writer) AddEntity(ent tt.Entity) (string, error) {
	eids, err := writer.AddEntities([]tt.Entity{ent})
	if err != nil {
		return "", err
	}
	if len(eids) == 0 {
		return "", errors.New("did not write expected number of entities")
	}
	return eids[0], nil
}

// AddEntities writes entities to the database.
func (writer *Writer) AddEntities(ents []tt.Entity) ([]string, error) {
	ctx := context.TODO()
	if len(ents) == 0 {
		return []string{}, nil
	}
	eids := []string{}
	ients := make([]interface{}, len(ents))
	for i, ent := range ents {
		// Set some fields to not null for compatibility
		notNullFilter.Filter(ent, nil)

		// Routes may need a default AgencyID set before writing to database.
		if v, ok := ent.(*gtfs.Route); ok && !v.AgencyID.Valid {
			v.AgencyID.SetInt(writer.defaultAgencyID)
		}
		// Set FeedVersion, Timestamps
		if v, ok := ent.(CanSetFeedVersion); ok {
			v.SetFeedVersionID(writer.FeedVersionID)
		}
		ients[i] = ent
	}
	retids, err := writer.Adapter.MultiInsert(ctx, ients)
	if err != nil {
		return eids, err
	}
	if len(retids) != len(ients) {
		return []string{}, errors.New("failed to write expected entities")
	}
	for i := 0; i < len(ents); i++ {
		eids = append(eids, strconv.Itoa(retids[i]))
		// Update ID
		if v, ok := ents[i].(CanSetID); ok {
			v.SetID(int(retids[i]))
		}
	}
	// Set default agency ID
	// TODO: handle this in ApplyDefaultAgencyFilter
	for _, ent := range ents {
		if a, ok := ent.(*gtfs.Agency); ok {
			if writer.defaultAgencyID == 0 {
				writer.defaultAgencyID = a.ID
			}
			break
		}
	}
	return eids, nil
}
