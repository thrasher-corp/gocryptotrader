package exchange

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrExchangeIsNil refers to when an exchange is not set.
	ErrExchangeIsNil                 = errors.New("exchange is nil")
	errOrderBuilderIsNil             = errors.New("order builder is nil")
	errPriceUnset                    = errors.New("price is unset")
	errAmountUnset                   = errors.New("amount is unset")
	errOrderTypeUnset                = errors.New("order type is unset")
	errAssetTypeUnset                = errors.New("asset type is unset")
	errOrderTypeUnsupported          = errors.New("order type unsupported")
	errSubmissionConfigInvalid       = errors.New("submission config invalid")
	errAmountInvalid                 = errors.New("amount invalid")
	errAmountTooLow                  = errors.New("amount too low")
	errAmountTooHigh                 = errors.New("amount too high")
	errPriceInvalid                  = errors.New("price invalid")
	errAmountStepIncrementSizeIsZero = errors.New("amount step increment size is zero")
	errQuoteStepIncrementSizeIsZero  = errors.New("quote step increment size is zero")
)

// OrderAmounts is the result of the order builder calculations.
type OrderAmounts struct {
	// PreOrderAmount depending on the exchange requirements could be a base or
	// quote amount.
	PreOrderAmount float64
	// PreOrderFeeAdjustedAmount is the PreOrderAmount adjusted to the fee if
	// the fee is taken from the selling currency. This will restrict the amount
	// to account for that fee percentage, so that the order will execute.
	PreOrderFeeAdjustedAmount float64
	// PreOrderPrecisionAdjustedAmount is the PreOrderFeeAdjustedAmount adjusted
	// to the exchange precision.
	PreOrderPrecisionAdjustedAmount float64
	// PreOrderPrecisionAdjustedPrice is the price adjusted to the exchange
	// precision.
	PreOrderPrecisionAdjustedPrice float64
	// PostOrderExpectedPurchasedAmount is the expected amount of currency
	// purchased after the order is executed.
	PostOrderExpectedPurchasedAmount float64
	// PostOrderFeeAdjustedAmount is the PostOrderExpectedPurchasedAmount
	// adjusted to the fee if the fee is taken from the purchasing currency.
	PostOrderFeeAdjustedAmount float64
	// TODO: ActualPurchasedAmount is the actual amount of currency purchased after
	// the order is executed. This can be taken from a hook or the exchange
	// might return this value. The hook can be sending a HTTP request for
	// balance which is expensive or if a websocket connection is active should
	// be updated via a callback.
	// ActualPurchasedAmount float64
}

// OrderBuilder is a helper struct to assist in building orders. All values
// will be checked in validate and/or Submit. If the exchange does not support
// the order type or order side it will return an error.
type OrderBuilder struct {
	exch               IBotExchange
	pair               currency.Pair
	orderType          order.Type
	price              float64
	assetType          asset.Item
	purchasing         bool
	exchangingCurrency currency.Code
	currencyAmount     float64
	feePercentage      float64
	config             order.SubmissionConfig
	aspect             *currency.OrderAspect
}

// NewOrderBuilder returns a new order builder which attempts to provide a more
// intuative way to build orders. This is not supported by all exchanges. Spot
// orders are only supported at this time. NOTE: Market orders require a price
// to be set, this is used to calculate the expected amount of currency to be
// purchased or sold depending on the order side and requirements of the exchange.
// TOOD: Add hook definitions base paramater here so that the order builder can
// be used without the need to integrate with GCT's services. i.e. only an
// exchange wrapper library is imported and used.
func (b *Base) NewOrderBuilder(exch IBotExchange) (*OrderBuilder, error) { // TODO: Might not return an error and then just validate on submit
	if b == nil {
		return nil, ErrExchangeIsNil
	}
	if exch == nil {
		return nil, ErrExchangeIsNil
	}
	if b.SubmissionConfig == (order.SubmissionConfig{}) {
		return nil, common.ErrFunctionNotSupported
	}
	return &OrderBuilder{exch: exch, config: b.SubmissionConfig}, nil
}

// Pair sets the currency pair
func (o *OrderBuilder) Pair(pair currency.Pair) *OrderBuilder {
	o.pair = pair
	return o
}

// Price sets the price.
// NOTE: This is currently mandatory for market orders as well,
// depending on purchasing and selling this will convert currency amounts and
// will be used to calculate the expected amount of currency to be purchased (NOT-ACCURATE).
// Until price finding is implemented you should pre-calculate the price from
// potential options below; accurate to least accurate (websocket preferred):
//  1. Orderbook -
//     a. Depending on liquidity side of order book and the deployment amount
//     calculate the potential average price across tranches.
//     b. Use best bid or ask price depending on side.
//  2. Ticker -
//     a. Use best bid or ask price depending on side.
//     b. Use last price.
func (o *OrderBuilder) Price(price float64) *OrderBuilder {
	o.price = price
	return o
}

// Market sets the order type to market order.
func (o *OrderBuilder) Market() *OrderBuilder {
	o.orderType = order.Market
	return o
}

// Purchase defines the currency you would like to purchase and the amount.
func (o *OrderBuilder) Purchase(c currency.Code, amount float64) *OrderBuilder {
	o.purchasing = true
	o.exchangingCurrency = c
	o.currencyAmount = amount
	return o
}

// Sell defines the currency you would like to sell and the amount.
func (o *OrderBuilder) Sell(c currency.Code, amount float64) *OrderBuilder {
	o.purchasing = false
	o.exchangingCurrency = c
	o.currencyAmount = amount
	return o
}

// Asset defines the asset type
func (o *OrderBuilder) Asset(a asset.Item) *OrderBuilder {
	o.assetType = a
	return o
}

// Fee defines the fee percentage to be used for the order. This is used to
// calculate the fee adjusted amount. If this is not set this might not execute
// due to insufficient funds and or the returned amount might not be closer to
// the expected amount.
func (o *OrderBuilder) FeePercentage(f float64) *OrderBuilder {
	o.feePercentage = f
	return o
}

// FeeRate defines the fee rate to be used for the order. This is used to
// calculate the fee adjusted amount. If this is not set this might not execute
// due to insufficient funds and or the returned amount might not be closer to
// the expected amount.
func (o *OrderBuilder) FeeRate(f float64) *OrderBuilder {
	o.feePercentage = f * 100
	return o
}

// TODO: Add *order.Submit pre allocation and subsequent Submit call.

// validate will check the order builder values and return an error if values
// are not set correctly.
func (o *OrderBuilder) validate() error {
	if o == nil {
		return errOrderBuilderIsNil
	}

	if o.exch == nil {
		return ErrExchangeIsNil
	}

	if o.price <= 0 { // TODO: Price hook to get price for Market orders
		return fmt.Errorf("%w: please use method 'Price'", errPriceUnset)
	}

	if o.pair.IsEmpty() {
		return fmt.Errorf("%w: please use method 'Pair'", currency.ErrCurrencyPairEmpty)
	}

	if o.orderType == order.UnknownType {
		return fmt.Errorf("%w: please use method(s) 'Market'", errOrderTypeUnset)
	}

	if o.orderType != order.Market {
		return fmt.Errorf("%w: only order type 'Market' is supported for now", errOrderTypeUnsupported)
	}

	if o.assetType == 0 {
		return fmt.Errorf("%w: please use method `Asset`", errAssetTypeUnset)
	}

	if o.exchangingCurrency.IsEmpty() {
		return fmt.Errorf("%w: please use method(s) `Purchase` or `Sell`", currency.ErrCurrencyCodeEmpty)
	}

	if o.currencyAmount <= 0 {
		return fmt.Errorf("%w: please use method(s) `Purchase` or `Sell`", errAmountUnset)
	}

	var err error
	switch {
	case o.orderType == order.Market:
		if o.purchasing {
			o.aspect, err = o.pair.MarketBuyOrderAspect(o.exchangingCurrency)
		} else {
			o.aspect, err = o.pair.MarketSellOrderAspect(o.exchangingCurrency)
		}
	// TODO: case o.orderType == order.Limit:
	default:
		return fmt.Errorf("%w: only asset type 'Spot' is supported for now", asset.ErrNotSupported)
	}

	// Note: Fee is optional and can be <= 0 for rebates.
	// TODO: Add rebate functionality ad-hoc
	return err
}

// Receipt is the result of submitting an order to the exchange.
type Receipt struct {
	// Builder is the pre-order initial state
	Builder *OrderBuilder
	// Outbound is the order that was submitted to the exchange
	Outbound *order.Submit
	// Response is the response from the exchange
	Response *order.SubmitResponse
	// OrderAmounts is the calculated amounts for the order
	OrderAmounts
}

// Submit will attempt to submit the order to the exchange. If the exchange
// does not support the order type or order side it will return an error.
func (o *OrderBuilder) Submit(ctx context.Context) (*Receipt, error) {
	err := o.validate()
	if err != nil {
		return nil, err
	}

	// TODO: Add fee pre-order hook.

	// TODO: Balance check pre-order hook. If the balance is not sufficient to
	// cover the order return an error.

	termAdjusted, err := o.convertOrderAmountToTerm(o.currencyAmount)
	if err != nil {
		return nil, err
	}

	preOrderFeeAdjusted, err := o.reduceOrderAmountByFee(termAdjusted, true /*IsPreOrder*/)
	if err != nil {
		return nil, err
	}

	preOrderPrecisionAdjustedAmount, preOrderPrecisionAdjustedPrice, err := o.orderAmountPriceAdjustToPrecision(preOrderFeeAdjusted, o.price)
	if err != nil {
		return nil, err
	}

	side := order.Buy
	if !o.aspect.BuySide {
		side = order.Sell
	}

	postOnly := false // TODO: PostOnly option to be added to order builder
	if o.orderType == order.Limit {
		postOnly = true
	}

	submit := &order.Submit{
		Exchange:  o.exch.GetName(),
		Pair:      o.pair,
		Side:      side,
		Amount:    preOrderPrecisionAdjustedAmount,
		AssetType: asset.Spot,
		Type:      o.orderType,
		Price:     preOrderPrecisionAdjustedPrice,
		PostOnly:  postOnly,
	}

	expectedPurchasedAmount, err := o.postOrderAdjustToPurchased(preOrderPrecisionAdjustedAmount, preOrderPrecisionAdjustedPrice)
	if err != nil {
		return nil, err
	}

	expectedPurchasedAmountFeeAdjusted, err := o.reduceOrderAmountByFee(expectedPurchasedAmount, false /*IsPreOrder*/)
	if err != nil {
		return nil, err
	}

	resp, err := o.exch.SubmitOrder(ctx, submit)
	if err != nil {
		return nil, err
	}

	// TODO: Order check post-order hook. See if the order has actually been
	// placed and get details.

	// TODO: Balance check post-order hook. See what has actually been purchased.
	return &Receipt{
		Builder:  o,
		Outbound: submit,
		Response: resp,
		OrderAmounts: OrderAmounts{
			PreOrderAmount:                   termAdjusted,
			PreOrderFeeAdjustedAmount:        preOrderFeeAdjusted,
			PreOrderPrecisionAdjustedAmount:  preOrderPrecisionAdjustedAmount,
			PreOrderPrecisionAdjustedPrice:   preOrderPrecisionAdjustedPrice,
			PostOrderExpectedPurchasedAmount: expectedPurchasedAmount,
			PostOrderFeeAdjustedAmount:       expectedPurchasedAmountFeeAdjusted,
		},
	}, nil
}

// convertOrderAmountToTerm adjusts the order amount to the required term
// (base or quote) based on the exchange configuration.
func (o *OrderBuilder) convertOrderAmountToTerm(amount float64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("convertOrderAmountToTerm %w: %v", errAmountInvalid, amount)
	}
	switch {
	case o.config.OrderBaseAmountsRequired:
		if o.pair.Quote.Equal(o.exchangingCurrency) {
			// Amount is currently in quote terms and needs to be converted to
			// base terms. For example, if 1 BTC is priced at 25k USD, the
			// amount in base terms (USD) needed to be sold to purchase 1 BTC is
			// 25k USD.
			amount /= o.price
		}
	case o.config.OrderSellingAmountsRequired:
		if o.purchasing {
			// Selling amount is needed for this specific exchange.
			if o.pair.Base.Equal(o.exchangingCurrency) {
				// Amount is currently in base terms and needs to be converted
				// to quote terms. For example, if 25k USD (wishing to be
				// purchased) is priced at 1 BTC, the amount in quote terms (USD)
				// needed to be sold to purchase 1 BTC is 25k USD.
				amount *= o.price
			} else {
				// Amount is currently in quote terms and needs to be converted
				// to base terms. For example, if 1 BTC is priced at 25k USD,
				// the amount in base terms (BTC) needed to be sold to purchase
				// 1 BTC is 25k USD.
				amount /= o.price
			}
		}
	default:
		return 0, fmt.Errorf("convertOrderAmountToTerm %w", errSubmissionConfigInvalid)
	}
	return amount, nil
}

// reduceOrderAmountByFee reduces the amount by the fee percentage to ensure
// either the order is not rejected due to insufficient funds `pre order` or the
// purchased amount is correctly reduced.
func (o *OrderBuilder) reduceOrderAmountByFee(amount float64, preOrder bool) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("reduceOrderAmountByFee %w: %v", errAmountInvalid, amount)
	}
	switch {
	case o.config.FeeAppliedToSellingCurrency:
		if !preOrder {
			return amount, nil // No fee reduction required
		}
	case o.config.FeeAppliedToPurchasedCurrency:
		if preOrder {
			return amount, nil // No fee reduction required
		}
		if o.config.FeePostOrderRequiresPrecisionOnAmount {
			var err error
			amount, err = o.orderPurchasedAmountAdjustToPrecision(amount)
			if err != nil {
				return 0, err
			}
		}
	default:
		return 0, fmt.Errorf("reduceOrderAmountByFee %w", errSubmissionConfigInvalid)
	}
	return math.ReduceByPercentage(amount, o.feePercentage), nil
}

// orderAmountAdjustToPrecision changes the amount to the required exchange
// defined precision.
func (o *OrderBuilder) orderAmountPriceAdjustToPrecision(amount, price float64) (float64, float64, error) {
	if amount <= 0 {
		return 0, 0, fmt.Errorf("orderAmountAdjustToPrecision %w: %v", errAmountInvalid, amount)
	}

	if price <= 0 {
		return 0, 0, fmt.Errorf("orderAmountAdjustToPrecision %w: %v", errPriceInvalid, amount)
	}

	limits, err := o.exch.GetOrderExecutionLimits(o.assetType, o.pair)
	if err != nil {
		if !o.config.RequiresParameterLimits && errors.Is(err, order.ErrExchangeLimitNotLoaded) {
			return amount, price, nil
		}
		return 0, 0, err
	}

	switch {
	case o.config.OrderBaseAmountsRequired:
		if limits.AmountStepIncrementSize != 0 {
			amount = AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
		}
		err = CheckAmounts(&limits, amount, false /*isQuote*/)
	case o.config.OrderSellingAmountsRequired:
		if o.aspect.SellingCurrency.Equal(o.pair.Base) {
			if limits.AmountStepIncrementSize != 0 {
				amount = AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
			}
			err = CheckAmounts(&limits, amount, false /*isQuote*/)
		} else {
			if limits.QuoteStepIncrementSize != 0 {
				amount = AdjustToFixedDecimal(amount, limits.QuoteStepIncrementSize)
			}
			err = CheckAmounts(&limits, amount, true /*isQuote*/)
		}
	default:
		return 0, 0, fmt.Errorf("orderAmountAdjustToPrecision %w", errSubmissionConfigInvalid)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("orderAmountAdjustToPrecision %w", err)
	}

	if o.orderType != order.Market && limits.PriceStepIncrementSize != 0 {
		// Client inputed limit order price needs to be adjusted to the required
		// precision.
		price = AdjustToFixedDecimal(price, limits.PriceStepIncrementSize)
	}

	return amount, price, nil
}

// orderPurchasedAmountAdjustToPrecision changes the amount to the required
// exchange defined precision. This is for when an exchange slams your expected
// purchased amount with a holy math.floor bat. If this is occuring you will
// actually incur a higher fee rate as a result. Happy days.
func (o *OrderBuilder) orderPurchasedAmountAdjustToPrecision(amount float64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w: %v", errAmountInvalid, amount)
	}

	limits, err := o.exch.GetOrderExecutionLimits(o.assetType, o.pair)
	if err != nil { // Precision in this case is definately needed.
		return 0, err
	}

	switch {
	case o.config.OrderBaseAmountsRequired:
		return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", common.ErrNotYetImplemented)
	case o.config.OrderSellingAmountsRequired:
		if o.aspect.PurchasingCurrency.Equal(o.pair.Base) {
			if limits.AmountStepIncrementSize == 0 {
				return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", errAmountStepIncrementSizeIsZero)
			}
			amount = AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
		} else {
			if limits.QuoteStepIncrementSize == 0 {
				return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", errQuoteStepIncrementSizeIsZero)
			}
			amount = AdjustToFixedDecimal(amount, limits.QuoteStepIncrementSize)
		}
	default:
		return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", errSubmissionConfigInvalid)
	}

	return amount, nil
}

// PostOrderAdjustToPurchased converts the amount to the purchased amount.
func (o *OrderBuilder) postOrderAdjustToPurchased(amount, price float64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("PostOrderAdjustToPurchased %w: %v", errAmountInvalid, amount)
	}
	if price <= 0 {
		return 0, fmt.Errorf("PostOrderAdjustToPurchased %w: %v", errPriceInvalid, price)
	}
	switch {
	case o.config.OrderBaseAmountsRequired:
		if !o.aspect.PurchasingCurrency.Equal(o.pair.Base) {
			return amount * price, nil
		}
		return amount, nil
	case o.config.OrderSellingAmountsRequired:
		if o.aspect.SellingCurrency.Equal(o.pair.Base) {
			return amount * price, nil
		}
		return amount / price, nil
	default:
		return 0, fmt.Errorf("PostOrderAdjustToPurchased %w", errSubmissionConfigInvalid)
	}
}

// AdjustToFixedDecimal adjusts the amount to the required precision. Uses
// decimal package to ensure precision is maintained.
// TODO: Shift to math package
func AdjustToFixedDecimal(amount, precision float64) float64 {
	decAmount := decimal.NewFromFloat(amount)
	step := decimal.NewFromFloat(precision)
	// derive modulus
	mod := decAmount.Mod(step)
	// subtract modulus to get floor
	return decAmount.Sub(mod).InexactFloat64()
}

// CheckAmounts checks the amount against the limits.
func CheckAmounts(limits *order.MinMaxLevel, amount float64, quote bool) error {
	if quote {
		if limits.MinimumQuoteAmount != 0 && amount < limits.MinimumQuoteAmount {
			return fmt.Errorf("check amounts quote %w: amount to deploy: %v minimum amount: %v", errAmountTooLow, amount, limits.MinimumQuoteAmount)
		}
		if limits.MaximumQuoteAmount != 0 && amount > limits.MaximumQuoteAmount {
			return fmt.Errorf("check amounts quote %w: amount to deploy: %v maximum amount: %v", errAmountTooHigh, amount, limits.MaximumQuoteAmount)
		}
	} else {
		if limits.MinimumBaseAmount != 0 && amount < limits.MinimumBaseAmount {
			return fmt.Errorf("check amounts base %w: amount to deploy: %v minimum amount: %v", errAmountTooLow, amount, limits.MinimumBaseAmount)
		}
		if limits.MaximumBaseAmount != 0 && amount > limits.MaximumBaseAmount {
			return fmt.Errorf("check amounts base %w: amount to deploy: %v maximum amount: %v", errAmountTooHigh, amount, limits.MaximumBaseAmount)
		}
	}
	return nil
}
