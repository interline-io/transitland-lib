package dbfinder

import (
	"context"
	"os"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/dbutil"
	sq "github.com/irees/squirrel"
)

// Temporary instrumentation for interline-io/tlv2#354: measure how often the
// AllowPreviousOnestopIds resolution actually does work beyond an exact lookup,
// and how much of that work a geohash+name search would reproduce. Enable by
// setting TL_LOG_ALLOWPREV_PROBE to any non-empty value; it is off otherwise and
// adds no overhead. When on, it runs one extra read-only query per bare-osid
// AllowPrev request and emits structured log lines ("allowprev_probe" /
// "allowprev_probe_meaningful"). Safe to delete once the measurement is done.
var allowPrevProbeEnabled = os.Getenv("TL_LOG_ALLOWPREV_PROBE") != ""

// Match the parameters the candidate search-key resolver would use, so
// search_would_match reflects what switching to search would actually return.
const (
	allowPrevProbeRadiusM    = 100.0
	allowPrevProbeSimilarity = 0.4
)

type allowPrevProbeRow struct {
	RequestedOsid    string `db:"requested_osid"`
	StopDbID         int    `db:"stop_db_id"`
	StopGtfsID       string `db:"stop_gtfs_id"`
	CurrentOsid      string `db:"current_osid"`
	Differs          bool   `db:"differs"`
	SearchWouldMatch bool   `db:"search_would_match"`
}

// allowPrevProbeQuery builds the diagnostic query. Args bind in order: the osid
// VALUES rows (req CTE), then the radius and similarity threshold in the
// search_would_match expression. squirrel emits CTE args before column args, so
// placeholder positions line up. The hist CTE collapses the per-feed-version
// history to one row per (requested osid, entity_id, feed) before joining to
// stops, mirroring the production CTE's DISTINCT ON and avoiding fan-out for
// pairs present in many feed versions.
func allowPrevProbeQuery(osids []string) sq.SelectBuilder {
	ph := make([]string, len(osids))
	args := make([]interface{}, len(osids))
	for i, osid := range osids {
		if i == 0 {
			ph[i] = "(?::text)"
		} else {
			ph[i] = "(?)"
		}
		args[i] = osid
	}
	return sq.StatementBuilder.
		Select(
			"hist.requested_osid",
			"gs.id as stop_db_id",
			"gs.stop_id as stop_gtfs_id",
			"coalesce(cur.onestop_id,'') as current_osid",
			"(cur.onestop_id IS DISTINCT FROM hist.requested_osid) as differs",
		).
		Column(sq.Expr(
			"(ST_DWithin(gs.geometry, ST_PointFromGeoHash(split_part(hist.requested_osid,'-',2))::geography, ?) "+
				"and (split_part(coalesce(cur.onestop_id,''),'-',3) = split_part(hist.requested_osid,'-',3) "+
				"or word_similarity(split_part(hist.requested_osid,'-',3), split_part(coalesce(cur.onestop_id,''),'-',3)) > ?)) as search_would_match",
			allowPrevProbeRadiusM, allowPrevProbeSimilarity,
		)).
		Distinct().
		WithCTE(sq.CTE{
			Alias:      "req",
			ColumnList: []string{"requested_osid"},
			Expression: sq.Expr("VALUES "+strings.Join(ph, ", "), args...),
		}).
		WithCTE(sq.CTE{
			Alias: "hist",
			Expression: sq.Expr("SELECT DISTINCT req.requested_osid, fvs.entity_id, fv.feed_id " +
				"FROM req " +
				"JOIN feed_version_stop_onestop_ids fvs ON fvs.onestop_id = req.requested_osid " +
				"JOIN feed_versions fv ON fv.id = fvs.feed_version_id"),
		}).
		From("hist").
		Join("gtfs_stops gs on gs.stop_id = hist.entity_id").
		Join("feed_versions cur_fv on cur_fv.id = gs.feed_version_id and cur_fv.feed_id = hist.feed_id").
		Join("feed_states fs on fs.materialized_feed_version_id = gs.feed_version_id").
		JoinClause("left join feed_version_stop_onestop_ids cur on cur.entity_id = gs.stop_id and cur.feed_version_id = gs.feed_version_id")
}

// probeAllowPrev resolves the requested Onestop IDs via the same (feed, stop_id)
// continuity the AllowPrev query uses, then classifies each resolved current
// stop:
//   - differs: the stop's current Onestop ID is not the requested one, i.e. an
//     exact lookup would have missed it and AllowPrev did real work.
//   - search_would_match: a geohash-point + name search would also have found
//     this stop, so a search-key resolver would not lose it.
//
// "meaningful" work (logged per row) is differs AND NOT search_would_match: the
// resolutions that only stored history can recover. Stops with no current
// Onestop ID (a stats data gap, where differs is ill-defined) are counted
// separately as n_no_current and excluded from both, so they do not inflate the
// meaningful count. It measures raw continuity resolution (no permission/license
// filtering) against active feed versions.
func (f *Finder) probeAllowPrev(ctx context.Context, osids []string) {
	if len(osids) == 0 {
		return
	}
	q := allowPrevProbeQuery(osids)

	var rows []allowPrevProbeRow
	if err := dbutil.Select(ctx, f.db, q, &rows); err != nil {
		log.For(ctx).Warn().Err(err).Msg("allowprev_probe query failed")
		return
	}

	differs, meaningful, noCurrent := 0, 0, 0
	for _, r := range rows {
		if r.CurrentOsid == "" {
			noCurrent++
			continue
		}
		if r.Differs {
			differs++
			if !r.SearchWouldMatch {
				meaningful++
			}
		}
	}
	log.For(ctx).Info().
		Int("n_requested", len(osids)).
		Int("n_resolved", len(rows)).
		Int("n_differs", differs).
		Int("n_meaningful", meaningful).
		Int("n_no_current", noCurrent).
		Strs("requested_osids", osids).
		Msg("allowprev_probe")
	for _, r := range rows {
		if r.CurrentOsid != "" && r.Differs && !r.SearchWouldMatch {
			log.For(ctx).Info().
				Str("requested_osid", r.RequestedOsid).
				Str("current_osid", r.CurrentOsid).
				Str("stop_gtfs_id", r.StopGtfsID).
				Int("stop_db_id", r.StopDbID).
				Msg("allowprev_probe_meaningful")
		}
	}
}
