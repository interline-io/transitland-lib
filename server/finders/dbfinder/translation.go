package dbfinder

import (
	"context"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	sq "github.com/irees/squirrel"
)

// TranslationsByFeedVersionIDs returns the translations of each feed version.
func (f *Finder) TranslationsByFeedVersionIDs(ctx context.Context, limit *int, after *model.Cursor, where *model.TranslationFilter, keys []int) ([][]*model.Translation, error) {
	var ents []*model.Translation
	q := lateralWrap(translationSelect(limit, after, f.PermFilter(ctx), where), "feed_versions", "id", "gtfs_translations", "feed_version_id", keys)
	err := dbutil.Select(ctx, f.db, q, &ents)
	return arrangeGroup(keys, ents, func(ent *model.Translation) int { return ent.FeedVersionID }), queryErr(ctx, err)
}

func translationSelect(limit *int, after *model.Cursor, permFilter *model.PermFilter, where *model.TranslationFilter) sq.SelectBuilder {
	q := sq.StatementBuilder.Select(
		"gtfs_translations.id",
		"gtfs_translations.feed_version_id",
		"gtfs_translations.table_name",
		"gtfs_translations.field_name",
		"gtfs_translations.language",
		"gtfs_translations.translation",
		"gtfs_translations.record_id",
		"gtfs_translations.record_sub_id",
		"gtfs_translations.field_value",
		"feed_versions.sha1 AS feed_version_sha1",
		"current_feeds.onestop_id AS feed_onestop_id",
	).From("gtfs_translations").
		Join("feed_versions ON feed_versions.id = gtfs_translations.feed_version_id").
		Join("current_feeds ON current_feeds.id = feed_versions.feed_id")
	if after != nil && after.Valid && after.ID > 0 {
		q = q.Where(sq.Gt{"gtfs_translations.id": after.ID})
	}
	if where != nil {
		if len(where.Ids) > 0 {
			q = q.Where(In("gtfs_translations.id", where.Ids))
		}
		if where.TableName != nil {
			q = q.Where(sq.Eq{"gtfs_translations.table_name": *where.TableName})
		}
		if where.FieldName != nil {
			q = q.Where(sq.Eq{"gtfs_translations.field_name": *where.FieldName})
		}
		if where.Language != nil {
			q = q.Where(sq.ILike{"gtfs_translations.language": dbutil.EscapeLike(*where.Language, false, false)})
		}
		if where.RecordID != nil {
			q = q.Where(sq.Eq{"gtfs_translations.record_id": *where.RecordID})
		}
	}
	q = joinImported(q)
	q = pfJoinCheckFv(q, permFilter)
	return q.OrderBy("gtfs_translations.id ASC").Limit(finderCheckLimit(limit))
}
