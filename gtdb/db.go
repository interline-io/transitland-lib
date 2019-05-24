package gtdb

import "github.com/interline-io/gotransit"

var bufferSize = 1000

func init() {
	// Register readers and writers
	r := func(url string) (gotransit.Reader, error) { return NewReader(url) }
	gotransit.RegisterReader("sqlite3", r)
	gotransit.RegisterReader("postgres", r)
	w := func(url string) (gotransit.Writer, error) { return NewWriter(url) }
	gotransit.RegisterWriter("sqlite3", w)
	gotransit.RegisterWriter("postgres", w)
}
