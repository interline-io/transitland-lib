package ext

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
)

// Extension defines two methods that specify the entities in an Extension and how to Create the necessary output structures, e.g. in a database.
type Extension interface {
}

type readerFactory func(dburl string) (adapters.Reader, error)
type writerFactory func(dburl string) (adapters.Writer, error)
type extensionFactory func(string) (Extension, error)

var readerFactories = map[string]readerFactory{}
var writerFactories = map[string]writerFactory{}
var extensionFactories = map[string]extensionFactory{}

// RegisterReader registers a Reader.
func RegisterReader(name string, factory readerFactory) error {
	if factory == nil {
		return fmt.Errorf("reader '%s' does not exist", name)
	}
	_, registered := readerFactories[name]
	if registered {
		return fmt.Errorf("reader '%s' already registered", name)
	}
	log.Tracef("registering reader: %s", name)
	readerFactories[name] = factory
	return nil
}

// RegisterWriter registers a Writer.
func RegisterWriter(name string, factory writerFactory) error {
	if factory == nil {
		return fmt.Errorf("writer '%s' does not exist", name)
	}
	_, registered := writerFactories[name]
	if registered {
		return fmt.Errorf("writer '%s' already registered", name)
	}
	log.Tracef("registering writer: %s", name)
	writerFactories[name] = factory
	return nil
}

// RegisterExtension registers an Extension.
func RegisterExtension(name string, factory extensionFactory) error {
	_, registered := extensionFactories[name]
	if registered {
		return fmt.Errorf("extension '%s' already registered", name)
	}
	log.Tracef("registering extension: %s", name)
	extensionFactories[name] = factory
	return nil
}

// NewReader uses the scheme prefix as the driver name, defaulting to csv.
func NewReader(addr string) (adapters.Reader, error) {
	scheme := strings.Split(addr, "://")
	driver := "csv"
	if len(scheme) > 1 {
		driver = scheme[0]
	}
	if f, ok := readerFactories[driver]; ok {
		return f(addr)
	}
	return nil, fmt.Errorf("no reader for %s", driver)
}

// OpenReader returns an opened reader.
func OpenReader(addr string) (adapters.Reader, error) {
	r, err := NewReader(addr)
	if err != nil {
		return nil, err
	}
	if err := r.Open(); err != nil {
		return nil, fmt.Errorf("could not open reader '%s': %s", addr, err.Error())
	}
	return r, nil
}

// NewWriter uses the scheme prefix as the driver name, defaulting to csv.
func NewWriter(addr string) (adapters.Writer, error) {
	scheme := strings.Split(addr, "://")
	driver := "csv"
	if len(scheme) > 1 {
		driver = scheme[0]
	}
	if f, ok := writerFactories[driver]; ok {
		return f(addr)
	}
	return nil, fmt.Errorf("no writer for %s", driver)
}

// OpenWriter returns an opened writer.
func OpenWriter(addr string, create bool) (adapters.Writer, error) {
	w, err := NewWriter(addr)
	if err != nil {
		return nil, err
	}
	if err := w.Open(); err != nil {
		return nil, fmt.Errorf("could not open writer '%s': %s", addr, err.Error())
	}
	if create {
		if err := w.Create(); err != nil {
			return nil, fmt.Errorf("could not create database '%s': %s", addr, err.Error())
		}
	}
	return w, nil
}

// GetExtension returns an Extension.
func GetExtension(name string, args string) (Extension, error) {
	if f, ok := extensionFactories[name]; ok {
		return f(args)
	}
	return nil, fmt.Errorf("no extension for %s", name)
}

func ParseExtensionArgs(value string) (string, string, error) {
	sp := strings.SplitN(value, ":", 2)
	if len(sp) < 2 {
		return value, "", nil
	}
	extName := sp[0]
	extArgs := sp[1]
	if strings.HasPrefix(extArgs, "{") {
		// Treat as JSON, but check validity
		a := make(map[string]interface{})
		if err := json.Unmarshal([]byte(extArgs), &a); err != nil {
			return "", "", err
		}
	} else {
		// Treat as key=value,key=value pairs
		a := make(map[string]interface{})
		for _, kv := range strings.Split(extArgs, ",") {
			k := strings.SplitN(kv, "=", 2)
			if len(k) < 2 {
				k = append(k, "")
			}
			// Attempt to convert to numeric
			if v, err := strconv.ParseFloat(k[1], 64); err == nil {
				a[k[0]] = v
			} else {
				a[k[0]] = k[1]
			}
		}
		j, err := json.Marshal(&a)
		if err != nil {
			return "", "", err
		}
		extArgs = string(j)
	}
	return extName, extArgs, nil
}
