package redate

import (
	"encoding/json"

	"github.com/interline-io/transitland-lib/ext"
)

func init() {
	e := func(args string) (ext.Extension, error) {
		opts := &redateOptions{}
		if err := json.Unmarshal([]byte(args), opts); err != nil {
			return nil, err
		}
		a, _ := opts.SourceDays.Int64()
		b, _ := opts.TargetDays.Int64()
		return NewRedateFilter(opts.SourceDate.Time, opts.TargetDate.Time, int(a), int(b))
	}
	ext.RegisterExtension("redate", e)
	ext.RegisterExtension("service_merge", func(string) (ext.Extension, error) {
		return NewServiceMergeFilter()
	})
}
