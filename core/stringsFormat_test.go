package core

import (
	"testing"
)

func Test_format_billions(t *testing.T) {
	v := Format(2_123_423_121)

	if v != "2 123 423 121" {
		t.Error("Expected 2 123 423 121, got ", v)
	}
}

func Test_format_negative_millions(t *testing.T) {
	v := Format(-7_121_234)

	if v != "-7 121 234" {
		t.Error("Expected -7 121 234, got ", v)
	}
}
