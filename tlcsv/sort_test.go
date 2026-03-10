package tlcsv

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
)

func TestSortCSVFiles(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "sort_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	adapter := NewDirAdapter(tmpdir)

	// Test stops.txt default sort
	stopsData := [][]string{
		{"stop_id", "stop_name"},
		{"S2", "Stop 2"},
		{"S1", "Stop 1"},
		{"S3", "Stop 3"},
	}
	if err := adapter.WriteRows("stops.txt", stopsData); err != nil {
		t.Fatal(err)
	}

	// Test stop_times.txt default sort (complex key)
	stopTimesData := [][]string{
		{"trip_id", "stop_sequence", "stop_id"},
		{"T1", "2", "S2"},
		{"T1", "1", "S1"},
		{"T2", "1", "S1"},
		{"T2", "2", "S2"},
	}
	if err := adapter.WriteRows("stop_times.txt", stopTimesData); err != nil {
		t.Fatal(err)
	}

	// Apply sort
	adapter.SetStandardizedSortOptions(adapters.StandardizedSortOptions{StandardizedSort: "asc"})
	if err := adapter.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify stops.txt
	verifyFile(t, filepath.Join(tmpdir, "stops.txt"), [][]string{
		{"stop_id", "stop_name"},
		{"S1", "Stop 1"},
		{"S2", "Stop 2"},
		{"S3", "Stop 3"},
	})

	// Verify stop_times.txt
	verifyFile(t, filepath.Join(tmpdir, "stop_times.txt"), [][]string{
		{"trip_id", "stop_sequence", "stop_id"},
		{"T1", "1", "S1"},
		{"T1", "2", "S2"},
		{"T2", "1", "S1"},
		{"T2", "2", "S2"},
	})
}

func TestCustomColumnSort(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "custom_sort_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	adapter := NewDirAdapter(tmpdir)

	// Data with a column we want to sort by
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

func verifyFile(t *testing.T, path string, expected [][]string) {
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
