package dmfr

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dimchansky/utfbom"
	"github.com/interline-io/gotransit/internal/log"
)

// Registry represents a parsed Distributed Mobility Feed Registry (DMFR) file
type Registry struct {
	Schema                string `json:"$schema"`
	Feeds                 []Feed
	LicenseSpdxIdentifier string `json:"license_spdx_identifier"`
}

// NewRegistry TODO
func NewRegistry(reader io.Reader) (*Registry, error) {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var registry Registry
	if err := json.Unmarshal([]byte(contents), &registry); err != nil {
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		return nil, err
	}
	log.Debug("Loaded a DMFR file containing %d feeds", len(registry.Feeds))
	if registry.LicenseSpdxIdentifier != "CC0-1.0" {
		log.Debug("Loading a DMFR file without the standard CC0-1.0 license. Proceed with caution!")
	}
	for i := 0; i < len(registry.Feeds); i++ {
		registry.Feeds[i].OtherIDs = map[string]string{}
		feedSpec := strings.ToLower(registry.Feeds[i].Spec)
		if feedSpec == "gtfs" || feedSpec == "gtfs-rt" || feedSpec == "gbfs" || feedSpec == "mds" {
			continue
		} else {
			log.Fatal("At least one feed in the DMFR file is not of a valid spec (GTFS, GTFS-RT, GBFS, or MDS)")
		}

	}
	return &registry, nil
}

func (registry *Registry) writeToJSONFile(path string) error {
	registryJSON, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, registryJSON, 0644)
}

// LoadAndParseRegistry loads and parses a Distributed Mobility Feed Registry (DMFR) file from either a file system path or a URL
func LoadAndParseRegistry(path string) (*Registry, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if body, err := ioutil.ReadAll(utfbom.SkipOnly(resp.Body)); err != nil {
			return nil, err
		} else {
			reader := bytes.NewReader(body)
			readerSkippingBOM := utfbom.SkipOnly(reader)
			return NewRegistry(readerSkippingBOM)
		}
	} else {
		if reader, err := os.Open(path); err != nil {
			return nil, err
		} else {
			readerSkippingBOM, enc := utfbom.Skip(reader)
			log.Info("DETECT: %s", enc)
			return NewRegistry(readerSkippingBOM)
		}
		return NewRegistry(bytes.NewReader(body))
	}
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewRegistry(reader)
}

// ParseString TODO
func ParseString(contents string) (*Registry, error) {
	reader := strings.NewReader(contents)
	return NewRegistry(reader)
}
