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

func TestSortCSVFiles(t *testing.T) {
	tmpdir := t.TempDir()
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

	w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: adapters.SortAsc})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	verifyColumns(t, filepath.Join(tmpdir, "stops.txt"),
		[]string{"stop_id"},
		[][]string{{"S1"}, {"S2"}, {"S3"}})
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
	tmpdir := t.TempDir()
	w, err := NewWriter(tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"S2", "S1", "S3"} {
		if _, err := w.AddEntity(&gtfs.Stop{StopID: tt.NewString(s)}); err != nil {
			t.Fatal(err)
		}
	}
	w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: adapters.SortDesc})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	verifyColumns(t, filepath.Join(tmpdir, "stops.txt"),
		[]string{"stop_id"},
		[][]string{{"S3"}, {"S2"}, {"S1"}})
}

// Empty cells in numeric columns rank as +∞: NULLS LAST in asc, NULLS FIRST in desc.
func TestSortCSVFilesEmptyNumericSqlConvention(t *testing.T) {
	stopTimes := func() []*gtfs.StopTime {
		return []*gtfs.StopTime{
			{TripID: tt.NewString("T1"), StopSequence: tt.NewInt(5)},
			{TripID: tt.NewString("T1")}, // empty stop_sequence
			{TripID: tt.NewString("T1"), StopSequence: tt.NewInt(2)},
			{TripID: tt.NewString("T1")}, // empty stop_sequence
		}
	}
	run := func(t *testing.T, direction string, want [][]string) {
		dir := t.TempDir()
		w, err := NewWriter(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, st := range stopTimes() {
			if _, err := w.AddEntity(st); err != nil {
				t.Fatal(err)
			}
		}
		w.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: direction})
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		verifyColumns(t, filepath.Join(dir, "stop_times.txt"), []string{"stop_sequence"}, want)
	}

	t.Run("asc puts empties last", func(t *testing.T) {
		run(t, adapters.SortAsc, [][]string{{"2"}, {"5"}, {""}, {""}})
	})
	t.Run("desc puts empties first", func(t *testing.T) {
		run(t, adapters.SortDesc, [][]string{{""}, {""}, {"5"}, {"2"}})
	})
}

// User-supplied columns work on files without captured entity metadata.
func TestCustomColumnSort(t *testing.T) {
	tmpdir := t.TempDir()
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
		StandardizedSort:        adapters.SortAsc,
		StandardizedSortColumns: []string{"custom"},
	})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	verifyAllRows(t, filepath.Join(tmpdir, "custom.txt"), [][]string{
		{"id", "val", "custom"},
		{"2", "B", "1"},
		{"3", "C", "2"},
		{"1", "A", "3"},
	})
}

// Files written outside the typed Writer path are left alone without an override.
func TestSortCSVFilesSkipsUnknownFiles(t *testing.T) {
	tmpdir := t.TempDir()
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
	adapter.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: adapters.SortAsc})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}
	verifyAllRows(t, filepath.Join(tmpdir, "unknown.txt"), original)
}

// helpers

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

func verifyAllRows(t *testing.T, path string, want [][]string) {
	t.Helper()
	got := readAll(t, path)
	if len(got) != len(want) {
		t.Fatalf("%s: expected %d rows, got %d", path, len(want), len(got))
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Errorf("%s row %d: expected %d cols, got %d", path, i, len(want[i]), len(got[i]))
			continue
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("%s row %d col %d: expected %q, got %q", path, i, j, want[i][j], got[i][j])
			}
		}
	}
}

// verifyColumns checks values for a subset of columns by header name.
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
			cell := ""
			if idx < len(rows[i+1]) {
				cell = rows[i+1][idx]
			}
			if cell != want[i][k] {
				t.Errorf("%s row %d col %s: expected %q, got %q", path, i, names[k], want[i][k], cell)
			}
		}
	}
}
