package gql

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/testdata"
)

func setupGbfs(ctx context.Context, gbf model.GbfsFinder) error {
	// Setup
	sourceFeedId := "gbfs-test"
	ts := httptest.NewServer(gbfs.NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer ts.Close()
	opts := gbfs.Options{}
	opts.FeedURL = fmt.Sprintf("%s/%s", ts.URL, "gbfs.json")
	feeds, _, err := gbfs.Fetch(ctx, nil, opts)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		key := fmt.Sprintf("%s:%s", sourceFeedId, feed.SystemInformation.Language.Val)
		gbf.AddData(ctx, key, feed)
	}
	return nil
}

func TestGbfsBikeResolver(t *testing.T) {
	testcases := []testcase{
		{
			name: "basic",
			query: `{
				bikes(where: {near:{lon: -122.396445, lat:37.793250, radius:100}}) {
				  bike_id
				}
			}`,
			selector:     "bikes.#.bike_id",
			selectExpect: []string{"2e09a0ed99c8ad32cca516661618645e"},
		},
		{
			name: "feed",
			query: `{
				bikes(where: {near:{lon: -122.396445, lat:37.793250, radius:100}}) {
				  bike_id
				  feed {
					system_information {
						name
					}
				  }
				}
			}`,
			selector:     "bikes.#.feed.system_information.name",
			selectExpect: []string{"Bay Wheels"},
		},
		{
			name: "limit 5",
			query: `{
				bikes(limit:5, where: {near:{lon: -122.396445, lat:37.793250, radius:1000}}) {
				  bike_id
				}
			}`,
			selector:     "bikes.#.bike_id",
			selectExpect: []string{"0cbf9b08f8b71a6362e20c8173c071a6", "1682088b2335fa5365610e6d299fde2d", "1bc913bf913729a147458cd6b2f91773", "1d61a000cb330f6c260fc439d29b20ab", "21667e59d3c6bc814b6716d87621ddde"},
		},
		{
			name: "limit 1",
			query: `{
				bikes(limit:1, where: {near:{lon: -122.396445, lat:37.793250, radius:1000}}) {
				  bike_id
				}
			}`,
			selector:     "bikes.#.bike_id",
			selectExpect: []string{"0cbf9b08f8b71a6362e20c8173c071a6"},
		},
	}
	c, cfg := newTestClient(t)
	setupGbfs(context.Background(), cfg.GbfsFinder)
	queryTestcases(t, c, testcases)
}

func TestGbfsStationResolver(t *testing.T) {
	testcases := []testcase{
		{
			name: "basic",
			query: `{
				docks(where: {near: {lon: -121.908666, lat: 37.336289, radius: 100}}) {
				  station_id
				  address
				  capacity
				  contact_phone
				  cross_street
				  is_charging_station
				  is_valet_station
				  is_virtual_station
				  lat
				  lon
				  name
				  parking_hoop
				  parking_type
				  post_code
				  rental_methods
				  short_name
				  station_area
				}
			  }
			`,
			selector:     "docks.#.station_id",
			selectExpect: []string{"d75591d7-080d-46cb-8ada-0fbe6af676fc"},
		},
		{
			name: "feed",
			query: `{
				docks(where: {near:{lon: -121.908666, lat:37.336289, radius:100}}) {
				  station_id
				  feed {
					system_information {
						name
					}
				  }
				}
			}`,
			selector:     "docks.#.feed.system_information.name",
			selectExpect: []string{"Bay Wheels"},
		},
		{
			name: "region",
			query: `{
				docks(where: {near: {lon: -121.908666, lat: 37.336289, radius: 100}}) {
				  station_id
				  region {
					name
					region_id
				  }
				}
			  }
			  `,
			selector:     "docks.#.region.name",
			selectExpect: []string{"San Jose"},
		},
		{
			name: "calendars",
			query: `{
				docks(where: {near: {lon: -121.908666, lat: 37.336289, radius: 100}}) {
				  station_id
				  feed {
					calendars {
					  end_day
					  end_month
					  end_year
					  start_day
					  start_month
					  start_year
					}
				  }
				}
			  }`,
			selector:     "docks.0.feed.calendars.0.end_month",
			selectExpect: []string{"12"},
		},
		{
			name: "status",
			query: `{
				docks(where: {near: {lon: -121.908666, lat: 37.336289, radius: 100}}) {
				  station_id
				  status {
					is_installed
					is_renting
					is_returning
					last_reported
					num_bikes_available
					num_bikes_disabled
					num_docks_available
					num_docks_disabled
					station_id
				  }
				}
			  }
			`,
			selector:     "docks.0.status.num_bikes_available",
			selectExpect: []string{"11"},
		},
		{
			name: "limit 5",
			query: `{
				docks(limit: 5, where: {near: {lon: -121.908666, lat: 37.336289, radius: 1000}}) {
				  station_id
				}
			  }
			  `,
			selector:     "docks.#.station_id",
			selectExpect: []string{"27045384-791c-4519-8087-fce2f7c48a69", "28988488-fb74-4bbc-9e69-613698b2dd8c", "2c7560e6-62c6-4403-8b97-8016471948b5", "3ebc4f3f-2941-47cd-a173-83f01a91bf57", "a96032c0-9ff2-4fbe-8f03-6b3f9816947d"},
		},
		{
			name: "limit 1",
			query: `{
				docks(limit: 1, where: {near: {lon: -121.908666, lat: 37.336289, radius: 1000}}) {
				  station_id
				}
			  }
			  `,
			selector:     "docks.#.station_id",
			selectExpect: []string{"27045384-791c-4519-8087-fce2f7c48a69"},
		},
	}
	c, cfg := newTestClient(t)
	setupGbfs(context.Background(), cfg.GbfsFinder)
	queryTestcases(t, c, testcases)
}
