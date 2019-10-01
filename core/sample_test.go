package core

import (
	"testing"
)

func testSum(t *testing.T) {
	v := sum(2, 4)

	if v != 6 {
		t.Error("Expected 6, got ", v)
	}
}

func testSum2(t *testing.T) {
	v := sum(5, 0)

	if v != 5 {
		t.Error("Expected 5, got ", v)
	}
}
