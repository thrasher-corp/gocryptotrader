package okx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetDualInvestmentCurrencyPairs(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentCurrencyPairs(contextGenerate())
	require.NoError(t, err, "GetDualInvestmentCurrencyPairs must not error")
	if mockTests {
		exp := []DualInvestmentCurrencyPairResponse{{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, OptionType: "C", Underlying: "BTC-USD"}}
		assert.Equal(t, exp, result, "GetDualInvestmentCurrencyPairs should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDualInvestmentCurrencyPairs should return currency pairs")
	}
}

func TestGetDualInvestmentProducts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DualInvestmentProductsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing base currency", arg: &DualInvestmentProductsRequest{QuoteCurrency: currency.USDT, OptionType: "P"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing quote currency", arg: &DualInvestmentProductsRequest{BaseCurrency: currency.BTC, OptionType: "P"}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing option type", arg: &DualInvestmentProductsRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT}, err: errMissingRequiredParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDualInvestmentProducts(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDualInvestmentProducts should reject an invalid request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentProducts(contextGenerate(), &DualInvestmentProductsRequest{BaseCurrency: currency.BTC, QuoteCurrency: currency.USDT, OptionType: "P"})
	require.NoError(t, err, "GetDualInvestmentProducts must not error")
	if mockTests {
		exp := []DualInvestmentProductResponse{
			{
				AbsoluteYield:       0.00232413,
				AnnualisedYield:     0.0541,
				BaseCurrency:        currency.BTC,
				QuoteCurrency:       currency.USDT,
				NotionalCurrency:    currency.USDT,
				ExpiryTime:          types.Time(time.UnixMilli(1774598400000)),
				InterestAccrualTime: types.Time(time.UnixMilli(1773244800000)),
				ListTime:            types.Time(time.UnixMilli(1743150759000)),
				MinimumSize:         10,
				MaximumSize:         6000000,
				OptionType:          "P",
				ProductID:           "BTC-USDT-260327-54500-P",
				QuoteTime:           types.Time(time.UnixMilli(1773243808703)),
				RedeemStartTime:     types.Time(time.UnixMilli(1773244800000)),
				RedeemEndTime:       types.Time(time.UnixMilli(1774594800000)),
				StepSize:            1,
				TradeEndTime:        types.Time(time.UnixMilli(1774584000000)),
				Underlying:          "BTC-USD",
				StrikePrice:         54500,
			},
		}
		assert.Equal(t, exp, result, "GetDualInvestmentProducts should decode every field")
	} else {
		assert.NotNil(t, result, "GetDualInvestmentProducts should return products")
	}
}

func TestRequestDualInvestmentQuote(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DualInvestmentQuoteRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing product ID", arg: &DualInvestmentQuoteRequest{NotionalSize: 0.001, NotionalCurrency: currency.BTC}, err: errMissingRequiredParameter},
		{name: "missing notional size", arg: &DualInvestmentQuoteRequest{ProductID: "BTC-USDT-260312-72000-C", NotionalCurrency: currency.BTC}, err: limits.ErrAmountBelowMin},
		{name: "missing notional currency", arg: &DualInvestmentQuoteRequest{ProductID: "BTC-USDT-260312-72000-C", NotionalSize: 0.001}, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RequestDualInvestmentQuote(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RequestDualInvestmentQuote should reject an invalid request")
		})
	}
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/sfp/dcd/quote", `{"productId":"BTC-USDT-260327-77000-C","notionalSz":"1.5","notionalCcy":"BTC"}`, []map[string]string{{"quoteId": "quoterbpDCD-QUOTE17732116652401234"}})
		_, err := ex.RequestDualInvestmentQuote(t.Context(), &DualInvestmentQuoteRequest{ProductID: "BTC-USDT-260327-77000-C", NotionalSize: 1.5, NotionalCurrency: currency.BTC})
		require.NoError(t, err, "RequestDualInvestmentQuote must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.RequestDualInvestmentQuote(contextGenerate(), &DualInvestmentQuoteRequest{ProductID: "BTC-USDT-260312-72000-C", NotionalSize: 0.001, NotionalCurrency: currency.BTC})
	require.NoError(t, err, "RequestDualInvestmentQuote must not error")
	if mockTests {
		exp := &DualInvestmentQuoteResponse{
			AbsoluteYield:       0.00135182,
			AnnualisedYield:     69.65,
			InterestAccrualTime: types.Time(time.UnixMilli(1773241200000)),
			NotionalSize:        0.001,
			NotionalCurrency:    currency.BTC,
			ProductID:           "BTC-USDT-260312-72000-C",
			QuoteID:             "qtbcDCD-QUOTE17732395560537636",
			ValidUntil:          types.Time(time.UnixMilli(1774584000000)),
			IndexPrice:          69000,
		}
		assert.Equal(t, exp, result, "RequestDualInvestmentQuote should decode every field")
	} else {
		assert.NotEmpty(t, result.QuoteID, "RequestDualInvestmentQuote should return a quote ID")
	}
}

func TestTradeDualInvestment(t *testing.T) {
	t.Parallel()
	_, err := e.TradeDualInvestment(t.Context(), "")
	require.ErrorIs(t, err, errMissingRequiredParameter, "TradeDualInvestment must reject a missing quote ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TradeDualInvestment(contextGenerate(), "quoterbpDCD-QUOTE17732116652401234")
	require.NoError(t, err, "TradeDualInvestment must not error")
	if mockTests {
		exp := &DualInvestmentTradeResponse{QuoteID: "quoterbpDCD-QUOTE17732116652401234", OrderID: "987654321", State: "live"}
		assert.Equal(t, exp, result, "TradeDualInvestment should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "TradeDualInvestment should return an order ID")
	}
}

func TestRequestDualInvestmentRedeemQuote(t *testing.T) {
	t.Parallel()
	_, err := e.RequestDualInvestmentRedeemQuote(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "RequestDualInvestmentRedeemQuote must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.RequestDualInvestmentRedeemQuote(contextGenerate(), "987654321")
	require.NoError(t, err, "RequestDualInvestmentRedeemQuote must not error")
	if mockTests {
		exp := &DualInvestmentRedeemQuoteResponse{
			OrderID:        "987654321",
			QuoteID:        "quoterbcDCD-REDEEM17732116652401234",
			RedeemSize:     1.4856,
			RedeemCurrency: currency.BTC,
			TermRate:       -0.5,
			ValidUntil:     types.Time(time.UnixMilli(1774598400000)),
		}
		assert.Equal(t, exp, result, "RequestDualInvestmentRedeemQuote should decode every field")
	} else {
		assert.NotEmpty(t, result.QuoteID, "RequestDualInvestmentRedeemQuote should return a quote ID")
	}
}

func TestRedeemDualInvestment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DualInvestmentRedeemRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing order ID", arg: &DualInvestmentRedeemRequest{QuoteID: "quoterbcDCD-REDEEM17732116652401234"}, err: order.ErrOrderIDNotSet},
		{name: "missing quote ID", arg: &DualInvestmentRedeemRequest{OrderID: "987654321"}, err: errMissingRequiredParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RedeemDualInvestment(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "RedeemDualInvestment should reject an invalid request")
		})
	}
	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/finance/sfp/dcd/redeem", `{"ordId":"987654322","quoteId":"quoterbcDCD-REDEEM17732116652401235"}`, []map[string]string{{"ordId": "987654322"}})
		_, err := ex.RedeemDualInvestment(t.Context(), &DualInvestmentRedeemRequest{OrderID: "987654322", QuoteID: "quoterbcDCD-REDEEM17732116652401235"})
		require.NoError(t, err, "RedeemDualInvestment must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RedeemDualInvestment(contextGenerate(), &DualInvestmentRedeemRequest{OrderID: "987654321", QuoteID: "quoterbcDCD-REDEEM17732116652401234"})
	require.NoError(t, err, "RedeemDualInvestment must not error")
	if mockTests {
		assert.Equal(t, &DualInvestmentRedeemResponse{OrderID: "987654321", State: "pending_redeem_booking"}, result, "RedeemDualInvestment should decode every field")
	} else {
		assert.NotEmpty(t, result.State, "RedeemDualInvestment should return the order state")
	}
}

func TestGetDualInvestmentOrderState(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentOrderState(t.Context(), "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetDualInvestmentOrderState must reject a missing order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentOrderState(contextGenerate(), "987654321")
	require.NoError(t, err, "GetDualInvestmentOrderState must not error")
	if mockTests {
		assert.Equal(t, &DualInvestmentOrderStateResponse{OrderID: "987654321", State: "live"}, result, "GetDualInvestmentOrderState should decode every field")
	} else {
		assert.NotEmpty(t, result.State, "GetDualInvestmentOrderState should return the order state")
	}
}

func TestGetDualInvestmentOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetDualInvestmentOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDualInvestmentOrderHistory must reject a nil request")
	_, err = e.GetDualInvestmentOrderHistory(t.Context(), &DualInvestmentOrderHistoryRequest{Begin: time.UnixMilli(1773300000000), End: time.UnixMilli(1773200000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetDualInvestmentOrderHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetDualInvestmentOrderHistory(contextGenerate(), &DualInvestmentOrderHistoryRequest{
		ProductID:  "BTC-USDT-260327-77000-C",
		Underlying: "BTC-USD",
		State:      "settled",
		BeginID:    "987654320",
		EndID:      "987654322",
		Begin:      time.UnixMilli(1773200000000),
		End:        time.UnixMilli(1773300000000),
		Limit:      1,
	})
	require.NoError(t, err, "GetDualInvestmentOrderHistory must not error")
	if mockTests {
		exp := []DualInvestmentOrderHistoryResponse{
			{
				OrderID:            "987654321",
				QuoteID:            "quoterbpDCD-QUOTE17732116652401234",
				State:              "settled",
				ProductID:          "BTC-USDT-260327-77000-C",
				BaseCurrency:       currency.BTC,
				QuoteCurrency:      currency.USDT,
				Underlying:         "BTC-USD",
				StrikePrice:        77000,
				NotionalSize:       1.5,
				NotionalCurrency:   currency.BTC,
				AbsoluteYield:      0.00806038,
				AnnualisedYield:    0.1834,
				YieldSize:          0.01209057,
				YieldCurrency:      currency.BTC,
				SettlementSize:     1.51209057,
				SettlementCurrency: currency.BTC,
				SettlementPrice:    76500,
				ExpiryTime:         types.Time(time.UnixMilli(1774598400000)),
				SettlementTime:     types.Time(time.UnixMilli(1774598400000)),
				RedeemStartTime:    types.Time(time.UnixMilli(1774598400000)),
				RedeemEndTime:      types.Time(time.UnixMilli(1774598400000)),
				CreationTime:       types.Time(time.UnixMilli(1773212400000)),
				UpdateTime:         types.Time(time.UnixMilli(1773212400000)),
			},
		}
		assert.Equal(t, exp, result, "GetDualInvestmentOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetDualInvestmentOrderHistory should return orders")
	}
}
