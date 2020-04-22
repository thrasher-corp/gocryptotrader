package binance

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestRateLimit_Limit(t *testing.T) {
	symbol := "BTC-USDT"

	testTable := map[string]struct {
		Expected request.EndpointLimit
		Limit    request.EndpointLimit
	}{
		"All Orderbooks Ticker": {Expected: limitOrderbookTickerAll, Limit: bestPriceLimit("")},
		"Orderbook Ticker":      {Expected: limitDefault, Limit: bestPriceLimit(symbol)},
		"All Open Orders":       {Expected: limitOpenOrdersAll, Limit: openOrdersLimit("")},
		"Open Orders":           {Expected: limitOrder, Limit: openOrdersLimit(symbol)},
		"Orderbook Depth 5":     {Expected: limitDefault, Limit: orderbookLimit(5)},
		"Orderbook Depth 10":    {Expected: limitDefault, Limit: orderbookLimit(10)},
		"Orderbook Depth 20":    {Expected: limitDefault, Limit: orderbookLimit(20)},
		"Orderbook Depth 50":    {Expected: limitDefault, Limit: orderbookLimit(50)},
		"Orderbook Depth 100":   {Expected: limitDefault, Limit: orderbookLimit(100)},
		"Orderbook Depth 500":   {Expected: limitOrderbookDepth500, Limit: orderbookLimit(500)},
		"Orderbook Depth 1000":  {Expected: limitOrderbookDepth1000, Limit: orderbookLimit(1000)},
		"Orderbook Depth 5000":  {Expected: limitOrderbookDepth5000, Limit: orderbookLimit(5000)},
		"All Symbol Prices":     {Expected: limitSymbolPriceAll, Limit: symbolPriceLimit("")},
		"Symbol Price":          {Expected: limitDefault, Limit: symbolPriceLimit(symbol)},
	}
	for name, tt := range testTable {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			exp, got := tt.Expected, tt.Limit
			if exp != got {
				t.Fatalf("incorrect limit applied.\nexp: %v\ngot: %v", exp, got)
			}

			l := SetRateLimit()
			if err := l.Limit(tt.Limit); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}

func TestRateLimit_LimitStatic(t *testing.T) {
	testTable := map[string]request.EndpointLimit{
		"Default":           limitDefault,
		"Historical Trades": limitHistoricalTrades,
		"All Price Changes": limitPriceChangeAll,
		"All Orders":        limitOrdersAll,
	}
	for name, tt := range testTable {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			l := SetRateLimit()
			if err := l.Limit(tt); err != nil {
				t.Fatalf("error applying rate limit: %v", err)
			}
		})
	}
}
