package tags

import (
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
	Renamed  string `csv:"new_name" alias:"old_name"`
	Collides string `csv:"collides" alias:"new_name"`
}

func TestCache_GetStructTagMap_Alias(t *testing.T) {
	c := NewCache(reflectx.NewMapperFunc("csv", ToSnakeCase))
	stg := c.GetStructTagMap(&aliasEntity{})
	renamed, ok := stg["new_name"]
	if !ok {
		t.Fatalf("did not get field for tag 'new_name'")
	}
	if renamed.Alias {
		t.Errorf("expected 'new_name' to be a field, not an alias")
	}
	old, ok := stg["old_name"]
	if !ok {
		t.Fatalf("did not get field for alias 'old_name'")
	}
	if !old.Alias {
		t.Errorf("expected 'old_name' to be marked as an alias")
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
