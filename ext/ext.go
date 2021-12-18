package ext

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
)

// Extension defines two methods that specify the entities in an Extension and how to Create the necessary output structures, e.g. in a database.
type Extension interface {
}

type readerFactory func(dburl string) (tl.Reader, error)
type writerFactory func(dburl string) (tl.Writer, error)
type extensionFactory func() Extension

var readerFactories = map[string]readerFactory{}
var writerFactories = map[string]writerFactory{}
var extensionFactories = map[string]extensionFactory{}

// RegisterReader registers a Reader.
func RegisterReader(name string, factory readerFactory) error {
	if factory == nil {
		return fmt.Errorf("factory '%s' does not exist", name)
	}
	_, registered := readerFactories[name]
	if registered {
		return fmt.Errorf("factory '%s' already registered", name)
	}
	log.Debug("Registering Reader factory: %s", name)
	readerFactories[name] = factory
	return nil
}

// RegisterWriter registers a Writer.
func RegisterWriter(name string, factory writerFactory) error {
	if factory == nil {
		return fmt.Errorf("factory '%s' does not exist", name)
	}
	_, registered := writerFactories[name]
	if registered {
		return fmt.Errorf("factory '%s' already registered", name)
	}
	log.Debug("Registering Writer factory: %s", name)
	writerFactories[name] = factory
	return nil
}

// RegisterExtension registers an Extension.
func RegisterExtension(name string, factory extensionFactory) error {
	_, registered := extensionFactories[name]
	if registered {
		return fmt.Errorf("extension '%s' already registered", name)
	}
	log.Debug("registering Extension factory: %s", name)
	extensionFactories[name] = factory
	return nil
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
		panic(fmt.Sprintf("no handler for reader '%s': %s", path, err.Error()))
	}
	if err := r.Open(); err != nil {
		panic(fmt.Sprintf("could not open reader '%s': %s", path, err.Error()))
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

// MustGetReader or exits.
func MustGetReader(inurl string) tl.Reader {
	if len(inurl) == 0 {
		log.Exit("No reader specified")
	}
	// Reader
	reader, err := NewReader(inurl)
	if err != nil {
		log.Exit("No known reader for '%s': %s", inurl, err)
	}
	if err := reader.Open(); err != nil {
		log.Exit("Could not open '%s': %s", inurl, err)
	}
	return reader
}

// MustGetWriter or exits.
func MustGetWriter(outurl string, create bool) tl.Writer {
	if len(outurl) == 0 {
		log.Exit("No writer specified")
	}
	// Writer
	writer, err := NewWriter(outurl)
	if err != nil {
		log.Exit("No known writer for '%s': %s", outurl, err)
	}
	if err := writer.Open(); err != nil {
		log.Exit("Could not open '%s': %s", outurl, err)
	}
	if create {
		if err := writer.Create(); err != nil {
			log.Exit("Could not create writer: %s", err)
		}
	}
	return writer
}
