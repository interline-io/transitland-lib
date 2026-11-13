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

// A value it can't convert clears it too.
func TestOptionScanClearsOnError(t *testing.T) {
	b := NewBool(true)
	if err := b.Scan("maybe"); err == nil {
		t.Fatal("expected an error")
	}
	if b.Valid || b.Val {
		t.Errorf("Scan(\"maybe\"): got %#v, want unset and false", b)
	}
}
