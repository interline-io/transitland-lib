package tags

import (
	"testing"

	"github.com/jmoiron/sqlx/reflectx"
)

type testEntity struct {
	Req         string `csv:"req,required"`
	Number      int    `csv:"this_is_a_number"`
	DefaultTag  string
	NotTagged   string `csv:"-"`
	notExported string
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
