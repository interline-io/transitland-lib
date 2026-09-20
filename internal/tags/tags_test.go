package tags

import (
	"sort"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx/reflectx"
)

type testEntity struct {
	Req        string `csv:"req,required"`
	Number     int    `csv:"this_is_a_number"`
	DefaultTag string
	NotTagged  string `csv:"-"`
}

// aliasEntity covers the two ways an alias can interact with a real field name:
// Renamed reads an old name that is no longer written, and Collides declares an
// alias that is another field's actual name.
type aliasEntity struct {
	Renamed  string `csv:"new_name,alias=old_name,required" enum:"1,2"`
	Collides string `csv:"collides,alias=new_name"`
}

// warnEntity covers the "warn" option, which moves a field's value checks from
// errors to warnings. It is independent of "required": a field can be both.
type warnEntity struct {
	Advisory string `csv:"advisory,warn"`
	Both     string `csv:"both,required,warn"`
	Renamed  string `csv:"renamed,alias=old_name,warn"`
	Plain    string `csv:"plain,required"`
}

func TestCache_GetStructTagMap_Warn(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	stg := c.GetStructTagMap(&warnEntity{})
	for _, tc := range []struct {
		field    string
		warn     bool
		required bool
	}{
		{"advisory", true, false},
		{"both", true, true},
		{"renamed", true, false},
		{"plain", false, true},
		// Validation tags stay on the field's own entry, so a value is checked
		// once, under the name the file actually uses.
		{"old_name", false, false},
	} {
		fi, ok := stg[tc.field]
		if !ok {
			t.Fatalf("did not get field for tag '%s'", tc.field)
		}
		if fi.Warn != tc.warn {
			t.Errorf("got Warn=%v for '%s', expected %v", fi.Warn, tc.field, tc.warn)
		}
		if fi.Required != tc.required {
			t.Errorf("got Required=%v for '%s', expected %v", fi.Required, tc.field, tc.required)
		}
	}
}

func TestCache_GetWarnFields(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	var got []string
	for _, fi := range c.GetWarnFields(&warnEntity{}) {
		got = append(got, fi.Name)
	}
	// An alias is another name for a field that is checked under its own name;
	// it must not make that field look like a second warn field here.
	sort.Strings(got)
	expect := []string{"advisory", "both", "renamed"}
	if len(got) != len(expect) {
		t.Fatalf("got warn fields %v, expected %v", got, expect)
	}
	for i := range got {
		if got[i] != expect[i] {
			t.Errorf("got %s at position %d, expected %s", got[i], i, expect[i])
		}
	}
	// A type with no warn tagged field is the common case: the warning pass
	// asks about every entity in a copy.
	if warnFields := c.GetWarnFields(&testEntity{}); len(warnFields) > 0 {
		t.Errorf("got warn fields %v for a type with none", warnFields)
	}

}

// Readers take the type snapshot without a lock, so one running while another
// goroutine parses a type and publishes a new snapshot has to see a complete
// answer either way. Run this with -race.
func TestCache_Concurrent(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	cases := []struct {
		ent       interface{}
		warnCount int
	}{
		{&warnEntity{}, 3},
		{&testEntity{}, 0},
		{&aliasEntity{}, 0},
	}
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tc := cases[i%len(cases)]
			for j := 0; j < 200; j++ {
				if got := len(c.GetWarnFields(tc.ent)); got != tc.warnCount {
					t.Errorf("got %d warn fields for %T, expected %d", got, tc.ent, tc.warnCount)
					return
				}
				if got := len(c.GetStructTagMap(tc.ent)); got == 0 {
					t.Errorf("got an empty field map for %T", tc.ent)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestCache_GetStructTagMap_Alias(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	stg := c.GetStructTagMap(&aliasEntity{})
	renamed, ok := stg["new_name"]
	if !ok {
		t.Fatalf("did not get field for tag 'new_name'")
	}
	if renamed.IsAlias() {
		t.Errorf("expected 'new_name' to be a field, not an alias")
	}
	old, ok := stg["old_name"]
	if !ok {
		t.Fatalf("did not get field for alias 'old_name'")
	}
	if !old.IsAlias() {
		t.Errorf("expected 'old_name' to be marked as an alias")
	}
	if old.Name != "old_name" || old.AliasOf != "new_name" {
		t.Errorf("got alias entry Name=%q AliasOf=%q, expected 'old_name' and 'new_name'", old.Name, old.AliasOf)
	}
	// Validation tags stay on the field's own entry, so a value is checked once,
	// under the name the file actually uses.
	if old.Required {
		t.Errorf("expected the alias entry not to carry the field's required tag")
	}
	if len(old.EnumValues) > 0 {
		t.Errorf("expected the alias entry not to carry the field's enum values, got %v", old.EnumValues)
	}
	// An alias must never displace a field that owns the name. Compare by index
	// rather than pointer: the alias entry is a copy.
	if old.Index[0] != renamed.Index[0] {
		t.Errorf("alias 'old_name' resolves to field %v, expected %v", old.Index, renamed.Index)
	}
	if renamed.Index[0] != 0 {
		t.Errorf("'new_name' resolves to field %v, expected the field that declares it", renamed.Index)
	}
}

func TestCache_GetHeader_Alias(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	header, _ := c.GetHeader(&aliasEntity{})
	expect := []string{"new_name", "collides"}
	if len(header) != len(expect) {
		t.Fatalf("got header %v, expected %v", header, expect)
	}
	for i := range header {
		if header[i] != expect[i] {
			t.Errorf("got %s at position %d, expected %s", header[i], i, expect[i])
		}
	}
}

func TestCache_GetStructTagMap(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	ent := &testEntity{}
	stg := c.GetStructTagMap(ent)
	if a, ok := stg["req"]; !ok {
		t.Errorf("did not get field for tag 'req'")
	} else if !a.Required {
		t.Errorf("expected 'req' to be tagged as required")
	}
	if _, ok := stg["this_is_a_number"]; !ok {
		t.Errorf("did not get field for tag 'this_is_a_number'")
	}
	if _, ok := stg["default_tag"]; !ok {
		t.Errorf("did not get field for tag 'default_tag'")
	}
	if _, ok := stg["default_tag"]; !ok {
		t.Errorf("did not get field for tag 'default_tag'")
	}
	if _, ok := stg["not_tagged"]; ok {
		t.Errorf("got unexpected tag 'not_tagged'")
	}
	if _, ok := stg["not_exported"]; ok {
		t.Errorf("got unexpected tag 'not_exported'")
	}
}

func TestCache_GetHeader(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	ent := &testEntity{}
	header, _ := c.GetHeader(ent)
	expect := []string{"req", "this_is_a_number", "default_tag"}
	if len(header) != len(expect) {
		t.Errorf("got header %v expected %v", header, expect)
	}
	for i := range header {
		if header[i] != expect[i] {
			t.Errorf("got %s which did not match expected header %s", header[i], expect[i])
		}
	}
}

func TestCache_GetInsert(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	ent := &testEntity{Req: "ok", Number: 123, DefaultTag: "default"}
	header, _ := c.GetHeader(ent)
	values, _ := c.GetInsert(ent, header)
	if len(values) != 3 {
		t.Errorf("expected 3 items in values")
	}
	if values[0].(string) != "ok" {
		t.Errorf("got '%v', expected 'ok'", values[0])
	}
	if values[1].(int) != 123 {
		t.Errorf("got '%v', expected 123", values[1])
	}
	if values[2].(string) != "default" {
		t.Errorf("got '%v', expected 'default'", values[2])
	}
}
