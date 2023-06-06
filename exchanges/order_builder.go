package exchange

import (
	"context"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var (
	// ErrExchangeIsNil is a common error
	ErrExchangeIsNil        = errors.New("exchange is nil")
	errOrderBuilderIsNil    = errors.New("order builder is nil")
	errPriceUnset           = errors.New("price is unset")
	errAmountUnset          = errors.New("amount is unset")
	errOrderTypeUnset       = errors.New("order type is unset")
	errAssetTypeUnset       = errors.New("asset type is unset")
	errOrderTypeUnsupported = errors.New("order type unsupported")
)

// OrderBuilder is a helper struct to assist in building orders. All values
// will be checked in validate and/or Submit. If the exchange does not support
// the order type or order side it will return an error.
type OrderBuilder struct {
	pair               currency.Pair
	orderType          order.Type
	price              float64
	assetType          asset.Item
	purchasing         bool
	exchangingCurrency currency.Code
	currencyAmount     float64
	fee                float64
	config             order.SubmissionConfig

	aspect   *currency.OrderAspect
	outbound *order.Submit

	// TODO: Add pre and post order hooks
}

// NewOrderBuilder returns a new order builder which attempts to provide a more
// intuative way to build orders. This is not supported by all exchanges. Spot
// orders are only supported at this time. NOTE: Market orders require a price
// to be set, this is used to calculate the expected amount of currency to be
// purchased or sold depending on the order side and requirements of the exchange.
func (b *Base) NewOrderBuilder() (*OrderBuilder, error) {
	if b == nil {
		return nil, ErrExchangeIsNil
	}
	if b.SubmissionConfig == (order.SubmissionConfig{}) {
		return nil, common.ErrFunctionNotSupported
	}
	return &OrderBuilder{config: b.SubmissionConfig}, nil
}

// Pair sets the currency pair
func (o *OrderBuilder) Pair(pair currency.Pair) *OrderBuilder {
	o.pair = pair
	return o
}

// Price sets the limit/market price. This is used for market orders, amounts
// will be used to calculate the expected amount of currency to be purchased.
// If Market use last, best bid or ask for price.
func (o *OrderBuilder) Price(price float64) *OrderBuilder {
	o.price = price
	return o
}

// Limit sets the order type to limit order.
func (o *OrderBuilder) Limit() *OrderBuilder {
	o.orderType = order.Limit
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
	o.fee = f
	return o
}

// FeeRate defines the fee rate to be used for the order. This is used to
// calculate the fee adjusted amount. If this is not set this might not execute
// due to insufficient funds and or the returned amount might not be closer to
// the expected amount.
func (o *OrderBuilder) FeeRate(f float64) *OrderBuilder {
	o.fee = f * 100
	return o
}

// validate will check the order builder values and return an error if values
// are not set correctly.
func (o *OrderBuilder) validate(exch IBotExchange) error {
	if o == nil {
		return errOrderBuilderIsNil
	}

	if exch == nil {
		return ErrExchangeIsNil
	}

	if o.price <= 0 {
		return fmt.Errorf("%w: please use method 'Price'", errPriceUnset)
	}

	if o.pair.IsEmpty() {
		return fmt.Errorf("%w: please use method 'Pair'", currency.ErrCurrencyPairEmpty)
	}

	if o.orderType == order.UnknownType {
		return fmt.Errorf("%w: please use method(s) 'Market' or `Limit`", errOrderTypeUnset)
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
	case o.orderType == order.Limit:
		if o.purchasing {
			o.aspect, err = o.pair.LimitBuyOrderAspect(o.exchangingCurrency)
		} else {
			o.aspect, err = o.pair.LimitSellOrderAspect(o.exchangingCurrency)
		}
	default:
		return fmt.Errorf("%w: %v", errOrderTypeUnsupported, o.orderType)
	}

	// Note: Fee is optional and can be <= 0 for rebates.
	return err
}

// Receipt ...
type Receipt struct {
	Price                           float64
	PreOrderSellingAmount           float64
	PreOrderFeeAdjustedAmount       float64
	PreOrderBaseAdjustedAmount      float64
	PreOrderPrecisionAdjustedAmount float64

	PostOrderExpectedPurchasedAmount float64
	PostOrderFeeAdjustedAmount       float64

	Response                      *order.SubmitResponse
	PostResponseFeeAdjustedAmount float64 // Check diff to ActualPurchasedAmount

	ActualPurchasedAmount float64
}

// Submit will attempt to submit the order to the exchange. If the exchange
// does not support the order type or order side it will return an error.
func (o *OrderBuilder) Submit(ctx context.Context, exch IBotExchange) (*Receipt, error) {
	err := o.validate(exch)
	if err != nil {
		return nil, err
	}

	// TODO: Purchase -> Sell

	// TODO: Adjust to fee

	// TODO: Adjust to precision

	// return &Receipt{
	// 	Price:                            price,
	// 	PreOrderSellingAmount:            amount,
	// 	PreOrderFeeAdjustedAmount:        feeAdjustedAmount,
	// 	PreOrderBaseAdjustedAmount:       baseAdjustedAmount,
	// 	PreOrderPrecisionAdjustedAmount:  precisionAdjustedAmount,
	// 	PostOrderExpectedPurchasedAmount: expectedPurchase,
	// 	PostOrderFeeAdjustedAmount:       purchasedFeeAdjusted,
	// 	Response:                         resp,
	// 	PostResponseFeeAdjustedAmount:    respFeeAdjusted,
	// 	ActualPurchasedAmount:            actualBalance,
	// }, nil

	return nil, nil
}

// TODO: Add *order.Submit pre allocation and subsequent Submit call.

// AdjustByFee reduces the selling amount by the fee percentage to ensure either
// the order is not rejected due to insufficient funds `pre order` or the
// purchased amount is correctly reduced.
func (o *OrderBuilder) preOrderAdjustByFee(amount float64, preOrder bool) (float64, error) {
	return 0, nil // TODO
}

// PreOrderAdjustToBase changes the amount to base currency if required.
func (o *OrderBuilder) preOrderAdjustToBase(amount, price float64) (float64, error) {
	return 0, nil // TODO
}

// PreOrderAdjustToPrecision changes the amount to the required precision.
func (o *OrderBuilder) preOrderAdjustToPrecision(amount float64) (float64, error) {
	return 0, nil // TODO
}

// PostOrderAdjustToPurchased converts the amount to the purchased amount.
func (o *OrderBuilder) postOrderAdjustToPurchased(amount, price float64) (float64, error) {
	return 0, nil // TODO
}

func (o *OrderBuilder) postOrderAdjustByFee(amount float64, preOrder bool) (float64, error) {
	return 0, nil // TODO
}
