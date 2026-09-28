package gbfs

import (
	"encoding/json"
	"io"
)

// LocalizedText is a text field in one language.
type LocalizedText struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// LocalizedString is a text field in each language a system publishes, the
// system's default language first.
type LocalizedString []LocalizedText

// UnmarshalJSON accepts a 3.x array of localized strings or a 1.x/2.x string.
//
// Anything else is kept as its JSON text, as tt.String did, rather than failing
// the whole file.
func (l *LocalizedString) UnmarshalJSON(b []byte) error {
	var s string
	var v []LocalizedText
	switch {
	case string(b) == "null":
		*l = nil
	case json.Unmarshal(b, &s) == nil:
		*l = LocalizedString{{Text: s}}
	case json.Unmarshal(b, &v) == nil:
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
	if len(l) == 0 {
		w.Write([]byte("null"))
		return
	}
	b, _ := json.Marshal(l.Default())
	w.Write(b)
}

// UnmarshalGQL reads a string as untagged text.
func (l *LocalizedString) UnmarshalGQL(v any) error {
	s, _ := v.(string)
	*l = LocalizedString{{Text: s}}
	return nil
}

// textsByID returns f's localized fields, keyed by the entity they belong to.
func textsByID(f *GbfsFeed) map[string][]*LocalizedString {
	ret := map[string][]*LocalizedString{}
	if e := f.SystemInformation; e != nil {
		ret["system"] = []*LocalizedString{&e.Name, &e.ShortName, &e.Operator, &e.TermsURL, &e.PrivacyURL}
	}
	for _, e := range f.StationInformation {
		ret["station:"+e.StationID.Val] = []*LocalizedString{&e.Name, &e.ShortName}
	}
	for _, e := range f.VehicleTypes {
		ret["vehicle_type:"+e.VehicleTypeID.Val] = []*LocalizedString{&e.Name, &e.Make, &e.Model}
	}
	for _, e := range f.Regions {
		ret["region:"+e.RegionID.Val] = []*LocalizedString{&e.Name}
	}
	for _, e := range f.Plans {
		ret["plan:"+e.PlanID.Val] = []*LocalizedString{&e.Name, &e.Description}
	}
	for _, e := range f.Alerts {
		ret["alert:"+e.AlertID.Val] = []*LocalizedString{&e.URL, &e.Summary, &e.Description}
	}
	return ret
}

// setLanguage tags f's untagged text, which is all of a 1.x/2.x file set's,
// with lang.
func setLanguage(f *GbfsFeed, lang string) {
	for _, fields := range textsByID(f) {
		for _, l := range fields {
			for i := range *l {
				if (*l)[i].Language == "" {
					(*l)[i].Language = lang
				}
			}
		}
	}
}

// addTranslations appends o's text to the entity with the same id in f. The
// first language decides which entities a system has.
func addTranslations(f *GbfsFeed, o *GbfsFeed) {
	dst := textsByID(f)
	for k, src := range textsByID(o) {
		d, ok := dst[k]
		if !ok {
			continue
		}
		for i := range src {
			*d[i] = append(*d[i], *src[i]...)
		}
	}
}
