package dmfr

import (
	"testing"
)

func Test_AddCrosswalkIDs(t *testing.T) {
	fakeTransitlandRegistry, err := ParseString(`{
		"feeds": [
			{
				"id": "f-eth-tenerife~titsa",
				"spec": "gtfs",
				"url": "http://www.titsa.com/Google_transit.zip"
			}
		]
	}`)
	if err != nil {
		t.Error(err)
	}
	fakeTransitFeedsRegistry, err := ParseString(`{
		"feeds": [
			{
				"id": "transportes-interurbanos-de-tenerife/1058",
				"spec": "gtfs",
				"url": "http://www.titsa.com/Google_transit.zip"
			}
		]
	}`)
	if err != nil {
		t.Error(err)
	}
	v := map[string]*Registry{"transitfeeds": fakeTransitFeedsRegistry}
	fakeTransitlandRegistry = AddCrosswalkIDs(fakeTransitlandRegistry, v)
	if len(fakeTransitFeedsRegistry.Feeds) != 1 {
		t.Error("oops, there should be 1 feed in fakeTransitlandRegistry after it has been crosswalked with  fakeTransitFeedsRegistry")
	}
	if fakeTransitlandRegistry.Feeds[0].OtherIDs["transitfeeds"] != "transportes-interurbanos-de-tenerife/1058" {
		t.Error("didn't assign the crosswalk'ed ID to the feed")
	}
}
