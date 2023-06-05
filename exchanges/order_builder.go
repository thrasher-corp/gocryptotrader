package exchange

import (
	"context"
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type OrderBuilder struct {
	submit order.Submit
	config order.SubmissionConfig
	pre    OrderHooks
	post   OrderHooks
	fee    float64
}

func (b *Base) NewOrderBuilder() (*OrderBuilder, error) {
	if b.SubmissionConfig == (order.SubmissionConfig{}) {
		return nil, errors.New("cannot use method")
	}
	return &OrderBuilder{config: b.SubmissionConfig}, nil
}

type OrderHooks struct {
}

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

func (o *OrderBuilder) Pair(pair currency.Pair) *OrderBuilder {
	o.submit.Pair = pair
	return o
}

func (o *OrderBuilder) Price(price float64) *OrderBuilder {
	o.submit.Price = price
	return o
}

func (o *OrderBuilder) Amount(amount float64) *OrderBuilder {
	o.submit.Amount = amount
	return o
}

func (o *OrderBuilder) Limit() *OrderBuilder {
	o.submit.Type = order.Limit
	return o
}

func (o *OrderBuilder) Market() *OrderBuilder {
	o.submit.Type = order.Market
	return o
}

func (o *OrderBuilder) Buy() *OrderBuilder {
	o.submit.Side = order.Buy
	return o
}

func (o *OrderBuilder) Sell() *OrderBuilder {
	o.submit.Side = order.Sell
	return o
}

func (o *OrderBuilder) Asset(a asset.Item) *OrderBuilder {
	o.submit.AssetType = a
	return o
}

func (o *OrderBuilder) CustomID(id string) *OrderBuilder {
	o.submit.ClientOrderID = id
	return o
}

func (o *OrderBuilder) Fee(f float64) *OrderBuilder {
	o.fee = f
	return o
}

var errExchangeIsNil = errors.New("exchange is nil")

func (o *OrderBuilder) Submit(ctx context.Context, exch IBotExchange) (*Receipt, error) {
	if exch == nil {
		return nil, errExchangeIsNil
	}

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

// AdjustByFee reduces the selling amount by the fee percentage to ensure either
// the order is not rejected due to insufficient funds `pre order` or the
// purchased amount is correctly reduced.
func (o *OrderBuilder) AdjustByFee(amount float64, preOrder bool) (float64, error) {
	return 0, nil // TODO
}

// PreOrderAdjustToBase changes the amount to base currency if required.
func (o *OrderBuilder) PreOrderAdjustToBase(amount, price float64) (float64, error) {
	return 0, nil // TODO
}

// PreOrderAdjusToPrecision changes the amount to the required precision.
func (o *OrderBuilder) PreOrderAdjustToPrecision(amount float64) (float64, error) {
	return 0, nil // TODO
}

// PostOrderAdjustToPurchased converts the amount to the purchased amount
func (o *OrderBuilder) PostOrderAdjustToPurchased(amount, price float64) (float64, error) {
	return 0, nil // TODO
}
