package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
)

// WriterTesterRoundTrip reads the example feed into the writer, then checks if it was all written
func WriterTesterRoundTrip(reader gotransit.Reader, writer gotransit.Writer, t *testing.T) {
	cp := copier.NewCopier(reader, writer)
	cp.Copy()

	cpreader, err := writer.NewReader()
	if err != nil {
		t.Error(err)
		return
	}
	cpreader.Open()
	defer cpreader.Close()
	ReaderTester(cpreader, t)
}

// WriterTester checks implementations of Writer interface against the example feed.
func WriterTester(writer gotransit.Writer, t *testing.T) {
	writer.Delete()
	defer writer.Delete()
	t.Run("AddAgency", func(t *testing.T) { writerTesterAddAgency(writer, t) })
	t.Run("AddStop", func(t *testing.T) { writerTesterAddStop(writer, t) })
}

func writerTesterAddAgency(writer gotransit.Writer, t *testing.T) {
	ent := gotransit.Agency{
		AgencyID:   "test",
		AgencyName: "Test Agency",
	}
	eid, err := writer.AddEntity(&ent)
	if err != nil {
		t.Error(err)
	}
	if len(eid) == 0 {
		t.Error("no id assigned")
	}
}

func writerTesterAddStop(writer gotransit.Writer, t *testing.T) {
	stop := gotransit.Stop{
		StopID:   "test",
		StopName: "test",
	}
	stop.Geometry = gotransit.NewPoint(-73.898583, 40.889248)
	eid, err := writer.AddEntity(&stop)
	if err != nil {
		t.Error(err)
	}
	if len(eid) == 0 {
		t.Error("no id assigned")
	}
}
