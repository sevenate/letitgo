package core

import (
	"math"
	"testing"
)

func TestExample(t *testing.T) {
	var v float64

	v = math.Pow(2.0, 4.0)

	if v != 16.0 {
		t.Error("Expected 16.0, got ", v)
	}
}
