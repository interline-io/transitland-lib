package gbfs

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestLocalizedString_UnmarshalJSON(t *testing.T) {
	tcs := []struct {
		name   string
		json   string
		expect LocalizedString
	}{
		{"2.x string", `"Café \/ 1st"`, LocalizedString{{Text: "Café / 1st"}}},
		{"3.x array", `[{"text":"Gare","language":"fr"},{"text":"Station","language":"en"}]`, LocalizedString{{Text: "Gare", Language: "fr"}, {Text: "Station", Language: "en"}}},
		{"number", `12`, LocalizedString{{Text: "12"}}},
		{"null", `null`, nil},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var l LocalizedString
			assert.NoError(t, json.Unmarshal([]byte(tc.json), &l))
			assert.Equal(t, tc.expect, l)
		})
	}
}

// Every entity is tagged, including two that share an id; text that already
// has a language keeps it; and a null entry in a list is skipped rather than
// dereferenced.
func TestSetLanguage(t *testing.T) {
	f := &GbfsFeed{Plans: []*SystemPricingPlan{
		{PlanID: tt.NewString("p"), Name: LocalizedString{{Text: "A"}}},
		nil,
		{PlanID: tt.NewString("p"), Name: LocalizedString{{Text: "B"}, {Text: "Be", Language: "fr"}}},
	}}
	setLanguage(f, "en")
	assert.Equal(t, LocalizedString{{Text: "A", Language: "en"}}, f.Plans[0].Name)
	assert.Equal(t, LocalizedString{{Text: "B", Language: "en"}, {Text: "Be", Language: "fr"}}, f.Plans[2].Name)
}

// localizedFields returns every LocalizedString reachable from v.
func localizedFields(v reflect.Value) []*LocalizedString {
	var ret []*LocalizedString
	switch {
	case v.Type() == reflect.TypeOf(LocalizedString{}):
		ret = append(ret, v.Addr().Interface().(*LocalizedString))
	case v.Kind() == reflect.Pointer && !v.IsNil():
		ret = localizedFields(v.Elem())
	case v.Kind() == reflect.Slice:
		for i := range v.Len() {
			ret = append(ret, localizedFields(v.Index(i))...)
		}
	case v.Kind() == reflect.Struct:
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				ret = append(ret, localizedFields(v.Field(i))...)
			}
		}
	}
	return ret
}

// Tagging and merging reach every localized field, found by reflection so a
// field added to the model without a matching entry in eachText fails here.
func TestEachText_EveryField(t *testing.T) {
	system := func(lang string) *GbfsFeed {
		f := &GbfsFeed{
			SystemInformation:  &SystemInformation{},
			StationInformation: []*StationInformation{{StationID: tt.NewString("s")}},
			VehicleTypes:       []*VehicleType{{VehicleTypeID: tt.NewString("v")}},
			Regions:            []*SystemRegion{{RegionID: tt.NewString("r")}},
			Plans:              []*SystemPricingPlan{{PlanID: tt.NewString("p")}},
			Alerts:             []*SystemAlert{{AlertID: tt.NewString("a")}},
		}
		for _, l := range localizedFields(reflect.ValueOf(f)) {
			*l = LocalizedString{{Text: lang}}
		}
		setLanguage(f, lang)
		return f
	}
	f := system("en")
	addTranslations(f, system("fr"))
	fields := localizedFields(reflect.ValueOf(f))
	assert.Len(t, fields, 16)
	for _, l := range fields {
		assert.Equal(t, LocalizedString{{Text: "en", Language: "en"}, {Text: "fr", Language: "fr"}}, *l)
	}
}

// A translation joins the entity with the same id; one the first language lacks
// is dropped, and null entries on either side are skipped.
func TestAddTranslations(t *testing.T) {
	station := func(id string, name string, lang string) *StationInformation {
		return &StationInformation{StationID: tt.NewString(id), Name: LocalizedString{{Text: name, Language: lang}}}
	}
	f := &GbfsFeed{StationInformation: []*StationInformation{station("s1", "Station", "en"), nil}}
	o := &GbfsFeed{StationInformation: []*StationInformation{nil, station("s1", "Gare", "fr"), station("s2", "Ailleurs", "fr")}}
	addTranslations(f, o)
	assert.Equal(t, LocalizedString{{Text: "Station", Language: "en"}, {Text: "Gare", Language: "fr"}}, f.StationInformation[0].Name)
	assert.Len(t, f.StationInformation, 2)
}
