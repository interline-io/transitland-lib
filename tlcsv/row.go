package tlcsv

import (
	"encoding/csv"
	"io"
	"iter"
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

type csvOptFn func(*csv.Reader)

// ReadRows iterates through csv rows with callback.
func ReadRows(in io.Reader, cb func(Row)) error {
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
	// Go
	return readRows(r, cb)
}

func readRows(r *csv.Reader, cb func(Row)) error {
	// Go for it.
	firstRow, err := r.Read()
	if err != nil {
		return err
	}
	// Copy header, since we will reuse the backing array
	header := []string{}
	for _, v := range firstRow {
		header = append(header, strings.TrimSpace(v))
	}
	// Map the header to row index
	hindex := map[string]int{}
	for k, i := range header {
		hindex[i] = k
	}
	for {
		row, err := r.Read()
		if err == nil {
			// ok
		} else if err == io.EOF {
			break
		} else if _, ok := err.(*csv.ParseError); ok {
			// Parse error: clear row, add error to row
			row = []string{}
		} else {
			// Serious error: break and return with error
			return err
		}
		// Remove whitespace
		for i := 0; i < len(row); i++ {
			v := row[i]
			// This is dumb but saves substantial time.
			if len(v) > 0 && (v[0] == ' ' || v[len(v)-1] == ' ' || v[0] == '\t' || v[len(v)-1] == '\t') {
				row[i] = strings.TrimSpace(v)
			}
		}
		// Pass parse errors to row
		line, _ := r.FieldPos(0)
		cb(Row{Row: row, Line: line, Header: header, Hindex: hindex, Err: err})
	}
	return nil
}

func ReadRowsIter(in io.Reader, optFns ...csvOptFn) iter.Seq2[Row, error] {
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
	// Add additional options
	for _, optFn := range optFns {
		optFn(r)
	}
	return func(yield func(Row, error) bool) {
		// Go for it.
		firstRow, err := r.Read()
		if err != nil {
			yield(Row{}, err)
			return
		}
		// Copy header, since we will reuse the backing array
		header := []string{}
		for _, v := range firstRow {
			header = append(header, strings.TrimSpace(v))
		}
		// Map the header to row index
		hindex := map[string]int{}
		for k, i := range header {
			hindex[i] = k
		}
		for {
			row, err := r.Read()
			if err == nil {
				// ok
			} else if err == io.EOF {
				break
			} else if _, ok := err.(*csv.ParseError); ok {
				// Parse error: clear row, add error to row
				row = []string{}
			} else {
				// Serious error: break and return with error
				yield(Row{}, err)
				return
			}
			// Remove whitespace
			for i := 0; i < len(row); i++ {
				v := row[i]
				// This is dumb but saves substantial time.
				if len(v) > 0 && (v[0] == ' ' || v[len(v)-1] == ' ' || v[0] == '\t' || v[len(v)-1] == '\t') {
					row[i] = strings.TrimSpace(v)
				}
			}
			// Pass parse errors to row
			line, _ := r.FieldPos(0)
			cbrow := Row{Row: row, Line: line, Header: header, Hindex: hindex, Err: err}
			if !yield(cbrow, nil) {
				return
			}
		}
	}
}
