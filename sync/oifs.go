package sync

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
)

var nameTilde = "[-:&@/]"
var nameFilter = "[^[:alnum:]~><]"

// GeneratedOperatorOnestopID is the onestop id sync gives the operator it
// generates for an agency that has no onestop id of its own.
func GeneratedOperatorOnestopID(feedOnestopID, agencyName string) string {
	fsid := "unknown"
	if strings.HasPrefix(feedOnestopID, "f-") && len(feedOnestopID) > 2 {
		fsid = feedOnestopID[2:]
	}
	return fmt.Sprintf("o-%s-%s", fsid, filterName(agencyName))
}

// filterName .
func filterName(name string) string {
	re1 := regexp.MustCompile(nameTilde)
	re2 := regexp.MustCompile(nameFilter)
	return strings.ToLower(re2.ReplaceAllString(re1.ReplaceAllString(name, "~"), ""))
}

// getPlaces returns the resolved places of agency row id, as FormatAgencyPlaces
// formats them.
func getPlaces(ctx context.Context, atx tldb.Adapter, id int) (string, error) {
	places := []builders.AgencyPlace{}
	if err := atx.Select(ctx, &places, "select name,adm0name,adm1name,rank from tl_agency_places where agency_id = ? order by id", id); err != nil {
		return "", err
	}
	return FormatAgencyPlaces(places), nil
}

// updateOifs brings an operator's rows in line with its associated feeds, as
// PlanOperatorOifs plans, and says whether any row changed.
func updateOifs(ctx context.Context, atx tldb.Adapter, operator dmfr.Operator) (bool, error) {
	existing := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(ctx, &existing, "select * from current_operators_in_feed where operator_id = ?", operator.ID); err != nil {
		return false, err
	}
	feeds := map[string]FeedAgencies{}
	for _, oif := range operator.AssociatedFeeds {
		osid := oif.FeedOnestopID.Val
		if _, ok := feeds[osid]; ok {
			continue
		}
		feedID := 0
		if err := atx.Get(ctx, &feedID, "select id from current_feeds where onestop_id = ?", osid); err == sql.ErrNoRows {
			log.For(ctx).Info().Msgf("Warning: no feed for '%s'", osid)
			continue
		} else if err != nil {
			return false, err
		}
		agencies := []gtfs.Agency{}
		if err := atx.Select(ctx, &agencies, "select gtfs_agencies.* from gtfs_agencies inner join feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id where feed_states.feed_id = ?", feedID); err != nil {
			return false, err
		}
		feed := FeedAgencies{FeedID: feedID}
		for _, agency := range agencies {
			feed.Agencies = append(feed.Agencies, FeedAgency{Agency: agency})
		}
		feeds[osid] = feed
	}
	return writeOifPlan(ctx, atx, PlanOperatorOifs(operator, existing, feeds))
}

// feedAgencies returns the agencies of a feed's active version, with their
// onestop ids.
func feedAgencies(ctx context.Context, atx tldb.Adapter, feedID int) ([]FeedAgency, error) {
	q := atx.Sqrl().
		Select("gtfs_agencies.*", "feed_version_agency_onestop_ids.onestop_id as onestop_id").
		From("gtfs_agencies").
		Join("feed_states on feed_states.materialized_feed_version_id = gtfs_agencies.feed_version_id").
		JoinClause("left join feed_version_agency_onestop_ids on feed_version_agency_onestop_ids.entity_id = gtfs_agencies.agency_id and feed_version_agency_onestop_ids.feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"feed_states.feed_id": feedID})
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	agencies := []FeedAgency{}
	if err := atx.Select(ctx, &agencies, qstr, qargs...); err != nil {
		return nil, err
	}
	return agencies, nil
}

// writeOifPlan applies a plan to current_operators_in_feed, with each written
// row's places resolved from its agency, and says whether any row changed.
func writeOifPlan(ctx context.Context, atx tldb.Adapter, plan OifPlan) (bool, error) {
	updated := false
	for _, row := range plan.Refreshes {
		places, err := getPlaces(ctx, atx, row.AgencyRowID)
		if err != nil {
			return false, err
		}
		row.ResolvedPlaces.Set(places)
		changed, err := refreshOif(ctx, atx, row.ID, row.OperatorAssociatedFeed)
		if err != nil {
			return false, err
		}
		updated = updated || changed
	}
	for _, row := range plan.Inserts {
		places, err := getPlaces(ctx, atx, row.AgencyRowID)
		if err != nil {
			return false, err
		}
		row.ResolvedPlaces.Set(places)
		if _, err := atx.Insert(ctx, &row.OperatorAssociatedFeed); err != nil {
			return false, err
		}
		updated = true
	}
	if len(plan.Deletes) > 0 {
		updated = true
		if _, err := atx.Sqrl().Delete("current_operators_in_feed").Where(sq.Eq{"id": plan.Deletes}).ExecContext(ctx); err != nil {
			return false, err
		}
	}
	return updated, nil
}

// refreshOif writes oif's resolved fields to row id where they differ from what
// the row holds, and says whether any did.
func refreshOif(ctx context.Context, atx tldb.Adapter, id int, oif dmfr.OperatorAssociatedFeed) (bool, error) {
	// Compare in SQL: tt reads '' back as unset, so only the database can tell
	// NULL from ''. An unchanged row matches no condition and isn't written.
	q := atx.Sqrl().
		Update("current_operators_in_feed").
		Set("resolved_onestop_id", oif.ResolvedOnestopID).
		Set("resolved_name", oif.ResolvedName).
		Set("resolved_short_name", oif.ResolvedShortName).
		Set("resolved_places", oif.ResolvedPlaces).
		Set("resolved_gtfs_agency_id", oif.ResolvedGtfsAgencyID).
		Where(sq.Eq{"id": id}).
		Where(sq.Or{
			sq.Expr("resolved_onestop_id is distinct from ?", oif.ResolvedOnestopID),
			sq.Expr("resolved_name is distinct from ?", oif.ResolvedName),
			sq.Expr("resolved_short_name is distinct from ?", oif.ResolvedShortName),
			sq.Expr("resolved_places is distinct from ?", oif.ResolvedPlaces),
			sq.Expr("resolved_gtfs_agency_id is distinct from ?", oif.ResolvedGtfsAgencyID),
		})
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return false, err
	}
	r, err := atx.DBX().ExecContext(ctx, qstr, qargs...)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	return n > 0, err
}

// feedDeletedOperators returns the ids of the soft-deleted operators that have
// rows in a feed.
func feedDeletedOperators(ctx context.Context, atx tldb.Adapter, feedID int) ([]int, error) {
	q := atx.Sqrl().
		Select("distinct current_operators.id").
		From("current_operators").
		Join("current_operators_in_feed on current_operators_in_feed.operator_id = current_operators.id").
		Where(sq.Eq{"current_operators_in_feed.feed_id": feedID}).
		Where("current_operators.deleted_at is not null")
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	ids := []int{}
	if err := atx.Select(ctx, &ids, qstr, qargs...); err != nil {
		return nil, err
	}
	return ids, nil
}

// feedUpdateOifs brings a feed's generated rows in line with the agencies of its
// active version, as PlanFeedOifs plans, and says whether any row changed.
func feedUpdateOifs(ctx context.Context, atx tldb.Adapter, feed dmfr.Feed) (bool, error) {
	existing := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(ctx, &existing, "select * from current_operators_in_feed where feed_id = ?", feed.ID); err != nil {
		return false, err
	}
	deleted, err := feedDeletedOperators(ctx, atx, feed.ID)
	if err != nil {
		return false, err
	}
	agencies, err := feedAgencies(ctx, atx, feed.ID)
	if err != nil {
		return false, err
	}
	return writeOifPlan(ctx, atx, PlanFeedOifs(feed.ID, feed.FeedID, existing, agencies, deleted))
}
