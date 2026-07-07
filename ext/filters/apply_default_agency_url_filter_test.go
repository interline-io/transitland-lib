package filters

import (
	"testing"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestApplyDefaultAgencyURLFilter_Filter(t *testing.T) {
	f, err := NewApplyDefaultAgencyURLFilter("https://example.com/feeds/f-test")
	if err != nil {
		t.Fatal(err)
	}

	missing := &gtfs.Agency{AgencyID: tt.NewString("a1")}
	if err := f.Filter(missing, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "https://example.com/feeds/f-test", missing.AgencyURL.Val, "missing agency_url should be filled")

	present := &gtfs.Agency{AgencyID: tt.NewString("a2"), AgencyURL: tt.NewUrl("http://example.com")}
	if err := f.Filter(present, tt.NewEntityMap()); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "http://example.com", present.AgencyURL.Val, "existing agency_url should be left unchanged")
}

func TestApplyDefaultAgencyURLFilter_Validation(t *testing.T) {
	_, err := NewApplyDefaultAgencyURLFilter("")
	assert.Error(t, err, "empty url should be rejected")

	_, err = NewApplyDefaultAgencyURLFilter("not-a-url")
	assert.Error(t, err, "invalid url should be rejected")
}

// TestApplyDefaultAgencyURLFilter_Extension exercises the registered extension as it
// is invoked from the CLI, e.g. --ext=ApplyDefaultAgencyURL:url=https://example.com.
func TestApplyDefaultAgencyURLFilter_Extension(t *testing.T) {
	name, args, err := ext.ParseExtensionArgs("ApplyDefaultAgencyURL:url=https://example.com/feeds/f-test")
	if err != nil {
		t.Fatal(err)
	}
	e, err := ext.GetExtension(name, args)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := e.(*ApplyDefaultAgencyURLFilter)
	if !ok {
		t.Fatalf("got %T", e)
	}
	assert.Equal(t, "https://example.com/feeds/f-test", f.url)
}
