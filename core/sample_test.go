package core

import (
	"testing"
)

func TestSum(t *testing.T) {
	var v int;

	v = Sum(2, 4)

	if v != 6 {
		t.Error("Expected 6, got ", v)
	}
}

func TestSum2(t *testing.T) {

	v := Sum(5, 0)

	if v != 5 {
		t.Error( "Expected 5, got ", v)
	}
}