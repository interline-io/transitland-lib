package cmds

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	sq "github.com/irees/squirrel"
	"github.com/spf13/pflag"
)

// FeedVersionArgs is the shared CLI surface for commands that operate on feed
// versions. Feed version ids are the primary input, given as positional arguments
// (fvid1 fvid2 ...) or read from --fvid-file. --fvid and --fv-sha1(-file) are
// deprecated compatibility flags. There is no feed-level selection: resolve feeds
// to their versions first.
type FeedVersionArgs struct {
	FVIDs      []string
	FVSHA1     []string
	fvidfile   string
	fvsha1file string
}

func (a *FeedVersionArgs) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&a.fvidfile, "fvid-file", "", "Read feed version IDs from a csv-like file (the feed_version_id column if present, else the first column; a non-numeric header row is ignored)")
	fl.StringSliceVar(&a.FVIDs, "fvid", nil, "Feed version ID")
	fl.StringSliceVar(&a.FVSHA1, "fv-sha1", nil, "Select feed versions by SHA1")
	fl.StringVar(&a.fvsha1file, "fv-sha1-file", "", "Read feed version SHA1s from a file, one per line")
	_ = fl.MarkDeprecated("fvid", "pass feed version ids as arguments instead")
	_ = fl.MarkDeprecated("fv-sha1", "resolve to feed version ids and pass them as arguments instead")
	_ = fl.MarkDeprecated("fv-sha1-file", "resolve to feed version ids and pass them as arguments instead")
}

// Parse reads positional feed version ids and the --fvid-file / --fv-sha1-file lists.
func (a *FeedVersionArgs) Parse(args []string) error {
	a.FVIDs = append(a.FVIDs, tlcli.NewNArgs(args).Args()...)
	if a.fvidfile != "" {
		ids, err := readFVIDFile(a.fvidfile)
		if err != nil {
			return err
		}
		a.FVIDs = appendNonEmpty(a.FVIDs, ids)
	}
	if a.fvsha1file != "" {
		lines, err := tlcli.ReadFileLines(a.fvsha1file)
		if err != nil {
			return err
		}
		a.FVSHA1 = appendNonEmpty(a.FVSHA1, lines)
	}
	return nil
}

// Empty reports whether no feed version was selected.
func (a *FeedVersionArgs) Empty() bool {
	return len(a.FVIDs) == 0 && len(a.FVSHA1) == 0
}

// SelectIDs resolves the selectors to existing feed version ids, warning about any
// explicitly requested id or sha1 that was not found.
func (a *FeedVersionArgs) SelectIDs(ctx context.Context, adapter tldb.Adapter) ([]int, error) {
	if a.Empty() {
		return nil, nil
	}
	or := sq.Or{}
	if len(a.FVIDs) > 0 {
		or = append(or, sq.Eq{"feed_versions.id": a.FVIDs})
	}
	if len(a.FVSHA1) > 0 {
		or = append(or, sq.Eq{"feed_versions.sha1": a.FVSHA1})
	}
	q := adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Where(or).
		OrderBy("feed_versions.id")
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	fvids := []int{}
	if err := adapter.Select(ctx, &fvids, qstr, qargs...); err != nil {
		return nil, err
	}
	if expected := explicitSelectorCount(a.FVIDs, a.FVSHA1); expected > len(fvids) {
		log.For(ctx).Warn().
			Int("requested", expected).
			Int("found", len(fvids)).
			Msg("some feed versions were not found and will be skipped")
	}
	return fvids, nil
}

// excludeLiveVersions drops any feed version that is its feed's active or
// materialized version, for callers that must never touch a live version.
func excludeLiveVersions(ctx context.Context, adapter tldb.Adapter, fvids []int) ([]int, error) {
	if len(fvids) == 0 {
		return fvids, nil
	}
	q := adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		Join("feed_states ON feed_states.feed_id = feed_versions.feed_id").
		Where(sq.Eq{"feed_versions.id": fvids}).
		Where("feed_versions.id IN (feed_states.active_feed_version_id, feed_states.materialized_feed_version_id)")
	qstr, qargs, err := q.ToSql()
	if err != nil {
		return nil, err
	}
	live := []int{}
	if err := adapter.Select(ctx, &live, qstr, qargs...); err != nil {
		return nil, err
	}
	if len(live) == 0 {
		return fvids, nil
	}
	liveSet := make(map[int]bool, len(live))
	for _, id := range live {
		liveSet[id] = true
	}
	out := make([]int, 0, len(fvids))
	for _, id := range fvids {
		if !liveSet[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// explicitSelectorCount returns the number of distinct explicit selectors when
// exactly one of fvids/sha1s is set. With both or neither set it returns 0: with
// both, they are OR'd at the SQL level and a "requested" count isn't meaningful;
// with neither, no explicit selection was made.
func explicitSelectorCount(fvids, sha1s []string) int {
	var sel []string
	switch {
	case len(fvids) > 0 && len(sha1s) == 0:
		sel = fvids
	case len(sha1s) > 0 && len(fvids) == 0:
		sel = sha1s
	default:
		return 0
	}
	seen := map[string]bool{}
	for _, v := range sel {
		seen[v] = true
	}
	return len(seen)
}

func appendNonEmpty(dst, lines []string) []string {
	for _, line := range lines {
		if line != "" {
			dst = append(dst, line)
		}
	}
	return dst
}

// readFVIDFile reads feed version ids from a file with tlcsv.ReadRows. If the header
// has a feed_version_id column, that column is used. Otherwise, if the first field of
// the header row parses as an integer, the "header" was really the first data row, so
// the first column is used and that value is kept as the first id. A real,
// non-numeric header with no feed_version_id column yields nothing.
func readFVIDFile(fn string) ([]string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ids []string
	col := -1
	if err := tlcsv.ReadRows(f, func(row tlcsv.Row) {
		if col < 0 {
			if i, ok := row.Hindex["feed_version_id"]; ok {
				col = i
			} else if len(row.Header) > 0 {
				if h := strings.TrimSpace(row.Header[0]); h != "" {
					if _, err := strconv.Atoi(h); err == nil {
						col = 0
						ids = append(ids, h)
					}
				}
			}
		}
		if col >= 0 && col < len(row.Row) {
			if v := strings.TrimSpace(row.Row[col]); v != "" {
				ids = append(ids, v)
			}
		}
	}); err != nil && err != io.EOF { // io.EOF: empty file, no rows
		return nil, err
	}
	return ids, nil
}
