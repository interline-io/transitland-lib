package cmds

import (
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/importer"
)

func TestImportResultsError(t *testing.T) {
	ok := ImportCommandResult{Result: importer.Result{FeedVersionImport: dmfr.FeedVersionImport{Success: true}}}
	unsuccessful := ImportCommandResult{Result: importer.Result{FeedVersionImport: dmfr.FeedVersionImport{Success: false, ExceptionLog: "bad data"}}}
	fatal := ImportCommandResult{FatalError: errors.New("required minimum entities not met")}

	tests := []struct {
		name            string
		results         []ImportCommandResult
		fail            bool
		continueOnError bool
		wantErr         bool
		wantFailed      int
	}{
		{name: "all ok", results: []ImportCommandResult{ok, ok}, wantErr: false, wantFailed: 0},
		{name: "fatal fails by default", results: []ImportCommandResult{ok, fatal}, wantErr: true, wantFailed: 1},
		{name: "fatal tolerated with continue-on-error", results: []ImportCommandResult{ok, fatal}, continueOnError: true, wantErr: false, wantFailed: 1},
		{name: "unsuccessful is not fatal by default", results: []ImportCommandResult{ok, unsuccessful}, wantErr: false, wantFailed: 1},
		{name: "unsuccessful fails with --fail", results: []ImportCommandResult{ok, unsuccessful}, fail: true, wantErr: true, wantFailed: 1},
		{name: "continue-on-error overrides --fail", results: []ImportCommandResult{unsuccessful, fatal}, fail: true, continueOnError: true, wantErr: false, wantFailed: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err, failed := importResultsError(tc.results, tc.fail, tc.continueOnError)
			if tc.wantErr != (err != nil) {
				t.Errorf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
			if failed != tc.wantFailed {
				t.Errorf("got failedCount=%d, want %d", failed, tc.wantFailed)
			}
		})
	}
}
