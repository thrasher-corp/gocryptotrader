package main

import (
	"fmt"
	"testing"
)

func TestIsEnabled(t *testing.T) {
	t.Parallel()
	expected := "Enabled"
	actual := IsEnabled(true)
	if actual != expected {
		t.Error(fmt.Sprintf("Test failed. Expected %s. Actual %s", expected, actual))
	}

	expected = "Disabled"
	actual = IsEnabled(false)
	if actual != expected {
		t.Error(fmt.Sprintf("Test failed. Expected %s. Actual %s", expected, actual))
	}
}
