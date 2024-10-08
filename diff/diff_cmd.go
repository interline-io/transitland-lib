// Package diff provides tools and utilities for comparing GTFS feeds.
package diff

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/spf13/pflag"
)

type Command struct {
	Outpath     string
	RawDiff     bool
	ShowDiff    bool
	ShowSame    bool
	ShowAdded   bool
	ShowDeleted bool
	CheckFiles  []string
	readerPathA string
	readerPathB string
}

func (cmd *Command) HelpDesc() (string, string) {
	a := "Calculate difference between two feeds, writing output in a GTFS-like format"
	b := "This command is experimental; it may provide incorrect results or crash on large feeds."
	return a, b
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <feed1> <feed2> <output>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.ShowSame, "same", false, "Show entities present in both files and identical")
	fl.BoolVar(&cmd.ShowDiff, "diff", false, "Show entities present in both files but different")
	fl.BoolVar(&cmd.ShowAdded, "added", false, "Show entities added in second file")
	fl.BoolVar(&cmd.ShowDeleted, "deleted", false, "Show entities deleted from first file")
	fl.BoolVar(&cmd.RawDiff, "raw", false, "Diff based on raw CSV contents")
}

func (cmd *Command) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires two input readers")
	}
	if fl.NArg() < 3 {
		return errors.New("requires output directory")
	}
	if !cmd.ShowAdded && !cmd.ShowDeleted && !cmd.ShowSame && !cmd.ShowDiff {
		log.Print("Using default mode of -same -diff -added -deleted")
		cmd.ShowAdded = true
		cmd.ShowDeleted = true
		cmd.ShowSame = true
		cmd.ShowDiff = true
	}
	cmd.readerPathA = fl.Arg(0)
	cmd.readerPathB = fl.Arg(1)
	cmd.Outpath = fl.Arg(2)
	return nil
}

func (cmd *Command) Run() error {
	readerA, err := tlcsv.NewReader(cmd.readerPathA)
	if err != nil {
		return err
	}
	if err := readerA.Open(); err != nil {
		return err
	}
	readerB, err := tlcsv.NewReader(cmd.readerPathB)
	if err != nil {
		return err
	}
	if err := readerB.Open(); err != nil {
		return err
	}
	var df1 *diffAdapter
	var df2 *diffAdapter
	if cmd.RawDiff {
		var err error
		df1, err = checkDiffRaw(readerA, cmd.CheckFiles)
		if err != nil {
			return err
		}
		df2, err = checkDiffRaw(readerB, cmd.CheckFiles)
		if err != nil {
			return err
		}
	} else {
		var err error
		df1, err = checkDiff(readerA)
		if err != nil {
			return err
		}
		df2, err = checkDiff(readerB)
		if err != nil {
			return err
		}
	}
	combinedFiles := map[string]bool{}
	for k := range df1.headers {
		combinedFiles[k] = true
	}
	for k := range df2.headers {
		combinedFiles[k] = true
	}
	files := []string{}
	for k := range combinedFiles {
		files = append(files, k)
	}
	sort.Strings(files)
	outWriter := tlcsv.NewDirAdapter(cmd.Outpath)
	if err := outWriter.Open(); err != nil {
		return err
	}
	defer outWriter.Close()
	for _, fn := range files {
		// Compare
		combinedKeys := map[string]bool{}
		for k := range df1.ents {
			if k.efn == fn {
				combinedKeys[k.eid] = true
			}
		}
		for k := range df2.ents {
			if k.efn == fn {
				combinedKeys[k.eid] = true
			}
		}
		presentBoth := []diffEnt{}
		presentDiff := []diffEnt{}
		deletedRows := []diffEnt{}
		addedRows := []diffEnt{}
		for k := range combinedKeys {
			ent1, ok1 := df1.ents[diffKey{fn, k}]
			ent2, ok2 := df2.ents[diffKey{fn, k}]
			// log.Traceln("========", fn, "key:", k)
			// log.Traceln("ent1:", ent1)
			// log.Traceln("ent2:", ent2)
			hh1 := hashRow(ent1.row)
			hh2 := hashRow(ent2.row)
			if ok1 && ok2 {
				if hh1 == hh2 && cmd.ShowSame {
					// log.Traceln("same")
					ent1.row = append(ent1.row, readerB.String(), "same")
					presentBoth = append(presentBoth, ent1)
				} else if hh1 != hh2 && cmd.ShowDiff {
					// log.Traceln("diff")
					ent1.row = append(ent1.row, readerA.String(), "diff")
					ent2.row = append(ent2.row, readerB.String(), "diff")
					presentDiff = append(presentDiff, ent1, ent2)
				}
			} else if ok1 && !ok2 && cmd.ShowDeleted {
				// log.Traceln("deleted")
				ent1.row = append(ent1.row, readerA.String(), "deleted")
				deletedRows = append(deletedRows, ent1)
			} else if ok2 && !ok1 && cmd.ShowAdded {
				// log.Traceln("added")
				ent2.row = append(ent2.row, readerB.String(), "added")
				addedRows = append(addedRows, ent2)
			}
		}
		// Write
		if len(presentBoth) == 0 && len(presentDiff) == 0 && len(addedRows) == 0 && len(deletedRows) == 0 {
			continue
		}
		header := []string{}
		h1 := df1.headers[fn]
		h2 := df2.headers[fn]
		if len(h1.row) == 0 && len(h2.row) > 0 {
			h1 = h2
		}
		if len(h2.row) == 0 && len(h1.row) > 0 {
			h2 = h1
		}
		if hashRow(h1.row) != hashRow(h2.row) {
			log.Traceln("headers are different:")
			log.Traceln(h1.row)
			log.Traceln(h2.row)
			continue
		}
		header = append(header, h1.row...)
		header = append(header, "diff_filename", "diff_status")
		outWriter.WriteRows(fn, [][]string{header})
		for _, row := range presentBoth {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range presentDiff {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range deletedRows {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range addedRows {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
	}
	return nil
}

type canFileInfos interface {
	tlcsv.Adapter
	FileInfos() ([]os.FileInfo, error)
}

func checkDiffRaw(reader tl.Reader, checkFiles []string) (*diffAdapter, error) {
	v, ok := reader.(*tlcsv.Reader)
	if !ok {
		return nil, errors.New("must be csv input")
	}
	var fis []os.FileInfo
	if afi, ok := v.Adapter.(canFileInfos); ok {
		fis, _ = afi.FileInfos()
	} else {
		return nil, errors.New("reader does not support file infos")
	}
	df := newDiffAdapter()
	if err := df.Open(); err != nil {
		return nil, err
	}
	defer df.Close()

	cfMap := map[string]bool{}
	for _, k := range checkFiles {
		cfMap[k] = true
	}

	for _, fi := range fis {
		// Only compare files with lowercase names that end with .txt
		if fi.Name() != strings.ToLower(fi.Name()) || !strings.HasSuffix(fi.Name(), ".txt") {
			continue
		}
		if len(cfMap) > 0 && !cfMap[fi.Name()] {
			continue
		}
		header := false
		v.Adapter.ReadRows(fi.Name(), func(row tlcsv.Row) {
			if !header {
				df.WriteRows(fi.Name(), [][]string{row.Header})
				header = true
			}
			// log.Traceln(fi.Name(), row.Row)
			var row2 []string
			row2 = append(row2, row.Row...)
			df.WriteRows(fi.Name(), [][]string{row2})
		})
	}
	return df, nil
}

func checkDiff(reader tl.Reader) (*diffAdapter, error) {
	df := newDiffAdapter()
	writer, err := tlcsv.NewWriter("")
	if err != nil {
		return nil, err
	}
	writer.WriterAdapter = df
	copier, err := copier.NewCopier(reader, writer, copier.Options{
		AllowEntityErrors:    true,
		AllowReferenceErrors: true,
	})
	if err != nil {
		return nil, err
	}
	copier.Copy()
	return df, nil
}

type diffAdapter struct {
	headers    map[string]diffEnt
	ents       map[diffKey]diffEnt
	entityKeys map[string]string
}

func newDiffAdapter() *diffAdapter {
	return &diffAdapter{
		headers: map[string]diffEnt{},
		ents:    map[diffKey]diffEnt{},
		entityKeys: map[string]string{
			"agency.txt":          "agency_id",
			"stops.txt":           "stop_id",
			"routes.txt":          "route_id",
			"calendar.txt":        "service_id",
			"trips.txt":           "trip_id",
			"pathways.txt":        "pathway_id",
			"fare_attributes.txt": "fare_id",
			"levels.txt":          "level_id",
		},
	}
}

func (adapter *diffAdapter) String() string                         { return "diff" }
func (adapter *diffAdapter) OpenFile(string, func(io.Reader)) error { return nil }
func (adapter *diffAdapter) ReadRows(string, func(tlcsv.Row)) error { return nil }
func (adapter *diffAdapter) Open() error                            { return nil }
func (adapter *diffAdapter) Close() error                           { return nil }
func (adapter *diffAdapter) Exists() bool                           { return false }
func (adapter *diffAdapter) Path() string                           { return "" }
func (adapter *diffAdapter) SHA1() (string, error)                  { return "", nil }
func (adapter *diffAdapter) DirSHA1() (string, error)               { return "", nil }

func (adapter *diffAdapter) WriteRows(efn string, rows [][]string) error {
	headerIndex := -1
	if hkey, ok := adapter.entityKeys[efn]; ok {
		if header, ok := adapter.headers[efn]; ok {
			for hki, hk := range header.row {
				if hk == hkey {
					headerIndex = hki
				}
			}
		}
	}
	for _, row := range rows {
		if _, ok := adapter.headers[efn]; !ok {
			adapter.headers[efn] = diffEnt{row: row}
		} else {
			k := ""
			if headerIndex >= 0 {
				k = row[headerIndex]
			} else {
				k = hashRow(row)
			}
			key := diffKey{efn, k}
			adapter.ents[key] = diffEnt{row: row}
		}
	}
	return nil
}

type diffKey struct {
	efn string
	eid string
}

type diffEnt struct {
	row []string
}

func hashRow(row []string) string {
	h := sha1.New()
	for _, c := range row {
		h.Write([]byte(c))
	}
	return hex.EncodeToString(h.Sum(nil))
}
