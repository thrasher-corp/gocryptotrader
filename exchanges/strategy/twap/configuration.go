package twap

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	strategy "github.com/thrasher-corp/gocryptotrader/exchanges/strategy/common"
)

// Config defines the base elements required to undertake strategy
type Config struct {
	// Exchange defines the exchange that the strategy is acting on
	Exchange exchange.IBotExchange
	// Pair defines the currency pair the strategy is acting on
	Pair currency.Pair
	// Asset is the current asset type the strategy is acting on
	Asset asset.Item
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
	// TWAP is the interval to construct the TWAP price which will be the
	// average(interval * 30)
	TWAP kline.Interval
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
	// occur *AWAY FROM TWAP PRICE*. Usage to limit price effect on trading
	// activity.
	MaxImpactSlippage float64
	// MaxNominalSlippage is the max allowable nominal
	// (initial cost to average order cost) splippage percentage that
	// can occur.
	MaxNominalSlippage float64
	// Buy if you are buying and lifting the asks else hitting those pesky bids.
	Buy bool
	// MaxSpreadPercentage defines the max spread percentage between best bid
	// and ask. If exceeded will not execute an order.
	MaxSpreadPercentage float64
	// CandleStickAligned defines if the strategy will truncate to UTC candle
	// stick standards without execution offsets/drift. e.g. 1 day candle
	// interval will execute signal generation at candle close/open 00:00 UTC.
	CandleStickAligned bool
	// RetryAttempts will execute a retry order submission attempt N times
	// before critical failure.
	RetryAttempts int64

	// TODO:
	// - Randomize and obfuscate amounts
	// - Hybrid and randomize execution order types (limit/market)
	//
	// When TWAP becomes applicable to use as a position allocator with margin.
	// ReduceOnly does not add to the size of position.
	// ReduceOnly bool
}

// Check validates all config fields before undertaking specfic strategy
func (c *Config) Check(ctx context.Context) error {
	if c == nil {
		return strategy.ErrConfigIsNil
	}

	if c.Exchange == nil {
		return strategy.ErrExchangeIsNil
	}

	if c.Pair.IsEmpty() {
		return currency.ErrPairIsEmpty
	}

	if !c.Asset.IsValid() {
		return fmt.Errorf("'%v' %w", c.Asset, asset.ErrNotSupported)
	}

	if !c.End.IsZero() && c.End.Before(time.Now()) {
		return strategy.ErrEndBeforeTimeNow
	}

	if c.Interval == 0 {
		return kline.ErrUnsetInterval
	}

	if c.TWAP == 0 {
		return fmt.Errorf("TWAP %w", kline.ErrUnsetInterval)
	}

	if c.FullAmount && c.Amount != 0 {
		return strategy.ErrCannotSetAmount
	}

	if !c.FullAmount && c.Amount <= 0 {
		return strategy.ErrInvalidAmount
	}

	if c.MaxImpactSlippage < 0 || !c.Buy && c.MaxImpactSlippage > 100 {
		return fmt.Errorf("impact '%v' %w", c.MaxImpactSlippage, strategy.ErrInvalidSlippage)
	}

	if c.MaxNominalSlippage < 0 || !c.Buy && c.MaxNominalSlippage > 100 {
		return fmt.Errorf("nominal '%v' %w", c.MaxNominalSlippage, strategy.ErrInvalidSlippage)
	}

	if c.PriceLimit < 0 {
		return fmt.Errorf("price '%v' %w", c.PriceLimit, strategy.ErrInvalidPriceLimit)
	}

	if c.MaxSpreadPercentage < 0 {
		return fmt.Errorf("max spread '%v' %w", c.MaxSpreadPercentage, strategy.ErrInvalidSpread)
	}

	if c.RetryAttempts <= 0 {
		return strategy.ErrInvalidRetryAttempts
	}

	return nil
}

// GetDeploymentAmount will truncate and equally distribute amounts at or around
// TWAP price.
func (c *Config) GetDistrbutionAmount(ctx context.Context, allocatedAmount float64, book *orderbook.Depth) (*Allocation, error) {
	if c == nil {
		return nil, strategy.ErrConfigIsNil
	}
	if c.Exchange == nil {
		return nil, strategy.ErrExchangeIsNil
	}
	if allocatedAmount <= 0 {
		return nil, fmt.Errorf("allocation amount: %w", strategy.ErrInvalidAmount)
	}
	if book == nil {
		return nil, strategy.ErrOrderbookIsNil
	}
	if c.Interval <= 0 {
		return nil, kline.ErrUnsetInterval
	}

	window := c.End.Sub(c.Start)
	if int64(window) <= int64(c.Interval) {
		return nil, strategy.ErrInvalidOperatingWindow
	}

	deployments := int64(window) / int64(c.Interval)
	deploymentAmount := allocatedAmount / float64(deployments)

	twapPrice, err := c.getTwapPrice(ctx)
	if err != nil {
		return nil, err
	}

	// The checks below determines if the allocation spread over time can or
	// *should* be deployed on the exchange.
	deploymentAmountInBase, _, err := c.VerifyBookDeployment(book, deploymentAmount, twapPrice)
	if err != nil {
		return nil, err
	}

	// NOTE: Don't need to return conformed amount if returning quote holdings
	_, err = c.VerifyExecutionLimitsReturnConformed(deploymentAmountInBase)
	if err != nil {
		return nil, err
	}

	return &Allocation{
		Total:       allocatedAmount,
		Deployment:  deploymentAmount,
		Window:      window,
		Deployments: deployments,
	}, nil
}

// VerifyBookDeployment verifies book liquidity and structure with deployment
// amount and returns base amount and details.
func (c *Config) VerifyBookDeployment(book *orderbook.Depth, deploymentAmount, twapPrice float64) (float64, *orderbook.Movement, error) {
	if c == nil {
		return 0, nil, strategy.ErrConfigIsNil
	}
	if book == nil {
		return 0, nil, strategy.ErrOrderbookIsNil
	}
	if deploymentAmount <= 0 {
		return 0, nil, fmt.Errorf("deployment: %w", strategy.ErrInvalidAmount)
	}

	var details *orderbook.Movement
	var err error
	if c.Buy {
		// Quote needs to be converted to base for deployment checks.
		details, err = book.LiftTheAsksFromBest(deploymentAmount, false)
		if err != nil {
			return 0, nil, err
		}
		deploymentAmount = details.Purchased
	} else {
		details, err = book.HitTheBidsFromBest(deploymentAmount, false)
		if err != nil {
			return 0, nil, err
		}
	}

	if details.FullBookSideConsumed {
		return 0, nil, strategy.ErrExceedsLiquidity
	}

	if c.MaxNominalSlippage != 0 && c.MaxNominalSlippage < details.NominalPercentage {
		return 0, nil, fmt.Errorf("%w: book slippage: %f requested max slippage %f",
			strategy.ErrMaxNominalExceeded,
			details.NominalPercentage,
			c.MaxNominalSlippage)
	}

	if c.PriceLimit != 0 {
		if c.Buy && details.StartPrice > c.PriceLimit {
			return 0, nil, fmt.Errorf("ask book head price: %f price limit: %f %w",
				details.StartPrice,
				c.PriceLimit,
				strategy.ErrPriceLimitExceeded)
		} else if !c.Buy && details.StartPrice < c.PriceLimit {
			return 0, nil, fmt.Errorf("bid book head price: %f price limit: %f %w",
				details.StartPrice,
				c.PriceLimit,
				strategy.ErrPriceLimitExceeded)
		}
	}

	// NOTE: If spread is quite wide this might indicate problems with liquidity.
	if c.MaxSpreadPercentage != 0 {
		spread, err := book.GetSpreadPercentage()
		if err != nil {
			return 0, nil, err
		}
		if spread > c.MaxSpreadPercentage {
			return 0, nil, fmt.Errorf("%w: book slippage: %f requested max slippage %f",
				strategy.ErrMaxSpreadExceeded,
				spread,
				c.MaxSpreadPercentage)
		}
	}
	return deploymentAmount, details, nil
}

// VerifyExecutionLimitsReturnConformed verifies if the deployment amount
// exceeds the exchange execution limits. TODO:  This will need to be expanded
// and abstracted further.
func (c *Config) VerifyExecutionLimitsReturnConformed(deploymentAmountInBase float64) (float64, error) {
	if c == nil {
		return 0, strategy.ErrConfigIsNil
	}
	if deploymentAmountInBase <= 0 {
		return 0, fmt.Errorf("base deployment: %w", strategy.ErrInvalidAmount)
	}

	minMax, err := c.Exchange.GetOrderExecutionLimits(c.Asset, c.Pair)
	if err != nil {
		return 0, err
	}

	if minMax.MinAmount != 0 && minMax.MinAmount > deploymentAmountInBase {
		return 0, fmt.Errorf("%w; %s",
			strategy.ErrUnderMinimumAmount,
			strategy.MinimumSizeResponse)
	}

	if minMax.MaxAmount != 0 && minMax.MaxAmount < deploymentAmountInBase {
		return 0, fmt.Errorf("%w; %s",
			strategy.ErrOverMaximumAmount,
			strategy.MaximumSizeResponse)
	}
	return minMax.ConformToAmount(deploymentAmountInBase), nil
}

// SignalTWAP defines the main signal for the TWAP strategy
type SignalTWAP struct {
	Price            float64 `json:"twap"`
	Interval         string  `json:"twapInterval"`
	Period           int     `json:"period"`
	Window           string  `json:"window"`
	EndPrice         float64 `json:"endPrice"`
	PercentageImpact float64 `json:"percentageImpact"`
	Exceeded         bool    `json:"parametersExceeded"`
}

// CheckTWAP checks the potential orderbook tranche end price after a potential
// amount deployment.
func (c *Config) CheckTWAP(twap float64, trancheEndPrice float64) *SignalTWAP {
	impact := math.Abs((twap - trancheEndPrice) / trancheEndPrice * 100)
	return &SignalTWAP{
		Price:            twap,
		Interval:         c.TWAP.String(),
		Period:           30,
		Window:           (30 * c.TWAP.Duration()).String(),
		EndPrice:         trancheEndPrice,
		PercentageImpact: impact,
		Exceeded:         impact > c.MaxImpactSlippage,
	}
}
