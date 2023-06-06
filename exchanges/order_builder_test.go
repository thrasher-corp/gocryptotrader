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
					SpecificSellingAmountsRequired: true,
					FeeAppliedToPurchasedCurrency:  true,
					RequiresParameterLimits:        true,
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

	if received.Price != expected.Price {
		t.Fatalf("received: %v expected: %v", received.Price, expected.Price)
	}
}

// var testExchange IBotExchange

// btcusdt := currency.NewPair(currency.BTC, currency.USDT)

// receipt, err := builder.
// 	Pair(btcusdt).
// 	Market().
// 	Sell(currency.BTC, 1).
// 	Asset(asset.Spot).
// 	FeePercentage(0.1).
// 	Submit(context.Background(), testExchange)
// if err != nil {
// 	t.Fatal(err)
// }

// fmt.Println(receipt)

// receipt, err = builder.
// 	Pair(btcusdt).
// 	Market().
// 	Purchase(currency.BTC, 1).
// 	Asset(asset.Spot).
// 	FeePercentage(0.1).
// 	Submit(context.Background(), testExchange)
// if err != nil {
// 	t.Fatal(err)
// }

// fmt.Println(receipt)

// receipt, err = builder.
// 	Pair(btcusdt).
// 	Limit().
// 	Price(40000).
// 	Sell(currency.USDT, 100).
// 	Asset(asset.Spot).
// 	FeePercentage(0.1).
// 	Submit(context.Background(), testExchange)
// if err != nil {
// 	t.Fatal(err)
// }

// fmt.Println(receipt)

// receipt, err = builder.
// 	Pair(btcusdt).
// 	Limit().
// 	Price(35000).
// 	Purchase(currency.USDT, 100).
// 	Asset(asset.Spot).
// 	FeePercentage(0.1).
// 	Submit(context.Background(), testExchange)
// if err != nil {
// 	t.Fatal(err)
// }

// fmt.Println(receipt)
