package diff

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"sort"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

type Command struct {
	Outpath     string
	ShowDiff    bool
	ShowSame    bool
	ShowMissing bool
	readerPathA string
	readerPathB string
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("diff", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: diff <input1> <input2> <output>")
		fl.PrintDefaults()
	}
	fl.BoolVar(&cmd.ShowSame, "same", false, "Show entities present in both files and identical")
	fl.BoolVar(&cmd.ShowDiff, "diff", false, "Show entities present in both files but different")
	fl.BoolVar(&cmd.ShowMissing, "missing", false, "Show entities present in one file but not the other")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		log.Exit("Requires two input readers")
	}
	if fl.NArg() < 3 {
		fl.Usage()
		log.Exit("Requires output directory")
	}
	if !cmd.ShowDiff && !cmd.ShowSame && !cmd.ShowMissing {
		fl.Usage()
		log.Exit("You must use at least one of: -same -diff -missing")
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
	readerB, err := tlcsv.NewReader(cmd.readerPathB)
	if err != nil {
		return err
	}
	df1, err := checkDiff(readerA)
	if err != nil {
		return err
	}
	df2, err := checkDiff(readerB)
	if err != nil {
		return err
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
		presentA := []diffEnt{}
		presentB := []diffEnt{}
		for k := range combinedKeys {
			ent1, ok1 := df1.ents[diffKey{fn, k}]
			ent2, ok2 := df2.ents[diffKey{fn, k}]
			if ok1 && ok2 {
				if ent1.hash == ent2.hash && cmd.ShowSame {
					ent1.row = append(ent1.row, "", "same")
					presentBoth = append(presentBoth, ent1)
				} else if ent1.hash != ent2.hash && cmd.ShowDiff {
					ent1.row = append(ent1.row, readerA.Path(), "diff")
					ent2.row = append(ent2.row, readerB.Path(), "diff")
					presentDiff = append(presentDiff, ent1, ent2)
				}
			} else if ok1 && !ok2 && cmd.ShowMissing {
				ent1.row = append(ent1.row, readerA.Path(), "present-A")
				presentA = append(presentA, ent1)
			} else if ok2 && !ok1 && cmd.ShowMissing {
				ent2.row = append(ent2.row, readerB.Path(), "present-B")
				presentB = append(presentB, ent2)
			}
		}
		// Write
		if len(presentBoth) == 0 && len(presentDiff) == 0 && len(presentA) == 0 && len(presentB) == 0 {
			continue
		}
		header := []string{}
		if df1.headers[fn].hash != df2.headers[fn].hash {
			fmt.Println("headers are different:")
			fmt.Println(df1.headers[fn].row)
			fmt.Println(df2.headers[fn].row)
			continue
		}
		fmt.Println("writing:", fn)
		header = append(header, df1.headers[fn].row...)
		header = append(header, "diff_filename", "diff_status")
		outWriter.WriteRows(fn, [][]string{header})
		for _, row := range presentBoth {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range presentDiff {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range presentA {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
		for _, row := range presentB {
			outWriter.WriteRows(fn, [][]string{row.row})
		}
	}
	return nil
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

type diffKey struct {
	efn string
	eid string
}

type diffEnt struct {
	hash string
	row  []string
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
		h := sha1.New()
		for _, c := range row {
			h.Write([]byte(c))
		}
		hh := hex.EncodeToString(h.Sum(nil))
		if _, ok := adapter.headers[efn]; !ok {
			adapter.headers[efn] = diffEnt{hash: hh, row: row}
		} else {
			k := ""
			if headerIndex >= 0 {
				k = row[headerIndex]
			} else {
				k = hh
			}
			key := diffKey{efn, k}
			adapter.ents[key] = diffEnt{hash: hh, row: row}
		}
	}
	return nil
}
