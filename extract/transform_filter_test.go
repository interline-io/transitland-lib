package extract

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func TestTransformFilter_Filter(t *testing.T) {
	tests := []struct {
		name        string
		setupFilter func() *TransformFilter
		entity      tt.Entity
		checkResult func(t *testing.T, ent tt.Entity)
		shouldError bool
	}{
		{
			name: "no matching transform rules",
			setupFilter: func() *TransformFilter {
				return NewTransformFilter()
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("Test Agency"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "Test Agency" {
					t.Errorf("expected agency name to remain unchanged, got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "uppercase transform on agency name",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "agency_name", "uppercase")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("test agency"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "TEST AGENCY" {
					t.Errorf("expected 'TEST AGENCY', got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "lowercase transform on agency name",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "agency_name", "lowercase")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("TEST AGENCY"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "test agency" {
					t.Errorf("expected 'test agency', got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "trim transform on agency name",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "agency_name", "trim")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("  test agency  "),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "test agency" {
					t.Errorf("expected 'test agency', got '%s'", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "urlescape transform on agency name",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "agency_name", "urlescape")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("test agency & co"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "test+agency+%26+co" {
					t.Errorf("expected 'test+agency+%%26+co', got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "replace spaces with underscores transform",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "agency_name", "replace_spaces_with_underscores")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("agency1"),
				AgencyName: tt.NewString("test agency name"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "test_agency_name" {
					t.Errorf("expected 'test_agency_name', got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "wildcard entity matching",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "*", "agency_name", "uppercase")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:   tt.NewString("any_agency"),
				AgencyName: tt.NewString("test agency"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "TEST AGENCY" {
					t.Errorf("expected 'TEST AGENCY', got %s", agency.AgencyName.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "wildcard field matching",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("agency.txt", "agency1", "*", "uppercase")
				return tf
			},
			entity: &gtfs.Agency{
				AgencyID:    tt.NewString("agency1"),
				AgencyName:  tt.NewString("test agency"),
				AgencyPhone: tt.NewString("555-1234"),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				agency := ent.(*gtfs.Agency)
				if agency.AgencyName.Val != "TEST AGENCY" {
					t.Errorf("expected agency name 'TEST AGENCY', got %s", agency.AgencyName.Val)
				}
				if agency.AgencyPhone.Val != "555-1234" {
					t.Errorf("expected agency phone '555-1234', got %s", agency.AgencyPhone.Val)
				}
			},
			shouldError: false,
		},
		{
			name: "multiple field transforms on stop",
			setupFilter: func() *TransformFilter {
				tf := NewTransformFilter()
				tf.AddValue("stops.txt", "stop1", "stop_name", "uppercase")
				tf.AddValue("stops.txt", "stop1", "stop_desc", "trim")
				return tf
			},
			entity: &gtfs.Stop{
				StopID:   tt.NewString("stop1"),
				StopName: tt.NewString("main street"),
				StopDesc: tt.NewString("  bus stop  "),
			},
			checkResult: func(t *testing.T, ent tt.Entity) {
				stop := ent.(*gtfs.Stop)
				if stop.StopName.Val != "MAIN STREET" {
					t.Errorf("expected stop name 'MAIN STREET', got %s", stop.StopName.Val)
				}
				if stop.StopDesc.Val != "bus stop" {
					t.Errorf("expected stop desc 'bus stop', got '%s'", stop.StopDesc.Val)
				}
			},
			shouldError: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			tf := testCase.setupFilter()
			emap := tt.NewEntityMap()
			err := tf.Filter(testCase.entity, emap)

			if testCase.shouldError && err == nil {
				t.Error("expected error but got none")
			}
			if !testCase.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			testCase.checkResult(t, testCase.entity)
		})
	}
}

func TestTransformFilter_AddValue(t *testing.T) {
	tf := NewTransformFilter()

	// Test adding invalid transform function (should log error but not crash)
	tf.AddValue("test.txt", "entity1", "field1", "invalid_function")

	// The invalid function should not cause a panic or error in AddValue itself
	// (it logs an error and returns early, but AddValue doesn't return an error)

	// Verify that applying the filter with the invalid function doesn't crash
	agency := &gtfs.Agency{
		AgencyID:   tt.NewString("entity1"),
		AgencyName: tt.NewString("test"),
	}

	err := tf.Filter(agency, tt.NewEntityMap())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// The agency name should remain unchanged since the invalid function was rejected
	if agency.AgencyName.Val != "test" {
		t.Errorf("expected agency name to remain 'test', got %s", agency.AgencyName.Val)
	}
}

func TestTransformFilter_AddValuesFromFile(t *testing.T) {
	// Test with non-existent file
	tf := NewTransformFilter()
	err := tf.AddValuesFromFile("nonexistent.csv")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
