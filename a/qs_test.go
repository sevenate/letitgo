package a

import "testing"

func TestMax(t *testing.T) {
	v := Max(2.0, 4.0)

	if v != 4.0 {
		t.Error("Expected 4.0, got ", v)
	}
}
