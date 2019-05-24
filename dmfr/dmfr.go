package dmfr

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/interline-io/gotransit/internal/log"
)

// Registry represents a parsed Distributed Mobility Feed Registry (DMFR) file
type Registry struct {
	Schema                string `json:"$schema"`
	Feeds                 []Feed
	LicenseSpdxIdentifier string `json:"license_spdx_identifier"`
}

// Feed listed in a parsed DMFR file
type Feed struct {
	Spec        string
	ID          string
	URL         string
	IDCrosswalk map[string]string `json:"id_crosswalk"`
}

// NewRegistry TODO
func NewRegistry(reader io.Reader) (*Registry, error) {
	contents, _ := ioutil.ReadAll(reader)
	var registry Registry
	if err := json.Unmarshal([]byte(contents), &registry); err != nil {
		return nil, err
	}
	log.Info("Loaded a DMFR file containing %d feeds", len(registry.Feeds))
	if registry.LicenseSpdxIdentifier != "CC0-1.0" {
		log.Info("Loading a DMFR file without the standard CC0-1.0 license. Proceed with caution!")
	}
	// for _, feed := range registry.Feeds {
	for i := 0; i < len(registry.Feeds); i++ {
		registry.Feeds[i].IDCrosswalk = map[string]string{}
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
	registryJSON, _ := json.Marshal(registry)
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
		if body, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		} else {
			return NewRegistry(bytes.NewReader(body))
		}
	} else {
		if reader, err := os.Open(path); err != nil {
			return nil, err
		} else {
			return NewRegistry(reader)
		}
	}
}

// ParseString TODO
func ParseString(contents string) (*Registry, error) {
	reader := strings.NewReader(contents)
	return NewRegistry(reader)
}
