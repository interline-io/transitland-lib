package tldb

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/tl"
)

// Writer takes a Reader and saves it to a database.
type Writer struct {
	FeedVersionID int
	Adapter       Adapter
}

// NewWriter returns a Writer appropriate for the given connection url.
func NewWriter(dburl string) (*Writer, error) {
	return &Writer{Adapter: newAdapter(dburl)}, nil
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
	// TODO: Special case. Remove and use NullInt.
	if v, ok := ent.(*tl.FareAttribute); ok && v.Transfers == "" {
		v.Transfers = "-1"
	}
	// Set the FeedVersionID
	if z, ok := ent.(canSetFeedVersion); ok {
		z.SetFeedVersionID(writer.FeedVersionID)
	}
	// Update Timestamps
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
	}
	// Save
	eid, err := writer.Adapter.Insert(ent)
	// Update ID
	if v, ok := ent.(canSetID); ok {
		v.SetID(int(eid))
	}
	return strconv.Itoa(eid), err
}

// AddEntities writes entities to the database.
func (writer *Writer) AddEntities(ents []tl.Entity) ([]string, error) {
	eids := []string{}
	ients := make([]interface{}, len(ents))
	for i, ent := range ents {
		if v, ok := ent.(canSetFeedVersion); ok {
			v.SetFeedVersionID(writer.FeedVersionID)
		}
		if v, ok := ent.(canUpdateTimestamps); ok {
			v.UpdateTimestamps()
		}
		ients[i] = ent
	}
	retids, err := writer.Adapter.MultiInsert(ients)
	if err != nil {
		return eids, err
	}
	if len(retids) != len(ients) {
		panic("failed to write expected entities")
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
	feed := tl.Feed{}
	feed.FeedID = fmt.Sprintf("%d", time.Now().UnixNano())
	feed.ID, err = writer.Adapter.Insert(&feed)
	if err != nil {
		return 0, err
	}
	fvid := 0
	fv, err := tl.NewFeedVersionFromReader(reader)
	if err != nil {
		return 0, err
	}
	fv.FeedID = feed.ID
	fvid, err = writer.Adapter.Insert(&fv)
	writer.FeedVersionID = fvid
	return fvid, err
}
