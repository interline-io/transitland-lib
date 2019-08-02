package extract

import (
	"testing"

	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/graph"
)

type mss = map[string][]string
type node = graph.Node

func nn(filename, eid string) node {
	return *graph.NewNode(filename, eid)
}

func TestExtract_Filter_BART(t *testing.T) {
	em := NewMarker()
	reader, err := gtcsv.NewReader("../../testdata/external/bart.zip")
	if err != nil {
		t.Error(err)
	}
	fm := map[string][]string{}
	fm["trips.txt"] = []string{"3792107WKDY"}
	em.Filter(reader, fm)
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

func TestExtract_Filter_ExampleFeed(t *testing.T) {
	reader, err := gtcsv.NewReader("../../testdata/extract-examples")
	if err != nil {
		t.Error(err)
	}
	testcases := []struct {
		name   string
		filter mss
		nodes  []node
	}{
		{
			"agency.txt:OK",
			mss{"agency.txt": {"OK"}},
			[]node{
				nn("calendar.txt", "OK"),
				nn("shapes.txt", "OK1"),
				nn("stops.txt", "OK1"),
				nn("stops.txt", "OK2"),
				nn("agency.txt", "OK"),
				nn("routes.txt", "OK1"),
				nn("trips.txt", "OK1"),
			},
		}, {
			"routes.txt:DTA1",
			mss{"routes.txt": {"DTA1"}},
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
		}, {
			"trips.txt:DTA_STOP4",
			mss{"trips.txt": {"DTA_STOP4"}},
			[]node{
				nn("trips.txt", "DTA_STOP4"),
				nn("routes.txt", "DTA1"),
				nn("calendar.txt", "DTA"),
				nn("stops.txt", "DTA3"),
				nn("stops.txt", "DTA4"),
				nn("agency.txt", "DTA"),
			},
		}, {
			"stops.txt:UNUSED2",
			mss{"stops.txt": {"UNUSED2"}},
			[]node{nn("stops.txt", "UNUSED2")},
		}, {
			"agency.txt:UNUSED",
			mss{"agency.txt": {"UNUSED"}},
			[]node{
				nn("agency.txt", "UNUSED"),
				nn("routes.txt", "UNUSED"),
			},
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			em := NewMarker()
			em.Filter(reader, testcase.filter)
			// for n := range em.found {
			// 	fmt.Printf("%#v\n", n)
			// }
			if len(em.found) != len(testcase.nodes) {
				t.Errorf("got %d nodes expect %d nodes", len(em.found), len(testcase.nodes))
			}
			for _, n := range testcase.nodes {
				if ok := em.IsMarked(n.Filename, n.ID); !ok {
					t.Errorf("expected %s %s but not found", n.Filename, n.ID)
				}
			}
		})
	}
}
