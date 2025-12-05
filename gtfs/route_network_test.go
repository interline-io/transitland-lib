package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestRouteNetwork_Errors(t *testing.T) {
	newRouteNetwork := func(fn func(*RouteNetwork)) *RouteNetwork {
		rn := &RouteNetwork{
			NetworkID: tt.NewKey("n1"),
			RouteID:   tt.NewKey("r1"),
		}
		if fn != nil {
			fn(rn)
		}
		return rn
	}

	tests := []struct {
		name           string
		rn             *RouteNetwork
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid route network",
			rn:             newRouteNetwork(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing network_id",
			rn: newRouteNetwork(func(rn *RouteNetwork) {
				rn.NetworkID = tt.Key{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:network_id"),
		},
		{
			name: "Missing route_id",
			rn: newRouteNetwork(func(rn *RouteNetwork) {
				rn.RouteID = tt.Key{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:route_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.rn)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestRouteNetwork_Methods(t *testing.T) {
	rn := &RouteNetwork{
		NetworkID: tt.NewKey("n1"),
		RouteID:   tt.NewKey("r1"),
	}

	if got := rn.Filename(); got != "route_networks.txt" {
		t.Errorf("Filename() = %v, want %v", got, "route_networks.txt")
	}
	if got := rn.TableName(); got != "gtfs_route_networks" {
		t.Errorf("TableName() = %v, want %v", got, "gtfs_route_networks")
	}
	if got := rn.DuplicateKey(); got != "network_id:'n1' route_id:'r1'" {
		t.Errorf("DuplicateKey() = %v, want %v", got, "network_id:'n1' route_id:'r1'")
	}
}
