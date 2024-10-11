package dmfr

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dimchansky/utfbom"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tt"
)

// Registry represents a parsed Distributed Mobility Feed Registry (DMFR) file
type Registry struct {
	Schema                string     `json:"$schema,omitempty"`
	Feeds                 []Feed     `json:"feeds,omitempty"`
	Operators             []Operator `json:"operators,omitempty"`
	Secrets               []Secret   `json:"secrets,omitempty"`
	LicenseSpdxIdentifier string     `json:"license_spdx_identifier,omitempty"`
}

// ReadRegistry TODO
func ReadRegistry(reader io.Reader) (*Registry, error) {
	loadReg, err := ReadRawRegistry(reader)
	if err != nil {
		return nil, err
	}

	// Apply nested operator rules
	reg := Registry{}
	reg.LicenseSpdxIdentifier = loadReg.LicenseSpdxIdentifier
	reg.Schema = loadReg.Schema
	reg.Operators = loadReg.Operators
	reg.Secrets = loadReg.Secrets
	if reg.Schema == "" {
		reg.Schema = "https://dmfr.transit.land/json-schema/dmfr.schema-v0.5.0.json"
	}
	operators := []Operator{}
	for _, rfeed := range loadReg.Feeds {
		reg.Feeds = append(reg.Feeds, rfeed.Feed) // add feed without operator
		fsid := rfeed.FeedID
		for _, operator := range rfeed.Operators {
			foundParent := false
			for i, oif := range operator.AssociatedFeeds {
				if oif.FeedOnestopID.Val == "" {
					oif.FeedOnestopID = tt.NewString(fsid)
				}
				if oif.FeedOnestopID.Val == fsid {
					foundParent = true
				}
				operator.AssociatedFeeds[i] = oif
			}
			if !foundParent {
				operator.AssociatedFeeds = append(operator.AssociatedFeeds, OperatorAssociatedFeed{FeedOnestopID: tt.NewString(fsid)})
			}
			operators = append(operators, operator)
		}
	}
	// Merge operators
	operators = append(operators, reg.Operators...)
	mergeOperators := map[string]Operator{}
	for _, operator := range operators {
		osid := operator.OnestopID.Val
		a, ok := mergeOperators[osid]
		if ok {
			operator.AssociatedFeeds = append(operator.AssociatedFeeds, a.AssociatedFeeds...)
			if operator.Name.Val == "" {
				operator.Name = a.Name
			}
			if operator.ShortName.Val == "" {
				operator.ShortName = a.ShortName
			}
			if operator.Website.Val == "" {
				operator.Website = a.Website
			}
		}
		mergeOperators[osid] = operator
	}
	reg.Operators = nil
	for _, operator := range mergeOperators {
		reg.Operators = append(reg.Operators, operator)
	}

	// Check license and required feeds
	log.Debugf("Loaded a DMFR file containing %d feeds", len(loadReg.Feeds))
	if loadReg.LicenseSpdxIdentifier != "CC0-1.0" {
		log.Debugf("Loading a DMFR file without the standard CC0-1.0 license. Proceed with caution!")
	}
	for i := 0; i < len(loadReg.Feeds); i++ {
		feedSpec := strings.ToLower(loadReg.Feeds[i].Spec)
		if feedSpec == "gtfs" || feedSpec == "gtfs-rt" || feedSpec == "gbfs" || feedSpec == "mds" {
			continue
		} else {
			return nil, errors.New("at least one feed in the DMFR file is not of a valid spec (GTFS, GTFS-RT, GBFS, or MDS)")
		}
	}
	return &reg, nil
}

// Format raw registry, before additional processing is applied
func (r *Registry) Write(w io.Writer) error {
	rr := RawRegistry{}
	rr.Operators = r.Operators
	rr.Secrets = r.Secrets
	rr.Schema = r.Schema
	rr.LicenseSpdxIdentifier = r.LicenseSpdxIdentifier
	for _, feed := range r.Feeds {
		rr.Feeds = append(rr.Feeds, RawRegistryFeed{Feed: feed})
	}
	return rr.Write(w)
}

// LoadAndParseRegistry loads and parses a Distributed Mobility Feed Registry (DMFR) file from either a file system path or a URL
func LoadAndParseRegistry(path string) (*Registry, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(utfbom.SkipOnly(resp.Body))
		if err != nil {
			return nil, err
		}
		reader := bytes.NewReader(body)
		readerSkippingBOM := utfbom.SkipOnly(reader)
		return ReadRegistry(readerSkippingBOM)
	}
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	readerSkippingBOM, _ := utfbom.Skip(reader)
	reg, err := ReadRegistry(readerSkippingBOM)
	if err != nil {
		return nil, err
	}
	return reg, nil
}
