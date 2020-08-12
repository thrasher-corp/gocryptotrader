package backtest

import "testing"

type testBT struct{}

func (bt *testBT) Init() *Config {
	return &Config{}
}

func (bt *testBT) OnData(t Data, b *Backtest) (bool, error) {
	return true, nil
}

func (s *testBT) OnEnd(b *Backtest) {
}

func TestBacktest_Run(t *testing.T) {
	bt := &testBT{}
	err := Run(bt)
	if err != nil {
		t.Fatal(err)
	}
}