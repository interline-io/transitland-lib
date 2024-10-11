package tlutil

// func TestEntityErrors(t *testing.T) {
// 	reader, err := tlcsv.NewReader(testutil.RelPath("testdata/bad-entities"))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if err := reader.Open(); err != nil {
// 		t.Error(err)
// 	}
// 	testutil.AllEntities(reader, func(ent tt.Entity) {
// 		t.Run(fmt.Sprintf("%s:%s", ent.Filename(), ent.EntityID()), func(t *testing.T) {
// 			var errs []error
// 			if extEnt, ok := ent.(tt.EntityWithErrors); ok {
// 				errs = extEnt.Errors()
// 			}
// 			expecterrs := testutil.GetExpectErrors(ent)
// 			testutil.CheckErrors(expecterrs, errs, t)
// 		})
// 	})
// 	if err := reader.Close(); err != nil {
// 		t.Error(err)
// 	}
// }
