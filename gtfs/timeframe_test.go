package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestTimeframe_Errors(t *testing.T) {
	newTimeframe := func(fn func(*Timeframe)) *Timeframe {
		tf := &Timeframe{
			TimeframeGroupID: tt.NewString("tf_group"),
			StartTime:        tt.NewSeconds(3600), // 01:00:00
			EndTime:          tt.NewSeconds(7200), // 02:00:00
			ServiceID:        tt.NewKey("service_1"),
		}
		if fn != nil {
			fn(tf)
		}
		return tf
	}

	tests := []struct {
		name           string
		timeframe      *Timeframe
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid timeframe",
			timeframe:      newTimeframe(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing timeframe_group_id",
			timeframe: newTimeframe(func(tf *Timeframe) {
				tf.TimeframeGroupID = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:timeframe_group_id"),
		},
		{
			name: "Missing service_id",
			timeframe: newTimeframe(func(tf *Timeframe) {
				tf.ServiceID = tt.Key{}
			}),
			expectedErrors: PE("RequiredFieldError:service_id"),
		},
		{
			name: "Valid without start_time",
			timeframe: newTimeframe(func(tf *Timeframe) {
				tf.StartTime = tt.Seconds{}
			}),
			expectedErrors: nil,
		},
		{
			name: "Valid without end_time",
			timeframe: newTimeframe(func(tf *Timeframe) {
				tf.EndTime = tt.Seconds{}
			}),
			expectedErrors: nil,
		},
		{
			name: "Invalid end_time < start_time",
			timeframe: newTimeframe(func(tf *Timeframe) {
				tf.StartTime = tt.NewSeconds(7200)
				tf.EndTime = tt.NewSeconds(3600)
			}),
			expectedErrors: PE("InvalidFieldError:end_time"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.timeframe)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
