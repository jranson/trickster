package numbers

import (
	"math"
	"testing"
)

func TestSafeAdd(t *testing.T) {
	const i1 = math.MaxInt
	const i2 = 500
	const i3 = 1
	i, ok := SafeAdd(i1, i3)
	if i != 0 {
		t.Errorf("expected %d got %d", 0, i)
	}
	if ok {
		t.Error("expected false")
	}

	i, ok = SafeAdd(i2, i3)
	if i != 501 {
		t.Errorf("expected %d got %d", 501, i)
	}
	if !ok {
		t.Error("expected true")
	}
}
