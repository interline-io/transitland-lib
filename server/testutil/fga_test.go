package testutil

import "testing"

func TestFGAServer(t *testing.T) {
	endpoint := FGAServer(t)
	if endpoint == "" {
		t.Fatal("expected non-empty endpoint")
	}
}
