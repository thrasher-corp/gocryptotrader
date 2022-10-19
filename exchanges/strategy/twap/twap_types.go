package twap

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/strategy"
)

// Strategy defines a TWAP strategy that handles the accumulation/de-accumulation
// of assets via a time weighted average price.
type Strategy struct {
	strategy.Base
	*Config
	holdings  map[currency.Code]*account.ProtectedBalance
	Reporter  chan Report
	Candles   kline.Item
	orderbook *orderbook.Depth
}

// Config defines the base elements required to undertake the TWAP strategy
type Config struct {
	Exchange exchange.IBotExchange
	Pair     currency.Pair
	Asset    asset.Item
	Verbose  bool

	// Simulate will run the strategy and order execution in simulation mode.
	Simulate bool

	// Start time will commence strategy operations after time.Now().
	Start time.Time

	// End will cease strategy operations unless AllowTradingPastEndTime is true
	// then will cease operations after balance is deployed.
	End time.Time

	// AllowTradingPastEndTime if volume has not been met exceed end time.
	AllowTradingPastEndTime bool

	// Interval between market orders.
	Interval kline.Interval

	// Amount if buying refers to quotation used to buy, if selling it will
	// refer to the base amount to sell.
	Amount float64

	// FullAmount if buying refers to all available quotation used to buy, if
	// selling it will refer to all the base amount to sell.
	FullAmount bool

	// PriceLimit if lifting the asks it will not execute an order above this
	// price. If hitting the bids this will not execute an order below this
	// price.
	PriceLimit float64

	// MaxImpactSlippage is the max allowable distance through book that can
	// occur. Usage to limit price effect on trading activity.
	MaxImpactSlippage float64

	// MaxNominalSlippage is the max allowable nominal
	// (initial cost to average order cost) splippage percentage that
	// can occur.
	MaxNominalSlippage float64

	// ReduceOnly does not add to the size of position.
	ReduceOnly bool

	// Buy if you are buying and lifting the asks else hitting those pesky bids.
	Buy bool

	// MaxSpreadpercentage defines the max spread percentage between best bid
	// and ask. If exceeded will not execute an order.
	MaxSpreadpercentage float64

	// TODO:
	// - Randomize and obfuscate amounts
	// - Hybrid and randomize execution order types (limit/market)
}

type Holdings struct {
	Current map[currency.Code]*account.ProtectedBalance
}
