package utils

import (
	"fmt"
	"runtime"
	"testing"
)

func TestAdjustGoMaxProcs(t *testing.T) {
	// Test default settings
	curr := runtime.GOMAXPROCS(-1)
	numCPUs := runtime.NumCPU()

	// This func both checks for an error of AdjustGoMaxProcs, plus
	// ensures that the value it sets is the one that is expected
	checker := func(setting, expected int) error {
		if err := AdjustGoMaxProcs(setting); err != nil {
			return err
		}
		if i := runtime.GOMAXPROCS(expected); i != expected {
			return fmt.Errorf("expected %d, got %d", expected, i)
		}
		return nil
	}

	tester := []struct {
		Setting  int
		Expected int
	}{
		{
			// Test setting to current runtime val
			Setting:  curr,
			Expected: curr,
		},
		{
			// Test setting to num of logical CPUs
			Setting:  numCPUs,
			Expected: numCPUs,
		},
		{
			// Test crazy value and make sure it defaults to numCPUs
			Setting:  1000,
			Expected: numCPUs,
		},
		{
			// Test another crazy value and make sure it defaults to numCPUs
			Setting:  -1,
			Expected: numCPUs,
		},
	}

	for x := range tester {
		if err := checker(tester[x].Setting, tester[x].Expected); err != nil {
			t.Errorf("%d failed. %s", x, err)
		}
	}
}
