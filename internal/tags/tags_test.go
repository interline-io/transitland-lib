package tags

// type testEntity struct {
// 	Req        string `csv:"req,required"`
// 	Number     string `csv:"number"`
// 	DefaultTag string
// }

// func (ent *testEntity) Filename() string {
// 	return "ok.txt"
// }

// func Test_newStructTagMap(t *testing.T) {
// 	ent := testEntity{}
// 	m := newStructTagMap(&ent)
// 	a := m["req"]
// 	if a.Csv != "req" {
// 		t.Error("expected: csv = req")
// 	}
// 	if a.Required != true {
// 		t.Error("expected: required = true")
// 	}
// 	//
// 	b := m["number"]
// 	if b.Csv != "number" {
// 		t.Error("expected: csv = number")
// 	}
// 	c := m["default_tag"]
// 	if c.Csv != "default_tag" {
// 		t.Error("expected DefaultTag as default_tag")
// 	}

// }

// func Test_getStructTagMap(t *testing.T) {
// 	ent := testEntity{}
// 	tk := fmt.Sprintf("*%T", ent)
// 	if _, ok := structTagMapCache[tk]; ok {
// 		t.Error("already cached")
// 	}
// 	m := GetStructTagMap(&ent)
// 	if _, ok := structTagMapCache[tk]; !ok {
// 		t.Error("failed to cache")
// 	}
// 	a := m["req"]
// 	if a.Csv != "req" {
// 		t.Error("expected: csv = req")
// 	}
// }
