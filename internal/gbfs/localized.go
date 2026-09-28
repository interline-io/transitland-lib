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

// A textFile is a file that holds translatable text.
type textFile struct {
	name string
	// each is eachText for this file's entities.
	each func(f *GbfsFeed, fn func(key string, fields []*LocalizedString))
	// fill takes o's entities from this file if f has none.
	fill func(f *GbfsFeed, o *GbfsFeed)
}

// entityText returns a textFile for a file that holds a list of entities, each
// read by text as its id and localized fields.
func entityText[T any](name string, list func(*GbfsFeed) *[]*T, text func(*T) (string, []*LocalizedString)) textFile {
	return textFile{
		name: name,
		each: func(f *GbfsFeed, fn func(string, []*LocalizedString)) {
			for _, e := range *list(f) {
				if e != nil {
					id, fields := text(e)
					fn(name+":"+id, fields)
				}
			}
		},
		fill: func(f *GbfsFeed, o *GbfsFeed) {
			if *list(f) == nil {
				*list(f) = *list(o)
			}
		},
	}
}

// textFiles are the files that hold translatable text.
var textFiles = []textFile{
	{
		name: fileSystemInformation,
		each: func(f *GbfsFeed, fn func(string, []*LocalizedString)) {
			if e := f.SystemInformation; e != nil {
				fn(fileSystemInformation, []*LocalizedString{&e.Name, &e.ShortName, &e.Operator, &e.TermsURL, &e.PrivacyURL})
			}
		},
		// The system comes from the language whose system_information fetched.
		fill: func(*GbfsFeed, *GbfsFeed) {},
	},
	entityText(fileStationInformation,
		func(f *GbfsFeed) *[]*StationInformation { return &f.StationInformation },
		func(e *StationInformation) (string, []*LocalizedString) {
			return e.StationID.Val, []*LocalizedString{&e.Name, &e.ShortName}
		}),
	entityText(fileVehicleTypes,
		func(f *GbfsFeed) *[]*VehicleType { return &f.VehicleTypes },
		func(e *VehicleType) (string, []*LocalizedString) {
			return e.VehicleTypeID.Val, []*LocalizedString{&e.Name, &e.Make, &e.Model}
		}),
	entityText(fileSystemRegions,
		func(f *GbfsFeed) *[]*SystemRegion { return &f.Regions },
		func(e *SystemRegion) (string, []*LocalizedString) {
			return e.RegionID.Val, []*LocalizedString{&e.Name}
		}),
	entityText(fileSystemPricingPlans,
		func(f *GbfsFeed) *[]*SystemPricingPlan { return &f.Plans },
		func(e *SystemPricingPlan) (string, []*LocalizedString) {
			return e.PlanID.Val, []*LocalizedString{&e.Name, &e.Description}
		}),
	entityText(fileSystemAlerts,
		func(f *GbfsFeed) *[]*SystemAlert { return &f.Alerts },
		func(e *SystemAlert) (string, []*LocalizedString) {
			return e.AlertID.Val, []*LocalizedString{&e.URL, &e.Summary, &e.Description}
		}),
}

// eachText calls fn with each of f's entities that has localized fields: its
// key, which names the same entity in another language, and those fields.
func eachText(f *GbfsFeed, fn func(key string, fields []*LocalizedString)) {
	for _, t := range textFiles {
		t.each(f, fn)
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
