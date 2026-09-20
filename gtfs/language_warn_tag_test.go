package gtfs

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

// A tt.Language field reports an unrecognized region, script or variant as an
// advisory rather than a rejection, because the language is still usable. The
// "warn" tag is the only thing that keeps such a report from dropping the
// entity and everything referencing it, so every language field has to carry
// it. Adding one without the tag would turn a typo like "en-EN" back into lost
// data.
func TestLanguageFieldsAreWarnTagged(t *testing.T) {
	languageType := reflect.TypeOf(tt.Language{})
	checked := 0
	for _, ent := range AllEntities() {
		entType := reflect.TypeOf(ent).Elem()
		for i := 0; i < entType.NumField(); i++ {
			field := entType.Field(i)
			if field.Type != languageType {
				continue
			}
			checked++
			csvTag := field.Tag.Get("csv")
			if opts := strings.Split(csvTag, ","); !slices.Contains(opts[1:], "warn") {
				t.Errorf("%s.%s is a tt.Language without the warn tag: csv:%q", entType.Name(), field.Name, csvTag)
			}
		}
	}
	if checked == 0 {
		t.Fatal("found no tt.Language fields, so this test is no longer checking anything")
	}
}
