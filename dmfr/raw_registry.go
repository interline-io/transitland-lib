package dmfr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"

	"github.com/iancoleman/orderedmap"
	"github.com/interline-io/log"
)

type RawRegistry struct {
	Schema                string            `json:"$schema,omitempty"`
	Feeds                 []RawRegistryFeed `json:"feeds,omitempty"`
	Operators             []Operator        `json:"operators,omitempty"`
	Secrets               []Secret          `json:"secrets,omitempty"`
	LicenseSpdxIdentifier string            `json:"license_spdx_identifier,omitempty"`
}

// feed.Operators should be loaded but not exported
type RawRegistryFeed struct {
	Feed
	Operators []Operator `json:"operators"`
}

func ReadRawRegistry(reader io.Reader) (*RawRegistry, error) {
	ctx := context.TODO()
	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var loadReg RawRegistry
	if err := json.Unmarshal([]byte(contents), &loadReg); err != nil {
		if e, ok := err.(*json.SyntaxError); ok {
			log.For(ctx).Debug().Msgf("syntax error at byte offset %d", e.Offset)
		}
		return nil, err
	}
	return &loadReg, nil
}

// Format raw registry, before additional processing is applied
func (r *RawRegistry) Write(w io.Writer) error {
	// Sort feeds
	sort.Slice(r.Feeds, func(i, j int) bool {
		return r.Feeds[i].FeedID < r.Feeds[j].FeedID
	})
	// Sort feed fields
	for _, feed := range r.Feeds {
		sort.Strings(feed.Languages)
		// Sort nested operators
		sort.Slice(feed.Operators, func(i, j int) bool {
			return feed.Operators[i].OnestopID.Val < feed.Operators[j].OnestopID.Val
		})
		// Sort nested operator fields
		for _, op := range feed.Operators {
			sort.Slice(op.AssociatedFeeds, func(i, j int) bool {
				return op.AssociatedFeeds[i].FeedOnestopID.Val < op.AssociatedFeeds[j].FeedOnestopID.Val
			})
		}
	}
	// Sort operators
	sort.Slice(r.Operators, func(i, j int) bool {
		return r.Operators[i].OnestopID.Val < r.Operators[j].OnestopID.Val
	})
	// Sort operator fields
	for _, op := range r.Operators {
		sort.Slice(op.AssociatedFeeds, func(i, j int) bool {
			return op.AssociatedFeeds[i].FeedOnestopID.Val < op.AssociatedFeeds[j].FeedOnestopID.Val
		})
	}
	// Convert to JSON, process as MapSlice to remove empty elements, write back as json
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	// Load JSON back into OrderedMap before removing null values
	m := orderedmap.OrderedMap{}
	m.SetEscapeHTML(false)
	json.Unmarshal(b, &m)
	m = removeNulls(m)

	// Convert back to JSON, then apply indent
	// OrderedMap doesn't support MarshalIndent directly
	mb, err := m.MarshalJSON()
	if err != nil {
		return err
	}
	var mbi bytes.Buffer
	if err := json.Indent(&mbi, mb, "", "  "); err != nil {
		return err
	}
	if _, err = w.Write(mbi.Bytes()); err != nil {
		return err
	}
	_, err = w.Write([]byte{'\n'})
	return err
}

func removeNulls(m orderedmap.OrderedMap) orderedmap.OrderedMap {
	// Create a new output OrderedMap,
	// go through every element in input map, and remove any null or empty maps
	m2 := orderedmap.New()
	m2.SetEscapeHTML(false)
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		if vx, ok := v.(orderedmap.OrderedMap); ok {
			p := removeNulls(vx)
			if len(p.Keys()) > 0 {
				v = p
			} else {
				v = nil
			}
		} else if vx, ok := v.([]interface{}); ok {
			var vll []interface{}
			for i := 0; i < len(vx); i++ {
				vxx := vx[i]
				if vxxx, ok := vxx.(orderedmap.OrderedMap); ok {
					p := removeNulls(vxxx)
					if len(p.Keys()) > 0 {
						vll = append(vll, p)
					}
				} else {
					vll = append(vll, vxx)
				}
			}
			if len(vll) > 0 {
				v = vll
			} else {
				v = nil
			}
		}
		if v != nil {
			m2.Set(k, v)
		}
	}
	return *m2
}
