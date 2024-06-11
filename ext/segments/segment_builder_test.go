package segments

import (
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func newMockCopier(url string) (*copier.Copier, *direct.Writer, error) {
	reader, err := tlcsv.NewReader(url)
	if err != nil {
		return nil, nil, err
	}
	writer := direct.NewWriter()
	cp, err := copier.NewCopier(reader, writer, copier.Options{})
	if err != nil {
		return nil, nil, err
	}
	return cp, writer, nil
}

func TestSegmentBuilder(t *testing.T) {
	cp, writer, err := newMockCopier("RG.zip")
	_ = writer
	if err != nil {
		t.Fatal(err)
	}
	e := NewSegmentBuilder()
	cp.AddExtension(e)
	cpr := cp.Copy()
	if cpr.WriteError != nil {
		t.Fatal(err)
	}
}
