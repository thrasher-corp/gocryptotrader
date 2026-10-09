package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
)

func TestOrderbookRenderingDepth(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		bids, asks, limit, rows int
	}{
		{name: "empty", limit: 50},
		{name: "single level", bids: 1, asks: 1, limit: 50, rows: 1},
		{name: "last level", bids: 3, asks: 3, limit: 50, rows: 3},
		{name: "bids only", bids: 3, limit: 50, rows: 3},
		{name: "asks only", asks: 3, limit: 50, rows: 3},
		{name: "unequal sides", bids: 1, asks: 3, limit: 50, rows: 3},
		{name: "more bids", bids: 3, asks: 1, limit: 50, rows: 3},
		{name: "at limit", bids: 3, asks: 3, limit: 3, rows: 3},
		{name: "custom limit", bids: 3, asks: 3, limit: 2, rows: 2},
		{name: "zero limit", bids: 3, asks: 3},
		{name: "stream ceiling", bids: 51, asks: 51, limit: 50, rows: 50},
		{name: "snapshot ceiling", bids: 101, asks: 101, limit: 100, rows: 100},
	} {
		for _, exchangeStyle := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/exchangeStyle=%t", tc.name, exchangeStyle), func(t *testing.T) {
				t.Parallel()
				book := &gctrpc.OrderbookResponse{Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"}}
				for i := range tc.bids {
					book.Bids = append(book.Bids, &gctrpc.OrderbookItem{Price: float64(100 - i), Amount: float64(i + 1)})
				}
				for i := range tc.asks {
					book.Asks = append(book.Asks, &gctrpc.OrderbookItem{Price: float64(200 + i), Amount: float64(i + 2)})
				}
				var output strings.Builder
				if exchangeStyle {
					renderOrderbookExchangeStyle(&output, book, "test", "spot", tc.limit)
				} else {
					renderOrderbookStream(&output, book, "test", tc.limit)
				}
				var got []string
				for line := range strings.SplitSeq(output.String(), "\n") {
					if strings.Contains(line, " @ ") || strings.HasPrefix(line, redText) || strings.HasPrefix(line, greenText) {
						got = append(got, line)
					}
				}
				var expected []string
				if exchangeStyle {
					for i := tc.rows; i > 0; i-- {
						price, amount := 0.0, 0.0
						if i-1 < tc.asks {
							price, amount = float64(200+i-1), float64(i+1)
						}
						expected = append(expected, fmt.Sprintf("%s%.8f\t\t%.8f", redText, price, amount))
					}
					for i := range tc.rows {
						price, amount := 0.0, 0.0
						if i < tc.bids {
							price, amount = float64(100-i), float64(i+1)
						}
						expected = append(expected, fmt.Sprintf("%s%.8f\t\t%.8f", greenText, price, amount))
					}
				} else {
					for i := range tc.rows {
						bidPrice, bidAmount, askPrice, askAmount := 0.0, 0.0, 0.0, 0.0
						if i < tc.bids {
							bidPrice, bidAmount = float64(100-i), float64(i+1)
						}
						if i < tc.asks {
							askPrice, askAmount = float64(200+i), float64(i+2)
						}
						expected = append(expected, fmt.Sprintf("%.8f BTC @ %.8f USD\t\t%.8f BTC @ %.8f USD", bidAmount, bidPrice, askAmount, askPrice))
					}
				}
				assert.Equal(t, expected, got, "rendering should retain every level up to the limit in display order")
				assert.Len(t, book.Bids, tc.bids, "rendering should not truncate the underlying bids")
				assert.Len(t, book.Asks, tc.asks, "rendering should not truncate the underlying asks")
			})
		}
	}
}
