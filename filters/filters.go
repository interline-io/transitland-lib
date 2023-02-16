package filters

import (
	"github.com/interline-io/transitland-lib/ext"
)

func init() {
	ext.RegisterExtension("Redate", func(args string) (ext.Extension, error) { return newRedateFilterFromJson(args) })
	ext.RegisterExtension("Prefix", func(args string) (ext.Extension, error) { return newPrefixFilterFromJson(args) })
	ext.RegisterExtension("ApplyDefaultAgency", func(string) (ext.Extension, error) { return &ApplyDefaultAgencyFilter{}, nil })
	ext.RegisterExtension("ApplyParentTimezone", func(string) (ext.Extension, error) { return &ApplyParentTimezoneFilter{}, nil })
	ext.RegisterExtension("BasicRouteType", func(string) (ext.Extension, error) { return &BasicRouteTypeFilter{}, nil })
	ext.RegisterExtension("NormalizeTimezone", func(string) (ext.Extension, error) { return &NormalizeTimezoneFilter{}, nil })
}
