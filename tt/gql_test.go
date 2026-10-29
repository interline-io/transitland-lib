package tt

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalGQL_Unencodable(t *testing.T) {
	for _, tc := range []struct {
		v    Float
		want string
	}{
		{NewFloat(1.5), "1.5"},
		{Float{}, "null"},
		{NewFloat(math.NaN()), "null"},
		{NewFloat(math.Inf(1)), "null"},
	} {
		var buf bytes.Buffer
		tc.v.MarshalGQL(&buf)
		assert.Equal(t, tc.want, buf.String())
	}
}
