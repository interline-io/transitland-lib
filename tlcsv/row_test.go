package tlcsv

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadRowsHeader(t *testing.T) {
	tcs := []struct {
		name       string
		data       string
		wantHeader []string
		wantRows   [][]string
		wantErr    error
	}{
		{
			name:       "header and rows",
			data:       "a,b\n1,2\n3,4\n",
			wantHeader: []string{"a", "b"},
			wantRows:   [][]string{{"1", "2"}, {"3", "4"}},
		},
		{
			// The callback must not fire, or every header-only GTFS file would yield
			// a blank entity.
			name:       "header only, no data rows",
			data:       "a,b\n",
			wantHeader: []string{"a", "b"},
		},
		{
			name:       "single row is the header",
			data:       "1\n",
			wantHeader: []string{"1"},
		},
		{
			name:       "ragged rows",
			data:       "a,b,c\n1\n2,3\n",
			wantHeader: []string{"a", "b", "c"},
			wantRows:   [][]string{{"1"}, {"2", "3"}},
		},
		{
			name:    "empty file",
			data:    "",
			wantErr: io.EOF,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var rows [][]string
			header, err := ReadRowsHeader(strings.NewReader(tc.data), func(row Row) {
				// Records are reused between reads, so copy before keeping.
				rows = append(rows, append([]string(nil), row.Row...))
			})
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantHeader, header)
			assert.Equal(t, tc.wantRows, rows)

			// ReadRows wraps ReadRowsHeader and must keep reporting the same rows.
			rows = nil
			err = ReadRows(strings.NewReader(tc.data), func(row Row) {
				rows = append(rows, append([]string(nil), row.Row...))
			})
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantRows, rows)
		})
	}
}
