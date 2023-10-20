package dmfr

import (
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dimchansky/utfbom"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// FeedVersionFileInfo .
type FeedVersionFileInfo struct {
	ID            int
	FeedVersionID int
	Name          string
	Size          int64
	Rows          int64
	Columns       int
	Header        string
	CSVLike       bool
	SHA1          string
	ValuesUnique  tt.Counts
	ValuesCount   tt.Counts
	tl.Timestamps
}

// EntityID .
func (fvi *FeedVersionFileInfo) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionFileInfo) TableName() string {
	return "feed_version_file_infos"
}

type canFileInfo interface {
	FileInfos() ([]os.FileInfo, error)
	OpenFile(string, func(io.Reader)) error
	ReadRows(string, func(tlcsv.Row)) error
}

// NewFeedVersionFileInfosFromReader calculates statistics about the contents of a feed version
func NewFeedVersionFileInfosFromReader(reader *tlcsv.Reader) ([]FeedVersionFileInfo, error) {
	ret := []FeedVersionFileInfo{}
	adapter, ok := reader.Adapter.(canFileInfo)
	if !ok {
		return ret, nil // errors.New("adapter does not support FileInfo")
	}
	fis, err := adapter.FileInfos()
	if err != nil {
		return ret, err
	}
	for _, fi := range fis {
		// Only generate stats for files with lowercase names that end with .txt
		if fi.Name() != strings.ToLower(fi.Name()) || !strings.HasSuffix(fi.Name(), ".txt") {
			continue
		}

		// First pass to generate SHA1
		h := sha1.New()
		adapter.OpenFile(fi.Name(), func(r io.Reader) {
			io.Copy(h, r)
		})

		// Prepare result
		fvfi := FeedVersionFileInfo{}
		fvfi.Name = fi.Name()
		fvfi.Size = fi.Size()
		fvfi.SHA1 = fmt.Sprintf("%x", h.Sum(nil))
		fvfi.CSVLike = true

		// Check if file has a csv-like header
		header := []string{}
		adapter.OpenFile(fi.Name(), func(r io.Reader) {
			csvr := csv.NewReader(utfbom.SkipOnly(r))
			csvr.TrimLeadingSpace = false
			if csvh, err := csvr.Read(); err == nil {
				hsize := 0
				for _, v := range csvh {
					t := strings.TrimSpace(v)
					hsize += len(t)
					if !utf8.ValidString(t) {
						fvfi.CSVLike = false
					} else if len(t) > 100 {
						// max column name size
						fvfi.CSVLike = false
					} else if hsize > 1000 {
						// max header row bytes size
						fvfi.CSVLike = false
					} else if strings.Contains(t, " ") {
						// no spaces in column names
						fvfi.CSVLike = false
					} else {
						header = append(header, t)
					}
				}
			}
			// Minimum 2 and maximum 100 columns for stats generation
			if len(header) < 2 || len(header) > 100 {
				fvfi.CSVLike = false
			}
		})

		// Generate statistics if file is csv-like
		if fvfi.CSVLike {
			valuesCount := tt.Counts{}
			valuesUnique := map[string]map[string]struct{}{}
			rows := int64(0)
			adapter.ReadRows(fi.Name(), func(row tlcsv.Row) {
				rows++
				for i, v := range row.Row {
					// Skip empty values
					if i >= len(row.Header) || v == "" {
						continue
					}
					k := row.Header[i]
					valuesCount[k] += 1
					vc, ok := valuesUnique[k]
					if !ok {
						vc = map[string]struct{}{}
						valuesUnique[k] = vc
					}
					// Store at most 1 million unique values
					if len(vc) >= 1_000_000 {
						continue
					}
					// Hash value if over 100 bytes
					if len(v) > 100 {
						vh := sha1.New()
						vh.Write([]byte(v))
						v = fmt.Sprintf("%x", string(vh.Sum(nil)))
					}
					vc[v] = struct{}{}
				}
			})
			fvfi.Rows = rows
			fvfi.Header = strings.Join(header, ",")
			fvfi.ValuesCount = valuesCount
			fvfi.ValuesUnique = tt.Counts{}
			for k, v := range valuesUnique {
				fvfi.ValuesUnique[k] = len(v)
			}
		}
		ret = append(ret, fvfi)
	}
	return ret, nil
}
