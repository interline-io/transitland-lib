package validator

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestSaveValidationReport(t *testing.T) {
	reader, err := tlcsv.NewReader(testutil.RelPath("test/data/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(testutil.RelPath(filepath.Join("test/data/rt", r.URL.Path)))
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))

	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 05, 0, 0, tz)
	opts := Options{
		SaveStaticValidationReport:   true,
		SaveRealtimeValidationReport: true,
		IncludeRealtimeJson:          true,
		EvaluateAt:                   now,
		ValidateRealtimeMessages: []string{
			ts.URL + "/ct-trip-updates.pb.json",
			ts.URL + "/ct-vehicle-positions.pb.json",
		},
	}

	v, _ := NewValidator(reader, opts)
	result, err := v.Validate()
	if err != nil {
		t.Fatal(err)
	}
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		if err := SaveValidationReport(atx, result, now, 1, true, true); err != nil {
			t.Fatal(err)
		}
		return nil
	})
}

// func TestContinuousValidation(t *testing.T) {
// 	dburl := os.Getenv("TL_TEST_DATABASE_URL")
// 	if dburl == "" {
// 		t.Skip("TL_TEST_DATABASE_URL is not set")
// 		return
// 	}
// 	writer := testdb.MustOpenWriter(dburl, false)
// 	atx := writer.Adapter

// 	rtDir := "/Users/irees/tmp" // testutil.RelPath("test/data/rt")
// 	reader, err := tlcsv.NewReader(filepath.Join(rtDir, "ct.zip"))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		buf, err := os.ReadFile(filepath.Join(rtDir, r.URL.Path))
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		w.Write(buf)
// 	}))

// 	// Prepare options
// 	tz, _ := time.LoadLocation("America/Los_Angeles")
// 	// now := time.Date(2023, 11, 7, 17, 05, 0, 0, tz)
// 	opts := Options{
// 		SaveStaticValidationReport:   true,
// 		SaveRealtimeValidationReport: true,
// 	}

// 	// Run
// 	now := time.Now()
// 	// Initial validation
// 	v, _ := NewValidator(reader, opts)
// 	result, err := v.Validate()
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if err := SaveValidationReport(atx, result, now, 1, true, true); err != nil {
// 		t.Fatal(err)
// 	}

// 	// Test continuous validation
// 	fns, err := getFiles("/Users/irees/tmp")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	for _, fn := range fns {
// 		subFn := ts.URL + strings.Replace(fn, rtDir, "", 1)
// 		fmt.Println(subFn)
// 		rtTest, err := rt.ReadURL(subFn)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		now = time.Unix(int64(*rtTest.Header.Timestamp), 0).In(tz)
// 		rtResult, err := v.ValidateRT(subFn, now)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 		// jr, _ := json.Marshal(rtResult)
// 		// fmt.Println(string(jr))
// 		result2 := Result{}
// 		result2.Realtime = append(result2.Realtime, rtResult)
// 		if err := SaveValidationReport(atx, &result2, now, 1, true, true); err != nil {
// 			t.Fatal(err)
// 		}

// 	}
// }

func getFiles(path string) ([]string, error) {
	files := []string{}
	if err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), ".pb") {
				files = append(files, path)
			}
			return nil
		}); err != nil {
		return nil, err
	}
	return files, nil
}
