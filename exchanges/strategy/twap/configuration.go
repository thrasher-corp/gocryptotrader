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
	errParamsAreNil               = errors.New("params are nil")
	errExchangeIsNil              = errors.New("exchange is nil")
	errEndBeforeNow               = errors.New("end time is before current time")
	errCannotSetAmount            = errors.New("specific amount cannot be set, full amount bool set")
	errInvalidVolume              = errors.New("invalid volume")
	errInvalidMaxSlippageValue    = errors.New("invalid max slippage percentage value")
	errInvalidPriceLimit          = errors.New("invalid price limit")
	errInvalidMaxSpreadPercentage = errors.New("invalid spread percentage")
	errUnderMinimumAmount         = errors.New("strategy deployment amount is under the exchange minimum")
	errOverMaximumAmount          = errors.New("strategy deployment amount is over the exchange maximum")
	errInvalidOperationWindow     = errors.New("start to end time window is cannot be less than or equal to interval")
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
			// We can schedule a future process
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

	return nil
}

var errInvalidAllocatedAmount = errors.New("allocated amount must be greater than zero")

// GetDeploymentAmount will truncate and equally distribute amounts across time.
func (c *Config) GetDistrbutionAmount(allocatedAmount float64, book *orderbook.Depth, quote bool) (float64, error) {
	if c == nil {
		return 0, errConfigurationIsNil
	}
	if allocatedAmount <= 0 {
		return 0, errInvalidAllocatedAmount
	}
	if c.Interval <= 0 {
		return 0, kline.ErrUnsetInterval // This can panic on zero value.
	}

	window := c.End.Sub(c.Start)
	if int64(window) <= int64(c.Interval) {
		return 0, errInvalidOperationWindow
	}
	segment := int64(window) / int64(c.Interval)

	fmt.Println("segment", segment)

	iterationAmount := allocatedAmount / float64(segment)

	fmt.Println("iteration amount", iterationAmount)

	iterationAmountInBase := iterationAmount
	if quote {
		// Quote needs to be converted to base for deployment capabilities.
		details, err := book.LiftTheAsksFromBest(iterationAmount, false)
		if err != nil {
			return 0, nil
		}
		iterationAmountInBase = details.Purchased
	}

	minMax, err := c.Exchange.GetOrderExecutionLimits(c.Asset, c.Pair)
	if err != nil {
		return 0, err
	}

	if minMax.MinAmount != 0 && minMax.MinAmount > iterationAmountInBase {
		return 0, fmt.Errorf("%w; %s", errUnderMinimumAmount, minimumSizeResponse)
	}

	if minMax.MaxAmount != 0 && minMax.MaxAmount < iterationAmountInBase {
		return 0, fmt.Errorf("%w; %s", errOverMaximumAmount, maximumSizeResponse)
	}

	fmt.Printf("minmax stuff: %+v\n", minMax)

	conformedAmount := minMax.ConformToAmount(iterationAmountInBase)

	fmt.Printf("conformed amount: %f iteration amount: %f changed by: %f\n",
		conformedAmount,
		iterationAmountInBase,
		iterationAmountInBase-conformedAmount,
	)

	return iterationAmount, nil
}
