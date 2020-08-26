package gtdb

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/interline-io/gotransit"
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
func (writer *Writer) NewReader() (gotransit.Reader, error) {
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
func (writer *Writer) AddEntity(ent gotransit.Entity) (string, error) {
	// Set the FeedVersionID
	if z, ok := ent.(canSetFeedVersion); ok {
		z.SetFeedVersionID(writer.FeedVersionID)
	}
	// Save
	eid, err := writer.Adapter.Insert(ent)
	return strconv.Itoa(eid), err
}

// AddEntities provides a generic interface for adding Entities to the database.
func (writer *Writer) AddEntities(ents []gotransit.Entity) error {
	ients := make([]interface{}, 0)
	for _, ent := range ents {
		if z, ok := ent.(canSetFeedVersion); ok {
			z.SetFeedVersionID(writer.FeedVersionID)
		}
		ients = append(ients, ent)
	}
	return writer.Adapter.BatchInsert(ients)
}

// CreateFeedVersion creates a new Feed Version and inserts into the database.
func (writer *Writer) CreateFeedVersion(reader gotransit.Reader) (int, error) {
	if reader == nil {
		return 0, errors.New("reader required")
	}
	var err error
	feed := gotransit.Feed{}
	feed.FeedID = fmt.Sprintf("%d", time.Now().UnixNano())
	feed.ID, err = writer.Adapter.Insert(&feed)
	if err != nil {
		return 0, err
	}
	fvid := 0
	fv, err := gotransit.NewFeedVersionFromReader(reader)
	if err != nil {
		return 0, err
	}
	fv.FeedID = feed.ID
	fvid, err = writer.Adapter.Insert(&fv)
	writer.FeedVersionID = fvid
	return fvid, err
}
