package tldb

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tlutil"
)

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
func (writer *Writer) NewReader() (tl.Reader, error) {
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
func (writer *Writer) AddEntity(ent tl.Entity) (string, error) {
	if v, ok := ent.(*tl.Route); ok && v.AgencyID == "" {
		v.AgencyID = strconv.Itoa(writer.defaultAgencyID)
	}
	// Set the FeedVersionID
	if z, ok := ent.(canSetFeedVersion); ok {
		z.SetFeedVersionID(writer.FeedVersionID)
	}
	// Save
	eid, err := writer.Adapter.Insert(ent)
	// Update ID
	if v, ok := ent.(canSetID); ok {
		v.SetID(eid)
	}
	// Set a default AgencyID if possible.
	if _, ok := ent.(*tl.Agency); ok && writer.defaultAgencyID == 0 {
		writer.defaultAgencyID = eid
	}
	return strconv.Itoa(eid), err
}

// AddEntities writes entities to the database.
func (writer *Writer) AddEntities(ents []tl.Entity) ([]string, error) {
	if len(ents) == 0 {
		return []string{}, nil
	}
	eids := []string{}
	ients := make([]interface{}, len(ents))
	for i, ent := range ents {
		// Routes may need a default AgencyID set before writing to database.
		if v, ok := ent.(*tl.Route); ok && v.AgencyID == "" {
			v.AgencyID = strconv.Itoa(writer.defaultAgencyID)
		}
		// Set FeedVersion, Timestamps
		if v, ok := ent.(canSetFeedVersion); ok {
			v.SetFeedVersionID(writer.FeedVersionID)
		}
		ients[i] = ent
	}
	retids, err := writer.Adapter.MultiInsert(ients)
	if err != nil {
		return eids, err
	}
	if len(retids) != len(ients) {
		return []string{}, errors.New("failed to write expected entities")
	}
	for i := 0; i < len(ents); i++ {
		eids = append(eids, strconv.Itoa(retids[i]))
		// Update ID
		if v, ok := ents[i].(canSetID); ok {
			v.SetID(int(retids[i]))
		}
	}
	return eids, nil
}

// CreateFeedVersion creates a new FeedVersion and inserts into the database.
func (writer *Writer) CreateFeedVersion(reader tl.Reader) (int, error) {
	if reader == nil {
		return 0, errors.New("reader required")
	}
	var err error
	feed := dmfr.Feed{}
	feed.FeedID = fmt.Sprintf("%d", time.Now().UnixNano())
	feed.ID, err = writer.Adapter.Insert(&feed)
	if err != nil {
		return 0, err
	}
	fvid := 0
	fv, err := tlutil.NewFeedVersionFromReader(reader)
	if err != nil {
		return 0, err
	}
	fv.FeedID = feed.ID
	fvid, err = writer.Adapter.Insert(&fv)
	writer.FeedVersionID = fvid
	return fvid, err
}
