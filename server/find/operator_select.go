package find

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/jmoiron/sqlx"
)

func FindOperators(atx sqlx.Ext, limit *int, after *int, ids []int, where *model.OperatorFilter) (ents []*model.Operator, err error) {
	q := OperatorSelect(limit, after, ids, where)
	MustSelect(model.DB, q, &ents)
	return ents, nil
}

func OperatorSelect(limit *int, after *int, ids []int, where *model.OperatorFilter) sq.SelectBuilder {
	q := quickSelectOrder("tl_mv_active_agency_operators", limit, after, ids, "")
	if where != nil {
		if where.Search != nil && len(*where.Search) > 0 {
			rank, wc := tsQuery(*where.Search)
			q = q.Column(rank).Where(wc)
		}
		if where.Merged != nil && *where.Merged == true {
			q = q.Distinct().Options("on (onestop_id)")
		} else {
			q = q.OrderBy("id asc")
		}
		if where.FeedVersionSha1 != nil {
			q = q.Where(sq.Eq{"feed_version_sha1": *where.FeedVersionSha1})
		}
		if where.FeedOnestopID != nil {
			q = q.Where(sq.Eq{"feed_onestop_id": *where.FeedOnestopID})
		}
		if where.AgencyID != nil {
			q = q.Where(sq.Eq{"agency_id": *where.AgencyID})
		}
		if where.OnestopID != nil {
			q = q.Where(sq.Eq{"onestop_id": where.OnestopID})
		}
	}
	return q
}
