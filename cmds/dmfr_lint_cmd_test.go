package cmds

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDmfrLintCommandMalformedJSON(t *testing.T) {
	ctx := context.TODO()
	tdir := t.TempDir()
	good := filepath.Join(tdir, "good.dmfr.json")
	bad := filepath.Join(tdir, "bad.dmfr.json")
	if err := os.WriteFile(bad, []byte(`{"feeds": [ this is not json ]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte(`{"feeds":[{"spec":"gtfs","id":"f-9q5-a","urls":{"static_current":"http://example.com/g.zip"}}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Run the formatter over it so the file is correctly formatted by
	// definition, rather than pinning a copy of the formatter's output here.
	fmtCmd := DmfrFormatCommand{Save: true}
	if err := fmtCmd.Parse([]string{good}); err != nil {
		t.Fatal(err)
	}
	if err := fmtCmd.Run(ctx); err != nil {
		t.Fatal(err)
	}

	// A file that will not parse must be reported, not panic, and must not stop
	// the remaining files from being checked.
	cmd := DmfrLintCommand{}
	if err := cmd.Parse([]string{bad, good}); err != nil {
		t.Fatal(err)
	}
	err := cmd.Run(ctx)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "bad.dmfr.json")
		assert.NotContains(t, err.Error(), "good.dmfr.json")
	}
}
