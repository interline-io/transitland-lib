package tlcsv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// Round trip Writer test.
func TestWriter(t *testing.T) {
	fe, reader := testreader.NewMinimalTestFeed()
	tmpdir, err := os.MkdirTemp("", "gtfs")
	if err != nil {
		t.Error(err)
	}
	writer, err := NewWriter(tmpdir)
	if err != nil {
		t.Error(err)
	}
	testreader.TestWriter(t, *fe, func() adapters.Reader { return reader }, func() adapters.Writer { return writer })
	// Clean up and double check
	if err := os.RemoveAll(tmpdir); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(tmpdir); !os.IsNotExist(err) {
		t.Error("did not remove temporary directory!", tmpdir)
	}
}

func TestWriterExtraColumn(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "gtfs")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(tmpdir)
	writer, err := NewWriter(tmpdir)
	writer.WriteExtraColumns(true)
	if err != nil {
		t.Error(err)
	}
	if err := writer.Open(); err != nil {
		t.Error(err)
	}
	if err := writer.Create(); err != nil {
		t.Error(err)
	}
	testEnt := gtfs.Stop{}
	// test ordering on output
	extraVals := []string{
		"ok", "hello",
		"foo", "bar",
		"abc", "123",
		"", "", // ignored
		"z", "",
	}
	for i := 0; i < len(extraVals); i += 2 {
		testEnt.SetExtra(extraVals[i], extraVals[i+1])
	}
	if _, err := writer.AddEntity(&testEnt); err != nil {
		t.Fatal(err)
	}
	reader, err := NewReader(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for ent := range reader.Stops() {
		a, _ := ent.GetExtra("ok")
		assert.Equal(t, "hello", a)
		b, _ := ent.GetExtra("foo")
		assert.Equal(t, "bar", b)
		for i := 0; i < len(extraVals); i += 2 {
			c, _ := ent.GetExtra(extraVals[i])
			assert.Equal(t, extraVals[i+1], c)
		}
		d, e := ent.GetExtra("")
		assert.Equal(t, "", d)
		assert.Equal(t, false, e)
		found = true
	}
	if !found {
		t.Error("expected to get a stop with extra columns")
	}
}

// The GTFS field is "reversed_signposted_as"; a previous refactor derived the column
// name from the Go field name and silently wrote (and read) "reverse_signposted_as",
// which spec-compliant consumers ignore. See gtfs/pathway.go.
func TestWriterPathwayReversedSignpostedAs(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "gtfs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	writer, err := NewWriter(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Open(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Create(); err != nil {
		t.Fatal(err)
	}
	testEnt := gtfs.Pathway{
		PathwayID:           tt.NewString("pathway1"),
		FromStopID:          tt.NewString("stop1"),
		ToStopID:            tt.NewString("stop2"),
		PathwayMode:         tt.NewInt(1),
		IsBidirectional:     tt.NewInt(1),
		SignpostedAs:        tt.NewString("To Platform 1"),
		ReverseSignpostedAs: tt.NewString("To Exit"),
	}
	if _, err := writer.AddEntity(&testEnt); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	// Written header must use the spec field name.
	data, err := os.ReadFile(filepath.Join(tmpdir, "pathways.txt"))
	if err != nil {
		t.Fatal(err)
	}
	header, _, _ := strings.Cut(string(data), "\n")
	cols := strings.Split(strings.TrimSpace(header), ",")
	assert.Contains(t, cols, "reversed_signposted_as")
	assert.NotContains(t, cols, "reverse_signposted_as")

	// And a feed using the spec field name must round trip.
	reader, err := NewReader(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for ent := range reader.Pathways() {
		assert.Equal(t, "To Platform 1", ent.SignpostedAs.Val)
		assert.Equal(t, "To Exit", ent.ReverseSignpostedAs.Val)
		found = true
	}
	assert.True(t, found, "expected to read back a pathway")
}
