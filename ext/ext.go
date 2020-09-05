package ext

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
)

// Extension defines two methods that specify the entities in an Extension and how to Create the necessary output structures, e.g. in a database.
type Extension interface {
	Create(tl.Writer) error
	Entities() []tl.Entity
}

type readerFactory func(dburl string) (tl.Reader, error)
type writerFactory func(dburl string) (tl.Writer, error)
type extensionFactory func() Extension
type entityFilterFactory func() tl.EntityFilter

var readerFactories = map[string]readerFactory{}
var writerFactories = map[string]writerFactory{}
var extensionFactories = map[string]extensionFactory{}
var entityFilterFactories = map[string]entityFilterFactory{}

// RegisterReader registers a Reader.
func RegisterReader(name string, factory readerFactory) {
	if factory == nil {
		log.Fatal("Factory %s does not exist", name)
	}
	_, registered := readerFactories[name]
	if registered {
		log.Fatal("Factory %s already registered", name)
	}
	log.Debug("Registering Reader factory: %s", name)
	readerFactories[name] = factory
}

// RegisterWriter registers a Writer.
func RegisterWriter(name string, factory writerFactory) {
	if factory == nil {
		log.Fatal("Factory %s does not exist", name)
	}
	_, registered := writerFactories[name]
	if registered {
		log.Fatal("Factory %s already registered", name)
	}
	log.Debug("Registering Writer factory: %s", name)
	writerFactories[name] = factory
}

// RegisterExtension registers an Extension.
func RegisterExtension(name string, factory extensionFactory) {
	_, registered := extensionFactories[name]
	if registered {
		panic("failed")
	}
	log.Debug("Registering Extension factory: %s", name)
	extensionFactories[name] = factory
}

// RegisterEntityFilter registers a EntityFilter.
func RegisterEntityFilter(name string) {
	log.Debug("Registering EntityFilter factory: %s", name)
}

// NewReader uses the scheme prefix as the driver name, defaulting to csv.
func NewReader(url string) (tl.Reader, error) {
	scheme := strings.Split(url, "://")
	if len(scheme) > 1 {
		return GetReader(scheme[0], url)
	}
	return GetReader("csv", url)
}

// MustOpenReaderOrPanic is a helper that returns an opened reader or panics.
func MustOpenReaderOrPanic(path string) tl.Reader {
	r, err := NewReader(path)
	if err != nil {
		panic(fmt.Sprintf("No handler for reader '%s': %s", path, err.Error()))
	}
	if err := r.Open(); err != nil {
		panic(fmt.Sprintf("Could not open reader '%s': %s", path, err.Error()))
	}
	return r
}

// MustOpenReaderOrExit is a helper that returns an opened a reader or exits.
func MustOpenReaderOrExit(path string) tl.Reader {
	r, err := NewReader(path)
	if err != nil {
		log.Exit("No handler for reader '%s': %s", path, err.Error())
	}
	if err := r.Open(); err != nil {
		log.Exit("Could not open reader '%s': %s", path, err.Error())
	}
	return r
}

// NewWriter uses the scheme prefix as the driver name, defaulting to csv.
func NewWriter(dburl string) (tl.Writer, error) {
	url := strings.Split(dburl, "://")
	if len(url) > 1 {
		return GetWriter(url[0], dburl)
	}
	return GetWriter("csv", dburl)
}

// MustOpenWriterOrPanic is a helper that returns an opened writer or panics.
func MustOpenWriterOrPanic(path string) tl.Writer {
	r, err := NewWriter(path)
	if err != nil {
		panic(fmt.Sprintf("No handler for reader '%s': %s", path, err.Error()))
	}
	if err := r.Open(); err != nil {
		panic(fmt.Sprintf("Could not open reader '%s': %s", path, err.Error()))
	}
	return r
}

// MustOpenWriterOrExit is a helper that returns an opened a writer or exits.
func MustOpenWriterOrExit(path string) tl.Writer {
	r, err := NewWriter(path)
	if err != nil {
		log.Exit("No handler for writer '%s': %s", path, err.Error())
	}
	if err := r.Open(); err != nil {
		log.Exit("Could not open writer '%s': %s", path, err.Error())
	}
	return r
}

// GetReader returns a Reader for the URL.
func GetReader(driver string, dburl string) (tl.Reader, error) {
	if f, ok := readerFactories[driver]; ok {
		return f(dburl)
	}
	return nil, fmt.Errorf("no Reader factory for %s", driver)
}

// GetWriter returns a Writer for the URL.
func GetWriter(driver string, dburl string) (tl.Writer, error) {
	if f, ok := writerFactories[driver]; ok {
		return f(dburl)
	}
	return nil, fmt.Errorf("no Writer factory for %s", driver)
}

// GetExtension returns an Extension.
func GetExtension(name string) (Extension, error) {
	if f, ok := extensionFactories[name]; ok {
		return f(), nil
	}
	return nil, fmt.Errorf("no Extension factory for %s", name)
}

// GetEntityFilter returns a Transform.
func GetEntityFilter(name string) (tl.EntityFilter, error) {
	if f, ok := entityFilterFactories[name]; ok {
		return f(), nil
	}
	return nil, fmt.Errorf("no EntityFilter factory for %s", name)
}
