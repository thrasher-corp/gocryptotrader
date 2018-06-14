package bitfinex

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestGetExchangeHistory(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
	_, err := b.GetExchangeHistory(p, "SPOT", time.Time{}, 0)
	if err != nil {
		t.Error("test failed - Bitfinex GetExchangeHistory() error", err)
	}
}
