package exchange

import (
	"context"
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type TestExchange struct {
	IBotExchange
	limitError error
	empty      bool
	wow        Base
}

// {"name":"BTCUSDT","alias":"BTCUSDT","baseCurrency":"BTC","quoteCurrency":"USDT","basePrecision":"0.000001","quotePrecision":"0.00000001","minTradeQuantity":"0.000048","minTradeAmount":"1","maxTradeQuantity":"71.73956243","maxTradeAmount":"2000000","minPricePrecision":"0.01","category":1,"showStatus":true,"innovation":false}
func (t *TestExchange) GetOrderExecutionLimits(a asset.Item, p currency.Pair) (order.MinMaxLevel, error) {
	if t.limitError != nil {
		return order.MinMaxLevel{}, t.limitError
	}
	if t.empty {
		return order.MinMaxLevel{}, nil
	}
	return order.MinMaxLevel{
		Asset:                   a,
		Pair:                    p,
		AmountStepIncrementSize: 0.000001,
		QuoteStepIncrementSize:  0.00000001,
		MinimumBaseAmount:       0.000048,
		MaximumBaseAmount:       71.73956243,
		MinimumQuoteAmount:      1,
		MaximumQuoteAmount:      2000000,
		PriceStepIncrementSize:  0.01,
	}, nil
}

func (t *TestExchange) ConstructOrder() (*OrderBuilder, error) {
	return t.wow.NewOrderBuilder(t)
}

func (t *TestExchange) SubmitOrder(ctx context.Context, s *order.Submit) (*order.SubmitResponse, error) {
	return s.DeriveSubmitResponse("TEST")
}

func (t *TestExchange) GetName() string { return "TestExchange" }

func TestNewOrderBuilder(t *testing.T) {
	t.Parallel()

	var b *Base
	_, err := b.NewOrderBuilder(nil)
	if !errors.Is(err, ErrExchangeIsNil) {
		t.Fatalf("received: %v expected: %v", err, ErrExchangeIsNil)
	}

	b = &Base{}
	_, err = b.NewOrderBuilder(&TestExchange{})
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Fatalf("received: %v expected: %v", err, common.ErrFunctionNotSupported)
	}

	b.SubmissionConfig.FeeAppliedToSellingCurrency = true

	builder, err := b.NewOrderBuilder(&TestExchange{})
	if err != nil {
		t.Fatal(err)
	}

	if builder == nil {
		t.Fatal("expected builder")
	}

	if builder.config == (order.SubmissionConfig{}) {
		t.Fatal("expected config")
	}

	if !builder.config.FeeAppliedToSellingCurrency {
		t.Fatal("expected true")
	}
}

func TestValidate(t *testing.T) {
	var builder *OrderBuilder
	err := builder.validate()
	if !errors.Is(err, errOrderBuilderIsNil) {
		t.Fatalf("received: %v expected: %v", err, errOrderBuilderIsNil)
	}

	builder = &OrderBuilder{}
	err = builder.validate()
	if !errors.Is(err, ErrExchangeIsNil) {
		t.Fatalf("received: %v expected: %v", err, ErrExchangeIsNil)
	}

	builder.exch = &TestExchange{}
	err = builder.validate()
	if !errors.Is(err, errPriceUnset) {
		t.Fatalf("received: %v expected: %v", err, errPriceUnset)
	}

	builder.Price(1)
	err = builder.validate()
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	builder.Pair(currency.NewPair(currency.BTC, currency.USDT))
	err = builder.validate()
	if !errors.Is(err, errOrderTypeUnset) {
		t.Fatalf("received: %v expected: %v", err, errOrderTypeUnset)
	}

	builder.Market()
	err = builder.validate()
	if !errors.Is(err, errAssetTypeUnset) {
		t.Fatalf("received: %v expected: %v", err, errAssetTypeUnset)
	}

	builder.Asset(asset.Spot)
	err = builder.validate()
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("received: %v expected: %v", err, currency.ErrCurrencyCodeEmpty)
	}

	builder.exchangingCurrency = currency.BTC
	err = builder.validate()
	if !errors.Is(err, errAmountUnset) {
		t.Fatalf("received: %v expected: %v", err, errAmountUnset)
	}

	builder.Sell(currency.BTC, 1)
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.Market()
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.Purchase(currency.BTC, 1)
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.FeePercentage(0.1)
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.FeeRate(0.001)
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	// builder.Limit().Purchase(currency.BTC, 1)
	err = builder.validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.orderType = order.IOS
	err = builder.validate()
	if !errors.Is(err, errOrderTypeUnsupported) {
		t.Fatalf("received: %v expected: %v", err, errOrderTypeUnsupported)
	}
}

func TestConvertOrderAmountToTerm(t *testing.T) {
	t.Parallel()

	var builder = &OrderBuilder{
		pair:  currency.NewPair(currency.BTC, currency.USDT),
		price: 25000, // 1 BTC = 25000 USDT
	}

	_, err := builder.convertOrderAmountToTerm(0)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, err = builder.convertOrderAmountToTerm(25000)
	if !errors.Is(err, errSubmissionConfigInvalid) {
		t.Fatalf("received: %v expected: %v", err, errSubmissionConfigInvalid)
	}

	builder.config.OrderBaseAmountsRequired = true

	// 25k USD wanting to be sold
	builder.exchangingCurrency = currency.USDT
	term, err := builder.convertOrderAmountToTerm(25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	// 1 BTC wanting to be sold
	builder.exchangingCurrency = currency.BTC
	term, err = builder.convertOrderAmountToTerm(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	builder.purchasing = true

	// 25k USD wanting to be purchased
	builder.exchangingCurrency = currency.USDT
	term, err = builder.convertOrderAmountToTerm(25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	// 1 BTC wanting to be purchased
	builder.exchangingCurrency = currency.BTC
	term, err = builder.convertOrderAmountToTerm(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	builder.config.OrderBaseAmountsRequired = false
	builder.config.OrderSellingAmountsRequired = true
	builder.purchasing = false

	// 25k USD wanting to be sold
	builder.exchangingCurrency = currency.USDT
	term, err = builder.convertOrderAmountToTerm(25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 25000 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	// 1 BTC wanting to be sold
	builder.exchangingCurrency = currency.BTC
	term, err = builder.convertOrderAmountToTerm(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	builder.purchasing = true

	// 25k USD wanting to be purchased
	builder.exchangingCurrency = currency.USDT
	term, err = builder.convertOrderAmountToTerm(25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 1 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}

	// 1 BTC wanting to be purchased
	builder.exchangingCurrency = currency.BTC
	term, err = builder.convertOrderAmountToTerm(1)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if term != 25000 {
		t.Fatalf("received: %v expected: %v", term, 1)
	}
}

func TestReduceOrderAmountByFee(t *testing.T) {
	t.Parallel()

	var builder = &OrderBuilder{}
	_, err := builder.reduceOrderAmountByFee(0, false)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, err = builder.reduceOrderAmountByFee(1, false)
	if !errors.Is(err, errSubmissionConfigInvalid) {
		t.Fatalf("received: %v expected: %v", err, errSubmissionConfigInvalid)
	}

	builder.config.FeeAppliedToSellingCurrency = true
	builder.feePercentage = 10

	amount, err := builder.reduceOrderAmountByFee(100, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 90 {
		t.Fatalf("received: %v expected: %v", amount, 90)
	}

	amount, err = builder.reduceOrderAmountByFee(100, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 100 {
		t.Fatalf("received: %v expected: %v", amount, 100)
	}

	builder.config.FeeAppliedToSellingCurrency = false
	builder.config.FeeAppliedToPurchasedCurrency = true

	amount, err = builder.reduceOrderAmountByFee(100, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 100 {
		t.Fatalf("received: %v expected: %v", amount, 90)
	}

	amount, err = builder.reduceOrderAmountByFee(100, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 90 {
		t.Fatalf("received: %v expected: %v", amount, 100)
	}
}

func TestOrderAmountAdjustToPrecision(t *testing.T) {
	t.Parallel()

	var builder = &OrderBuilder{pair: currency.NewPair(currency.BTC, currency.USDT)}
	_, _, err := builder.orderAmountPriceAdjustToPrecision(0, 0)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 0)
	if !errors.Is(err, errPriceInvalid) {
		t.Fatalf("received: %v expected: %v", err, errPriceInvalid)
	}

	var errTest = errors.New("test error") // Return strange error
	builder.exch = &TestExchange{limitError: errTest}
	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 25000)
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v expected: %v", err, errTest)
	}

	builder.config.RequiresParameterLimits = true // Do not skip if not deployed
	builder.exch = &TestExchange{limitError: order.ErrExchangeLimitNotLoaded}
	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 25000)
	if !errors.Is(err, order.ErrExchangeLimitNotLoaded) {
		t.Fatalf("received: %v expected: %v", err, order.ErrExchangeLimitNotLoaded)
	}

	builder.config.RequiresParameterLimits = false // Skip if not deployed
	amount, price, err := builder.orderAmountPriceAdjustToPrecision(1, 25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 1 {
		t.Fatalf("received: %v expected: %v", amount, 1)
	}

	if price != 25000 {
		t.Fatalf("received: %v expected: %v", price, 25000)
	}

	// purchase/sell 1 BTC market order
	builder.config.OrderBaseAmountsRequired = true
	builder.orderType = order.Market
	builder.exch = &TestExchange{}
	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 1 {
		t.Fatalf("received: %v expected: %v", amount, 1)
	}

	if price != 25000.0033 { // This shouldn't be adjusted in a market order because this technically should be a ticker or ob price.
		t.Fatalf("received: %v expected: %v", price, 25000.0033)
	}

	// purchase/sell 1 BTC limit order
	builder.orderType = order.Limit
	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 1 {
		t.Fatalf("received: %v expected: %v", amount, 1)
	}

	if price != 25000 {
		t.Fatalf("received: %v expected: %v", price, 25000)
	}

	// base under minimum 0.000048
	_, _, err = builder.orderAmountPriceAdjustToPrecision(0.0000477777, 25000.0033)
	if !errors.Is(err, errAmountTooLow) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooLow)
	}

	// base over maximum 71.73956243
	_, _, err = builder.orderAmountPriceAdjustToPrecision(71.7395633333, 25000.0033)
	if !errors.Is(err, errAmountTooHigh) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooHigh)
	}

	builder.config.OrderBaseAmountsRequired = false
	builder.config.OrderSellingAmountsRequired = true
	builder.aspect = &currency.OrderAspect{
		SellingCurrency: currency.BTC,
	}

	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 1 {
		t.Fatalf("received: %v expected: %v", amount, 1)
	}

	if price != 25000 {
		t.Fatalf("received: %v expected: %v", price, 25000)
	}

	builder.aspect = &currency.OrderAspect{
		SellingCurrency: currency.USDT,
	}

	amount, price, err = builder.orderAmountPriceAdjustToPrecision(25000.0000000001, 25000.0033)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 25000 {
		t.Fatalf("received: %v expected: %v", amount, 25000)
	}

	if price != 25000 {
		t.Fatalf("received: %v expected: %v", price, 25000)
	}

	// quote under minimum 1
	_, _, err = builder.orderAmountPriceAdjustToPrecision(0.50000000001, 25000.0033)
	if !errors.Is(err, errAmountTooLow) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooLow)
	}

	// quote over maximum 2000000
	_, _, err = builder.orderAmountPriceAdjustToPrecision(2000001.0000000001, 25000.0033)
	if !errors.Is(err, errAmountTooHigh) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooHigh)
	}
}

func TestOrderPurchasedAmountAdjustToPrecision(t *testing.T) {
	t.Parallel()

	var builder = &OrderBuilder{
		pair: currency.NewPair(currency.BTC, currency.USDT),
		exch: &TestExchange{empty: true},
	}

	_, err := builder.orderPurchasedAmountAdjustToPrecision(0)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, err = builder.orderPurchasedAmountAdjustToPrecision(1)
	if !errors.Is(err, errSubmissionConfigInvalid) {
		t.Fatalf("received: %v expected: %v", err, errSubmissionConfigInvalid)
	}

	builder.config.OrderBaseAmountsRequired = true
	_, err = builder.orderPurchasedAmountAdjustToPrecision(1)
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: %v expected: %v", err, common.ErrNotYetImplemented)
	}

	builder.config.OrderBaseAmountsRequired = false
	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.BTC,
	}
	builder.config.OrderSellingAmountsRequired = true
	_, err = builder.orderPurchasedAmountAdjustToPrecision(1)
	if !errors.Is(err, errAmountStepIncrementSizeIsZero) {
		t.Fatalf("received: %v expected: %v", err, errAmountStepIncrementSizeIsZero)
	}

	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.USDT,
	}
	_, err = builder.orderPurchasedAmountAdjustToPrecision(1)
	if !errors.Is(err, errQuoteStepIncrementSizeIsZero) {
		t.Fatalf("received: %v expected: %v", err, errQuoteStepIncrementSizeIsZero)
	}

	builder.exch = &TestExchange{}

	// Expected purchasing amount in BTC
	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.BTC,
	}
	amount, err := builder.orderPurchasedAmountAdjustToPrecision(0.00018968651647541208)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 0.000189 { // Might have to change this to quote there are things under the hood that I dont have enough information on.
		t.Fatalf("received: %v expected: %v", amount, 0.000189)
	}

	// Expected purchasing amount in USDT
	builder.config.OrderSellingAmountsRequired = true
	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.USDT,
	}
	amount, err = builder.orderPurchasedAmountAdjustToPrecision(4.99662702001)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if amount != 4.99662702 {
		t.Fatalf("received: %v expected: %v", amount, 4.99662702)
	}

}

func TestPostOrderAdjustToPurchased(t *testing.T) {
	t.Parallel()

	builder := &OrderBuilder{pair: currency.NewPair(currency.BTC, currency.USDT)}
	_, err := builder.postOrderAdjustToPurchased(0, 0)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, err = builder.postOrderAdjustToPurchased(1, 0)
	if !errors.Is(err, errPriceInvalid) {
		t.Fatalf("received: %v expected: %v", err, errPriceInvalid)
	}

	_, err = builder.postOrderAdjustToPurchased(1, 1)
	if !errors.Is(err, errSubmissionConfigInvalid) {
		t.Fatalf("received: %v expected: %v", err, errSubmissionConfigInvalid)
	}

	// Sell 1 BTC at 25000
	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.USDT,
	}
	builder.config.OrderBaseAmountsRequired = true
	balance, err := builder.postOrderAdjustToPurchased(1, 25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	if balance != 25000 {
		t.Fatalf("received: %v expected: %v", balance, 25000)
	}

	// Purchase 1 BTC at 25000
	builder.aspect = &currency.OrderAspect{
		PurchasingCurrency: currency.BTC,
	}
	balance, err = builder.postOrderAdjustToPurchased(1, 25000) // Already converted to base
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}
	if balance != 1 {
		t.Fatalf("received: %v expected: %v", balance, 25000)
	}

	builder.config.OrderBaseAmountsRequired = false

	// Selling amounts are used for these orders so they always need to be
	// converted.
	builder.config.OrderSellingAmountsRequired = true
	builder.aspect = &currency.OrderAspect{
		SellingCurrency: currency.USDT,
	}
	balance, err = builder.postOrderAdjustToPurchased(25000, 25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}
	if balance != 1 {
		t.Fatalf("received: %v expected: %v", balance, 25000)
	}

	builder.aspect = &currency.OrderAspect{
		SellingCurrency: currency.BTC,
	}
	balance, err = builder.postOrderAdjustToPurchased(1, 25000)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}
	if balance != 25000 {
		t.Fatalf("received: %v expected: %v", balance, 25000)
	}
}

func TestSubmit(t *testing.T) {
	t.Parallel()
	exch := &TestExchange{
		wow: Base{
			SubmissionConfig: order.SubmissionConfig{
				OrderSellingAmountsRequired:           true,
				FeeAppliedToPurchasedCurrency:         true,
				RequiresParameterLimits:               true,
				FeePostOrderRequiresPrecisionOnAmount: true,
			},
		},
	}
	pair := currency.NewPair(currency.BTC, currency.USDT)
	marketOrderOne, err := exch.ConstructOrder()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := marketOrderOne.
		Pair(pair).
		Market().
		Price(25000).
		Sell(currency.USDT, 5).
		Asset(asset.Spot).
		FeePercentage(0.1).
		Submit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	checkAmounts(t, receipt, OrderAmounts{})

	marketPurchaseQuote, err := exch.ConstructOrder()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = marketPurchaseQuote.
		Pair(pair).
		Market().
		Price(25000).
		Purchase(currency.USDT, 1).
		Asset(asset.Spot).
		FeePercentage(0.1).
		Submit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	checkAmounts(t, receipt, OrderAmounts{})

	marketSellBase, err := exch.ConstructOrder()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = marketSellBase.
		Pair(pair).
		Market().
		Price(25000).
		Sell(currency.BTC, 1).
		Asset(asset.Spot).
		FeePercentage(0.1).
		Submit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	checkAmounts(t, receipt, OrderAmounts{})

	marketSellQuote, err := exch.ConstructOrder()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = marketSellQuote.
		Pair(pair).
		Market().
		Price(25000).
		Sell(currency.USDT, 1).
		Asset(asset.Spot).
		FeePercentage(0.1).
		Submit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	checkAmounts(t, receipt, OrderAmounts{})

}

func checkAmounts(t *testing.T, received *Receipt, expected OrderAmounts) {
	t.Helper()

	if received == nil {
		if expected != (OrderAmounts{}) {
			t.Fatalf("received: %v expected: %v", received, expected)
		}
		return
	}

	if received.Builder == nil {
		t.Fatal("builder is nil")
	}

	if received.Outbound == nil {
		t.Fatal("outbound submit order is nil")
	}

	if received.Response == nil {
		t.Fatal("builder is nil")
	}

	if received.OrderAmounts == (OrderAmounts{}) {
		t.Fatal("order amounts is empty")
	}

	if received.OrderAmounts.PreOrderAmount != expected.PreOrderAmount {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PreOrderAmount, expected.PreOrderAmount)
	}

	if received.OrderAmounts.PreOrderFeeAdjustedAmount != expected.PreOrderFeeAdjustedAmount {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PreOrderFeeAdjustedAmount, expected.PreOrderFeeAdjustedAmount)
	}

	if received.OrderAmounts.PreOrderPrecisionAdjustedPrice != expected.PreOrderPrecisionAdjustedPrice {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PreOrderPrecisionAdjustedPrice, expected.PreOrderPrecisionAdjustedPrice)
	}

	if received.OrderAmounts.PreOrderPrecisionAdjustedAmount != expected.PreOrderPrecisionAdjustedAmount {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PreOrderPrecisionAdjustedAmount, expected.PreOrderPrecisionAdjustedAmount)
	}

	if received.OrderAmounts.PostOrderExpectedPurchasedAmount != expected.PostOrderExpectedPurchasedAmount {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PostOrderExpectedPurchasedAmount, expected.PostOrderExpectedPurchasedAmount)
	}

	if received.OrderAmounts.PostOrderFeeAdjustedAmount != expected.PostOrderFeeAdjustedAmount {
		t.Fatalf("received: %v expected: %v", received.OrderAmounts.PostOrderFeeAdjustedAmount, expected.PostOrderFeeAdjustedAmount)
	}
}
