package exchange

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/math"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
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
	errNilOrderBuilder               = errors.New("order builder is nil")
)

// OrderAmounts represents the calculated amounts and values related to an order.
type OrderAmounts struct {
	// PreOrderAmount is the amount before any adjustments, which could be a
	// base or quote amount depending on exchange requirements.
	PreOrderAmount float64
	// PreOrderFeeAdjustedAmount is the PreOrderAmount adjusted to account for
	// the fee, if the fee is taken from the selling currency. This adjustment
	// ensures that the order will execute correctly.
	PreOrderFeeAdjustedAmount float64
	// PreOrderPrecisionAdjustedAmount is the PreOrderFeeAdjustedAmount adjusted
	// to comply with the exchange precision.
	PreOrderPrecisionAdjustedAmount float64
	// PreOrderPrecisionAdjustedPrice is the price adjusted to comply with the
	// exchange precision.
	PreOrderPrecisionAdjustedPrice float64
	// PostOrderExpectedPurchasedAmount is the expected amount of currency that
	// will be purchased after the order is executed.
	PostOrderExpectedPurchasedAmount float64
	// PostOrderFeeAdjustedAmount  is the PostOrderExpectedPurchasedAmount
	// adjusted to account for the fee, if the fee is taken from the purchasing
	// currency.
	PostOrderFeeAdjustedAmount float64
}

// OrderTypeSetter sets the order type
type OrderTypeSetter interface {
	// Market sets the order type to market order
	Market() MarketPairSetter
	// TODO: Limit, Stop, StopLimit, TrailingStop, Iceberg etc.
}

// MarketPairSetter sets the currency pair for a specific market order
type MarketPairSetter interface {
	// Pair sets the currency pair
	Pair(pair currency.Pair) MarketPriceSetter
}

// MarketPriceSetter sets the price for a specific market order
type MarketPriceSetter interface {
	// Price sets the price for the order.
	// NOTE: This is currently mandatory for market orders. Depending on what
	// needs to be purchasedd or what needs to be sold, the currency amounts
	// will be converted and used to calculate the expected amount of currency
	// to be purchased (this calculation may not be accurate). Until price
	// finding is implemented, you should pre-calculate the price using the
	// potential options below, listed from most accurate to least accurate (websocket preferred):
	//  1. Orderbook -
	//     a. Calculate the potential average price across tranches based on the
	//     liquidity side of the order book and the deployment amount.
	//     b. Use the best bid or ask price depending on the order side.
	//  2. Ticker -
	//     a. Use the best bid or ask price depending on the order side.
	//     b. Use the last price.
	Price(price float64) MarketAmountSetter
}

// MarketAmountSetter sets the amount to sell or an amount to purchase for a
// specific market order.
type MarketAmountSetter interface {
	// Purchase defines the currency you would like to purchase and the amount.
	Purchase(c currency.Code, amount float64) MarketAssetSetter
	// Sell defines the currency you would like to sell and the amount.
	Sell(c currency.Code, amount float64) MarketAssetSetter
}

// MarketAssetSetter sets the asset type for a specific market order
type MarketAssetSetter interface {
	// Asset defines the asset type for the order.
	Asset(a asset.Item) MarketOptionsSetter
}

// Receipt is the result of submitting an order to the exchange.
type Receipt struct {
	// Outbound is the order that was submitted to the exchange
	Outbound *order.Submit
	// Response is the response from the exchange
	Response *order.SubmitResponse
	// OrderAmounts is the calculated amounts for the order
	OrderAmounts
}

// MarketOptionsSetter has a collection of optional parameters for a market
// order. This also includes finisher methods to pre-allocate and submit the
// order.
type MarketOptionsSetter interface {
	// FeePercentage defines the fee percentage to be used for the order. This is
	// used to calculate the fee adjusted amount. If this is not set this might not
	// execute due to insufficient funds and or the returned amount might not be
	// closer to the expected amount.
	FeePercentage(feePercentage float64) MarketOptionsSetter
	// FeeRate defines the fee rate to be used for the order. This is used to
	// calculate the fee adjusted amount. If this is not set this might not execute
	// due to insufficient funds and or the returned amount might not be closer to
	// the expected amount.
	FeeRate(feeRate float64) MarketOptionsSetter
	Submit(ctx context.Context) (*Receipt, error)
	PreAlloc() (Submitter, error)
}

// Submitter is an interface that allows pre-allocated orders to be submitted.
type Submitter interface {
	Submit(ctx context.Context) (*Receipt, error)
}

// MarketOrder is a market order that can be pre-allocated and submitted.
type MarketOrder struct {
	*OrderBuilder
}

// OrderBuilder provides a convenient way to construct orders. All values will
// be validated in the `Validate` and `Submit` methods. If the exchange does not
// support the specified order type or order side, an error will be returned.
type OrderBuilder struct {
	exch               IBotExchange           // The exchange associated with the order
	pair               currency.Pair          // The currency pair for the order
	orderType          order.Type             // The type of the order (e.g., market, limit)
	price              float64                // The price of the order (for limit orders)
	assetType          asset.Item             // The asset type (e.g., spot, margin)
	purchasing         bool                   // Indicates if the order is for purchasing or selling
	exchangingCurrency currency.Code          // The currency to be exchanged
	currencyAmount     float64                // The amount of currency to be traded
	feePercentage      float64                // The fee percentage for the order
	config             order.SubmissionConfig // The configuration for order submission
	orderParams        *currency.OrderParameters
	preAllocatedSubmit *order.Submit
	preAllocateAmounts *OrderAmounts
}

// NewOrderBuilder returns a new OrderBuilder, which provides a more intuitive
// way to construct orders. This feature is not supported by all exchanges.
// Currently, only spot orders are supported. Note that market orders require
// a price to be set, as it is used to calculate the expected amount of currency
// to be purchased or sold based on the order side and exchange requirements.
// Interfacing has been used to force standard usage of the builder. Depending
// on future implementations the need for required parameters will change.
// This will handle the following:
//  1. Precision calculations for the order. e.g. If the exchange requires a
//     precision amount only 8 decimal places the amount request will
//     automatically be rounded to the nearest 8 decimal places. Then the amount
//     will be checked against the minimum  and maximum order amounts for
//     deployability.
//  2. Fee calculations for the order. e.g. If a fee percentage is set and the
//     exchange fee applies to the selling currency or purchased currency then
//     amounts are adjusted to compensate for the fee. To ensure deployability.
//  3. Order amounts. If you define an amount you want to sell or purchase
//     the order builder will calculate the expected amount of currency to be
//     purchased or sold based on the order side and exchange requirements. e.g
//     If you want to sell 1000 USDT to purchase BTC and the exchange requires
//     base amounts for the amounts it will then calculate the expected amount
//     of BTC to be purchased from the set 'Price' method which is required for
//     market orders.
//
// TODO: Consider adding hook definitions as parameters to allow using the
// order builder without integrating with GCT's services. This would enable
// using only needing an exchange wrapper library.
func (b *Base) NewOrderBuilder(exch IBotExchange) OrderTypeSetter {
	if b == nil {
		return &OrderBuilder{}
	}
	return &OrderBuilder{exch: exch, config: b.SubmissionConfig}
}

// Market sets the order type to market order.
func (o *OrderBuilder) Market() MarketPairSetter { // TODO: Might merge this with purchase and sell methods and then have a separate method for limit orders.
	o.orderType = order.Market
	return &MarketOrder{o}
}

// Pair sets the currency pair
func (m *MarketOrder) Pair(pair currency.Pair) MarketPriceSetter {
	m.pair = pair
	return m
}

// Price sets the price for the order.
// NOTE: This is currently mandatory for market orders. Depending on what
// needs to be purchasedd or what needs to be sold, the currency amounts
// will be converted and used to calculate the expected amount of currency
// to be purchased (this calculation may not be accurate). Until price
// finding is implemented, you should pre-calculate the price using the
// potential options below, listed from most accurate to least accurate (websocket preferred):
//  1. Orderbook -
//     a. Calculate the potential average price across tranches based on the
//     liquidity side of the order book and the deployment amount.
//     b. Use the best bid or ask price depending on the order side.
//  2. Ticker -
//     a. Use the best bid or ask price depending on the order side.
//     b. Use the last price.
func (m *MarketOrder) Price(price float64) MarketAmountSetter {
	m.price = price
	return m
}

// Purchase defines the currency you would like to purchase and the amount.
func (m *MarketOrder) Purchase(c currency.Code, amount float64) MarketAssetSetter { // TODO: Might swap params around.
	m.purchasing = true
	m.exchangingCurrency = c
	m.currencyAmount = amount
	return m
}

// Sell defines the currency you would like to sell and the amount.
func (m *MarketOrder) Sell(c currency.Code, amount float64) MarketAssetSetter { // TODO: Might swap params around.
	m.purchasing = false
	m.exchangingCurrency = c
	m.currencyAmount = amount
	return m
}

// Asset defines the asset type
func (m *MarketOrder) Asset(a asset.Item) MarketOptionsSetter {
	m.assetType = a
	return m
}

// FeePercentage defines the fee percentage to be used for the order. This is
// used to calculate the fee adjusted amount. If this is not set this might not
// execute due to insufficient funds and or the returned amount might not be
// closer to the expected amount.
func (m *MarketOrder) FeePercentage(f float64) MarketOptionsSetter {
	m.feePercentage = f
	return m
}

// FeeRate defines the fee rate to be used for the order. This is used to
// calculate the fee adjusted amount. If this is not set this might not execute
// due to insufficient funds and or the returned amount might not be closer to
// the expected amount.
func (m *MarketOrder) FeeRate(f float64) MarketOptionsSetter {
	m.feePercentage = f * 100
	return m
}

// validate will check the order builder values and return an error if values
// are not set correctly.
func (o *OrderBuilder) validate() error {
	if o == nil {
		return errNilOrderBuilder
	}

	if o.exch == nil {
		return ErrExchangeIsNil
	}

	if o.config == (order.SubmissionConfig{}) {
		return common.ErrFunctionNotSupported
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
			o.orderParams, err = o.pair.MarketBuyOrderParameters(o.exchangingCurrency)
		} else {
			o.orderParams, err = o.pair.MarketSellOrderParameters(o.exchangingCurrency)
		}
	// TODO: case o.orderType == order.Limit:
	default:
		return fmt.Errorf("%w: only asset type 'Spot' is supported for now", asset.ErrNotSupported)
	}

	// Note: Fee is optional and can be <= 0 for rebates.
	// TODO: Add rebate functionality ad-hoc
	return err
}

// PreAlloc will attempt to pre-allocate the order for the exchange. If the
// exchange does not support the order type or order side it will return an
// error.
func (o *OrderBuilder) PreAlloc() (Submitter, error) {
	if o == nil {
		return nil, errOrderBuilderIsNil
	}

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
	if !o.orderParams.IsBuySide {
		side = order.Sell
	}

	expectedPurchasedAmount, err := o.postOrderAdjustToPurchased(preOrderPrecisionAdjustedAmount, preOrderPrecisionAdjustedPrice)
	if err != nil {
		return nil, err
	}

	expectedPurchasedAmountFeeAdjusted, err := o.reduceOrderAmountByFee(expectedPurchasedAmount, false /*IsPreOrder*/)
	if err != nil {
		return nil, err
	}

	o.preAllocatedSubmit = &order.Submit{
		Exchange:  o.exch.GetName(),
		Pair:      o.pair,
		Side:      side,
		Amount:    preOrderPrecisionAdjustedAmount,
		AssetType: asset.Spot,
		Type:      o.orderType,
		Price:     preOrderPrecisionAdjustedPrice,
	}

	o.preAllocateAmounts = &OrderAmounts{
		PreOrderAmount:                   termAdjusted,
		PreOrderFeeAdjustedAmount:        preOrderFeeAdjusted,
		PreOrderPrecisionAdjustedAmount:  preOrderPrecisionAdjustedAmount,
		PreOrderPrecisionAdjustedPrice:   preOrderPrecisionAdjustedPrice,
		PostOrderExpectedPurchasedAmount: expectedPurchasedAmount,
		PostOrderFeeAdjustedAmount:       expectedPurchasedAmountFeeAdjusted,
	}

	return o, nil
}

// Submit will attempt to submit the order to the exchange. If the exchange
// does not support the order type or order side it will return an error.
// TODO: Add batch support
func (o *OrderBuilder) Submit(ctx context.Context) (*Receipt, error) {
	if o == nil {
		return nil, errOrderBuilderIsNil
	}

	if o.preAllocatedSubmit == nil {
		_, err := o.PreAlloc()
		if err != nil {
			return nil, err
		}
	}

	resp, err := o.exch.SubmitOrder(ctx, o.preAllocatedSubmit)
	if err != nil {
		return nil, err
	}

	// TODO: Order check post-order hook. See if the order has actually been
	// placed and get details.

	// TODO: Balance check post-order hook. See what has actually been purchased.
	return &Receipt{
		Outbound:     o.preAllocatedSubmit,
		Response:     resp,
		OrderAmounts: *o.preAllocateAmounts,
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
		return 0, fmt.Errorf("convertOrderAmountToTerm: %w", errSubmissionConfigInvalid)
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
		return 0, fmt.Errorf("reduceOrderAmountByFee: %w", errSubmissionConfigInvalid)
	}
	return math.ReduceByPercentage(amount, o.feePercentage), nil
}

// orderAmountAdjustToPrecision changes the amount to the required exchange
// defined precision.
func (o *OrderBuilder) orderAmountPriceAdjustToPrecision(amount, price float64) (adjAmount, adjPrice float64, err error) {
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
			amount = math.AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
		}
		err = CheckAmounts(&limits, amount, false /*isQuote*/)
	case o.config.OrderSellingAmountsRequired:
		if o.orderParams.SellingCurrency.Equal(o.pair.Base) {
			if limits.AmountStepIncrementSize != 0 {
				amount = math.AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
			}
			err = CheckAmounts(&limits, amount, false /*isQuote*/)
		} else {
			if limits.QuoteStepIncrementSize != 0 {
				amount = math.AdjustToFixedDecimal(amount, limits.QuoteStepIncrementSize)
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
		// Client inputted limit order price needs to be adjusted to the required
		// precision.
		price = math.AdjustToFixedDecimal(price, limits.PriceStepIncrementSize)
	}

	return amount, price, nil
}

// orderPurchasedAmountAdjustToPrecision changes the amount to the required
// exchange defined precision. This is for when an exchange slams your expected
// purchased amount with a holy math.floor bat. If this is occurring you will
// actually incur a higher fee rate as a result.
func (o *OrderBuilder) orderPurchasedAmountAdjustToPrecision(amount float64) (float64, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w: %v", errAmountInvalid, amount)
	}

	limits, err := o.exch.GetOrderExecutionLimits(o.assetType, o.pair)
	if err != nil { // Precision in this case is definitely needed.
		return 0, err
	}

	switch {
	case o.config.OrderBaseAmountsRequired:
		return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", common.ErrNotYetImplemented)
	case o.config.OrderSellingAmountsRequired:
		if o.orderParams.PurchasingCurrency.Equal(o.pair.Base) {
			if limits.AmountStepIncrementSize == 0 {
				return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", errAmountStepIncrementSizeIsZero)
			}
			amount = math.AdjustToFixedDecimal(amount, limits.AmountStepIncrementSize)
		} else {
			if limits.QuoteStepIncrementSize == 0 {
				return 0, fmt.Errorf("orderPurchasedAmountAdjustToPrecision %w", errQuoteStepIncrementSizeIsZero)
			}
			amount = math.AdjustToFixedDecimal(amount, limits.QuoteStepIncrementSize)
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
		if !o.orderParams.PurchasingCurrency.Equal(o.pair.Base) {
			return amount * price, nil
		}
		return amount, nil
	case o.config.OrderSellingAmountsRequired:
		if o.orderParams.SellingCurrency.Equal(o.pair.Base) {
			return amount * price, nil
		}
		return amount / price, nil
	default:
		return 0, fmt.Errorf("PostOrderAdjustToPurchased %w", errSubmissionConfigInvalid)
	}
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
