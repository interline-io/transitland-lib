package extract

import (
	"testing"

	"github.com/interline-io/transitland-lib/adapters/tlcsv"
	"github.com/interline-io/transitland-lib/internal/graph"
	"github.com/interline-io/transitland-lib/internal/testutil"
)

type mss = map[string][]string
type node = graph.Node

func nn(filename, eid string) node {
	return *graph.NewNode(filename, eid)
}

func TestExtract_Filter_BART(t *testing.T) {
	em := NewMarker()
	reader, err := tlcsv.NewReader(testutil.ExampleFeedBART.URL)
	if err != nil {
		t.Error(err)
	}
	em.fm["trips.txt"] = []string{"3792107WKDY"}
	em.Filter(reader)
	if !em.IsMarked("stops.txt", "MCAR") {
		t.Error("expected stop MCAR")
	}
	if em.IsMarked("stops.txt", "FTVL") {
		t.Error("expected no stop FTVL")
	}
	if !em.IsMarked("agency.txt", "BART") {
		t.Error("expected agency BART")
	}
	if em.IsMarked("routes.txt", "03") {
		t.Error("expected no route 03")
	}
}

func TestExtract_Bbox(t *testing.T) {
	em := NewMarker()
	em.bbox = "-122.276929,37.794923,-122.259099,37.834413"
	reader, err := tlcsv.NewReader(testutil.ExampleFeedBART.URL)
	if err != nil {
		t.Error(err)
	}
	em.Filter(reader)
	if !em.IsMarked("stops.txt", "MCAR") {
		t.Error("expected stop MCAR")
	}
	if !em.IsMarked("stops.txt", "12TH") {
		t.Error("expected stop 12TH")
	}
	if !em.IsMarked("stops.txt", "LAKE") {
		t.Error("expected stop LAKE")
	}
	if em.IsMarked("stops.txt", "FTVL") {
		t.Error("expected no stop FTVL")
	}
	if em.IsMarked("stops.txt", "ROCK") {
		t.Error("expected no stop ROCK")
	}
	if !em.IsMarked("agency.txt", "BART") {
		t.Error("expected agency BART")
	}
}

func TestExtract_Filter_ExampleFeed(t *testing.T) {
	reader, err := tlcsv.NewReader(testutil.RelPath("testdata/extract-examples"))
	if err != nil {
		t.Error(err)
	}
	testcases := []struct {
		name    string
		filter  mss
		exclude mss
		nodes   []node
	}{
		{
			"agency.txt:OK",
			mss{"agency.txt": {"OK"}},
			nil,
			[]node{
				nn("calendar.txt", "OK"),
				nn("shapes.txt", "OK1"),
				nn("stops.txt", "OK1"),
				nn("stops.txt", "OK2"),
				nn("agency.txt", "OK"),
				nn("routes.txt", "OK1"),
				nn("trips.txt", "OK1"),
			},
		},
		{
			"routes.txt:DTA1",
			mss{"routes.txt": {"DTA1"}},
			nil,
			[]node{
				nn("stops.txt", "DTA3"),
				nn("routes.txt", "DTA1"),
				nn("trips.txt", "DTA1"),
				nn("agency.txt", "DTA"),
				nn("calendar.txt", "DTA"),
				nn("shapes.txt", "DTA1"),
				nn("stops.txt", "DTA1"),
				nn("stops.txt", "DTA2"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"),
			},
		},
		{
			"trips.txt:DTA_STOP4",
			mss{"trips.txt": {"DTA_STOP4"}},
			nil,
			[]node{
				nn("trips.txt", "DTA_STOP4"),
				nn("routes.txt", "DTA1"),
				nn("calendar.txt", "DTA"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"),
				nn("agency.txt", "DTA"),
			},
		},
		{
			"stops.txt:UNUSED2",
			mss{"stops.txt": {"UNUSED2"}},
			nil,
			[]node{nn("stops.txt", "UNUSED2")},
		},
		{
			"agency.txt:UNUSED",
			mss{"agency.txt": {"UNUSED"}},
			nil,
			[]node{
				nn("agency.txt", "UNUSED"),
				nn("routes.txt", "UNUSED"),
			},
		},
		// exclude marker tests
		{
			"include agency.txt:DTA exclude routes.txt:DTA1",
			mss{"agency.txt": {"DTA"}},
			mss{"routes.txt": {"DTA1"}},
			// DTA1 and all DTA1 trips should be excluded
			[]node{
				nn("agency.txt", "DTA"),
				nn("calendar.txt", "DTA"),
				nn("routes.txt", "DTA2"),
				nn("shapes.txt", "DTA1"),
				nn("stops.txt", "DTA1"),
				nn("stops.txt", "DTA2"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"),
				nn("trips.txt", "DTA2"),
			},
		},
		{
			"include agency.txt:DTA exclude trips.txt:DTA_STOP4",
			mss{"agency.txt": {"DTA"}},
			mss{"trips.txt": {"DTA_STOP4"}},
			// All of agency DTA should be included except for DTA_STOP4 trip
			// TODO: should stop DTA4 be included?
			[]node{
				nn("agency.txt", "DTA"),
				nn("calendar.txt", "DTA"),
				nn("routes.txt", "DTA1"),
				nn("routes.txt", "DTA2"),
				nn("shapes.txt", "DTA1"),
				nn("stops.txt", "DTA1"),
				nn("stops.txt", "DTA2"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"),
				nn("trips.txt", "DTA1"),
				nn("trips.txt", "DTA2"),
			},
		},
		{
			"include routes.txt:DTA1 exclude trips.txt:DTA_STOP4",
			mss{"routes.txt": {"DTA1"}},
			mss{"trips.txt": {"DTA_STOP4"}},
			// TODO: stop DTA4 should probably not be included,
			// as it's only reachable from route DTA1 / trip DTA_STOP4
			[]node{
				nn("agency.txt", "DTA"),
				nn("calendar.txt", "DTA"),
				nn("routes.txt", "DTA1"),
				nn("shapes.txt", "DTA1"),
				nn("stops.txt", "DTA1"),
				nn("stops.txt", "DTA2"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"), // should not be included?
				nn("trips.txt", "DTA1"),
			},
		},
		{
			"include routes.txt:DTA1 exclude stops.txt:DTA4",
			mss{"routes.txt": {"DTA1"}},
			mss{"stops.txt": {"DTA4"}},
			// excluding stop DTA4 also excludes trip DTA_STOP4
			[]node{
				nn("agency.txt", "DTA"),
				nn("calendar.txt", "DTA"),
				nn("routes.txt", "DTA1"),
				nn("shapes.txt", "DTA1"),
				nn("stops.txt", "DTA1"),
				nn("stops.txt", "DTA2"),
				nn("stops.txt", "DTA3"),
				nn("trips.txt", "DTA1"),
			},
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			em := NewMarker()
			em.fm = testcase.filter
			em.ex = testcase.exclude
			em.Filter(reader)
			count := 0
			for _, v := range em.found {
				if v {
					count += 1
				}
			}
			if count != len(testcase.nodes) {
				t.Logf("Found nodes:")
				for n, ok := range em.found {
					if ok {
						t.Logf(`nn("%s", "%s"),`+"\n", n.Filename, n.ID)
					}
				}
				t.Errorf("got %d nodes expect %d nodes", len(em.found), len(testcase.nodes))
			}
			for _, n := range testcase.nodes {
				ok := em.IsMarked(n.Filename, n.ID)
				if !ok {
					t.Errorf("expected %s %s but not found", n.Filename, n.ID)
				}
			}
		})
	}
}
