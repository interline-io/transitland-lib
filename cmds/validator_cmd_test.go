package cmds

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
)

func TestValidatorCommand_DMFRAuth(t *testing.T) {
	const expectedKey = "test-api-key"
	zipPath := testpath.RelPath(filepath.Join("testdata", "gtfs-examples", "example.zip"))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("ApiKey") != expectedKey {
			http.Error(w, "missing or invalid ApiKey", http.StatusUnauthorized)
			return
		}
		http.ServeFile(w, r, zipPath)
	}))
	defer ts.Close()

	feed := dmfr.Feed{
		FeedID: "f-auth-test",
		Spec:   "gtfs",
		URLs:   dmfr.FeedUrls{StaticCurrent: ts.URL + "/gtfs.zip"},
		Authorization: dmfr.FeedAuthorization{
			Type:      "header",
			ParamName: "ApiKey",
		},
	}

	writeFiles := func(t *testing.T, secrets []dmfr.Secret) (string, string) {
		t.Helper()
		dir := t.TempDir()
		dmfrBody, _ := json.Marshal(struct {
			Feeds []dmfr.Feed `json:"feeds"`
		}{Feeds: []dmfr.Feed{feed}})
		dmfrPath := filepath.Join(dir, "registry.dmfr.json")
		if err := os.WriteFile(dmfrPath, dmfrBody, 0644); err != nil {
			t.Fatal(err)
		}
		secretsPath := ""
		if len(secrets) > 0 {
			secretsBody, _ := json.Marshal(struct {
				Secrets []dmfr.Secret `json:"secrets"`
			}{Secrets: secrets})
			secretsPath = filepath.Join(dir, "secrets.json")
			if err := os.WriteFile(secretsPath, secretsBody, 0644); err != nil {
				t.Fatal(err)
			}
		}
		return dmfrPath, secretsPath
	}

	cases := []struct {
		name        string
		secrets     []dmfr.Secret
		errContains string
	}{
		{
			name:    "valid key fetches and validates",
			secrets: []dmfr.Secret{{FeedID: feed.FeedID, Key: expectedKey}},
		},
		{
			name:        "missing secret errors before fetch",
			secrets:     nil,
			errContains: "no matching secret found",
		},
		{
			name:        "wrong key returns fetch error",
			secrets:     []dmfr.Secret{{FeedID: feed.FeedID, Key: "wrong-key"}},
			errContains: "401",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dmfrPath, secretsPath := writeFiles(t, tc.secrets)
			cmd := ValidatorCommand{
				Quiet:       true,
				DMFRFile:    dmfrPath,
				FeedID:      feed.FeedID,
				SecretsFile: secretsPath,
			}
			if err := cmd.Parse(nil); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			err := cmd.Run(context.Background())
			if tc.errContains == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tc.errContains)
			}
		})
	}
}
