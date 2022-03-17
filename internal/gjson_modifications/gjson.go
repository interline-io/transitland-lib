package gjson_modifications

import (
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

// code from https://github.com/tidwall/gjson/issues/190#issuecomment-736801395
func AddSortModifier () {
		gjson.AddModifier("sort", func(json, arg string) string {
			r := gjson.Get(json, gjson.Get(arg, "array").String())
			if !r.IsArray() || r.Index == 0 {
				return json
			}
			orderBy := gjson.Get(arg, "orderBy").String()
			caseSensitive := gjson.Get(arg, "caseSensitive").Bool()
			desc := gjson.Get(arg, "desc").Bool()
			arr := r.Array()
			sort.SliceStable(arr, func(i, j int) bool {
				a := arr[i].Get(orderBy)
				b := arr[j].Get(orderBy)
				if desc {
					return b.Less(a, caseSensitive)
				}
				return a.Less(b, caseSensitive)
			})
			var sorted strings.Builder
			sorted.WriteByte('[')
			for i, item := range arr {
				if i > 0 {
					sorted.WriteByte(',')
				}
				sorted.WriteString(item.Raw)
			}
			sorted.WriteByte(']')
			return json[:r.Index] + sorted.String() + json[r.Index+len(r.Raw):]
		})
	}