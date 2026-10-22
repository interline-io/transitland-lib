package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

func vehiclePositionAgency(t *testing.T, ctx context.Context, cfg model.Config, onestopId string) *model.Agency {
	agencies, err := cfg.Finder.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: &onestopId})
	if err != nil {
		t.Fatal(err)
	}
	if len(agencies) != 1 {
		t.Fatalf("got %d agencies for %s, expected 1", len(agencies), onestopId)
	}
	return agencies[0]
}

// vehiclePositionIDs returns the entity ids attributed to an agency, which is
// what the attribution rules below are about.
func vehiclePositionIDs(ctx context.Context, cfg model.Config, agency *model.Agency) []string {
	var ret []string
	for _, ent := range cfg.RTFinder.FindVehiclePositionsForAgency(ctx, agency, nil, nil) {
		ret = append(ret, ent.ID)
	}
	return ret
}

// A realtime feed serving one operator is that operator's, so every vehicle in
// it is attributed even when the message names no route.
func TestVehiclePositions_ExclusiveFeed(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: testconfig.BAVehiclePositionsRTJson()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		assert.ElementsMatch(t,
			[]string{"1001", "1002", "1003", "1004", "1005"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}

// Regional feeds are associated with dozens of operators. Once a feed is shared
// no operator owns it outright, so only vehicles naming one of the agency's
// routes are attributed to it.
func TestVehiclePositions_SharedFeed(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: testconfig.BAVehiclePositionsRTJson()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		if _, err := cfg.Adapter.DBX().ExecContext(ctx, `
			insert into current_operators_in_feed(feed_id, resolved_onestop_id, resolved_name)
			select id, 'o-9q9-someoneelse', 'Someone Else' from current_feeds where onestop_id = 'BA'`); err != nil {
			t.Fatal(err)
		}
		// 1001 names route 01 and 1002 names route 05; 1005 names route 99,
		// which is not BART's, and the rest name no route at all.
		assert.ElementsMatch(t,
			[]string{"1001", "1002"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}

// A feed version carrying more than one agency cannot hand every vehicle to
// each of them either.
func TestVehiclePositions_MultiAgencyFeedVersion(t *testing.T) {
	testconfig.ConfigTxRollback(t, testconfig.Options{RTJsons: testconfig.BAVehiclePositionsRTJson()}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		agency := vehiclePositionAgency(t, ctx, cfg, "o-9q9-bayarearapidtransit")
		if _, err := cfg.Adapter.DBX().ExecContext(ctx, `
			insert into gtfs_agencies(feed_version_id, agency_id, agency_name, agency_url, agency_timezone)
			select feed_version_id, 'OTHER', 'Other Agency', 'http://example.com', agency_timezone
			from gtfs_agencies where id = $1`, agency.ID); err != nil {
			t.Fatal(err)
		}
		assert.ElementsMatch(t,
			[]string{"1001", "1002"},
			vehiclePositionIDs(ctx, cfg, agency),
		)
	})
}
