package cmds

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadFVIDFile(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "feed_version_id header",
			content: "feed_version_id,sha1\n1\n2\n3\n",
			want:    []string{"1", "2", "3"},
		},
		{
			name:    "feed_version_id not first column",
			content: "onestop_id,feed_version_id\nf-a,10\nf-b,20\n",
			want:    []string{"10", "20"},
		},
		{
			name:    "no header, first column integer (header row is data)",
			content: "100\n200\n300\n",
			want:    []string{"100", "200", "300"},
		},
		{
			name:    "no header, extra columns",
			content: "100,f-a\n200,f-b\n",
			want:    []string{"100", "200"},
		},
		{
			name:    "non-numeric header without feed_version_id yields nothing",
			content: "onestop_id,id\nf-a,1\nf-b,2\n",
			want:    nil,
		},
		{
			name:    "empty file",
			content: "",
			want:    nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := filepath.Join(t.TempDir(), "fvids")
			if err := os.WriteFile(fn, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := readFVIDFile(fn)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
