package backtest

import (
	"testing"
	"time"
)

var (
	testDataFromKline = DataFromKlineItem{
		Item: genOHCLVData(),
	}
)

func TestReset(t *testing.T) {
	testDataFromKline.Load()
	_, _ = testDataFromKline.Next()

	testDataFromKline.Reset()

	if testDataFromKline.latest != nil {
		t.Fatal("expected data to be inl after reset")
	}

	if testDataFromKline.offset != 0 {
		t.Fatal("expected offset to be 0 after reset")
	}
}

func TestDataFromKlineItem_Latest(t *testing.T) {
	testDataFromKline.Load()
	_, _ = testDataFromKline.Next()

	x := testDataFromKline.Latest()
	if x.Time().Month() != time.Now().Month() {
		t.Fatal("expected month value to be same as current month")
	}
}

func TestDataFromKlineItem_History(t *testing.T) {
	testDataFromKline.Load()
	_, _ = testDataFromKline.Next()
	t.Log(testDataFromKline.History()[0].Time())
}

func TestDataFromKlineItem_Stream(t *testing.T) {

}

func TestDataFromKlineItem_SetStream(t *testing.T) {

}
