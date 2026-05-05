package tlcsv

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// TestSortCSVFiles drives the default-sort path through the typed Writer so
// that sort metadata is captured at header-generation time, exercising the
// real end-to-end flow.
func TestSortCSVFiles(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "sort_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	w, err := NewWriter(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"S2", "S1", "S3"} {
		if _, err := w.AddEntity(&gtfs.Stop{StopID: tt.NewString(s)}); err != nil {
			t.Fatal(err)
		}
	}
	// Multi-digit sequences exercise numeric sort: "10" must sort after "2".
	stopTimes := []struct {
		trip string
		seq  int
	}{
		{"T1", 2}, {"T1", 10}, {"T1", 1}, {"T2", 1}, {"T2", 2},
	}
	for _, st := range stopTimes {
		ent := &gtfs.StopTime{TripID: tt.NewString(st.trip), StopSequence: tt.NewInt(st.seq)}
		if _, err := w.AddEntity(ent); err != nil {
			t.Fatal(err)
		}
	}

	w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: "asc"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	verifyColumn(t, filepath.Join(tmpdir, "stops.txt"), "stop_id", []string{"S1", "S2", "S3"})
	// stop_times sorts by (trip_id asc, stop_sequence asc) with stop_sequence
	// compared as int — so 10 sorts after 2 within trip T1.
	verifyColumns(t, filepath.Join(tmpdir, "stop_times.txt"),
		[]string{"trip_id", "stop_sequence"},
		[][]string{
			{"T1", "1"},
			{"T1", "2"},
			{"T1", "10"},
			{"T2", "1"},
			{"T2", "2"},
		})
}

func TestSortCSVFilesDescending(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "sort_desc_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	w, err := NewWriter(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"S2", "S1", "S3"} {
		if _, err := w.AddEntity(&gtfs.Stop{StopID: tt.NewString(s)}); err != nil {
			t.Fatal(err)
		}
	}
	w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: "desc"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	verifyColumn(t, filepath.Join(tmpdir, "stops.txt"), "stop_id", []string{"S3", "S2", "S1"})
}

// TestSortCSVFilesEmptyNumericLast verifies that for numeric sort columns,
// empty/unparseable cells sort after valid values regardless of direction.
func TestSortCSVFilesEmptyNumericLast(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "sort_empty_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	w, err := NewWriter(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	// Two stop_times have a missing stop_sequence (zero value with !Valid
	// renders as empty in CSV), one has a value.
	stopTimes := []*gtfs.StopTime{
		{TripID: tt.NewString("T1"), StopSequence: tt.NewInt(5)},
		{TripID: tt.NewString("T1")}, // empty stop_sequence
		{TripID: tt.NewString("T1"), StopSequence: tt.NewInt(2)},
		{TripID: tt.NewString("T1")}, // empty stop_sequence
	}
	for _, st := range stopTimes {
		if _, err := w.AddEntity(st); err != nil {
			t.Fatal(err)
		}
	}

	w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: "asc"})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Expect: 2, 5, then the two empties.
	got := readColumn(t, filepath.Join(tmpdir, "stop_times.txt"), "stop_sequence")
	want := []string{"2", "5", "", ""}
	if len(got) != len(want) {
		t.Fatalf("expected %d rows, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: expected %q, got %q (full: %v)", i, want[i], got[i], got)
		}
	}
}

// TestCustomColumnSort exercises the user-supplied StandardizedSortColumns
// override path, which works on any file regardless of whether the writer
// captured metadata for it.
func TestCustomColumnSort(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "custom_sort_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	adapter := NewDirAdapter(tmpdir)

	data := [][]string{
		{"id", "val", "custom"},
		{"1", "A", "3"},
		{"2", "B", "1"},
		{"3", "C", "2"},
	}
	if err := adapter.WriteRows("custom.txt", data); err != nil {
		t.Fatal(err)
	}

	adapter.SetStandardizedSortOptions(adapters.StandardizedSortOptions{
		StandardizedSort:        "asc",
		StandardizedSortColumns: []string{"custom"},
	})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}

	verifyFile(t, filepath.Join(tmpdir, "custom.txt"), [][]string{
		{"id", "val", "custom"},
		{"2", "B", "1"},
		{"3", "C", "2"},
		{"1", "A", "3"},
	})
}

// TestSortCSVFilesSkipsUnknownFiles confirms that a file written via raw
// WriteRows (without going through the typed Writer) is left untouched by
// the sort step when no user override is supplied.
func TestSortCSVFilesSkipsUnknownFiles(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "sort_skip_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	adapter := NewDirAdapter(tmpdir)
	original := [][]string{
		{"id", "val"},
		{"3", "C"},
		{"1", "A"},
		{"2", "B"},
	}
	if err := adapter.WriteRows("unknown.txt", original); err != nil {
		t.Fatal(err)
	}
	adapter.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: "asc"})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	verifyFile(t, filepath.Join(tmpdir, "unknown.txt"), original)
}

// helpers ---

func verifyFile(t *testing.T, path string, expected [][]string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	actual, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(actual) != len(expected) {
		t.Errorf("expected %d rows, got %d for %s", len(expected), len(actual), path)
		return
	}
	for i := range expected {
		for j := range expected[i] {
			if actual[i][j] != expected[i][j] {
				t.Errorf("row %d col %d: expected %s, got %s for %s", i, j, expected[i][j], actual[i][j], path)
			}
		}
	}
}

func readAll(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func readColumn(t *testing.T, path, name string) []string {
	t.Helper()
	rows := readAll(t, path)
	if len(rows) == 0 {
		return nil
	}
	idx := -1
	for i, h := range rows[0] {
		if h == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		t.Fatalf("column %q not found in %s (header: %v)", name, path, rows[0])
	}
	out := make([]string, 0, len(rows)-1)
	for _, r := range rows[1:] {
		if idx < len(r) {
			out = append(out, r[idx])
		} else {
			out = append(out, "")
		}
	}
	return out
}

func verifyColumn(t *testing.T, path, name string, want []string) {
	t.Helper()
	got := readColumn(t, path, name)
	if len(got) != len(want) {
		t.Fatalf("%s: expected %d rows in column %q, got %d (%v)", path, len(want), name, len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s column %q row %d: expected %q, got %q (full: %v)", path, name, i, want[i], got[i], got)
		}
	}
}

func verifyColumns(t *testing.T, path string, names []string, want [][]string) {
	t.Helper()
	rows := readAll(t, path)
	if len(rows) == 0 {
		t.Fatalf("%s: no rows", path)
	}
	header := rows[0]
	idxs := make([]int, len(names))
	for k, n := range names {
		idxs[k] = -1
		for i, h := range header {
			if h == n {
				idxs[k] = i
				break
			}
		}
		if idxs[k] == -1 {
			t.Fatalf("column %q not found in %s", n, path)
		}
	}
	if len(rows)-1 != len(want) {
		t.Fatalf("%s: expected %d data rows, got %d", path, len(want), len(rows)-1)
	}
	for i := range want {
		for k, idx := range idxs {
			if rows[i+1][idx] != want[i][k] {
				t.Errorf("%s row %d col %s: expected %q, got %q", path, i, names[k], want[i][k], rows[i+1][idx])
			}
		}
	}
}
