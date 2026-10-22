package sync

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

type agencyPlace struct {
	Name     tt.String
	Adm1name tt.String
	Adm0name tt.String
}

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

func getPlaces(ctx context.Context, atx tldb.Adapter, id int) (string, error) {
	agencyPlaces := []agencyPlace{}
	if err := atx.Select(ctx, &agencyPlaces, "select name,adm0name,adm1name from tl_agency_places where agency_id = ? AND rank > 0.2 order by rank desc, id", id); err != nil {
		return "", err
	}
	// Places stay in rank order, each listed once, so the same rows always give
	// the same string.
	places := []string{}
	for _, a := range agencyPlaces {
		suba := []string{}
		if a.Name.Valid {
			suba = append(suba, a.Name.Val)
		}
		if a.Adm1name.Valid {
			suba = append(suba, a.Adm1name.Val)
		}
		if a.Adm0name.Valid {
			suba = append(suba, a.Adm0name.Val)
		}
		place := strings.Join(suba, ", ")
		if len(suba) > 0 && !slices.Contains(places, place) {
			places = append(places, place)
		}
	}
	return strings.Join(places, " / "), nil
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
		agencies, err := feedAgencies(ctx, atx, feedID)
		if err != nil {
			return false, err
		}
		feeds[osid] = FeedAgencies{FeedID: feedID, Agencies: agencies}
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
		Join("current_feeds on current_feeds.id = feed_states.feed_id").
		JoinClause("left join feed_version_agency_onestop_ids on feed_version_agency_onestop_ids.entity_id = gtfs_agencies.agency_id and feed_version_agency_onestop_ids.feed_version_id = gtfs_agencies.feed_version_id").
		Where(sq.Eq{"current_feeds.id": feedID})
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
		places, err := getPlaces(ctx, atx, row.AgencyID)
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
		places, err := getPlaces(ctx, atx, row.AgencyID)
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

// deleteHiddenOifs removes a feed's rows of soft-deleted operators, which the
// finders already hide, and says whether there were any.
func deleteHiddenOifs(ctx context.Context, atx tldb.Adapter, feedID int) (bool, error) {
	q := atx.Sqrl().
		Delete("current_operators_in_feed").
		Where(sq.Eq{"feed_id": feedID}).
		Where("operator_id in (select id from current_operators where deleted_at is not null)")
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

// feedUpdateOifs brings a feed's generated rows in line with the agencies of its
// active version, as PlanFeedOifs plans, and says whether any row changed.
func feedUpdateOifs(ctx context.Context, atx tldb.Adapter, feed dmfr.Feed) (bool, error) {
	// Rows of soft-deleted operators go first, so none of them keeps an agency
	// from its generated row.
	hidden, err := deleteHiddenOifs(ctx, atx, feed.ID)
	if err != nil {
		return false, err
	}
	existing := []dmfr.OperatorAssociatedFeed{}
	if err := atx.Select(ctx, &existing, "select * from current_operators_in_feed where feed_id = ?", feed.ID); err != nil {
		return false, err
	}
	agencies, err := feedAgencies(ctx, atx, feed.ID)
	if err != nil {
		return false, err
	}
	updated, err := writeOifPlan(ctx, atx, PlanFeedOifs(feed.ID, feed.FeedID, existing, agencies))
	return hidden || updated, err
}
