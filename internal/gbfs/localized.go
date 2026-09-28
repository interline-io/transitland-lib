package gbfs

import (
	"encoding/json"
	"io"

	"github.com/interline-io/transitland-lib/tt"
)

// LocalizedText is a text field in one language.
type LocalizedText struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// LocalizedString is a text field in each language a system publishes, the
// system's default language first.
type LocalizedString []LocalizedText

// UnmarshalJSON accepts a 3.x localized array or a 1.x/2.x string. Anything
// else is kept as its JSON text rather than failing the whole file.
func (l *LocalizedString) UnmarshalJSON(b []byte) error {
	var s string
	var v []LocalizedText
	switch {
	case string(b) == "null":
		*l = nil
	case b[0] == '"' && json.Unmarshal(b, &s) == nil:
		*l = LocalizedString{{Text: s}}
	case b[0] == '[' && json.Unmarshal(b, &v) == nil:
		*l = v
	default:
		*l = LocalizedString{{Text: string(b)}}
	}
	return nil
}

// Default returns the text in the system's default language.
func (l LocalizedString) Default() string {
	if len(l) == 0 {
		return ""
	}
	return l[0].Text
}

// MarshalGQL writes the text in the system's default language.
func (l LocalizedString) MarshalGQL(w io.Writer) {
	var s tt.String
	if len(l) > 0 {
		s = tt.NewString(l.Default())
	}
	s.MarshalGQL(w)
}

// UnmarshalGQL reads a string as untagged text.
func (l *LocalizedString) UnmarshalGQL(v any) error {
	var s tt.String
	if err := s.UnmarshalGQL(v); err != nil {
		return err
	}
	*l = LocalizedString{{Text: s.Val}}
	return nil
}

// eachText calls fn with each of f's entities that has localized fields: its
// key, which names the same entity in another language, and those fields.
func eachText(f *GbfsFeed, fn func(key string, fields []*LocalizedString)) {
	if e := f.SystemInformation; e != nil {
		fn("system", []*LocalizedString{&e.Name, &e.ShortName, &e.Operator, &e.TermsURL, &e.PrivacyURL})
	}
	for _, e := range f.StationInformation {
		if e != nil {
			fn("station:"+e.StationID.Val, []*LocalizedString{&e.Name, &e.ShortName})
		}
	}
	for _, e := range f.VehicleTypes {
		if e != nil {
			fn("vehicle_type:"+e.VehicleTypeID.Val, []*LocalizedString{&e.Name, &e.Make, &e.Model})
		}
	}
	for _, e := range f.Regions {
		if e != nil {
			fn("region:"+e.RegionID.Val, []*LocalizedString{&e.Name})
		}
	}
	for _, e := range f.Plans {
		if e != nil {
			fn("plan:"+e.PlanID.Val, []*LocalizedString{&e.Name, &e.Description})
		}
	}
	for _, e := range f.Alerts {
		if e != nil {
			fn("alert:"+e.AlertID.Val, []*LocalizedString{&e.URL, &e.Summary, &e.Description})
		}
	}
}

// setLanguage tags f's untagged text with lang.
func setLanguage(f *GbfsFeed, lang string) {
	eachText(f, func(_ string, fields []*LocalizedString) {
		for _, l := range fields {
			for i := range *l {
				if (*l)[i].Language == "" {
					(*l)[i].Language = lang
				}
			}
		}
	})
}

// addTranslations appends o's text to the matching entities in f. Entities f
// lacks are dropped.
func addTranslations(f *GbfsFeed, o *GbfsFeed) {
	src := map[string][]*LocalizedString{}
	eachText(o, func(key string, fields []*LocalizedString) {
		src[key] = fields
	})
	eachText(f, func(key string, fields []*LocalizedString) {
		for i, l := range src[key] {
			*fields[i] = append(*fields[i], *l...)
		}
	})
}

// putFirst moves each field's text in lang to the front, where Default reads
// it.
func putFirst(f *GbfsFeed, lang string) {
	eachText(f, func(_ string, fields []*LocalizedString) {
		for _, l := range fields {
			for i, t := range *l {
				if t.Language == lang {
					copy((*l)[1:i+1], (*l)[:i])
					(*l)[0] = t
					break
				}
			}
		}
	})
}
