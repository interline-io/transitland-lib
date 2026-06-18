package dbfinder

import (
	"strings"

	sq "github.com/irees/squirrel"
	"github.com/mmcloughlin/geohash"
)

// Tunables for resolving previous/looked-up stop Onestop IDs as search keys.
// A stop Onestop ID is "s-<geohash>-<filtered_name>"; rather than storing every
// historical (feed_version, stop_id, onestop_id) association, we decode the id
// back into its content claim — a point (the geohash) and a name — and search
// current stops for a match. See interline-io/tlv2#354.
var (
	// Radius in meters around the decoded geohash point. Absorbs coordinate
	// jitter between feed versions (the geohash-10 cell is ~1m; observed
	// real-world renames move stops a few tens of meters).
	stopOnestopSearchRadius = 100.0
	// pg_trgm word_similarity threshold for matching the requested name
	// component against the stop's current name component. Both sides are
	// filterName() output, so the comparison is in the same normalized space.
	stopOnestopSearchSimilarity = 0.4
)

// parseStopOnestopID splits "s-<geohash>-<name>" into the geohash center point
// and the name component. The name never contains '-' because filterName maps
// '-' to '~', so a 3-way split is exact. Returns ok=false for anything that is
// not a well-formed stop Onestop ID (wrong prefix, invalid geohash), so route/
// operator ids or garbage do not decode to a bogus point and search there.
func parseStopOnestopID(osid string) (lat float64, lng float64, name string, ok bool) {
	parts := strings.SplitN(osid, "-", 3)
	if len(parts) != 3 || parts[0] != "s" {
		return 0, 0, "", false
	}
	if err := geohash.Validate(parts[1]); err != nil {
		return 0, 0, "", false
	}
	lat, lng = geohash.DecodeCenter(parts[1])
	return lat, lng, parts[2], true
}

// stopOnestopSearchCTE builds a small VALUES CTE of parsed inputs, one row per
// requested Onestop ID, so that any number of inputs can be resolved with a
// single spatial join (one GiST index probe per row) instead of per-input
// queries. Returns ok=false if no input parsed.
func stopOnestopSearchCTE(osids []string) (sq.CTE, bool) {
	var rows []string
	var args []interface{}
	for _, osid := range osids {
		lat, lng, name, ok := parseStopOnestopID(osid)
		if !ok {
			continue
		}
		if len(rows) == 0 {
			// Cast the first row so Postgres can resolve the VALUES column
			// types; an all-placeholder VALUES otherwise errors with
			// "could not determine data type of parameter".
			rows = append(rows, "(?::text,?::text,?::double precision,?::double precision)")
		} else {
			rows = append(rows, "(?,?,?,?)")
		}
		args = append(args, osid, name, lng, lat)
	}
	if len(rows) == 0 {
		return sq.CTE{}, false
	}
	return sq.CTE{
		Alias:        "stop_onestop_search",
		ColumnList:   []string{"input_onestop_id", "name_component", "lng", "lat"},
		Materialized: true,
		Expression:   sq.Expr("VALUES "+strings.Join(rows, ", "), args...),
	}, true
}
