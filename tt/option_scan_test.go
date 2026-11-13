package tt

import "testing"

// Scanning nothing unsets a value and clears what it held, so code reading
// Val doesn't see a value that's gone.
func TestOptionScanClears(t *testing.T) {
	for _, src := range []any{nil, ""} {
		v := NewString("Platform 1")
		if err := v.Scan(src); err != nil {
			t.Fatal(err)
		}
		if v.Valid || v.Val != "" {
			t.Errorf("Scan(%#v): got %#v, want unset and empty", src, v)
		}
	}
	n := NewInt(30)
	if err := n.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if n.Valid || n.Val != 0 {
		t.Errorf("Scan(nil): got %#v, want unset and zero", n)
	}
}
