package backtest

import (
	"testing"
)

func TestGenerateOutput(t *testing.T) {
	err := GenerateOutput([]byte{})
	if err != nil {
		t.Fatal(err)
	}
}