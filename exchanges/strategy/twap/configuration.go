package twap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
)

const (
	minimumSizeResponse = "reduce end date, increase granularity (interval) or increase deployable capital requirements"
	maximumSizeResponse = "increase end date, decrease granularity (interval) or decrease deployable capital requirements"
)

var (
	errParamsAreNil                    = errors.New("params are nil")
	errExchangeIsNil                   = errors.New("exchange is nil")
	errEndBeforeNow                    = errors.New("end time is before current time")
	errCannotSetAmount                 = errors.New("specific amount cannot be set, full amount bool set")
	errInvalidVolume                   = errors.New("invalid volume")
	errInvalidMaxSlippageValue         = errors.New("invalid max slippage percentage value")
	errInvalidPriceLimit               = errors.New("invalid price limit")
	errInvalidMaxSpreadPercentage      = errors.New("invalid spread percentage")
	errUnderMinimumAmount              = errors.New("strategy deployment amount is under the exchange minimum")
	errOverMaximumAmount               = errors.New("strategy deployment amount is over the exchange maximum")
	errInvalidOperationWindow          = errors.New("start to end time window is cannot be less than or equal to interval")
	errInvalidAllocatedAmount          = errors.New("allocated amount must be greater than zero")
	errOrderbookIsNil                  = errors.New("orderbook is nil")
	errMaxSpreadPercentageExceeded     = errors.New("max spread percentage exceeded")
	errMaxImpactPercentageExceeded     = errors.New("impact percentage exceeded")
	errMaxNominalPercentageExceeded    = errors.New("nominal percentage exceeded")
	errMaxPriceLimitExceeded           = errors.New("price limit exceeded")
	errExceedsTotalBookLiquidity       = errors.New("exceeds total orderbook liquidity")
	errBookSmallerThanDeploymentAmount = errors.New("orderbook cannot take in deployment amount")
	errConfigurationIsNil              = errors.New("strategy configuration is nil")
	errInvalidRetryAttempts            = errors.New("invalid retry attempts")
)

// Config defines the base elements required to undertake the TWAP strategy
type Config struct {
	Exchange exchange.IBotExchange
	Pair     currency.Pair
	Asset    asset.Item

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

	// TODO: When TWAP becomes applicable to use as a position allocator with
	// margin.
	// ReduceOnly does not add to the size of position.
	// ReduceOnly bool

	// Buy if you are buying and lifting the asks else hitting those pesky bids.
	Buy bool

	// MaxSpreadPercentage defines the max spread percentage between best bid
	// and ask. If exceeded will not execute an order.
	MaxSpreadPercentage float64

	// TODO:
	// - Randomize and obfuscate amounts
	// - Hybrid and randomize execution order types (limit/market)

	// CandleStickAligned defines if the strategy will truncate to UTC candle
	// stick standards without execution offsets/drift. e.g. 1 day candle
	// interval will execute signal generation at candle close/open 00:00 UTC.
	CandleStickAligned bool

	// RetryAttempts will execute a retry order submission attempt N times
	// before critical failure.
	RetryAttempts int64
}

// Check validates all parameter fields before undertaking specfic strategy
func (c *Config) Check(ctx context.Context) error {
	if c == nil {
		return errParamsAreNil
	}

	if c.Exchange == nil {
		return errExchangeIsNil
	}

	if c.Pair.IsEmpty() {
		return currency.ErrPairIsEmpty
	}

	if !c.Asset.IsValid() {
		return fmt.Errorf("'%v' %w", c.Asset, asset.ErrNotSupported)
	}

	err := common.StartEndTimeCheck(c.Start, c.End)
	if err != nil {
		if !errors.Is(err, common.ErrStartAfterTimeNow) {
			// NOTE: This can schedule a future task.
			return err
		}
	}

	if c.End.Before(time.Now()) {
		return errEndBeforeNow
	}

	if c.Interval == 0 {
		return kline.ErrUnsetInterval
	}

	if c.FullAmount && c.Amount != 0 {
		return errCannotSetAmount
	}

	if !c.FullAmount && c.Amount <= 0 {
		return errInvalidVolume
	}

	if c.MaxImpactSlippage < 0 || !c.Buy && c.MaxImpactSlippage > 100 {
		return fmt.Errorf("impact '%v' %w",
			c.MaxImpactSlippage, errInvalidMaxSlippageValue)
	}

	if c.MaxNominalSlippage < 0 || !c.Buy && c.MaxNominalSlippage > 100 {
		return fmt.Errorf("nominal '%v' %w",
			c.MaxNominalSlippage, errInvalidMaxSlippageValue)
	}

	if c.PriceLimit < 0 {
		return fmt.Errorf("price '%v' %w", c.PriceLimit, errInvalidPriceLimit)
	}

	if c.MaxSpreadPercentage < 0 {
		return fmt.Errorf("max spread '%v' %w",
			c.MaxSpreadPercentage, errInvalidMaxSpreadPercentage)
	}

	if c.RetryAttempts <= 0 {
		return errInvalidRetryAttempts
	}

	return nil
}

// GetDeploymentAmount will truncate and equally distribute amounts across time.
func (c *Config) GetDistrbutionAmount(allocatedAmount float64, book *orderbook.Depth) (float64, error) {
	if c == nil {
		return 0, errConfigurationIsNil
	}
	if c.Exchange == nil {
		return 0, errExchangeIsNil
	}
	if allocatedAmount <= 0 {
		return 0, errInvalidAllocatedAmount
	}
	if book == nil {
		return 0, errOrderbookIsNil
	}
	if c.Interval <= 0 {
		return 0, kline.ErrUnsetInterval // This can panic on zero value.
	}

	window := c.End.Sub(c.Start)
	if int64(window) <= int64(c.Interval) {
		return 0, errInvalidOperationWindow
	}
	deployments := int64(window) / int64(c.Interval)
	deploymentAmount := allocatedAmount / float64(deployments)

	// The checks below determines if the allocation spread over time can or
	// *should* be deployed on the exchange.
	deploymentAmountInBase, _, err := c.VerifyBookDeployment(book, deploymentAmount)
	if err != nil {
		return 0, err
	}

	// NOTE: Don't need to returned conformed amount if returning quote holdings
	_, err = c.VerifyExecutionLimitsReturnConformed(deploymentAmountInBase)
	if err != nil {
		return 0, err
	}
	return deploymentAmount, nil
}

// VerifyBookDeployment verifies book liquidity and structure with deployment
// amount and returns base amount and details.
func (c *Config) VerifyBookDeployment(book *orderbook.Depth, deploymentAmount float64) (float64, *orderbook.Movement, error) {
	if c == nil {
		return 0, nil, errConfigurationIsNil
	}
	if book == nil {
		return 0, nil, errOrderbookIsNil
	}
	if deploymentAmount <= 0 {
		return 0, nil, errInvalidAllocatedAmount
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

	if c.MaxImpactSlippage != 0 {
		// This check is nested because this cares about book impact slippage
		// levels and has the potential to wipe out complete book stored in
		// memory.
		if details.FullBookSideConsumed {
			return 0, nil, errExceedsTotalBookLiquidity
		}

		if c.MaxImpactSlippage < details.ImpactPercentage {
			return 0, nil, fmt.Errorf("%w: book slippage: %f requested max slippage %f",
				errMaxImpactPercentageExceeded,
				details.ImpactPercentage,
				c.MaxImpactSlippage)
		}
	}

	if c.MaxNominalSlippage != 0 && c.MaxNominalSlippage < details.NominalPercentage {
		return 0, nil, fmt.Errorf("%w: book slippage: %f requested max slippage %f",
			errMaxNominalPercentageExceeded,
			details.NominalPercentage,
			c.MaxNominalSlippage)
	}

	if c.PriceLimit != 0 {
		if c.Buy && details.StartPrice > c.PriceLimit {
			return 0, nil, fmt.Errorf("ask book head price: %f price limit: %f %w",
				details.StartPrice,
				c.PriceLimit,
				errMaxPriceLimitExceeded)
		} else if !c.Buy && details.StartPrice < c.PriceLimit {
			return 0, nil, fmt.Errorf("bid book head price: %f price limit: %f %w",
				details.StartPrice,
				c.PriceLimit,
				errMaxPriceLimitExceeded)
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
				errMaxSpreadPercentageExceeded,
				spread,
				c.MaxSpreadPercentage)
		}
	}

	if details.Sold < deploymentAmount {
		return details.Sold, details, errBookSmallerThanDeploymentAmount
	}

	return deploymentAmount, details, nil
}

// VerifyExecutionLimitsReturnConformed verifies if the deploument amount
// exceeds the exchange execution limits. TODO: This will need to be expanded.
func (c *Config) VerifyExecutionLimitsReturnConformed(deploymentAmountInBase float64) (float64, error) {
	if c == nil {
		return 0, errConfigurationIsNil
	}
	if deploymentAmountInBase <= 0 {
		return 0, errInvalidAllocatedAmount
	}

	minMax, err := c.Exchange.GetOrderExecutionLimits(c.Asset, c.Pair)
	if err != nil {
		return 0, err
	}

	if minMax.MinAmount != 0 && minMax.MinAmount > deploymentAmountInBase {
		return 0, fmt.Errorf("%w; %s", errUnderMinimumAmount, minimumSizeResponse)
	}

	if minMax.MaxAmount != 0 && minMax.MaxAmount < deploymentAmountInBase {
		return 0, fmt.Errorf("%w; %s", errOverMaximumAmount, maximumSizeResponse)
	}

	return minMax.ConformToAmount(deploymentAmountInBase), nil
}

// GetNextSchedule gets next signal/execution time, this will also allow
// truncation to interval for alignment to candle.
func (c *Config) GetNextSchedule(scheduled time.Time) (time.Duration, error) {
	if c == nil {
		return 0, errConfigurationIsNil
	}
	if c.Interval <= 0 {
		return 0, kline.ErrUnsetInterval
	}
	if scheduled.IsZero() {
		scheduled = time.Now()
	}
	if c.CandleStickAligned {
		scheduled = scheduled.Truncate(c.Interval.Duration())
	}
	return time.Until(scheduled.Add(c.Interval.Duration())), nil
}

var errTimerIsNil = errors.New("timer is nil")

// SetTimer sets timer at new interval time.
func (c *Config) SetTimer(timer *time.Timer) error {
	if c == nil {
		return errConfigurationIsNil
	}
	if timer == nil {
		return errTimerIsNil
	}
	schedule, err := c.GetNextSchedule(time.Now())
	if err != nil {
		return err
	}
	timer.Reset(schedule)
	return nil
}
