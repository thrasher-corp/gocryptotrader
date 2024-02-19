package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestPruneNil exercises List.PruneNil()
func TestListPruneNil(t *testing.T) {
	l := List{
		(*Subscription)(nil),
		&Subscription{
			Channel: TickerChannel,
		},
		(*Subscription)(nil),
		&Subscription{
			Channel: OrderbookChannel,
		},
	}
	require.Equal(t, 4, len(l), "List should start with 4 elements")
	l.PruneNil()
	require.Equal(t, 2, len(l), "List should have 2 elements after pruning")
	require.Equal(t, TickerChannel, l[0].Channel, "First element should be ticker")
	require.Equal(t, OrderbookChannel, l[1].Channel, "Second element should be orderbook")
}
