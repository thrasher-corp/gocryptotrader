package backtest

import (
	"testing"
)

type testBT struct{}

func (bt *testBT) Init() *Config {
	return &Config{}
}

func (bt *testBT) OnData(d DataEvent, b *Backtest) (bool, error) {
	return true, nil
}

func (bt *testBT) OnEnd(b *Backtest) {}

func TestBacktest_Run(t *testing.T) {
	bt := &testBT{}
	klineData := &DataFromKlineItem{}
	err := Run(bt, klineData)
	if err != nil {
		t.Fatal(err)
	}
}
