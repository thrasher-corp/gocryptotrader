package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// TestListStrings exercises List.Strings()
func TestListStrings(t *testing.T) {
	l := List{
		&Subscription{
			Channel: TickerChannel,
			Asset:   asset.Spot,
			Pairs:   currency.Pairs{ethusdcPair, btcusdtPair},
		},
		&Subscription{
			Channel: OrderbookChannel,
			Pairs:   currency.Pairs{ethusdcPair},
		},
	}
	exp := []string{"orderbook  ETH/USDC", "ticker spot ETH/USDC,BTC/USDT"}
	assert.ElementsMatch(t, exp, l.Strings(), "String must return correct sorted list")
}

// TestListGroupPairs exercises List.GroupPairs()
func TestListGroupPairs(t *testing.T) {
	l := List{
		{Asset: asset.Spot, Channel: TickerChannel, Pairs: currency.Pairs{ethusdcPair, btcusdtPair}},
	}
	for _, c := range []string{TickerChannel, OrderbookChannel} {
		for _, p := range []currency.Pair{ethusdcPair, btcusdtPair} {
			l = append(l, &Subscription{
				Channel: c,
				Asset:   asset.Spot,
				Pairs:   currency.Pairs{p},
			})
		}
	}
	n := l.GroupPairs()
	assert.Len(t, l, 5, "Orig list should not be changed")
	assert.Len(t, n, 2, "New list should be grouped")
	exp := []string{"ticker spot ETH/USDC,BTC/USDT", "orderbook spot ETH/USDC,BTC/USDT"}
	assert.ElementsMatch(t, exp, n.Strings(), "String must return correct sorted list")
}
