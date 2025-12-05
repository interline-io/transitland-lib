package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestNetwork_Errors(t *testing.T) {
	newNetwork := func(fn func(*Network)) *Network {
		network := &Network{
			NetworkID:   tt.NewString("n1"),
			NetworkName: tt.NewString("Network 1"),
		}
		if fn != nil {
			fn(network)
		}
		return network
	}

	tests := []struct {
		name           string
		network        *Network
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid network",
			network:        newNetwork(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing network_id",
			network: newNetwork(func(n *Network) {
				n.NetworkID = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:network_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.network)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestNetwork_Methods(t *testing.T) {
	network := &Network{
		NetworkID:   tt.NewString("n1"),
		NetworkName: tt.NewString("Network 1"),
	}
	network.ID = 123

	if got := network.EntityID(); got != "123" {
		t.Errorf("EntityID() = %v, want %v", got, "123")
	}
	network.ID = 0
	if got := network.EntityID(); got != "n1" {
		t.Errorf("EntityID() = %v, want %v", got, "n1")
	}

	if got := network.EntityKey(); got != "n1" {
		t.Errorf("EntityKey() = %v, want %v", got, "n1")
	}
	if got := network.Filename(); got != "networks.txt" {
		t.Errorf("Filename() = %v, want %v", got, "networks.txt")
	}
	if got := network.TableName(); got != "gtfs_networks" {
		t.Errorf("TableName() = %v, want %v", got, "gtfs_networks")
	}
}
