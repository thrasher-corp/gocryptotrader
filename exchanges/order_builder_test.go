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
	limitError error
	IBotExchange
}

// {"name":"BTCUSDT","alias":"BTCUSDT","baseCurrency":"BTC","quoteCurrency":"USDT","basePrecision":"0.000001","quotePrecision":"0.00000001","minTradeQuantity":"0.000048","minTradeAmount":"1","maxTradeQuantity":"71.73956243","maxTradeAmount":"2000000","minPricePrecision":"0.01","category":1,"showStatus":true,"innovation":false}
func (t *TestExchange) GetOrderExecutionLimits(a asset.Item, p currency.Pair) (order.MinMaxLevel, error) {
	if t.limitError != nil {
		return order.MinMaxLevel{}, t.limitError
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

func TestNewOrderBuilder(t *testing.T) {
	t.Parallel()

	var b *Base
	_, err := b.NewOrderBuilder()
	if !errors.Is(err, ErrExchangeIsNil) {
		t.Fatalf("received: %v expected: %v", err, ErrExchangeIsNil)
	}

	b = &Base{}
	_, err = b.NewOrderBuilder()
	if !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Fatalf("received: %v expected: %v", err, common.ErrFunctionNotSupported)
	}

	b.SubmissionConfig.FeeAppliedToSellingCurrency = true

	builder, err := b.NewOrderBuilder()
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
	err := builder.validate(nil)
	if !errors.Is(err, errOrderBuilderIsNil) {
		t.Fatalf("received: %v expected: %v", err, errOrderBuilderIsNil)
	}

	builder = &OrderBuilder{}
	err = builder.validate(nil)
	if !errors.Is(err, ErrExchangeIsNil) {
		t.Fatalf("received: %v expected: %v", err, ErrExchangeIsNil)
	}

	err = builder.validate(&TestExchange{})
	if !errors.Is(err, errPriceUnset) {
		t.Fatalf("received: %v expected: %v", err, errPriceUnset)
	}

	builder.Price(1)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: %v expected: %v", err, currency.ErrCurrencyPairEmpty)
	}

	builder.Pair(currency.NewPair(currency.BTC, currency.USDT))
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, errOrderTypeUnset) {
		t.Fatalf("received: %v expected: %v", err, errOrderTypeUnset)
	}

	builder.Limit()
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, errAssetTypeUnset) {
		t.Fatalf("received: %v expected: %v", err, errAssetTypeUnset)
	}

	builder.Asset(asset.Spot)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, currency.ErrCurrencyCodeEmpty) {
		t.Fatalf("received: %v expected: %v", err, currency.ErrCurrencyCodeEmpty)
	}

	builder.exchangingCurrency = currency.BTC
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, errAmountUnset) {
		t.Fatalf("received: %v expected: %v", err, errAmountUnset)
	}

	builder.Sell(currency.BTC, 1)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.Market()
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.Purchase(currency.BTC, 1)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.FeePercentage(0.1)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.FeeRate(0.001)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.Limit().Purchase(currency.BTC, 1)
	err = builder.validate(&TestExchange{})
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v expected: %v", err, nil)
	}

	builder.orderType = order.IOS
	err = builder.validate(&TestExchange{})
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
	_, _, err := builder.orderAmountPriceAdjustToPrecision(0, 0, nil)
	if !errors.Is(err, errAmountInvalid) {
		t.Fatalf("received: %v expected: %v", err, errAmountInvalid)
	}

	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 0, nil)
	if !errors.Is(err, errPriceInvalid) {
		t.Fatalf("received: %v expected: %v", err, errPriceInvalid)
	}

	var errTest = errors.New("test error") // Return strange error
	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 25000, &TestExchange{limitError: errTest})
	if !errors.Is(err, errTest) {
		t.Fatalf("received: %v expected: %v", err, errTest)
	}

	builder.config.RequiresParameterLimits = true // Do not skip if not deployed
	_, _, err = builder.orderAmountPriceAdjustToPrecision(1, 25000, &TestExchange{limitError: order.ErrExchangeLimitNotLoaded})
	if !errors.Is(err, order.ErrExchangeLimitNotLoaded) {
		t.Fatalf("received: %v expected: %v", err, order.ErrExchangeLimitNotLoaded)
	}

	builder.config.RequiresParameterLimits = false // Skip if not deployed
	amount, price, err := builder.orderAmountPriceAdjustToPrecision(1, 25000, &TestExchange{limitError: order.ErrExchangeLimitNotLoaded})
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
	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033, &TestExchange{})
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
	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033, &TestExchange{})
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
	_, _, err = builder.orderAmountPriceAdjustToPrecision(0.0000477777, 25000.0033, &TestExchange{})
	if !errors.Is(err, errAmountTooLow) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooLow)
	}

	// base over maximum 71.73956243
	_, _, err = builder.orderAmountPriceAdjustToPrecision(71.7395633333, 25000.0033, &TestExchange{})
	if !errors.Is(err, errAmountTooHigh) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooHigh)
	}

	builder.config.OrderBaseAmountsRequired = false
	builder.config.OrderSellingAmountsRequired = true
	builder.aspect = &currency.OrderAspect{
		SellingCurrency: currency.BTC,
	}

	amount, price, err = builder.orderAmountPriceAdjustToPrecision(1.0000000000001, 25000.0033, &TestExchange{})
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

	amount, price, err = builder.orderAmountPriceAdjustToPrecision(25000.0000000001, 25000.0033, &TestExchange{})
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
	_, _, err = builder.orderAmountPriceAdjustToPrecision(0.50000000001, 25000.0033, &TestExchange{})
	if !errors.Is(err, errAmountTooLow) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooLow)
	}

	// quote over maximum 2000000
	_, _, err = builder.orderAmountPriceAdjustToPrecision(2000001.0000000001, 25000.0033, &TestExchange{})
	if !errors.Is(err, errAmountTooHigh) {
		t.Fatalf("received: %v expected: %v", err, errAmountTooHigh)
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

	testCases := []struct {
		ExchangeName                string
		ExpectedMarketPurchaseOrder *Receipt
		ExpectedMarketSellOrder     *Receipt
		ExpectedLimitPurchaseOrder  *Receipt
		ExpectedLimitSellOrder      *Receipt
	}{
		{
			ExchangeName: "bybit",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.ExchangeName, func(t *testing.T) {
			var b *Base
			switch tc.ExchangeName {
			case "bybit":
				b = &Base{SubmissionConfig: order.SubmissionConfig{
					OrderSellingAmountsRequired:   true,
					FeeAppliedToPurchasedCurrency: true,
					RequiresParameterLimits:       true,
				}}
			default:
				t.Fatal("exchange not found")
			}

			pair := currency.NewPair(currency.BTC, currency.USDT)
			marketPurchaseBase, err := b.NewOrderBuilder()
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := marketPurchaseBase.
				Pair(pair).
				Market().
				Price(25000).
				Purchase(currency.BTC, 1).
				Asset(asset.Spot).
				FeePercentage(0.1).
				Submit(context.Background(), &TestExchange{})
			if err != nil {
				t.Fatal(err)
			}
			checkReceipts(t, receipt, tc.ExpectedMarketPurchaseOrder)

			marketPurchaseQuote, err := b.NewOrderBuilder()
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
				Submit(context.Background(), &TestExchange{})
			if err != nil {
				t.Fatal(err)
			}
			checkReceipts(t, receipt, tc.ExpectedMarketPurchaseOrder)

			marketSellBase, err := b.NewOrderBuilder()
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
				Submit(context.Background(), &TestExchange{})
			if err != nil {
				t.Fatal(err)
			}
			checkReceipts(t, receipt, tc.ExpectedMarketPurchaseOrder)

			marketSellQuote, err := b.NewOrderBuilder()
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
				Submit(context.Background(), &TestExchange{})
			if err != nil {
				t.Fatal(err)
			}
			checkReceipts(t, receipt, tc.ExpectedMarketPurchaseOrder)
		})
	}
}

func checkReceipts(t *testing.T, received, expected *Receipt) {
	t.Helper()

	if received == nil {
		if expected != nil {
			t.Fatalf("received: %v expected: %v", received, expected)
		}
		return
	}

	if received.Builder == nil {
		t.Fatal("builder is nil")
	}

}
