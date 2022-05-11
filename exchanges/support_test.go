package exchange

import "testing"

func TestIsSupported(t *testing.T) {
	if ok := IsSupported("BiTStaMp"); !ok {
		t.Error("supported exchange should be valid")
	}

	if ok := IsSupported("meowexch"); ok {
		t.Error("non-supported exchange should be in valid")
	}
}
