package tags

import (
	"fmt"
	"testing"
)

type testEntity struct {
	req    string `csv:"req" required:"true"`
	number int    `csv:"number" min:"-10" max:"10"`
	url    string `csv:"url" validator:"url"`
}

func (ent *testEntity) Filename() string {
	return "ok.txt"
}

func Test_newStructTagMap(t *testing.T) {
	ent := testEntity{}
	m := newStructTagMap(&ent)
	//
	a := m["req"]
	if a.Csv != "req" {
		t.Error("expected: csv = req")
	}
	if a.Required != true {
		t.Error("expected: required = true")
	}
	//
	b := m["number"]
	if b.Csv != "number" {
		t.Error("expected: csv = number")
	}
	if b.Min != -10 {
		t.Error("expected: min = -10")
	}
	if b.Max != 10 {
		t.Error("expected max = 10")
	}
	//
	c := m["url"]
	if c.Csv != "url" {
		t.Error("expected: csv = url")
	}
	if c.Validator != "url" {
		t.Error("expected: validator = url")
	}
}

func Test_getStructTagMap(t *testing.T) {
	// TODO: test lock?
	ent := testEntity{}
	tk := fmt.Sprintf("*%T", ent)
	if _, ok := structTagMapCache[tk]; ok {
		t.Error("already cached")
	}
	m := GetStructTagMap(&ent)
	if _, ok := structTagMapCache[tk]; !ok {
		t.Error("failed to cache")
	}
	a := m["req"]
	if a.Csv != "req" {
		t.Error("expected: csv = req")
	}
}
