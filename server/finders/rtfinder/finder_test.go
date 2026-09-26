package rtfinder

import (
	"testing"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMakeAlert_ActivePeriod(t *testing.T) {
	a := &pb.Alert{
		ActivePeriod: []*pb.TimeRange{
			{Start: proto.Uint64(100), End: proto.Uint64(200)},
			{Start: proto.Uint64(300)},
			{End: proto.Uint64(400)},
		},
	}
	got := makeAlert(a).ActivePeriod
	if !assert.Len(t, got, 3) {
		return
	}
	check := func(idx int, start, end *int) {
		assert.Equal(t, start, got[idx].Start, "active_period[%d].start", idx)
		assert.Equal(t, end, got[idx].End, "active_period[%d].end", idx)
	}
	check(0, intp(100), intp(200))
	check(1, intp(300), nil)
	check(2, nil, intp(400))
}

func intp(v int) *int {
	return &v
}
