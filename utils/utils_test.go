package utils

import (
	"runtime"
	"testing"
)

func TestAdjustGoMaxProcs(t *testing.T) {
	// Ensure that a supplied crazy number is set to a valid one and doesn't
	// return an error
	err := AdjustGoMaxProcs(1000)
	if err != nil {
		t.Fatalf("TestAdjustGoMaxProcs returned err: %v", err)
	}

	// This time use the num of logical CPU's and ensure it doesn't
	// return an error
	err = AdjustGoMaxProcs(runtime.NumCPU())
	if err != nil {
		t.Fatalf("TestAdjustGoMaxProcs returned err: %v", err)
	}
}
