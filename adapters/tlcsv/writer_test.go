package tlcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

// Round trip Writer test.
func TestWriter(t *testing.T) {
	fe, reader := testutil.NewMinimalTestFeed()
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		t.Error(err)
	}
	writer, err := NewWriter(tmpdir)
	if err != nil {
		t.Error(err)
	}
	testutil.TestWriter(t, *fe, func() adapters.Reader { return reader }, func() adapters.Writer { return writer })
	// Clean up and double check
	if err := os.RemoveAll(tmpdir); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(tmpdir); !os.IsNotExist(err) {
		t.Error("did not remove temporary directory!", tmpdir)
	}
}

func TestWriterExtraColumn(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		t.Error(err)
	}
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
	testEnt := tl.Stop{}
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
