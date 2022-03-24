package dmfr

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dimchansky/utfbom"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
)

// Registry represents a parsed Distributed Mobility Feed Registry (DMFR) file
type Registry struct {
	Schema                string `json:"$schema"`
	Feeds                 []tl.Feed
	Operators             []tl.Operator
	Secrets               []tl.Secret
	LicenseSpdxIdentifier string `json:"license_spdx_identifier"`
}

// feed.Operators should be loaded but not manipulated
type loadFeed struct {
	Operators []tl.Operator
	tl.Feed
}

type loadRegistry struct {
	Feeds []loadFeed
	Registry
}

// NewRegistry TODO
func NewRegistry(reader io.Reader) (*Registry, error) {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var loadReg loadRegistry
	if err := json.Unmarshal([]byte(contents), &loadReg); err != nil {
		if e, ok := err.(*json.SyntaxError); ok {
			log.Debugf("syntax error at byte offset %d", e.Offset)
		}
		return nil, err
	}

	// Apply nested operator rules
	reg := loadReg.Registry
	operators := []tl.Operator{}
	for _, rfeed := range loadReg.Feeds {
		reg.Feeds = append(reg.Feeds, rfeed.Feed) // add feed without operator
		fsid := rfeed.FeedID
		for _, operator := range rfeed.Operators {
			foundParent := false
			for i, oif := range operator.AssociatedFeeds {
				if oif.FeedOnestopID.String == "" {
					oif.FeedOnestopID = tl.NewString(fsid)
				}
				if oif.FeedOnestopID.String == fsid {
					foundParent = true
				}
				operator.AssociatedFeeds[i] = oif
			}
			if !foundParent {
				operator.AssociatedFeeds = append(operator.AssociatedFeeds, tl.OperatorAssociatedFeed{FeedOnestopID: tl.NewString(fsid)})
			}
			operators = append(operators, operator)
		}
	}
	// Merge operators
	operators = append(operators, reg.Operators...)
	mergeOperators := map[string]tl.Operator{}
	for _, operator := range operators {
		osid := operator.OnestopID.String
		a, ok := mergeOperators[osid]
		if ok {
			operator.AssociatedFeeds = append(operator.AssociatedFeeds, a.AssociatedFeeds...)
			if operator.Name.String == "" {
				operator.Name = a.Name
			}
			if operator.ShortName.String == "" {
				operator.ShortName = a.ShortName
			}
			if operator.Website.String == "" {
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
		return NewRegistry(readerSkippingBOM)
	}
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	readerSkippingBOM, _ := utfbom.Skip(reader)
	reg, err := NewRegistry(readerSkippingBOM)
	if err != nil {
		return nil, err
	}
	return reg, nil
}

// ParseString TODO
func ParseString(contents string) (*Registry, error) {
	reader := strings.NewReader(contents)
	return NewRegistry(reader)
}
