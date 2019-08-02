package testutil

import (
	"testing"

	"github.com/interline-io/gotransit"
)

// ReaderTester contains information about the number and types of identities expected in a Reader.
type ReaderTester struct {
	URL       string
	Counts    map[string]int
	EntityIDs map[string][]string
}

// TestReader tests implementations of the Reader interface.
func TestReader(t testing.TB, fe ReaderTester, newReader func() gotransit.Reader) {
	reader := newReader()
	if reader == nil {
		t.Error("no reader")
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	testReader(t, fe, reader)
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
}

// TestWriter tests implementations of the Writer interface.
func TestWriter(t testing.TB, fe ReaderTester, newReader func() gotransit.Reader, newWriter func() gotransit.Writer) {
	// Open writer
	writer := newWriter()
	if writer == nil {
		t.Error("no writer")
	}
	if err := writer.Open(); err != nil {
		t.Error(err)
	}
	if err := writer.Create(); err != nil {
		t.Error(err)
	}
	// Open reader
	reader := newReader()
	if reader == nil {
		t.Error("no reader")
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	// Copy
	if err := DirectCopy(reader, writer); err != nil {
		t.Error(err)
	}
	// Validate
	reader2, err := writer.NewReader()
	if err != nil {
		t.Error(err)
	}
	testReader(t, fe, reader2)
	// Close
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
	if err := writer.Close(); err != nil {
		t.Error(err)
	}
	if err := reader2.Close(); err != nil {
		t.Error(err)
	}
}

// Test checks the Reader against the expected values and records errors.
func testReader(t testing.TB, fe ReaderTester, reader gotransit.Reader) {
	ids := map[string]map[string]int{}
	add := func(ent gotransit.Entity) {
		ent.SetID(0) // TODO: This is a HORRIBLE UGLY HACK :( it sets db ID to zero value to get GTFS ID.
		m, ok := ids[ent.Filename()]
		if !ok {
			m = map[string]int{}
		}
		m[ent.EntityID()]++
		ids[ent.Filename()] = m
	}
	check := func(fn string, gotids map[string]int) {
		s := msisum(gotids)
		if exp, ok := fe.Counts[fn]; ok && s != exp {
			t.Errorf("got %d expected %d", s, exp)
		}
		for _, k := range fe.EntityIDs[fn] {
			if _, ok := gotids[k]; !ok {
				t.Errorf("did not find expected entity %s '%s'", fn, k)
			}
		}
	}
	AllEntities(reader, add)
	for k, v := range ids {
		check(k, v)
		// fmt.Println("checking:", k, "got", msisum(v))
	}
}

func getfn(ent gotransit.Entity) string {
	return ent.Filename()
}

func msisum(m map[string]int) int {
	count := 0
	keys := []string{}
	for k, v := range m {
		keys = append(keys, k)
		count += v
	}
	return count
}
