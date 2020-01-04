package gtcsv

import (
	"encoding/csv"
	"io"
	"strings"

	"github.com/dimchansky/utfbom"
)

// Row is a row value with a header.
type Row struct {
	Row    []string
	Header []string
	Hindex map[string]int
	Line   int
	Err    error
}

// Get a value from the row as a string.
func (row *Row) Get(k string) (string, bool) {
	if i, ok := row.Hindex[k]; ok {
		if len(row.Row) > i {
			return row.Row[i], true
		}
	}
	return "", false
}

// ReadRows iterates through csv rows with callback.
func ReadRows(in io.Reader, cb func(Row)) {
	// Handle byte-order-marks.
	r := csv.NewReader(utfbom.SkipOnly(in))
	// Allow variable columns - very common in GTFS
	r.FieldsPerRecord = -1
	// Trimming is done elsewhere
	r.TrimLeadingSpace = false
	// Reuse record
	r.ReuseRecord = true
	// Allow unescaped quotes
	r.LazyQuotes = true
	// Go for it.
	firstRow, err := r.Read()
	if err != nil {
		return
	}
	// Copy header, since we will reuse the backing array
	header := []string{}
	for _, v := range firstRow {
		header = append(header, v)
	}
	// Map the header to row index
	hindex := map[string]int{}
	for k, i := range header {
		hindex[i] = k
	}
	line := 2 // lines are 1-indexed, plus header
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		// Clear the line if there was a parse error
		if err != nil {
			row = []string{}
		}
		// Remove whitespace
		for i := 0; i < len(row); i++ {
			v := row[i]
			// This is dumb but saves substantial time.
			if len(v) > 0 && (v[0] == ' ' || v[len(v)-1] == ' ') {
				row[i] = strings.TrimSpace(v)
			}
		}
		// Pass parse errors to row
		cb(Row{Row: row, Line: line, Header: header, Hindex: hindex, Err: err})
		line++
	}
}
