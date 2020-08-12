package backtest

import "testing"

var (
	bt = &Backtest{}
)

type testBT struct{}

func (s *testBT) OnData(last DataEvent, b *Backtest) (bool, error) {
	return true, nil
}

func (s *testBT) OnEnd(b *Backtest) {
}

func TestBacktest_Run(t *testing.T) {
	err := bt.Run()
	if err != nil {
		t.Fatal(err)
	}
}